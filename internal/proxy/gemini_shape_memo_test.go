package proxy

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	json "encoding/json/v2"

	"9router/proxy/internal/handlerutil"
	"9router/proxy/internal/providers"
)

// antigravityFake behaves the way the backend was observed to: a request whose
// systemInstruction is large is refused with a bare RESOURCE_EXHAUSTED, a small
// one passes, and one whose instruction was moved into contents[0] passes.
// With refuseMoved set it does the opposite for requests that lost their
// systemInstruction, so the fallback can be tested.
type antigravityFake struct {
	srv         *httptest.Server
	hits        atomic.Int64
	withSI      atomic.Int64 // requests that arrived carrying a systemInstruction
	refuseMoved atomic.Bool
	acceptLarge atomic.Bool // a large systemInstruction passes as it is
}

func newAntigravityFake(t *testing.T) *antigravityFake {
	t.Helper()
	f := &antigravityFake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		raw, _ := io.ReadAll(r.Body)
		var env struct {
			Request jsontext.Value `json:"request"`
		}
		_ = json.Unmarshal(raw, &env)
		var inner map[string]any
		_ = json.Unmarshal(env.Request, &inner)
		si, _ := inner["systemInstruction"].(map[string]any)
		if si == nil {
			si, _ = inner["system_instruction"].(map[string]any)
		}
		// The official Antigravity identity is re-added to every request, moved or
		// not, as a fixed 427-byte prefix. Only the user's own instruction is what
		// the backend refuses, so only that is counted.
		siLen := 0
		if parts, ok := si["parts"].([]any); ok {
			for _, p := range parts {
				if pm, ok := p.(map[string]any); ok {
					if txt, ok := pm["text"].(string); ok && !strings.Contains(txt, "You are Antigravity") {
						siLen += len(txt)
					}
				}
			}
		}
		hasSI := siLen > 0
		if hasSI {
			f.withSI.Add(1)
		}
		switch {
		case hasSI && siLen > 1000 && !f.acceptLarge.Load():
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":{"code":429,"message":"Resource exhausted","status":"RESOURCE_EXHAUSTED"}}`)
		case !hasSI && f.refuseMoved.Load():
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"code":400,"message":"bad shape","status":"INVALID_ARGUMENT"}}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]},"finishReason":"STOP"}]}`)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func shapeRequest(t *testing.T, f *antigravityFake, systemPrompt string) {
	t.Helper()
	shapeRequestIn(t, f, systemPrompt, "conv-1")
}

// shapeRequestIn sends one request as part of a conversation. An empty session
// sends it with no session identity at all.
func shapeRequestIn(t *testing.T, f *antigravityFake, systemPrompt, session string) {
	t.Helper()
	ctx := context.Background()
	if session != "" {
		ctx = handlerutil.WithSessionID(ctx, session)
	}
	cfg := &providers.ProviderConfig{BaseURL: f.srv.URL}
	body := fmt.Sprintf(`{"model":"gemini-3-flash","messages":[{"role":"system","content":%q},{"role":"user","content":"hello"}]}`, systemPrompt)
	resp, err := ForwardGemini(ctx, f.srv.Client(), cfg, "ya29.fake", body, false, "proj-1", "gemini-3-flash", "")
	if err != nil {
		t.Fatalf("ForwardGemini() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ForwardGemini() status = %d, want 200", resp.StatusCode)
	}
}

func resetShapeMemo(t *testing.T) {
	t.Helper()
	prev := movedSystemInstructions
	movedSystemInstructions = newShapeMemo(movedSystemInstructionTTL, nil)
	t.Cleanup(func() { movedSystemInstructions = prev })
}

// learnedKey returns the one key the memo learned. ForwardGemini resolves the
// model to a tier before it keys the memo, so the test reads the key instead
// of guessing the resolved name.
func learnedKey(t *testing.T) shapeKey {
	t.Helper()
	movedSystemInstructions.mu.Lock()
	defer movedSystemInstructions.mu.Unlock()
	if len(movedSystemInstructions.entries) != 1 {
		t.Fatalf("the memo holds %d entries, want exactly 1", len(movedSystemInstructions.entries))
	}
	for k := range movedSystemInstructions.entries {
		return k
	}
	return shapeKey{}
}

var (
	bigSystemPrompt   = strings.Repeat("Follow the rules of the house. ", 120) // about 3.6 KB
	smallSystemPrompt = "Be brief."
)

// The first large request pays the rejected attempt and teaches the memo; the
// next one goes out already moved and needs a single round trip.
func TestForwardGemini_LearnsThatLargeRequestsNeedTheMovedShape(t *testing.T) {
	resetShapeMemo(t)
	f := newAntigravityFake(t)

	shapeRequest(t, f, bigSystemPrompt)
	if got := f.hits.Load(); got != 2 {
		t.Fatalf("first large request made %d upstream calls, want 2 (rejected + retry)", got)
	}

	before, withSIBefore := f.hits.Load(), f.withSI.Load()
	shapeRequest(t, f, bigSystemPrompt)
	if got := f.hits.Load() - before; got != 1 {
		t.Errorf("second large request made %d upstream calls, want 1 (sent moved straight away)", got)
	}
	if got := f.withSI.Load() - withSIBefore; got != 0 {
		t.Errorf("second large request still carried a systemInstruction (%d calls)", got)
	}
}

// A request that passes as it is must keep its systemInstruction: the memo
// only moves bodies at least as large as one that was refused.
func TestForwardGemini_ShapeMemoLeavesSmallRequestsAlone(t *testing.T) {
	resetShapeMemo(t)
	f := newAntigravityFake(t)

	shapeRequest(t, f, bigSystemPrompt) // teaches the memo
	before, withSIBefore := f.hits.Load(), f.withSI.Load()
	shapeRequest(t, f, smallSystemPrompt)
	if got := f.hits.Load() - before; got != 1 {
		t.Errorf("small request made %d upstream calls, want 1", got)
	}
	if got := f.withSI.Load() - withSIBefore; got != 1 {
		t.Errorf("small request lost its systemInstruction (carried in %d of 1 calls)", got)
	}
}

// If the moved shape is refused, the memo was wrong: the same call still goes
// through in its regular shape, and the memo is dropped so the next request
// starts with the regular shape again.
func TestForwardGemini_RefusedMovedShapeFallsBackAndForgets(t *testing.T) {
	resetShapeMemo(t)
	f := newAntigravityFake(t)

	shapeRequest(t, f, bigSystemPrompt) // learn: large bodies need the moved shape
	key := learnedKey(t)
	// The memo stores the size of the whole translated body, which is larger
	// than the prompt alone, so ask with a size well above it.
	atLeastABody := 2 * len(bigSystemPrompt)
	if !movedSystemInstructions.shouldMove(key, atLeastABody) {
		t.Fatal("the first large request did not teach the memo")
	}

	// The backend changes: the moved shape is refused and the regular one passes.
	f.refuseMoved.Store(true)
	f.acceptLarge.Store(true)

	before, withSIBefore := f.hits.Load(), f.withSI.Load()
	shapeRequest(t, f, bigSystemPrompt)
	if got := f.hits.Load() - before; got != 2 {
		t.Errorf("fallback made %d upstream calls, want 2 (moved shape refused, then regular shape)", got)
	}
	if got := f.withSI.Load() - withSIBefore; got != 1 {
		t.Errorf("the regular-shape retry carried a systemInstruction in %d calls, want 1", got)
	}
	if movedSystemInstructions.shouldMove(key, atLeastABody) {
		t.Error("the memo still moves large bodies after the moved shape was refused")
	}

	// And the next large request starts with the regular shape.
	before = f.hits.Load()
	shapeRequest(t, f, bigSystemPrompt)
	if got := f.hits.Load() - before; got != 1 {
		t.Errorf("request after the fallback made %d upstream calls, want 1", got)
	}
}

func TestShapeMemo_KeepsTheSmallestSizeExpiresAndForgets(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	m := newShapeMemo(10*time.Minute, func() time.Time { return now })
	k := shapeKey{model: "m", project: "p", session: "s"}

	if m.shouldMove(k, 5000) {
		t.Fatal("an unknown key must not move anything")
	}
	m.remember(k, 8000)
	m.remember(k, 3000)
	m.remember(k, 6000) // larger than the smallest seen: must not raise the threshold
	if !m.shouldMove(k, 3000) {
		t.Error("a body as large as the smallest refused one must be moved")
	}
	if m.shouldMove(k, 2999) {
		t.Error("a body smaller than any refused one must stay as it is")
	}

	now = now.Add(11 * time.Minute)
	if m.shouldMove(k, 9000) {
		t.Error("an expired entry must not apply")
	}

	m.remember(k, 4000)
	m.forget(k)
	if m.shouldMove(k, 9000) {
		t.Error("a forgotten entry must not apply")
	}
	m.remember(k, 0) // a zero size is not a measurement
	if m.shouldMove(k, 9000) {
		t.Error("remembering size 0 must not create an entry")
	}
}

// What one conversation learned is not applied to another, and a request with
// no session identity neither learns nor applies anything.
func TestForwardGemini_ShapeMemoIsPerConversation(t *testing.T) {
	resetShapeMemo(t)
	f := newAntigravityFake(t)

	shapeRequestIn(t, f, bigSystemPrompt, "conv-a") // learns for conv-a
	before := f.hits.Load()
	shapeRequestIn(t, f, bigSystemPrompt, "conv-b")
	if got := f.hits.Load() - before; got != 2 {
		t.Errorf("another conversation made %d upstream calls, want 2 (it starts from the regular shape)", got)
	}

	before = f.hits.Load()
	shapeRequestIn(t, f, bigSystemPrompt, "")
	shapeRequestIn(t, f, bigSystemPrompt, "")
	if got := f.hits.Load() - before; got != 4 {
		t.Errorf("two requests without a session made %d upstream calls, want 4 (no learning without a session)", got)
	}
}

func TestShapeMemo_StaysBounded(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	m := newShapeMemo(10*time.Minute, func() time.Time { return now })
	for i := 0; i < shapeMemoMaxEntries+50; i++ {
		m.remember(shapeKey{model: "m", project: "p", session: fmt.Sprintf("s%d", i)}, 1000)
	}
	if got := len(m.entries); got > shapeMemoMaxEntries {
		t.Errorf("memo holds %d entries, want at most %d", got, shapeMemoMaxEntries)
	}
}
