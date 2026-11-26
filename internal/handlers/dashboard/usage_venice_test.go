package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"9router/proxy/internal/venicebalance"
)

func TestFetchVeniceUsage_AdminKeyReadsBilling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer admin-key" {
			t.Errorf("Authorization = %q, want the admin key", got)
		}
		w.Header().Set(venicebalance.HeaderUSD, "25")
		_, _ = w.Write([]byte(`{
			"canConsume": true,
			"consumptionCurrency": "DIEM",
			"balances": {"diem": 90.5, "usd": 25, "bundledCredits": null, "earnedCredits": null},
			"diemEpochAllocation": 100
		}`))
	}))
	defer server.Close()
	old := veniceBalanceURL
	veniceBalanceURL = server.URL
	defer func() { veniceBalanceURL = old }()
	venicebalance.ResetForTest(nil)
	t.Cleanup(func() { venicebalance.ResetForTest(nil) })

	res := fetchVeniceUsage(t.Context(), "inference-key", "admin-key", "conn-1")
	if res.message != "" {
		t.Fatalf("unexpected message: %s", res.message)
	}
	diem, ok := res.quotas["diem"].(map[string]any)
	if !ok {
		t.Fatalf("missing diem row: %v", res.quotas)
	}
	if got := usageNum(diem["remainingPercentage"], 0); got != 90.5 {
		t.Errorf("diem remainingPercentage = %v, want 90.5", got)
	}
	if got := usageNum(diem["used"], 0); got != 9.5 {
		t.Errorf("diem used = %v, want 9.5", got)
	}
	reset, _ := diem["resetAt"].(string)
	resetAt, err := time.Parse(time.RFC3339, reset)
	if err != nil {
		t.Fatalf("resetAt %q is not RFC3339: %v", reset, err)
	}
	if resetAt.Hour() != 0 || resetAt.Minute() != 0 || resetAt.Location() != time.UTC {
		t.Errorf("resetAt = %v, want the next UTC midnight", resetAt)
	}
	usd, ok := res.quotas["usd"].(map[string]any)
	if !ok {
		t.Fatalf("missing usd row: %v", res.quotas)
	}
	if usd["isCreditBalance"] != true || usageNum(usd["total"], 0) != 25 {
		t.Errorf("usd row = %v, want a 25 USD credit balance", usd)
	}
	if _, hasBundled := res.quotas["bundled"]; hasBundled {
		t.Error("a null bundled balance must not render a row")
	}
}

func TestFetchVeniceUsage_InferenceKeyGetsActionableMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Admin API key required"}`))
	}))
	defer server.Close()
	old := veniceBalanceURL
	veniceBalanceURL = server.URL
	defer func() { veniceBalanceURL = old }()
	venicebalance.ResetForTest(nil)
	t.Cleanup(func() { venicebalance.ResetForTest(nil) })

	res := fetchVeniceUsage(t.Context(), "", "inference-key-message", "conn-2")
	if !strings.Contains(res.message, "ADMIN key") {
		t.Errorf("message = %q, want it to name the missing ADMIN key", res.message)
	}
}

func TestFetchVeniceUsage_FallsBackToLastInferenceBalance(t *testing.T) {
	venicebalance.ResetForTest(nil)
	t.Cleanup(func() { venicebalance.ResetForTest(nil) })
	usd := 3.25
	diem := 42.0
	venicebalance.RecordValues("conn-3", &usd, &diem, time.Now())

	res := fetchVeniceUsage(t.Context(), "", "", "conn-3")
	if res.message != "" {
		t.Fatalf("unexpected message: %s", res.message)
	}
	usdRow, ok := res.quotas["usd"].(map[string]any)
	if !ok || usageNum(usdRow["total"], 0) != 3.25 {
		t.Errorf("usd row = %v, want 3.25", res.quotas["usd"])
	}
	diemRow, ok := res.quotas["diem"].(map[string]any)
	if !ok || usageNum(diemRow["total"], 0) != 42 {
		t.Errorf("diem row = %v, want 42", res.quotas["diem"])
	}
}

// TestFetchVeniceUsage_InferenceKeyProbeIsMemoized pins the cost guard: the
// connection key is tried against the billing API once, its 401 verdict is
// memoed, and the snapshot answers the next refresh without another call.
func TestFetchVeniceUsage_InferenceKeyProbeIsMemoized(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Admin API key required"}`))
	}))
	defer server.Close()
	old := veniceBalanceURL
	veniceBalanceURL = server.URL
	defer func() { veniceBalanceURL = old }()
	venicebalance.ResetForTest(nil)
	t.Cleanup(func() { venicebalance.ResetForTest(nil) })
	usd := 7.5
	venicebalance.RecordValues("conn-probe", &usd, nil, time.Now())

	first := fetchVeniceUsage(t.Context(), "probe-key-unique", "", "conn-probe")
	if first.message != "" {
		t.Fatalf("first read should fall back to the snapshot, got %q", first.message)
	}
	second := fetchVeniceUsage(t.Context(), "probe-key-unique", "", "conn-probe")
	if second.message != "" {
		t.Fatalf("second read should still use the snapshot, got %q", second.message)
	}
	if calls != 1 {
		t.Errorf("billing API called %d times, want 1 (the rejection must be memoized)", calls)
	}
}

func TestFetchVeniceUsage_NoKeyNoBalanceExplainsWhy(t *testing.T) {
	venicebalance.ResetForTest(nil)
	t.Cleanup(func() { venicebalance.ResetForTest(nil) })

	res := fetchVeniceUsage(t.Context(), "", "", "conn-none")
	if !strings.Contains(res.message, "ADMIN") {
		t.Errorf("message = %q, want the ADMIN-key explanation", res.message)
	}
}
