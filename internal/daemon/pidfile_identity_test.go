package daemon

import (
	"os"
	"runtime"
	"testing"

	"9router/proxy/internal/proc"
)

// selfStartMicros is this process's start time, or a skip where the platform
// cannot report one. The start-time cases below only mean something where it can.
func selfStartMicros(t *testing.T) int64 {
	t.Helper()
	start, err := proc.StartTime(os.Getpid())
	if err != nil {
		t.Skipf("start time is not readable on %s: %v", runtime.GOOS, err)
	}
	return start
}

// pathLookupSupported reports whether the platform can read a live process's
// image path. macOS 27 cannot, so a claim without a start time is unverifiable
// there, and must not be trusted.
func pathLookupSupported() bool {
	_, err := proc.Executable(os.Getpid())
	return err == nil
}

// A claim carrying this process's own start time is this process.
func TestOwnerAcceptsOwnStartTime(t *testing.T) {
	c := claim{pid: os.Getpid(), startMicros: selfStartMicros(t)}
	if !c.owner() {
		t.Fatal("owner() = false for this process's own start time")
	}
}

// A recycled PID keeps the number and the binary but not the start time. The
// claim must not match, or `stop` would signal a stranger.
func TestOwnerRejectsRecycledPIDWithDifferentStartTime(t *testing.T) {
	c := claim{
		pid:         os.Getpid(),
		startMicros: selfStartMicros(t) + 1_000_000,
		exePath:     selfExecutable(t),
	}
	if c.owner() {
		t.Fatal("owner() = true for a start time that is not this process's")
	}
}

// Without a start time the executable decides. Where the image path cannot be
// read, the claim is unverifiable: it must be refused, not trusted on its word.
func TestOwnerLegacyExecutableClaim(t *testing.T) {
	c := claim{pid: os.Getpid(), exePath: selfExecutable(t)}
	want := pathLookupSupported()
	if got := c.owner(); got != want {
		t.Fatalf("owner() = %v for a legacy claim, want %v (image path readable: %v)", got, want, want)
	}
}

// What formatPID writes, parseClaim reads back to the same identity.
func TestFormatPIDRoundTripsIdentity(t *testing.T) {
	got := parseClaim(string(formatPID(os.Getpid())))
	if got.pid != os.Getpid() {
		t.Fatalf("pid = %d, want %d", got.pid, os.Getpid())
	}
	if start, err := proc.StartTime(os.Getpid()); err == nil && got.startMicros != start {
		t.Fatalf("startMicros = %d, want %d", got.startMicros, start)
	}
	if got.exePath != selfExecutable(t) {
		t.Fatalf("exePath = %q, want %q", got.exePath, selfExecutable(t))
	}
}
