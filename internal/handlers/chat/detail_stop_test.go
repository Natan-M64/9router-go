package chat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"9router/proxy/internal/db"
	"9router/proxy/internal/translator"
)

func TestLastFinishReason(t *testing.T) {
	stream := `data: {"choices":[{"delta":{"content":"a"},"finish_reason":null}]}` + "\n\n" +
		`data: {"choices":[{"delta":{},"finish_reason":"length"}]}` + "\n\n" +
		`data: [DONE]`
	cases := map[string]string{
		stream: "length",
		`{"choices":[{"message":{},"finish_reason": "stop"}]}`:  "stop",
		`data: {"choices":[{"delta":{},"finish_reason":null}]}`: "",
		"": "",
	}
	cases[`{"finish_reason":"tool_calls"} {"finish_reason":"length"}`] = "length"
	for in, want := range cases {
		if got := lastFinishReason(in); got != want {
			t.Errorf("lastFinishReason(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRequestedMaxTokens(t *testing.T) {
	a, b, zero := 100, 200, 0
	if got := requestedMaxTokens(&a, &b); got != 100 {
		t.Errorf("max_tokens must win, got %d", got)
	}
	if got := requestedMaxTokens(nil, &b); got != 200 {
		t.Errorf("fallback to max_completion_tokens, got %d", got)
	}
	if got := requestedMaxTokens(&zero, nil); got != 0 {
		t.Errorf("zero means unset, got %d", got)
	}
}

// The stored response text is cut at a fixed length, so a stop reason at the
// end of a long answer is only recoverable from the full buffer.
func TestLogUsageRecordsStopReasonAndRequestedLimit(t *testing.T) {
	database, cleanup := setupChatTestDB(t)
	defer cleanup()
	h := NewChatHandler(db.NewRepo(database))

	metrics := &streamMetrics{}
	_, _ = metrics.ResponseBuf.Write([]byte(strings.Repeat("x", 30000)))
	_, _ = metrics.ResponseBuf.Write([]byte(`data: {"choices":[{"delta":{},"finish_reason":"length"}]}`))

	info := &UsageLogInfo{Provider: "p", Model: "m", ConnectionID: "conn-stop-reason"}
	h.LogUsage(context.Background(), info, &translator.OpenAIUsage{PromptTokens: 10, CompletionTokens: 5}, 50,
		[]byte(`{"max_tokens":777,"messages":[{"role":"user","content":"hi"}]}`), metrics)

	var data string
	if err := database.QueryRow(`SELECT data FROM requestDetails WHERE connectionId = 'conn-stop-reason' ORDER BY rowid DESC LIMIT 1`).Scan(&data); err != nil {
		t.Fatalf("query requestDetails: %v", err)
	}
	var got struct {
		FinishReason string `json:"finishReason"`
		MaxTokens    int    `json:"maxTokens"`
	}
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.FinishReason != "length" || got.MaxTokens != 777 {
		t.Errorf("record carries finishReason=%q maxTokens=%d, want length and 777", got.FinishReason, got.MaxTokens)
	}
}
