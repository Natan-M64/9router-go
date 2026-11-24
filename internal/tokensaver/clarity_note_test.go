package tokensaver

import (
	json "encoding/json/v2"
	"strings"
	"testing"
)

// Contract for the bypass turn: the clarity note touches only the tail user
// message, so the system prompt (with the caveman prompt) is unchanged. The
// fallback wiring that calls these helpers is not covered here.
func TestClarityNoteLeavesSystemPromptIntact(t *testing.T) {
	base := []byte(`{"model":"m","messages":[` +
		`{"role":"system","content":"client system"},` +
		`{"role":"user","content":"first question"},` +
		`{"role":"assistant","content":"first answer"},` +
		`{"role":"user","content":"now explain the rm -rf risk"}]}`)

	prompt := GetCavemanPromptWithLang("full", "en")
	withCaveman, did := InjectSystemPrompt(base, prompt)
	if !did {
		t.Fatal("expected caveman prompt to be injected into system")
	}
	// Same request, bypass turn: system must still carry the caveman prompt.
	withNote, did := AppendTailUserNote(withCaveman, CavemanClarityNote)
	if !did {
		t.Fatal("expected clarity note on the tail user message")
	}

	sysBefore := firstMessageContent(t, withCaveman)
	sysAfter := firstMessageContent(t, withNote)
	if sysBefore != sysAfter {
		t.Fatalf("system prompt changed on bypass turn:\nbefore=%q\nafter=%q", sysBefore, sysAfter)
	}
	if !strings.Contains(sysAfter, prompt) {
		t.Fatal("caveman prompt missing from system on bypass turn")
	}
}

func TestAppendTailUserNoteTargetsLastUserOnly(t *testing.T) {
	body := []byte(`{"messages":[` +
		`{"role":"user","content":"a"},` +
		`{"role":"assistant","content":"b"},` +
		`{"role":"user","content":"c"}]}`)

	out, did := AppendTailUserNote(body, "NOTE")
	if !did {
		t.Fatal("expected note to be appended")
	}
	msgs := messagesOf(t, out)
	if got := msgs[0]["content"]; got != "a" {
		t.Errorf("first user message changed: %v", got)
	}
	if got := msgs[1]["content"]; got != "b" {
		t.Errorf("assistant message changed: %v", got)
	}
	last, _ := msgs[2]["content"].(string)
	if !strings.HasPrefix(last, "c") || !strings.HasSuffix(last, "NOTE") {
		t.Errorf("last user message = %q, want prefix c and suffix NOTE", last)
	}
}

func TestAppendTailUserNoteBlockContent(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)

	out, did := AppendTailUserNote(body, "NOTE")
	if !did {
		t.Fatal("expected note to be appended to block content")
	}
	blocks, ok := messagesOf(t, out)[0]["content"].([]any)
	if !ok || len(blocks) != 2 {
		t.Fatalf("content blocks = %v, want 2 blocks", messagesOf(t, out)[0]["content"])
	}
	if blk, _ := blocks[0].(map[string]any); blk["text"] != "hi" {
		t.Errorf("original block altered: %v", blk)
	}
	if blk, _ := blocks[1].(map[string]any); blk["text"] != "NOTE" {
		t.Errorf("appended block = %v, want text NOTE", blk)
	}
}

func TestAppendTailUserNoteNoUserMessageIsNoop(t *testing.T) {
	body := []byte(`{"messages":[{"role":"system","content":"s"},{"role":"assistant","content":"a"}]}`)
	out, did := AppendTailUserNote(body, "NOTE")
	if did {
		t.Fatal("expected no-op without a user message")
	}
	if string(out) != string(body) {
		t.Fatal("body must be returned unchanged")
	}
}

func TestAppendTailUserNoteIsIdempotent(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"x"}]}`)
	once, _ := AppendTailUserNote(body, "NOTE")
	twice, did := AppendTailUserNote(once, "NOTE")
	if did || string(once) != string(twice) {
		t.Fatal("appending the same note twice must not duplicate it")
	}
}

func firstMessageContent(t *testing.T, body []byte) string {
	t.Helper()
	msgs := messagesOf(t, body)
	s, _ := msgs[0]["content"].(string)
	return s
}

func messagesOf(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	arr, _ := req["messages"].([]any)
	out := make([]map[string]any, 0, len(arr))
	for _, m := range arr {
		mm, _ := m.(map[string]any)
		out = append(out, mm)
	}
	return out
}

// In a tool loop the last message is a tool or assistant turn. The nearest user
// message is mid-history, and editing it would rewrite the cached prefix, so the
// note must be skipped instead of moved there.
func TestAppendTailUserNoteSkipsWhenTailIsNotUser(t *testing.T) {
	body := []byte(`{"messages":[` +
		`{"role":"user","content":"do it"},` +
		`{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"}}]},` +
		`{"role":"tool","tool_call_id":"c1","content":"ok"}]}`)
	out, did := AppendTailUserNote(body, CavemanClarityNote)
	if did {
		t.Fatalf("note was appended although the tail is a tool message: %s", out)
	}
	if string(out) != string(body) {
		t.Fatalf("body changed: %s", out)
	}
	var req struct {
		Messages []struct {
			Content any `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(out, &req); err != nil {
		t.Fatal(err)
	}
	if s, _ := req.Messages[0].Content.(string); strings.Contains(s, CavemanClarityNote) {
		t.Fatal("mid-history user message was edited")
	}
}
