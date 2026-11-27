package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"9router/proxy/internal/models"
)

func TestRequireRateLimitReleasesIPConcurrencyAfterRequest(t *testing.T) {
	rl := NewRateLimiter(time.Minute)
	rl.SetIPLimits(IPLimits{Concurrency: 1})
	handler := RequireRateLimit(rl, nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := range 2 {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		req.RemoteAddr = "203.0.113.7:1234"
		req = req.WithContext(withAPIKey(req, &models.APIKey{ID: "key-1", IsActive: 1}))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want 204", i+1, rec.Code)
		}
	}
}

func TestRequireRateLimitReleasesIPSlotWhenKeyAdmissionRefuses(t *testing.T) {
	rl := NewRateLimiter(time.Minute)
	rl.SetIPLimits(IPLimits{Concurrency: 1})
	if _, ok := rl.Acquire(context.Background(), "key-1", 0, 1, 0); !ok {
		t.Fatal("failed to reserve key concurrency slot for the refusal case")
	}
	handler := RequireRateLimit(rl, nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	mkRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		req.RemoteAddr = "203.0.113.9:1234"
		return req.WithContext(withAPIKey(req, &models.APIKey{
			ID:                   "key-1",
			IsActive:             1,
			RateLimitConcurrency: new(1),
			QueueTimeoutMs:       new(1),
		}))
	}

	refused := httptest.NewRecorder()
	handler.ServeHTTP(refused, mkRequest())
	if refused.Code != http.StatusTooManyRequests {
		t.Fatalf("request with full key slot status = %d, want 429", refused.Code)
	}
	rl.ReleaseConcurrency("key-1")

	accepted := httptest.NewRecorder()
	handler.ServeHTTP(accepted, mkRequest())
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("request after key slot release status = %d, want 204 (IP slot was leaked)", accepted.Code)
	}
}
