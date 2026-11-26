package chat

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"9router/proxy/internal/db"
	"9router/proxy/internal/handlerutil"
	"9router/proxy/internal/translator"
)

func fp(system, tools string) prefixFingerprint {
	return prefixFingerprint{system: system, tools: tools}
}

func TestCacheBreakIgnoresHealthyAndSmallRequests(t *testing.T) {
	if r := cacheBreakReport(50000, 47000, fp("a", "t"), fp("a", "t"), true); r != nil {
		t.Errorf("a 94%% hit must not be reported: %v", r)
	}
	if r := cacheBreakReport(5000, 0, fp("b", "t"), fp("a", "t"), true); r != nil {
		t.Errorf("a small request must not be reported: %v", r)
	}
}

func TestCacheBreakNamesWhatMoved(t *testing.T) {
	cases := []struct {
		name      string
		cur, prev prefixFingerprint
		hasPrev   bool
		want      string
	}{
		{"system moved", fp("new", "t"), fp("old", "t"), true, "system-changed"},
		{"tools moved", fp("s", "new"), fp("s", "old"), true, "tools-changed"},
		{"both moved, system is named first", fp("new", "new"), fp("old", "old"), true, "system-changed"},
		{"prefix same, history cut", fp("s", "t"), fp("s", "t"), true, "history-rewritten"},
		{"no earlier request", fp("s", "t"), prefixFingerprint{}, false, "first-seen"},
	}
	for _, c := range cases {
		r := cacheBreakReport(60000, 20000, c.cur, c.prev, c.hasPrev)
		if r == nil {
			t.Fatalf("%s: no report", c.name)
		}
		if r["cause"] != c.want {
			t.Errorf("%s: cause = %v, want %s", c.name, r["cause"], c.want)
		}
		if r["hitPct"] != 33 {
			t.Errorf("%s: hitPct = %v, want 33", c.name, r["hitPct"])
		}
	}
}

func TestPrefixMemoComparesWithThePreviousRequest(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	m := newPrefixMemo(func() time.Time { return now })
	if _, ok := m.swap("conv", fp("s", "t")); ok {
		t.Fatal("first request must have no previous")
	}
	prev, ok := m.swap("conv", fp("s2", "t"))
	if !ok || prev.system != "s" {
		t.Fatalf("second request must see the first: ok=%v prev=%+v", ok, prev)
	}
	if _, ok := m.swap("other", fp("x", "y")); ok {
		t.Error("another conversation must not see this one")
	}
}

func TestPrefixMemoExpiresAndStaysBounded(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	m := newPrefixMemo(func() time.Time { return now })
	m.swap("conv", fp("s", "t"))
	now = now.Add(cacheBreakMemoTTL + time.Second)
	if _, ok := m.swap("conv", fp("s", "t")); ok {
		t.Error("an expired fingerprint must not be used")
	}
	for i := 0; i < cacheBreakMemoMax+50; i++ {
		m.swap(string(rune('a'+i%26))+time.Duration(i).String(), fp("s", "t"))
	}
	if n := len(m.entries); n > cacheBreakMemoMax {
		t.Errorf("memo holds %d entries, want at most %d", n, cacheBreakMemoMax)
	}
}

