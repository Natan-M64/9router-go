//go:build !darwin

package proc

import (
	"fmt"
	"runtime"
)

// startTime is not implemented outside Darwin yet. Linux and Windows claims are
// verified by executable path, as they always were, so nothing depends on this.
func startTime(pid int) (int64, error) {
	return 0, fmt.Errorf("proc.StartTime: not readable on %s (pid %d)", runtime.GOOS, pid)
}
