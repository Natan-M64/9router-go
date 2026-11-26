package dashboard

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"9router/proxy/internal/handlerutil"
	"9router/proxy/internal/models"
)

// usageReadCache serves a reopen of the quota tracker without spending another
// upstream read.
//
// The tracker remounts on every tab switch — App renders it under an `{#if}` —
// so leaving the page and coming back ten seconds later re-ran the whole
// fan-out, one gated upstream call per account. The numbers were the same; the
// only thing the user saw was the loading state again.
//
// The TTL sits just above the tracker's 60 s auto-refresh tick, so the second
// tick of any two is answered from cache: the dashboard still redraws every
// minute, while the provider sees one read per two minutes per account. That
// halves quota-API traffic without changing what the page does, and a manual
// refresh (`?force=1`) is always live.
//
// The key carries the connection's updatedAt, so editing a credential or the
// connection data invalidates the row without an explicit flush. Rotation does
// not touch updatedAt, so round-robin traffic cannot thrash the cache.
const (
	usageReadCacheTTL      = 70 * time.Second
	usageReadCacheCapacity = 512
)

type usageReadCacheEntry struct {
	key     string
	payload map[string]any
	expires time.Time
}

type usageReadCacheStore struct {
	mu      sync.Mutex
	entries map[string]usageReadCacheEntry
	now     func() time.Time
}

func newUsageReadCacheStore() *usageReadCacheStore {
	return &usageReadCacheStore{
		entries: make(map[string]usageReadCacheEntry),
		now:     time.Now,
	}
}

var usageReadCache = newUsageReadCacheStore()

// usageCacheKey fingerprints everything a cached payload depends on. A change
// in any part makes the previous entry unreachable.
//
// The credential data is hashed into the key because updatedAt has one-second
// granularity: two edits in the same second would otherwise share a key and
// leave the first edit's quota visible for the rest of the TTL.
func usageCacheKey(conn *models.ProviderConnection) string {
	if conn == nil {
		return ""
	}
	sum := sha256.Sum256([]byte(conn.Data))
	return conn.ID + "|" + conn.Provider + "|" + conn.AuthType + "|" + conn.UpdatedAt + "|" + hex.EncodeToString(sum[:8])
}

func (s *usageReadCacheStore) get(connID, key string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[connID]
	if !ok {
		return nil, false
	}
	if entry.key != key || !s.now().Before(entry.expires) {
		delete(s.entries, connID)
		return nil, false
	}
	return entry.payload, true
}

func (s *usageReadCacheStore) put(connID, key string, payload map[string]any) {
	if connID == "" || key == "" || payload == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if len(s.entries) >= usageReadCacheCapacity {
		s.pruneLocked(now)
	}
	s.entries[connID] = usageReadCacheEntry{
		key:     key,
		payload: payload,
		expires: now.Add(usageReadCacheTTL),
	}
}

// pruneLocked drops expired entries, then the soonest-expiring one if the map
// is still full. Called with the lock held, only at the capacity boundary.
func (s *usageReadCacheStore) pruneLocked(now time.Time) {
	evicted := ""
	for connID, entry := range s.entries {
		if !now.Before(entry.expires) {
			delete(s.entries, connID)
			continue
		}
		if evicted == "" || entry.expires.Before(s.entries[evicted].expires) {
			evicted = connID
		}
	}
	if len(s.entries) >= usageReadCacheCapacity && evicted != "" {
		delete(s.entries, evicted)
	}
}

// usageForceRequested reports whether the caller asked to bypass the cache.
func usageForceRequested(raw string) bool {
	return raw == "1" || raw == "true"
}

// writeUsagePayload stores a successful read and writes it to the client. Only
// successful payloads reach this helper, so a transient failure is never
// cached for the next reopen.
func writeUsagePayload(w http.ResponseWriter, conn *models.ProviderConnection, cacheKey string, payload map[string]any) {
	if _, hasMessage := payload["message"]; !hasMessage {
		usageReadCache.put(conn.ID, cacheKey, payload)
	}
	handlerutil.WriteJSON(w, http.StatusOK, payload)
}
