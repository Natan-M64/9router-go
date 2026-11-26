package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"9router/proxy/internal/db"
	"9router/proxy/internal/translator"
)

func compositionBody(system, toolsJSON string, bashOut, readOut string) []byte {
	return []byte(fmt.Sprintf(`{"tools":%s,"messages":[
		{"role":"system","content":%q},
		{"role":"user","content":"hi"},
		{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"bash","arguments":"{}"}},{"id":"c2","type":"function","function":{"name":"read","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"c1","content":%q},
		{"role":"tool","tool_call_id":"c2","content":%q}
	]}`, toolsJSON, system, bashOut, readOut))
}

func TestRequestCompositionCountsByRoleAndTool(t *testing.T) {
	bash := strings.Repeat("b", 400)
	read := strings.Repeat("r", 100)
	comp := requestComposition(compositionBody("sys", `[{"type":"function"}]`, bash, read))
	if comp == nil {
		t.Fatal("composition is nil for a chat body")
	}
	raw, _ := json.Marshal(comp)
	var got struct {
		Messages  int            `json:"messages"`
		Bytes     map[string]int `json:"bytes"`
		ToolBytes map[string]int `json:"toolBytes"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Messages != 5 {
		t.Errorf("messages = %d, want 5", got.Messages)
	}
	if got.ToolBytes["bash"] <= got.ToolBytes["read"] {
		t.Errorf("bash result must be counted under bash (%d) and bigger than read (%d)", got.ToolBytes["bash"], got.ToolBytes["read"])
	}
	if got.Bytes["tool"] != got.ToolBytes["bash"]+got.ToolBytes["read"] {
		t.Errorf("tool bytes do not add up: role=%d tools=%v", got.Bytes["tool"], got.ToolBytes)
	}
}

func TestRequestCompositionNeverStoresText(t *testing.T) {
	secret := "SECRET-PROMPT-TEXT-123"
	comp := requestComposition(compositionBody(secret, `[]`, secret, secret))
	raw, _ := json.Marshal(comp)
	if strings.Contains(string(raw), secret) {
		t.Fatalf("composition leaked prompt text: %s", raw)
	}
}

// The fingerprint must move only when the system messages or the tools move:
// a change to the conversation itself leaves the system hash alone, so the
// record can tell a changed prefix from a changed history.
func TestRequestCompositionPrefixFingerprints(t *testing.T) {
	a := requestComposition(compositionBody("sys-a", `[{"n":1}]`, "x", "y"))
	b := requestComposition(compositionBody("sys-a", `[{"n":1}]`, "different", "history"))
	c := requestComposition(compositionBody("sys-b", `[{"n":1}]`, "x", "y"))
	d := requestComposition(compositionBody("sys-a", `[{"n":2}]`, "x", "y"))
	pa, pb, pc, pd := a["prefix"].(map[string]string), b["prefix"].(map[string]string), c["prefix"].(map[string]string), d["prefix"].(map[string]string)
	if pa["system"] != pb["system"] || pa["tools"] != pb["tools"] {
		t.Error("a history-only change must not move the prefix fingerprints")
	}
	if pa["system"] == pc["system"] {
		t.Error("a system change must move the system fingerprint")
	}
	if pa["tools"] == pd["tools"] {
		t.Error("a tools change must move the tools fingerprint")
	}
}

func TestRequestCompositionCapsToolRows(t *testing.T) {
	var msgs []string
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("t%d", i)
		msgs = append(msgs,
			fmt.Sprintf(`{"role":"assistant","content":"","tool_calls":[{"id":%q,"type":"function","function":{"name":"tool%02d"}}]}`, id, i),
			fmt.Sprintf(`{"role":"tool","tool_call_id":%q,"content":%q}`, id, strings.Repeat("z", 10+i)))
	}
	body := []byte(`{"messages":[` + strings.Join(msgs, ",") + `]}`)
	comp := requestComposition(body)
	tb := comp["toolBytes"].(map[string]int)
	if len(tb) != compositionToolCap+1 {
		t.Fatalf("tool rows = %d, want %d (cap plus other)", len(tb), compositionToolCap+1)
	}
	if tb["other"] == 0 {
		t.Error("the tools past the cap must be folded into other")
	}
}

func TestRequestCompositionRejectsNonChatBodies(t *testing.T) {
	for _, body := range []string{`not json`, `{"prompt":"x"}`, `[]`} {
		if got := requestComposition([]byte(body)); got != nil {
			t.Errorf("requestComposition(%q) = %v, want nil", body, got)
		}
	}
}

// The composition must reach the stored request record, not just the helper.
func TestLogUsageStoresCompositionInTheRecord(t *testing.T) {
	database, cleanup := setupChatTestDB(t)
	defer cleanup()
	h := NewChatHandler(db.NewRepo(database))
	info := &UsageLogInfo{Provider: "p", Model: "m", ConnectionID: "conn-composition"}
	h.LogUsage(context.Background(), info, &translator.OpenAIUsage{PromptTokens: 10, CompletionTokens: 5}, 50,
		compositionBody("sys", `[{"type":"function"}]`, "bash output", "read output"), nil)

	var data string
	if err := database.QueryRow(`SELECT data FROM requestDetails WHERE connectionId = 'conn-composition' ORDER BY rowid DESC LIMIT 1`).Scan(&data); err != nil {
		t.Fatalf("query requestDetails: %v", err)
	}
	var got struct {
		Composition struct {
			Messages  int            `json:"messages"`
			ToolBytes map[string]int `json:"toolBytes"`
		} `json:"composition"`
	}
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatal(err)
	}
	if got.Composition.Messages != 5 || got.Composition.ToolBytes["bash"] == 0 {
		t.Errorf("record composition = %+v, want 5 messages and bash bytes", got.Composition)
	}
}

// The per-message chain must move at the first message that changed and stay
// put for a message that did not, because it is what addresses a cache break.
func TestRequestCompositionMessageChainMovesAtTheChangedMessage(t *testing.T) {
	bash := strings.Repeat("b", 400)
	read := strings.Repeat("r", 100)
	a := requestComposition(compositionBody("sys", `[]`, bash, read))["prefix"].(map[string]string)["messages"]
	b := requestComposition(compositionBody("sys", `[]`, bash+"x", read))["prefix"].(map[string]string)["messages"]
	if a == "" || b == "" {
		t.Fatal("chain is empty")
	}
	idx, ok := firstMovedMessage(a, b)
	if !ok {
		t.Fatal("a changed tool result must move the chain")
	}
	// system, user, assistant, tool(c1) are unchanged; the changed message is
	// the fourth entry (index 3).
	if idx != 3 {
		t.Errorf("first moved index = %d, want 3", idx)
	}
}

// The chain is bounded: a conversation past compositionMessageCap must not grow
// the record or the memo with its length.
func TestRequestCompositionMessageChainIsCapped(t *testing.T) {
	var msgs []string
	for i := 0; i < compositionMessageCap+10; i++ {
		msgs = append(msgs, fmt.Sprintf(`{"role":"user","content":"m%d"}`, i))
	}
	chain := requestComposition([]byte(`{"messages":[` + strings.Join(msgs, ",") + `]}`))["prefix"].(map[string]string)["messages"]
	if got, want := len(chain), compositionMessageCap*messageHashLen; got != want {
		t.Errorf("chain length = %d, want %d", got, want)
	}
}
