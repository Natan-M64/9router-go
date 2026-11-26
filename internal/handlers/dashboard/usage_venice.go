package dashboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"9router/proxy/internal/venicebalance"
)

// veniceBalanceURL is the account billing endpoint. It returns the spendable
// DIEM epoch balance and the USD / bundled / earned credit balances.
//
// It requires an ADMIN key: an inference key answers 401 "Admin API key
// required". A connection is normally configured with an inference key, so
// that response is not treated as a broken credential — the fetcher falls back
// to the balance Venice reports on inference responses, captured by
// internal/venicebalance.
var veniceBalanceURL = "https://api.venice.ai/api/v1/billing/balance"

// fetchVeniceUsage reads the Venice credit balance.
//
// Two sources, in order:
//   - an ADMIN key stored on the connection (providerSpecificData.adminApiKey)
//     reads the billing API, which is authoritative and includes the DIEM epoch
//     allocation;
//   - otherwise the last `x-venice-balance-*` observation from an inference
//     response, which works with an inference key and costs no request.
//
// With neither, the row explains what is missing instead of showing an error
// the operator cannot act on.
func fetchVeniceUsage(ctx context.Context, apiKey string, adminKey string, connectionID string) usageResult {
	// The connection key is tried as an ADMIN key too: Venice accepts one key
	// for both inference and billing, so an operator who pasted an ADMIN key
	// needs no second field. An inference key answers 401 "Admin API key
	// required"; that verdict is memoed so the tracker does not spend one
	// rejected call per refresh on it.
	key := strings.TrimSpace(firstNonEmptyStr(adminKey, apiKey))
	if key == "" || veniceAdminRejected(key) {
		return veniceUsageFromSnapshot(connectionID)
	}
	status, header, out, err := usageGet(ctx, veniceBalanceURL, map[string]string{
		"Authorization": "Bearer " + key,
		"Accept":        "application/json",
	})
	if err != nil {
		return usageResult{plan: "Venice AI", message: fmt.Sprintf("Venice error: %v", err)}
	}
	// Venice also reports the balance on this response; keep it even when the
	// body cannot be parsed.
	venicebalance.Record(connectionID, header)
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		if strings.Contains(strings.ToLower(string(out)), "admin api key") {
			memoVeniceAdminRejection(key)
			// An inference key still reports its balance on every inference
			// response; serve that instead of an error the operator cannot fix
			// without issuing another key.
			if snap := veniceUsageFromSnapshot(connectionID); snap.message == "" {
				return snap
			}
			return usageResult{
				plan: "Venice AI",
				message: "Venice reports this account's balance only to an ADMIN key, and no chat request has run yet. " +
					"Use an ADMIN key as the connection key, or send one Venice request and reopen this page.",
			}
		}
		return usageResult{plan: "Venice AI", message: "Venice authentication failed. Check the API key."}
	}
	if status < 200 || status >= 300 {
		msg := fmt.Sprintf("Venice balance API error (%d)", status)
		if t := strings.TrimSpace(string(out)); t != "" {
			if len(t) > 120 {
				t = t[:120]
			}
			msg += ": " + t
		}
		return usageResult{plan: "Venice AI", message: msg}
	}
	data := usageJSON(out)
	if data == nil {
		return usageResult{plan: "Venice AI", message: "Venice balance response was not JSON."}
	}
	quotas := veniceQuotasFromBalance(data)
	if len(quotas) == 0 {
		return usageResult{plan: "Venice AI", message: "Venice connected. No balance data returned."}
	}
	plan := "Venice AI"
	if canConsume, ok := data["canConsume"].(bool); ok && !canConsume {
		plan = "Venice AI (Out of Credits)"
	}
	return usageResult{plan: plan, quotas: quotas}
}

