// Package venicebalance keeps the last inference-reported credit balance for a
// Venice connection.
//
// Venice's billing API (GET /api/v1/billing/balance) requires an ADMIN key,
// while a connection is normally configured with an INFERENCE key. Every
// inference response carries `x-venice-balance-diem` and
// `x-venice-balance-usd` headers, so the balance can be read for free from
// traffic the gateway already serves; no extra request and no broader key.
//
// The store is in memory because the value is a last-seen observation, not
// ledger data: it is refreshed by the next request, it must never cost a disk
// write on the hot path, and a restart losing it only means "no balance until
// the next chat turn". It is bounded by connection count and a TTL.
package venicebalance

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Header names published by Venice on inference responses. The value is the
// balance before the request was processed; a header is omitted when that
// balance is zero.
const (
	HeaderUSD  = "x-venice-balance-usd"
	HeaderDIEM = "x-venice-balance-diem"
)

// Snapshot is the last balance observed for one connection.
type Snapshot struct {
	// USD is the prepaid fiat balance, nil when Venice did not report it.
	USD *float64
	// DIEM is the remaining staking allowance for the current UTC epoch, nil
	// when the account is not staking or the header was absent.
	DIEM *float64
	// At is when the observation was recorded.
	At time.Time
}

// TTL bounds how long an observation is trusted. A week is long enough that a
// low-traffic account still shows its last known balance and short enough that
// a stale figure cannot survive indefinitely.
const TTL = 7 * 24 * time.Hour

// maxEntries bounds the map. It is sized for a fleet of connections, not for
// traffic: the key is a connection id, so only one row per connection exists.
const maxEntries = 4096

var store = struct {
	mu      sync.Mutex
	entries map[string]Snapshot
	now     func() time.Time
}{
	entries: make(map[string]Snapshot),
	now:     time.Now,
}

// Record stores the balances carried by an upstream response. It is a no-op
// when the response carries neither header, which is the common case for
// non-Venice traffic and for error responses Venice documents as header-less.
func Record(connectionID string, header http.Header) {
	if connectionID == "" || header == nil {
		return
	}
	usd, usdOK := parseBalance(header.Get(HeaderUSD))
	diem, diemOK := parseBalance(header.Get(HeaderDIEM))
	if !usdOK && !diemOK {
		return
	}
	RecordValues(connectionID, usd, diem, store.now())
}

// RecordValues stores one observation. Exported for tests and for callers that
// already parsed the values.
func RecordValues(connectionID string, usd, diem *float64, at time.Time) {
	if connectionID == "" {
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.entries) >= maxEntries {
		pruneLocked(at)
	}
	store.entries[connectionID] = Snapshot{USD: usd, DIEM: diem, At: at}
}

// Latest returns the last observation for a connection. Expired entries are
// dropped on read so a dormant connection does not keep a row forever.
func Latest(connectionID string) (Snapshot, bool) {
	if connectionID == "" {
		return Snapshot{}, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	snap, ok := store.entries[connectionID]
	if !ok {
		return Snapshot{}, false
	}
	if store.now().Sub(snap.At) > TTL {
		delete(store.entries, connectionID)
		return Snapshot{}, false
	}
	return snap, true
}

// pruneLocked drops expired entries, then the oldest one if the map is still
// full. Called only with the lock held and only when the cap is reached, so the
// linear scan is paid at most once per insert past the cap.
func pruneLocked(now time.Time) {
	oldestKey := ""
	var oldest time.Time
	for key, snap := range store.entries {
		if now.Sub(snap.At) > TTL {
			delete(store.entries, key)
			continue
		}
		if oldestKey == "" || snap.At.Before(oldest) {
			oldestKey = key
			oldest = snap.At
		}
	}
	if len(store.entries) >= maxEntries && oldestKey != "" {
		delete(store.entries, oldestKey)
	}
}

// parseBalance reads one currency value. Venice formats the header as a plain
// decimal string; a non-numeric value is ignored rather than coerced to zero,
// because zero is a real balance and the two must not be confused.
func parseBalance(raw string) (*float64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}
	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || v < 0 {
		return nil, false
	}
	return &v, true
}

// ResetForTest clears the store and pins the clock. Test-only helper, kept
// unexported-by-convention: production code never calls it.
func ResetForTest(now func() time.Time) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.entries = make(map[string]Snapshot)
	if now == nil {
		now = time.Now
	}
	store.now = now
}
