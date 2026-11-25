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
	for _, m := range req.Messages {
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
	}

	return map[string]any{
		"messages":  len(req.Messages),
		"bytes":     roleBytes,
		"toolBytes": capToolBytes(toolBytes),
		"prefix": map[string]string{
			"system": shortHash(system.Sum(nil)),
			"tools":  shortHash(hashBytes(req.Tools)),
		},
	}
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
