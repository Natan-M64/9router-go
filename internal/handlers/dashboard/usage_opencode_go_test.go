package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func withOpenCodeGoUsageServer(t *testing.T, status int, body string) *string {
	t.Helper()
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	prev := openCodeGoUsageURL
	openCodeGoUsageURL = srv.URL + "/zen/go/v1/usage"
	t.Cleanup(func() { openCodeGoUsageURL = prev })
	return &auth
}

func TestFetchOpenCodeGoUsage_MapsThreeWindows(t *testing.T) {
	auth := withOpenCodeGoUsageServer(t, http.StatusOK,
		`{"usage":{"rolling":{"status":"ok","percent":25,"resetsAt":"2026-11-25T05:00:00Z"},
		          "weekly":{"status":"ok","percent":60,"resetsAt":"2026-11-30T00:00:00Z"},
		          "monthly":{"status":"ok","percent":10,"resetsAt":"2026-12-01T00:00:00Z"}}}`)

	res := fetchOpenCodeGoUsage(context.Background(), "sk-go")
	if res.plan != "OpenCode Go" {
		t.Errorf("plan = %q, want OpenCode Go", res.plan)
	}
	if len(res.quotas) != 3 {
		t.Fatalf("expected 3 quotas, got %d (%+v)", len(res.quotas), res.quotas)
	}
	if *auth != "Bearer sk-go" {
		t.Errorf("expected Bearer auth, got %q", *auth)
	}
	weekly, _ := res.quotas["Weekly"].(map[string]any)
	if weekly["used"] != 60.0 || weekly["remaining"] != 40.0 {
		t.Errorf("weekly mis-mapped: %+v", weekly)
	}
}

func TestFetchOpenCodeGoUsage_ErrorsCarryThePlanName(t *testing.T) {
	withOpenCodeGoUsageServer(t, http.StatusUnauthorized, `{"error":{"type":"AuthError"}}`)
	res := fetchOpenCodeGoUsage(context.Background(), "bad")
	if res.message != "OpenCode Go authentication failed. Check the API key." {
		t.Errorf("message = %q", res.message)
	}
	if len(res.quotas) != 0 {
		t.Errorf("an auth failure must not show quota rows: %+v", res.quotas)
	}
}

func TestFetchOpenCodeGoUsage_NoKeyNeverCallsOut(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	t.Cleanup(srv.Close)
	prev := openCodeGoUsageURL
	openCodeGoUsageURL = srv.URL
	t.Cleanup(func() { openCodeGoUsageURL = prev })

	res := fetchOpenCodeGoUsage(context.Background(), "   ")
	if hit {
		t.Error("an empty key must not reach the network")
	}
	if res.message != "OpenCode Go API key not available. Add a key to view usage." {
		t.Errorf("message = %q", res.message)
	}
}

func TestFetchOpenCodeGoUsage_SendsTheCLIIdentityAndAStableSession(t *testing.T) {
	var ua, client, session string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		client = r.Header.Get("x-opencode-client")
		session = r.Header.Get("x-opencode-session")
		_, _ = w.Write([]byte(`{"usage":{"rolling":{"percent":1,"resetsAt":"2026-11-25T05:00:00Z"},"weekly":{"percent":1,"resetsAt":"2026-11-30T00:00:00Z"},"monthly":{"percent":1,"resetsAt":"2026-12-01T00:00:00Z"}}}`))
	}))
	t.Cleanup(srv.Close)
	prev := openCodeGoUsageURL
	openCodeGoUsageURL = srv.URL
	t.Cleanup(func() { openCodeGoUsageURL = prev })

	fetchOpenCodeGoUsage(context.Background(), "sk-go")
	if ua != openCodeCLIUserAgent || client != "cli" {
		t.Errorf("identity = %q / %q, want the CLI identity", ua, client)
	}
	if session == "" {
		t.Fatal("no x-opencode-session header")
	}
	var again string
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		again = r.Header.Get("x-opencode-session")
		_, _ = w.Write([]byte(`{}`))
	})
	fetchOpenCodeGoUsage(context.Background(), "sk-go")
	if again != session {
		t.Errorf("session changed between reads of one key: %q then %q", session, again)
	}
}
