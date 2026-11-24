package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestIdleTracker_TriggersAfterTimeout(t *testing.T) {
	now := time.Unix(1000, 0)
	tracker := newIdleTracker(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), func() time.Time { return now })

	if tracker.ShouldShutdown(now, 10*time.Second) {
		t.Fatal("expected ShouldShutdown=false immediately at t=0")
	}

	now = now.Add(9 * time.Second)
	if tracker.ShouldShutdown(now, 10*time.Second) {
		t.Fatal("expected ShouldShutdown=false before timeout elapses")
	}

	now = now.Add(1 * time.Second)
	if !tracker.ShouldShutdown(now, 10*time.Second) {
		t.Fatal("expected ShouldShutdown=true once timeout elapses with 0 active requests")
	}
}

func TestIdleTracker_DoesNotShutdownDuringActiveRequest(t *testing.T) {
	now := time.Unix(1000, 0)
	started := make(chan struct{})
	release := make(chan struct{})

	tracker := newIdleTracker(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	}), func() time.Time { return now })

	done := make(chan struct{})
	go func() {
		req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
		rec := httptest.NewRecorder()
		tracker.ServeHTTP(rec, req)
		close(done)
	}()

	<-started
	now = now.Add(30 * time.Minute)
	if tracker.ShouldShutdown(now, 10*time.Second) {
		t.Fatal("expected ShouldShutdown=false while request is still active")
	}

	close(release)
	<-done

	if tracker.ShouldShutdown(now, 10*time.Second) {
		t.Fatal("expected ShouldShutdown=false immediately after request completes (activity timestamp refreshed)")
	}

	now = now.Add(10 * time.Second)
	if !tracker.ShouldShutdown(now, 10*time.Second) {
		t.Fatal("expected ShouldShutdown=true after idle timeout following request completion")
	}
}

func TestStartIdleWatcher_InvokesOnIdle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracker := newIdleTracker(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), time.Now)
	var called atomic.Bool
	done := make(chan struct{})

	startIdleWatcher(ctx, tracker, 120*time.Millisecond, func() {
		if called.CompareAndSwap(false, true) {
			close(done)
		}
	})

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for idle watcher to invoke onIdle")
	}
}
