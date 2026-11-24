package tokensaver

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func bigToolText(lines int) string {
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&sb, "entry %d: distinct payload %d\n", i, i*7919)
	}
	return sb.String()
}

// The cached result must be the result the pipeline would have produced.
func TestRTKCacheReturnsTheUncachedResult(t *testing.T) {
	text := bigToolText(400)
	cfg := DefaultRTKConfig()
	want := compressTextDetailedWithCommand(text, "", cfg)
	first := CompressTextDetailedWithCommand(text, "", cfg)
	second := CompressTextDetailedWithCommand(text, "", cfg)
	if !reflect.DeepEqual(first, want) || !reflect.DeepEqual(second, want) {
		t.Fatalf("cached result differs from the uncached one:\nwant %+v\nfirst %+v\nsecond %+v", want, first, second)
	}
}

func TestRTKCacheHitsOnRepeatAndKeysOnCommandAndConfig(t *testing.T) {
	text := bigToolText(300)
	cfg := DefaultRTKConfig()
	hits0 := rtkCache.hits.Load()
	CompressTextDetailedWithCommand(text, "", cfg)
	CompressTextDetailedWithCommand(text, "", cfg)
	if got := rtkCache.hits.Load() - hits0; got != 1 {
		t.Fatalf("hits after one repeat = %d, want 1", got)
	}
	// A different command, a different limit and a different text are all
	// different questions and must not be answered from that entry.
	hits1 := rtkCache.hits.Load()
	CompressTextDetailedWithCommand(text, "cat big.txt", cfg)
	other := cfg
	other.MaxLines = 7
	CompressTextDetailedWithCommand(text, "", other)
	CompressTextDetailedWithCommand(text+"x", "", cfg)
	if got := rtkCache.hits.Load() - hits1; got != 0 {
		t.Fatalf("a changed command, config or text was served from the cache (%d hits)", got)
	}
}

// Changing a returned result must not change what the next caller sees.
func TestRTKCacheResultIsIsolatedFromCallers(t *testing.T) {
	text := bigToolText(350)
	cfg := DefaultRTKConfig()
	a := CompressTextDetailedWithCommand(text, "", cfg)
	if len(a.TechniquesUsed) == 0 {
		t.Fatal("the fixture records no techniques, so this test would prove nothing")
	}
	a.TechniquesUsed[0] = "tampered"
	b := CompressTextDetailedWithCommand(text, "", cfg)
	if b.TechniquesUsed[0] == "tampered" {
		t.Fatal("a caller's change leaked into the cache")
	}
}

func TestRTKCacheSkipsSmallText(t *testing.T) {
	cfg := DefaultRTKConfig()
	hits0, miss0 := rtkCache.hits.Load(), rtkCache.misses.Load()
	CompressTextDetailedWithCommand("short output\n", "", cfg)
	CompressTextDetailedWithCommand("short output\n", "", cfg)
	if rtkCache.hits.Load() != hits0 || rtkCache.misses.Load() != miss0 {
		t.Fatal("a text under the threshold went through the cache")
	}
}

// The cache is bounded by entries and by the bytes it holds.
func TestRTKResultCacheEvictsOldest(t *testing.T) {
	c := newRTKResultCache(3, 1<<20)
	var keys [5]rtkCacheKey
	for i := range keys {
		keys[i][0] = byte(i + 1)
		c.put(keys[i], RTKCompressResult{Text: "t"})
	}
	if c.order.Len() != 3 {
		t.Fatalf("entries = %d, want 3", c.order.Len())
	}
	if _, ok := c.get(keys[0]); ok {
		t.Error("the oldest entry survived")
	}
	if _, ok := c.get(keys[4]); !ok {
		t.Error("the newest entry was evicted")
	}

	b := newRTKResultCache(100, 10)
	b.put(keys[0], RTKCompressResult{Text: "123456"})
	b.put(keys[1], RTKCompressResult{Text: "123456"})
	if b.bytes > 10 {
		t.Errorf("bytes held = %d, want at most 10", b.bytes)
	}
	b.put(keys[2], RTKCompressResult{Text: strings.Repeat("x", 11)})
	if _, ok := b.get(keys[2]); ok {
		t.Error("an entry larger than the whole budget was stored")
	}
}

func TestRTKCacheIsSafeUnderConcurrentUse(t *testing.T) {
	cfg := DefaultRTKConfig()
	texts := []string{bigToolText(300), bigToolText(310), bigToolText(320)}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				CompressTextDetailedWithCommand(texts[(g+i)%len(texts)], "", cfg)
			}
		}(g)
	}
	wg.Wait()
}

// Every configuration field that can change the output must change the key, or
// two settings would share a cached answer. A nil and an empty map are the same
// setting and must share one.
func TestRTKCacheKeyCoversEveryConfigToggle(t *testing.T) {
	text := bigToolText(300)
	base := DefaultRTKConfig()
	key := func(c RTKConfig) rtkCacheKey {
		k, ok := rtkCacheKeyFor(text, "", c)
		if !ok {
			t.Fatal("config not serialisable")
		}
		return k
	}
	want := key(base)

	variants := map[string]func(c *RTKConfig){
		"deduplicate off":  func(c *RTKConfig) { c.Deduplicate = false },
		"max chars":        func(c *RTKConfig) { c.MaxChars = 123 },
		"mode":             func(c *RTKConfig) { c.Mode = "advance" },
		"intensity":        func(c *RTKConfig) { c.Intensity = "aggressive" },
		"category toggled": func(c *RTKConfig) { c.Categories = map[string]bool{"git": false} },
		"filter toggled":   func(c *RTKConfig) { c.Filters = map[string]bool{"gh": false} },
	}
	for name, mutate := range variants {
		c := base
		c.Categories = map[string]bool{}
		for k, v := range base.Categories {
			c.Categories[k] = v
		}
		mutate(&c)
		if key(c) == want {
			t.Errorf("%s did not change the cache key", name)
		}
	}

	nilFilters, emptyFilters := base, base
	nilFilters.Filters = nil
	emptyFilters.Filters = map[string]bool{}
	if key(nilFilters) != key(emptyFilters) {
		t.Error("a nil and an empty Filters map must share a key")
	}
}
