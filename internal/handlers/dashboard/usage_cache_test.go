package dashboard

import (
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"9router/proxy/internal/models"
)

func TestUsageReadCache_TTLAndKeyChange(t *testing.T) {
	current := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	cache := newUsageReadCacheStore()
	cache.now = func() time.Time { return current }
	conn := &models.ProviderConnection{ID: "c1", Provider: "venice", AuthType: "apikey", UpdatedAt: "2026-11-25T11:00:00Z"}
	key := usageCacheKey(conn)
	payload := map[string]any{"plan": "Venice AI"}
	cache.put(conn.ID, key, payload)

	if got, ok := cache.get(conn.ID, key); !ok || got["plan"] != "Venice AI" {
		t.Fatal("expected a cache hit inside the TTL")
	}
	current = current.Add(usageReadCacheTTL + time.Second)
	if _, ok := cache.get(conn.ID, key); ok {
		t.Fatal("expected a miss past the TTL")
	}

	cache.put(conn.ID, key, payload)
	edited := *conn
	edited.UpdatedAt = "2026-11-25T12:30:00Z"
	if _, ok := cache.get(conn.ID, usageCacheKey(&edited)); ok {
		t.Fatal("expected a miss after the connection changed")
	}
}

func TestWriteUsagePayload_DoesNotCacheFailures(t *testing.T) {
	oldCache := usageReadCache
	usageReadCache = newUsageReadCacheStore()
	t.Cleanup(func() { usageReadCache = oldCache })

	conn := &models.ProviderConnection{ID: "transient", Provider: "deepseek"}
	payload := map[string]any{"message": "DeepSeek error: temporary network failure", "quotas": map[string]any{}}
	recorder := httptest.NewRecorder()
	writeUsagePayload(recorder, conn, usageCacheKey(conn), payload)

	if _, hit := usageReadCache.get(conn.ID, usageCacheKey(conn)); hit {
		t.Fatal("transient provider failure must not be cached")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
}

func TestUsageReadCache_StaysBounded(t *testing.T) {
	cache := newUsageReadCacheStore()
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return base }
	for i := 0; i < usageReadCacheCapacity+20; i++ {
		id := "c" + time.Duration(i).String()
		cache.put(id, id, map[string]any{"i": i})
	}
	if len(cache.entries) > usageReadCacheCapacity {
		t.Errorf("cache holds %d entries, want at most %d", len(cache.entries), usageReadCacheCapacity)
	}
}

// TestHandleGetConnectionUsage_CachesReopenAndHonoursForce pins the behaviour
// the tracker relies on: an unchanged connection is answered from the cache, a
// forced read bypasses it, and an edit invalidates it even when both writes land
// in the same second (updatedAt is second-granular, so the key hashes the
// credential data too).
func TestHandleGetConnectionUsage_CachesReopenAndHonoursForce(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	h := NewDashboardHandler(repo)

	oldCache := usageReadCache
	usageReadCache = newUsageReadCacheStore()
	t.Cleanup(func() { usageReadCache = oldCache })

	const id = "conn-cache"
	const seeded = `{"modelLock_demo-model":"2099-01-01T00:00:00Z"}`
	if err := repo.CreateProviderConnectionFull(id, "nous", "apikey", "locked", nil, seeded); err != nil {
		t.Fatalf("seed connection: %v", err)
	}

	read := func(query string) map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/usage/"+id+query, nil)
		req.SetPathValue("connectionId", id)
		rec := httptest.NewRecorder()
		h.HandleGetConnectionUsage(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return body
	}

	conn, err := repo.GetProviderConnectionByID(id)
	if err != nil || conn == nil {
		t.Fatalf("reload connection: %v", err)
	}
	// Preload a marker under the live key: a read that hits the cache must
	// return it instead of the lock row the connection data would produce.
	usageReadCache.put(id, usageCacheKey(conn), map[string]any{"plan": "cached-marker", "quotas": map[string]any{}})

	if body := read(""); body["plan"] != "cached-marker" {
		t.Fatalf("an unchanged reopen must be served from the cache, got %v", body)
	}
	forced := read("?force=1")
	if forced["plan"] == "cached-marker" {
		t.Fatalf("force=1 must bypass the cache, got %v", forced)
	}

	// Two edits in the same second share an updatedAt, which is why the key
	// hashes the data as well. The second edit clears the lock row.
	if err := repo.ReplaceProviderConnectionPayload(id, "locked", seeded); err != nil {
		t.Fatalf("replace payload: %v", err)
	}
	if err := repo.ReplaceProviderConnectionPayload(id, "unlocked", `{}`); err != nil {
		t.Fatalf("clear payload: %v", err)
	}
	body := read("")
	quotas, _ := body["quotas"].(map[string]any)
	if _, hasLock := quotas["demo-model"]; hasLock {
		t.Fatalf("an edited connection must not serve the previous payload: %v", body)
	}
}
