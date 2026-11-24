package proxy

import (
	"sync"
	"time"
)

// shapeMemoMaxEntries bounds the memo: one entry per conversation that was
// rejected, dropped by expiry or, past this size, by a reset.
const shapeMemoMaxEntries = 1024

// movedSystemInstructionTTL is how long a learned "this size needs the moved
// systemInstruction" stays in force. It is short on purpose: the rejection is
// a backend behaviour that can change, and expiring just costs one rejected
// attempt to learn it again.
const movedSystemInstructionTTL = 15 * time.Minute

// shapeKey names what was learned: a model on a project, for one conversation.
// The session is part of the key because the trigger of the rejection was not
// isolated (it did not reproduce with a large system prompt, with tools, or with
// both), so what is learned is only trusted for the conversation that showed it.
type shapeKey struct {
	model   string
	project string
	session string
}

type shapeEntry struct {
	// minBytes is the smallest translated request body that was rejected with a
	// bare RESOURCE_EXHAUSTED until its systemInstruction was moved into
	// contents[0]. Requests smaller than this are known to pass as they are.
	minBytes int
	until    time.Time
}

// shapeMemo remembers, per model and project, how large a request has to be
// before Antigravity refuses its systemInstruction. Without it every large
// request pays a rejected round trip (1 to 2.5 s measured) before the gateway
// retries with the instruction moved.
//
// The size matters: moving the system prompt into a user turn weakens how the
// model treats it, so a request that passes untouched must keep going out
// untouched. Only a body at least as large as one that already failed is moved
// ahead of time.
type shapeMemo struct {
	mu      sync.Mutex
	entries map[shapeKey]shapeEntry
	ttl     time.Duration
	now     func() time.Time
}

func newShapeMemo(ttl time.Duration, now func() time.Time) *shapeMemo {
	if now == nil {
		now = time.Now
	}
	return &shapeMemo{entries: make(map[shapeKey]shapeEntry), ttl: ttl, now: now}
}

// movedSystemInstructions is the process-wide memo used by ForwardGemini.
var movedSystemInstructions = newShapeMemo(movedSystemInstructionTTL, nil)

// remember records that a body of this size needed the moved shape, keeping the
// smallest such size seen, and (re)starts the expiry.
func (m *shapeMemo) remember(k shapeKey, size int) {
	if size <= 0 || k.session == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.entries) >= shapeMemoMaxEntries {
		for key, e := range m.entries {
			if !e.until.After(m.now()) {
				delete(m.entries, key)
			}
		}
		if len(m.entries) >= shapeMemoMaxEntries {
			m.entries = make(map[shapeKey]shapeEntry)
		}
	}
	if e, ok := m.entries[k]; ok && e.until.After(m.now()) && e.minBytes < size {
		size = e.minBytes
	}
	m.entries[k] = shapeEntry{minBytes: size, until: m.now().Add(m.ttl)}
}

// shouldMove reports whether a body of this size is at least as large as one
// that was already rejected for this key and not yet expired.
func (m *shapeMemo) shouldMove(k shapeKey, size int) bool {
	if k.session == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[k]
	if !ok {
		return false
	}
	if !e.until.After(m.now()) {
		delete(m.entries, k)
		return false
	}
	return size >= e.minBytes
}

// forget drops what was learned for the key, for when the moved shape is
// refused too and the memory is therefore wrong.
func (m *shapeMemo) forget(k shapeKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, k)
}
