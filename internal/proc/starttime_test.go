package proc

import (
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// A child started inside a known window must report a start time inside that
// window. This checks the seconds field and the byte layout together: a wrong
// offset reads as a far-off date or as garbage.
func TestStartTimeOfChildFallsInsideLaunchWindow(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("start time is only read on darwin")
	}
	before := time.Now().Add(-time.Second)
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	after := time.Now().Add(time.Second)

	micros, err := StartTime(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("StartTime(child): %v", err)
	}
	got := time.UnixMicro(micros)
	if got.Before(before) || got.After(after) {
		t.Fatalf("StartTime(child) = %v, want between %v and %v", got, before, after)
	}
}

// A process that has exited has no start time to report. Returning one would
// let a recycled PID pass as the daemon.
func TestStartTimeOfExitedProcessIsAnError(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("start time is only read on darwin")
	}
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("run child: %v", err)
	}
	pid := cmd.Process.Pid
	if _, err := StartTime(pid); err == nil {
		t.Fatalf("StartTime(exited pid %d) returned no error", pid)
	}
}

func TestStartTimeRejectsInvalidPID(t *testing.T) {
	if _, err := StartTime(0); err == nil {
		t.Fatal("StartTime(0) returned no error")
	}
}
