package chat

import (
	"sync"
	"time"
)

const (
	// cacheBreakMinPrompt is the prompt size below which a low hit rate is not
	// worth flagging: small requests never share a prefix with the conversation.
	cacheBreakMinPrompt = 20000
	// cacheBreakHitFloor is the hit rate under which a large request is flagged.
	cacheBreakHitFloor = 0.80
	// cacheBreakMemoMax and cacheBreakMemoTTL bound the per-conversation memory.
	cacheBreakMemoMax = 512
	cacheBreakMemoTTL = 30 * time.Minute
)

// prefixFingerprint is what the previous request of a conversation sent as its
// fixed part: the system messages and the tools definition.
type prefixFingerprint struct {
	system string
	tools  string
	at     time.Time
}

// prefixMemo remembers the last fingerprint per conversation so a request can
// be compared with the one before it. A conversation is the session id when the
// client sent one, else the connection and model.
type prefixMemo struct {
	mu      sync.Mutex
	entries map[string]prefixFingerprint
	now     func() time.Time
}

func newPrefixMemo(now func() time.Time) *prefixMemo {
	if now == nil {
		now = time.Now
	}
	return &prefixMemo{entries: make(map[string]prefixFingerprint), now: now}
}

// swap stores the fingerprint for key and returns the previous one, if any and
// not expired.
func (m *prefixMemo) swap(key string, fp prefixFingerprint) (prefixFingerprint, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	fp.at = now
	prev, ok := m.entries[key]
	if ok && now.Sub(prev.at) > cacheBreakMemoTTL {
		ok = false
	}
	if !ok && len(m.entries) >= cacheBreakMemoMax {
		for k, v := range m.entries {
			if now.Sub(v.at) > cacheBreakMemoTTL {
				delete(m.entries, k)
			}
		}
		if len(m.entries) >= cacheBreakMemoMax {
			m.entries = make(map[string]prefixFingerprint)
		}
	}
	m.entries[key] = fp
	return prev, ok
}

// processPrefixMemo is the shared memory the chat handler uses.
var processPrefixMemo = newPrefixMemo(nil)

// cacheBreakReport explains a low cache hit on a large request, or returns nil
// when there is nothing to report. It names what moved in the fixed prefix since
// the previous request of the conversation. When nothing moved, the cut is in
// the history: a prune or a compaction rewrote it.
func cacheBreakReport(prompt, cached int, cur prefixFingerprint, prev prefixFingerprint, hasPrev bool) map[string]any {
	if prompt < cacheBreakMinPrompt {
		return nil
	}
	hit := float64(cached) / float64(prompt)
	if hit >= cacheBreakHitFloor {
		return nil
	}
	report := map[string]any{"hitPct": int(hit * 100)}
	switch {
	case !hasPrev:
		report["cause"] = "first-seen"
	case cur.system != prev.system:
		report["cause"] = "system-changed"
	case cur.tools != prev.tools:
		report["cause"] = "tools-changed"
	default:
		report["cause"] = "history-rewritten"
	}
	return report
}
