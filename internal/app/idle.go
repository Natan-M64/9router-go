package app

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"9router/proxy/internal/log"
)

const (
	minIdlePollInterval = 100 * time.Millisecond
	maxIdlePollInterval = 15 * time.Second
)

// idleTracker tracks active in-flight HTTP requests and the timestamp of the
// last completed (or started) request so an idle server can shut down cleanly
// without interrupting active requests or SSE streams.
type idleTracker struct {
	next         http.Handler
	active       atomic.Int64
	lastActivity atomic.Int64
	now          func() time.Time
}

func newIdleTracker(next http.Handler, now func() time.Time) *idleTracker {
	if now == nil {
		now = time.Now
	}
	t := &idleTracker{
		next: next,
		now:  now,
	}
	t.lastActivity.Store(now().UnixNano())
	return t
}

func (t *idleTracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.active.Add(1)
	t.lastActivity.Store(t.now().UnixNano())
	defer func() {
		t.lastActivity.Store(t.now().UnixNano())
		t.active.Add(-1)
	}()
	t.next.ServeHTTP(w, r)
}

// ShouldShutdown reports whether there are zero active requests and the time
// elapsed since the last activity meets or exceeds timeout.
func (t *idleTracker) ShouldShutdown(now time.Time, timeout time.Duration) bool {
	if timeout <= 0 || t.active.Load() > 0 {
		return false
	}
	last := time.Unix(0, t.lastActivity.Load())
	return now.Sub(last) >= timeout
}

// startIdleWatcher polls tracker until ctx is cancelled or the server has been
// idle for at least timeout, then invokes onIdle once.
func startIdleWatcher(ctx context.Context, tracker *idleTracker, timeout time.Duration, onIdle func()) {
	if tracker == nil || timeout <= 0 || onIdle == nil {
		return
	}
	poll := min(max(timeout/4, minIdlePollInterval), maxIdlePollInterval)

	go func() {
		ticker := time.NewTicker(poll)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if tracker.ShouldShutdown(tracker.now(), timeout) {
					log.Info("idle", "no active requests for idle timeout; shutting down", "timeout", timeout.String())
					onIdle()
					return
				}
			}
		}
	}()
}
