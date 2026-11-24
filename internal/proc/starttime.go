package proc

import (
	"fmt"
	"runtime"
)

// StartTime reports when the process pid was started, in microseconds since the
// Unix epoch. A PID that is recycled for a new process always has a different
// start time, so the pair (pid, start time) names one process for its lifetime.
//
// It returns an error where the platform does not expose the start time or the
// process does not exist. Callers treat that as "unverified", not as a match.
func StartTime(pid int) (int64, error) {
	if pid <= 0 {
		return 0, fmt.Errorf("proc.StartTime: invalid pid %d", pid)
	}
	micros, err := startTime(pid)
	if err != nil {
		return 0, err
	}
	if micros <= 0 {
		return 0, fmt.Errorf("proc.StartTime: no start time for pid %d on %s", pid, runtime.GOOS)
	}
	return micros, nil
}
