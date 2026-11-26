package chat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// compositionToolCap bounds how many tool names get their own row. Tool names
// come from the client, so the record keeps the largest twelve and folds the
// rest into "other" to keep the stored shape bounded.
const compositionToolCap = 12

// compositionMessageCap bounds the per-message prefix chain. A prompt-cache
// cut happens at the first message that moved, and history rewrites land in the
// early turns; 32 eight-hex entries is 256 bytes in the record and in the
// per-conversation memo, so an operator can see *where* the prefix broke
// without this growing with the conversation.
const compositionMessageCap = 32

// requestComposition describes what the client sent, by role and by tool,
// without storing any of its text. Bytes are the JSON size of each message's
// content. The prefix fingerprints hash the system messages and the tools
// definition: when two turns of one conversation carry different fingerprints,
// the prompt cache can only reuse what came before the first difference, and
// the record shows which part moved. Returns nil when the body is not a chat
// request.
func requestComposition(body []byte) map[string]any {
	var req struct {
		Messages []map[string]json.RawMessage `json:"messages"`
		Tools    json.RawMessage              `json:"tools"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Messages == nil {
		return nil
	}

	// Tool results carry only a call id, so map each id back to its tool name.
	names := map[string]string{}
	for _, m := range req.Messages {
		var calls []struct {
			ID       string `json:"id"`
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		if json.Unmarshal(m["tool_calls"], &calls) == nil {
			for _, c := range calls {
				names[c.ID] = c.Function.Name
			}
		}
	}

	roleBytes := map[string]int{}
	toolBytes := map[string]int{}
	system := sha256.New()
	chain := make([]byte, 0, compositionMessageCap*8)
	for i, m := range req.Messages {
		var role string
		_ = json.Unmarshal(m["role"], &role)
		size := len(m["content"])
		switch role {
		case "system", "developer":
			roleBytes["system"] += size
			system.Write(m["content"])
		case "tool":
			roleBytes["tool"] += size
			var id string
			_ = json.Unmarshal(m["tool_call_id"], &id)
			name := names[id]
			if name == "" {
				name = "unknown"
			}
			toolBytes[name] += size
		case "user", "assistant":
			roleBytes[role] += size
		default:
			roleBytes["other"] += size
		}
		if i < compositionMessageCap {
			chain = append(chain, messageHash(m)...)
		}
	}

	return map[string]any{
		"messages":  len(req.Messages),
		"bytes":     roleBytes,
		"toolBytes": capToolBytes(toolBytes),
		"prefix": map[string]string{
			"system": shortHash(system.Sum(nil)),
			"tools":  shortHash(hashBytes(req.Tools)),
			// messages is the ordered chain the cache-break report diffs to
			// name the first message that moved. Hashes only, never the text.
			"messages": string(chain),
		},
	}
}

// messageHash is one message's 32-bit FNV-1a over its keys and raw values, in
// sorted key order so map iteration cannot reorder the input. content and
// tool_calls dominate by size; role and tool_call_id catch the pointer moves
// that matter when a message is replaced rather than appended. Written by hand
// to avoid one hasher allocation per message on the request path.
func messageHash(m map[string]json.RawMessage) []byte {
	const prime32 = 16777619
	h := uint32(2166136261)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for i := 0; i < len(k); i++ {
			h ^= uint32(k[i])
			h *= prime32
		}
		h ^= 0xff
		h *= prime32
		for _, b := range m[k] {
			h ^= uint32(b)
			h *= prime32
		}
	}
	const hexDigits = "0123456789abcdef"
	out := make([]byte, 8)
	for i := 0; i < 8; i++ {
		out[i] = hexDigits[(h>>uint(28-4*i))&0xf]
	}
	return out
}

// capToolBytes keeps the largest compositionToolCap tools and sums the rest.
func capToolBytes(m map[string]int) map[string]int {
	type kv struct {
		name string
		n    int
	}
	all := make([]kv, 0, len(m))
	for k, v := range m {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].n != all[j].n {
			return all[i].n > all[j].n
		}
		return all[i].name < all[j].name
	})
	out := map[string]int{}
	for i, e := range all {
		if i < compositionToolCap {
			out[e.name] = e.n
		} else {
			out["other"] += e.n
		}
	}
	return out
}

func hashBytes(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}

func shortHash(b []byte) string {
	return hex.EncodeToString(b)[:12]
}