// A low hit on a large request must reach the stored record, and the second
// request of the same conversation must be compared with the first.
func TestLogUsageStoresCacheBreakForTheConversation(t *testing.T) {
	database, cleanup := setupChatTestDB(t)
	defer cleanup()
	h := NewChatHandler(db.NewRepo(database))
	prev := processPrefixMemo
	processPrefixMemo = newPrefixMemo(nil)
	t.Cleanup(func() { processPrefixMemo = prev })

	first := compositionBody("sys-one", `[{"type":"function"}]`, "a", "b")
	second := compositionBody("sys-two", `[{"type":"function"}]`, "a", "b")
	ctx := handlerutil.WithSessionID(context.Background(), "conv-break")
	info := &UsageLogInfo{Provider: "p", Model: "m", ConnectionID: "conn-break"}

	// 30k prompt with 5k cached: a 17% hit, far below the floor.
	h.LogUsage(ctx, info, &translator.OpenAIUsage{PromptTokens: 30000, CompletionTokens: 5, CachedTokens: 5000}, 50, first, nil)
	// The record id is unix milliseconds plus the model, so two calls in the
	// same millisecond would share an id and the insert would keep only one.
	time.Sleep(2 * time.Millisecond)
	h.LogUsage(ctx, info, &translator.OpenAIUsage{PromptTokens: 30000, CompletionTokens: 5, CachedTokens: 5000}, 50, second, nil)

	rows, err := database.Query(`SELECT data FROM requestDetails WHERE connectionId = 'conn-break' ORDER BY rowid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var causes []string
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			t.Fatal(err)
		}
		var got struct {
			CacheBreak struct {
				Cause string `json:"cause"`
			} `json:"cacheBreak"`
		}
		if err := json.Unmarshal([]byte(data), &got); err != nil {
			t.Fatal(err)
		}
		causes = append(causes, got.CacheBreak.Cause)
	}
	if len(causes) != 2 || causes[0] != "first-seen" || causes[1] != "system-changed" {
		t.Errorf("stored causes = %v, want [first-seen system-changed]", causes)
	}
}

// The chain is what turns "history-rewritten" into an address: index 0 is the
// front of the conversation (a prune or a rewritten first turn), a larger index
// is a mid-history rewrite, and an unchanged chain means the gateway did not
// break the prefix at all.
func TestFirstMovedMessage(t *testing.T) {
	const h = messageHashLen
	chain := func(parts ...string) string {
		out := ""
		for _, p := range parts {
			out += p
		}
		return out
	}
	tests := []struct {
		name      string
		prev, cur string
		wantIdx   int
		wantOK    bool
	}{
		{"identical", chain("aaaaaaaa", "bbbbbbbb"), chain("aaaaaaaa", "bbbbbbbb"), 0, false},
		{"front moved", chain("aaaaaaaa", "bbbbbbbb"), chain("cccccccc", "bbbbbbbb"), 0, true},
		{"index three moved", chain("a1a1a1a1", "b2b2b2b2", "c3c3c3c3", "d4d4d4d4"), chain("a1a1a1a1", "b2b2b2b2", "c3c3c3c3", "ffffffff"), 3, true},
		{"append only", chain("aaaaaaaa"), chain("aaaaaaaa", "bbbbbbbb"), 0, false},
		{"empty previous", "", chain("aaaaaaaa"), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, ok := firstMovedMessage(tt.prev, tt.cur)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && idx != tt.wantIdx {
				t.Errorf("index = %d, want %d", idx, tt.wantIdx)
			}
		})
	}
	if h != 8 {
		t.Fatalf("messageHashLen = %d, fixtures are written for 8", h)
	}
}

func TestCacheBreakReportNamesTheFirstMovedMessage(t *testing.T) {
	prev := prefixFingerprint{system: "s", tools: "t", messages: "a1a1a1a1b2b2b2b2c3c3c3c3"}
	cur := prefixFingerprint{system: "s", tools: "t", messages: "a1a1a1a1b2b2b2b2ffffffff"}
	r := cacheBreakReport(60000, 20000, cur, prev, true)
	if r == nil || r["cause"] != "history-rewritten" {
		t.Fatalf("report = %v, want history-rewritten", r)
	}
	if r["firstMovedIndex"] != 2 {
		t.Errorf("firstMovedIndex = %v, want 2", r["firstMovedIndex"])
	}
}

// When every retained hash matches, the gateway forwarded the same prefix and
// naming it a rewrite would send the operator after a bug that is not there.
func TestCacheBreakReportPrefixUnchangedWhenHashesMatch(t *testing.T) {
	same := "a1a1a1a1b2b2b2b2"
	prev := prefixFingerprint{system: "s", tools: "t", messages: same}
	cur := prefixFingerprint{system: "s", tools: "t", messages: same}
	r := cacheBreakReport(60000, 2000, cur, prev, true)
	if r == nil {
		t.Fatal("no report")
	}
	if r["cause"] != "prefix-unchanged" {
		t.Errorf("cause = %v, want prefix-unchanged", r["cause"])
	}
	if _, ok := r["firstMovedIndex"]; ok {
		t.Error("an unchanged chain must not name a moved message")
	}
}