// veniceQuotasFromBalance renders the billing payload as quota rows.
//
// DIEM is a real percentage: the payload carries the epoch allocation, so the
// remaining fraction is meaningful and the row resets at the next UTC epoch
// boundary. USD, bundled and earned credits have no denominator and render as
// credit balances.
func veniceQuotasFromBalance(data map[string]any) map[string]any {
	balances, _ := data["balances"].(map[string]any)
	quotas := map[string]any{}
	diem, diemOK := usageNumOK(balances["diem"])
	allocation := usageNum(data["diemEpochAllocation"], 0)
	if diemOK && allocation > 0 {
		remaining := math.Max(0, math.Min(diem, allocation))
		quotas["diem"] = map[string]any{
			"name":                "DIEM (daily epoch)",
			"displayName":         "DIEM (daily epoch)",
			"used":                allocation - remaining,
			"total":               allocation,
			"remainingPercentage": math.Round(remaining/allocation*1000) / 10,
			"resetAt":             nextUTCMidnight().Format(time.RFC3339),
			"recurring":           true,
		}
	} else if diemOK {
		quotas["diem"] = veniceCreditRow("DIEM credits", "DIEM", diem)
	}
	if v, ok := usageNumOK(balances["usd"]); ok {
		quotas["usd"] = veniceCreditRow("USD credits", "USD", v)
	}
	if v, ok := usageNumOK(balances["bundledCredits"]); ok {
		quotas["bundled"] = veniceCreditRow("Bundled credits", "USD", v)
	}
	if v, ok := usageNumOK(balances["earnedCredits"]); ok {
		quotas["earned"] = veniceCreditRow("Earned credits", "USD", v)
	}
	return quotas
}

// veniceUsageFromSnapshot renders the last inference-reported balance. Without
// the epoch allocation a DIEM figure cannot be a percentage, so both currencies
// are credit balances.
func veniceUsageFromSnapshot(connectionID string) usageResult {
	snap, ok := venicebalance.Latest(connectionID)
	if !ok || (snap.USD == nil && snap.DIEM == nil) {
		return usageResult{
			plan: "Venice AI",
			message: "No balance yet. Venice reports the balance only to an ADMIN key; " +
				"with an inference key it is read from your next chat request.",
		}
	}
	quotas := map[string]any{}
	if snap.DIEM != nil {
		quotas["diem"] = veniceCreditRow("DIEM credits (last request)", "DIEM", *snap.DIEM)
	}
	if snap.USD != nil {
		quotas["usd"] = veniceCreditRow("USD credits (last request)", "USD", *snap.USD)
	}
	return usageResult{plan: "Venice AI", quotas: quotas}
}

func veniceCreditRow(name, currency string, amount float64) map[string]any {
	return map[string]any{
		"name":                name,
		"displayName":         name,
		"used":                0,
		"total":               math.Max(0, amount),
		"remainingPercentage": 100,
		"resetAt":             nil,
		"isCreditBalance":     true,
		"currency":            currency,
	}
}

// nextUTCMidnight is the DIEM epoch boundary: the allocation resets every UTC
// day.
func nextUTCMidnight() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
}

// veniceAdminRejection memo bounds the probe. An inference key answers 401 on
// every billing read, and the tracker refreshes each connection every minute;
// memoing the verdict for a few hours keeps one rejection per key instead of
// sixty an hour. Keyed by a hash so no credential is held in the map.
const (
	veniceRejectionTTL      = 6 * time.Hour
	veniceRejectionCapacity = 128
)

var veniceRejections = struct {
	mu      sync.Mutex
	entries map[string]time.Time
	now     func() time.Time
}{
	entries: make(map[string]time.Time),
	now:     time.Now,
}

func veniceKeyFingerprint(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:8])
}

func veniceAdminRejected(key string) bool {
	fp := veniceKeyFingerprint(key)
	veniceRejections.mu.Lock()
	defer veniceRejections.mu.Unlock()
	expires, ok := veniceRejections.entries[fp]
	if !ok {
		return false
	}
	if !veniceRejections.now().Before(expires) {
		delete(veniceRejections.entries, fp)
		return false
	}
	return true
}

func memoVeniceAdminRejection(key string) {
	fp := veniceKeyFingerprint(key)
	veniceRejections.mu.Lock()
	defer veniceRejections.mu.Unlock()
	if len(veniceRejections.entries) >= veniceRejectionCapacity {
		veniceRejections.entries = make(map[string]time.Time)
	}
	veniceRejections.entries[fp] = veniceRejections.now().Add(veniceRejectionTTL)
}
