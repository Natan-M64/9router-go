package proxy

import (
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

const (
	packedToolChunk = `data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"deepseek-v4.1-flash-free","choices":[{"index":0,"finish_reason":null,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"execute","arguments":"{}"}}]}}]}` + "\n"
	packedTermChunk = `data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"deepseek-v4.1-flash-free","choices":[{"index":0,"finish_reason":"tool_calls","delta":{}}]}` + "\n"
	packedDoneFrame = "data: [DONE]\n\n"
)

// packedWant is the canonical output for a tool-call chunk followed by its
// terminal chunk, then the sentinel: one blank line after each data line and
// exactly one [DONE]. The packed and already-separated inputs must both
// produce it, so the fix is also idempotent.
const packedWant = packedToolChunk + "\n" + packedTermChunk + "\n" + packedDoneFrame

// Regression for the ocg/deepseek-v4.1-flash-free tool-call streams seen in
// Kilo (2026-11-26, opencode.log L60457/L65749/L76477): the provider emitted
// the tool_calls delta chunk and the terminal finish_reason:"tool_calls" chunk
// back-to-back with only a single '\n' between them. Per SSE, two `data:` lines
// without a blank line are ONE event whose payload is both lines joined by
// '\n', so the client tries to JSON-parse two objects and fails with
// "Unable to parse JSON string". SSECopy used to forward that pair verbatim;
// it must split every consecutive `data:` line into its own event.
func TestSSECopy_SplitsPackedDataEvents(t *testing.T) {
	plainReader := func(s string) io.Reader { return strings.NewReader(s) }
	byteReader := func(s string) io.Reader { return iotest.OneByteReader(strings.NewReader(s)) }

	crlfTool := strings.TrimSuffix(packedToolChunk, "\n") + "\r\n"
	crlfTerm := strings.TrimSuffix(packedTermChunk, "\n") + "\r\n"

	// A provider that spreads one JSON object across two `data:` lines of the
	// same event must keep its single event: only the first line is incomplete
	// JSON, so no separator may be inserted.
	multiLine := `data: {"choices":` + "\n" +
		`data: [{"delta":{"content":"hi"},"finish_reason":"stop"}]}` + "\n\n"

	exact := func(want string) func(*testing.T, string) {
		return func(t *testing.T, out string) {
			if got := strings.Count(out, "data: [DONE]"); got != 1 {
				t.Fatalf("expected exactly one sentinel, got %d in %q", got, out)
			}
			if out != want {
				t.Fatalf("unexpected stream\n got: %q\nwant: %q", out, want)
			}
		}
	}

	tests := []struct {
		name   string
		input  string
		reader func(string) io.Reader
		check  func(*testing.T, string)
	}{
		{
			name:   "packed_single_read",
			input:  packedToolChunk + packedTermChunk + "\n" + packedDoneFrame,
			reader: plainReader,
			check:  exact(packedWant),
		},
		{
			name:   "packed_byte_at_a_time",
			input:  packedToolChunk + packedTermChunk + "\n" + packedDoneFrame,
			reader: byteReader,
			check:  exact(packedWant),
		},
		{
			name:   "already_separated_unchanged",
			input:  packedToolChunk + "\n" + packedTermChunk + "\n" + packedDoneFrame,
			reader: plainReader,
			check:  exact(packedWant),
		},
		{
			name:   "packed_eof_without_done",
			input:  packedToolChunk + packedTermChunk,
			reader: plainReader,
			check:  exact(packedWant),
		},
		{
			name:   "packed_crlf",
			input:  crlfTool + crlfTerm + "\r\n" + packedDoneFrame,
			reader: plainReader,
			check: func(t *testing.T, out string) {
				if got := strings.Count(out, "data: [DONE]"); got != 1 {
					t.Fatalf("expected exactly one sentinel, got %d in %q", got, out)
				}
				if strings.Contains(out, "}\r\ndata: ") {
					t.Fatalf("CRLF data lines must be separated by a blank line; got %q", out)
				}
				if !strings.Contains(out, crlfTerm) {
					t.Fatalf("CRLF terminal chunk must survive; got %q", out)
				}
			},
		},
		{
			name:   "comment_between_packed",
			input:  packedToolChunk + ": keepalive\n" + packedTermChunk + "\n" + packedDoneFrame,
			reader: plainReader,
			check: func(t *testing.T, out string) {
				if got := strings.Count(out, "data: [DONE]"); got != 1 {
					t.Fatalf("expected exactly one sentinel, got %d in %q", got, out)
				}
				// A comment does not terminate an event, so the following data
				// line is still split from the previous one.
				if !strings.Contains(out, ": keepalive\n\ndata: ") {
					t.Fatalf("data line after a comment must be split; got %q", out)
				}
			},
		},
		{
			name:   "multiline_json_event_not_split",
			input:  multiLine,
			reader: plainReader,
			check:  exact(multiLine + packedDoneFrame),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &mockResponseWriter{}
			if err := SSECopy(rec, tt.reader(tt.input), rec, nil); err != nil {
				t.Fatalf("SSECopy failed: %v", err)
			}
			tt.check(t, rec.String())
		})
	}
}
