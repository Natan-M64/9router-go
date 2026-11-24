package translator

import (
	json "encoding/json/v2"
	"testing"
)

// A plain-string system prompt cannot carry cache_control, so the head anchor
// silently never lands on it. The anchor must promote the string to a text
// block so the system prefix is cacheable.
func TestAnchorClaudeCachePromotesStringSystem(t *testing.T) {
	body := []byte(`{"model":"claude","system":"you are helpful","messages":[{"role":"user","content":"hi"}]}`)

	out := AnchorClaudeCacheWithTTL(body, "")

	var req map[string]any
	if err := json.Unmarshal(out, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	blocks, ok := req["system"].([]any)
	if !ok || len(blocks) != 1 {
		t.Fatalf("system = %#v, want one text block", req["system"])
	}
	blk, _ := blocks[0].(map[string]any)
	if blk["text"] != "you are helpful" {
		t.Errorf("system text = %v, want original string", blk["text"])
	}
	cc, _ := blk["cache_control"].(map[string]any)
	if cc == nil || cc["type"] != "ephemeral" {
		t.Errorf("system block cache_control = %v, want ephemeral anchor", blk["cache_control"])
	}
}

// Applying the anchor twice must not change the body: the translated path and
// the native path may both anchor the same request.
func TestAnchorClaudeCacheIsIdempotent(t *testing.T) {
	body := []byte(`{"model":"claude","system":"sys","tools":[{"name":"a"},{"name":"b"}],` +
		`"messages":[{"role":"user","content":"hi"}]}`)

	once := AnchorClaudeCacheWithTTL(body, ClaudeCacheTTL1h)
	twice := AnchorClaudeCacheWithTTL(once, ClaudeCacheTTL1h)
	if string(once) != string(twice) {
		t.Fatalf("second anchor changed the body:\nonce=%s\ntwice=%s", once, twice)
	}
}
