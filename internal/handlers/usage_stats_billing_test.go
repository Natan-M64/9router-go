package handlers

import (
	json "encoding/json/v2"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"9router/proxy/internal/db"
)

// TestHandleUsageStats_SplitsMeteredFromIncludedCost pins the cost basis split:
// `cost` is a list-price estimate for every provider, so a subscription or free
// quota (codex, antigravity) inflates the total as if it had been invoiced. The
// response must report the metered share apart from the included share, and an
// unclassifiable custom endpoint must not be folded into either.
func TestHandleUsageStats_SplitsMeteredFromIncludedCost(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	repo := db.NewRepo(database)

	dailyJSON := `{
		"byProvider": {
			"deepseek":                 {"requests": 5, "promptTokens": 1000, "completionTokens": 100, "cost": 0.07},
			"codex":                    {"requests": 2, "promptTokens": 9000, "completionTokens": 200, "cost": 1.20},
			"openai-compatible-local":  {"requests": 1, "promptTokens": 300, "completionTokens": 30, "cost": 0.03}
		}
	}`
	dateKey := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if _, err := database.Exec(`INSERT INTO usageDaily (dateKey, data) VALUES (?, ?)`, dateKey, dailyJSON); err != nil {
		t.Fatalf("seed usageDaily: %v", err)
	}

	handler := HandleUsageStats(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/usage/stats?period=7d", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp UsageStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	wantBilling := map[string]string{
		"deepseek":                "metered",
		"codex":                   "included",
		"openai-compatible-local": "unknown",
	}
	for provider, want := range wantBilling {
		item, ok := resp.ByProvider[provider]
		if !ok {
			t.Fatalf("expected byProvider[%q]: %v", provider, resp.ByProvider)
		}
		if item.Billing != want {
			t.Errorf("byProvider[%q].Billing = %q, want %q", provider, item.Billing, want)
		}
	}

	assertClose := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	assertClose("TotalCostMetered", resp.TotalCostMetered, 0.07)
	assertClose("TotalCostIncluded", resp.TotalCostIncluded, 1.20)
	assertClose("TotalCostUnclassified", resp.TotalCostUnclassified, 0.03)
	assertClose("TotalCost", resp.TotalCost, 1.30)
}
