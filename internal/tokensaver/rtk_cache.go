package tokensaver

import (
	"container/list"
	"crypto/sha256"
	"sync"
	"sync/atomic"

	json "encoding/json/v2"
)

// An agent resends the whole conversation on every turn, so the same tool
// results reach RTK again and again, and compressing one costs milliseconds per
// kilobyte (the filter patterns are matched against the full text). The result
// is a pure function of the text, the command and the configuration: the
// filters are embedded and nothing in the pipeline reads a clock, a random
// source or the filesystem. So the outcome is memoized.
const (
	// rtkCacheMinBytes is the smallest text worth caching. Below it the
	// compression costs less than hashing and bookkeeping would save.
	rtkCacheMinBytes = 4 << 10
	// rtkCacheMaxEntries and rtkCacheMaxBytes bound the cache. The bytes count
	// the compressed texts it holds, not the inputs it has seen.
	rtkCacheMaxEntries = 1024
	rtkCacheMaxBytes   = 64 << 20
)

type rtkCacheKey [sha256.Size]byte

type rtkCacheEntry struct {
	key rtkCacheKey
	res RTKCompressResult
}

type rtkResultCache struct {
	mu         sync.Mutex
	maxEntries int
	maxBytes   int
	bytes      int
	order      *list.List
	items      map[rtkCacheKey]*list.Element

	hits   atomic.Int64
	misses atomic.Int64
}

func newRTKResultCache(maxEntries, maxBytes int) *rtkResultCache {
	return &rtkResultCache{
		maxEntries: maxEntries,
		maxBytes:   maxBytes,
		order:      list.New(),
		items:      make(map[rtkCacheKey]*list.Element),
	}
}

var rtkCache = newRTKResultCache(rtkCacheMaxEntries, rtkCacheMaxBytes)

// rtkCacheKeyFor hashes everything the result depends on. ok is false when the
// configuration cannot be serialised, in which case the caller skips the cache.
func rtkCacheKeyFor(text, command string, cfg RTKConfig) (key rtkCacheKey, ok bool) {
	cfgJSON, err := json.Marshal(cfg, json.Deterministic(true))
	if err != nil {
		return key, false
	}
	h := sha256.New()
	// Length prefixes keep ("ab","c") and ("a","bc") from colliding.
	writeField := func(b []byte) {
		var n [8]byte
		l := uint64(len(b))
		for i := range n {
			n[i] = byte(l >> (8 * i))
		}
		h.Write(n[:])
		h.Write(b)
	}
	writeField([]byte(command))
	writeField(cfgJSON)
	writeField([]byte(text))
	copy(key[:], h.Sum(nil))
	return key, true
}

func (c *rtkResultCache) get(key rtkCacheKey) (RTKCompressResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		c.misses.Add(1)
		return RTKCompressResult{}, false
	}
	c.order.MoveToFront(el)
	c.hits.Add(1)
	return cloneRTKResult(el.Value.(*rtkCacheEntry).res), true
}

func (c *rtkResultCache) put(key rtkCacheKey, res RTKCompressResult) {
	size := len(res.Text)
	if size > c.maxBytes {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return
	}
	c.items[key] = c.order.PushFront(&rtkCacheEntry{key: key, res: cloneRTKResult(res)})
	c.bytes += size
	for c.order.Len() > c.maxEntries || c.bytes > c.maxBytes {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}
		e := oldest.Value.(*rtkCacheEntry)
		c.order.Remove(oldest)
		delete(c.items, e.key)
		c.bytes -= len(e.res.Text)
	}
}

// cloneRTKResult copies the one slice a caller could mutate, so a cached entry
// is never changed through a returned value.
func cloneRTKResult(r RTKCompressResult) RTKCompressResult {
	r.TechniquesUsed = append([]string(nil), r.TechniquesUsed...)
	return r
}

// CompressTextDetailedWithCommand compresses text as the output of command
// under cfg. Large inputs are memoized: see the note at the top of this file.
func CompressTextDetailedWithCommand(text string, command string, cfg RTKConfig) RTKCompressResult {
	if len(text) < rtkCacheMinBytes {
		return compressTextDetailedWithCommand(text, command, cfg)
	}
	key, ok := rtkCacheKeyFor(text, command, cfg)
	if !ok {
		return compressTextDetailedWithCommand(text, command, cfg)
	}
	if res, hit := rtkCache.get(key); hit {
		return res
	}
	res := compressTextDetailedWithCommand(text, command, cfg)
	rtkCache.put(key, res)
	return res
}
