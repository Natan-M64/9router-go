package venicebalance

import (
	"net/http"
	"testing"
	"time"
)

func floatPtr(v float64) *float64 { return &v }

func TestRecordAndLatest(t *testing.T) {
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	ResetForTest(func() time.Time { return base })
	t.Cleanup(func() { ResetForTest(nil) })

	header := http.Header{}
	header.Set(HeaderUSD, "12.5")
	header.Set(HeaderDIEM, "90.5")
	Record("conn-1", header)

	snap, ok := Latest("conn-1")
	if !ok {
		t.Fatal("Latest() = none, want a snapshot")
	}
	if snap.USD == nil || *snap.USD != 12.5 {
		t.Errorf("USD = %v, want 12.5", snap.USD)
	}
	if snap.DIEM == nil || *snap.DIEM != 90.5 {
		t.Errorf("DIEM = %v, want 90.5", snap.DIEM)
	}
	if !snap.At.Equal(base) {
		t.Errorf("At = %v, want %v", snap.At, base)
	}
}

func TestRecordIgnoresMissingAndInvalidHeaders(t *testing.T) {
	ResetForTest(func() time.Time { return time.Now() })
	t.Cleanup(func() { ResetForTest(nil) })

	Record("conn-2", http.Header{})
	Record("conn-2", http.Header{HeaderUSD: []string{"not-a-number"}})
	Record("conn-2", http.Header{HeaderUSD: []string{"-3"}})
	Record("", http.Header{HeaderUSD: []string{"1"}})
	if _, ok := Latest("conn-2"); ok {
		t.Error("Latest() = a snapshot, want none after only invalid headers")
	}
}

func TestLatestExpiresAfterTTL(t *testing.T) {
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	current := base
	ResetForTest(func() time.Time { return current })
	t.Cleanup(func() { ResetForTest(nil) })

	RecordValues("conn-3", floatPtr(5), nil, base)
	current = base.Add(TTL - time.Minute)
	if _, ok := Latest("conn-3"); !ok {
		t.Fatal("Latest() = none inside the TTL, want a snapshot")
	}
	current = base.Add(TTL + time.Minute)
	if _, ok := Latest("conn-3"); ok {
		t.Fatal("Latest() = a snapshot past the TTL, want none")
	}
}

func TestStoreStaysBounded(t *testing.T) {
	base := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	ResetForTest(func() time.Time { return base })
	t.Cleanup(func() { ResetForTest(nil) })

	for i := 0; i < maxEntries+10; i++ {
		RecordValues(string(rune('a'+i%26))+string(rune('0'+i%10))+string(rune(i)), floatPtr(1), nil, base.Add(time.Duration(i)*time.Second))
	}
	store.mu.Lock()
	size := len(store.entries)
	store.mu.Unlock()
	if size > maxEntries {
		t.Errorf("store holds %d entries, want at most %d", size, maxEntries)
	}
}
