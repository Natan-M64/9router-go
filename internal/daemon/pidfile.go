package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"9router/proxy/internal/constants"
	"9router/proxy/internal/log"
	"9router/proxy/internal/proc"
)

// The pid file ties a PID number to the process it was written for. It holds up
// to three space-separated fields on one line:
//
//	20130 t1795543887123456 /usr/local/bin/9router-go
//
// The PID alone is not a claim on a process: the OS recycles PIDs, so a file
// left behind by a killed daemon can name an unrelated program by the time the
// next command runs, and `stop` would signal that stranger.
//
// The start time (t<microseconds>) is the tie where the platform reports it. It
// names one process for its whole life, so a recycled PID never matches, and it
// keeps matching after a self-update replaces the binary on disk. Where it is
// not readable, the executable is the tie: cheap to read, needs no privileges,
// and it stops matching the moment the PID lands on something else. A claim that
// resolves neither way is unverifiable, never wrong, and nothing is signalled on
// its word. Files written before start times existed hold only the PID and the
// executable, and are read the same way.
const (
	pidFieldSep  = " "
	startTimeTag = "t"
)

// claim is what the pid file asserts. Zero startMicros and an empty exePath
// mean the field was absent; a claim with neither can never verify.
type claim struct {
	pid         int
	startMicros int64
	exePath     string
}

// writePID records the running daemon PID and the binary it runs, replacing a
// stale file. The write lands whole or not at all: a half-written line would
// read back as unverifiable and strand a daemon that is up.
func writePID(pid int) error {
	if err := os.MkdirAll(Dir(), constants.FilePermDir); err != nil {
		return err
	}
	return os.WriteFile(PIDFile(), formatPID(pid), constants.FilePermFile)
}

// clearPID removes the pid file.
func clearPID() {
	if err := os.Remove(PIDFile()); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Warn("daemon", "remove pid file failed", "path", PIDFile(), "error", err)
	}
}

// formatPID renders the claim. Each field is optional and dropped when it cannot
// be read: writing a placeholder that can never verify would make the daemon
// unfindable, so a missing field is left out rather than faked.
func formatPID(pid int) []byte {
	fields := []string{strconv.Itoa(pid)}
	start, startErr := proc.StartTime(pid)
	if startErr == nil {
		fields = append(fields, startTimeTag+strconv.FormatInt(start, 10))
	}
	self, exeErr := proc.SelfExecutable()
	if exeErr == nil {
		fields = append(fields, self)
	}
	if startErr != nil && exeErr != nil {
		log.Warn("daemon", "pid file falls back to a bare pid", "start_error", startErr, "exe_error", exeErr)
	}
	return []byte(strings.Join(fields, pidFieldSep))
}

// readClaim parses the pid file, tolerating the bare-number and the
// PID-plus-executable forms older builds wrote.
func readClaim() claim {
	data, err := os.ReadFile(PIDFile())
	if err != nil {
		return claim{}
	}
	return parseClaim(string(data))
}

// parseClaim reads the on-disk line: PID, then optionally a start-time field,
// then optionally the executable. The executable is everything after the start
// field, because a path may contain spaces.
func parseClaim(raw string) claim {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return claim{}
	}
	if nl := strings.IndexAny(trimmed, "\r\n"); nl >= 0 {
		trimmed = strings.TrimSpace(trimmed[:nl])
	}
	pidStr, rest := cutField(trimmed)
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return claim{}
	}
	c := claim{pid: pid}
	if field, after := cutField(rest); field != "" {
		if micros, ok := parseStartField(field); ok {
			c.startMicros = micros
			rest = after
		}
	}
	c.exePath = strings.TrimSpace(rest)
	return c
}

// cutField splits off the first space-or-tab-delimited field.
func cutField(s string) (first, rest string) {
	s = strings.TrimSpace(s)
	idx := strings.IndexAny(s, " \t")
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimSpace(s[idx+1:])
}

// parseStartField reads a "t<microseconds>" field. Anything else is not a start
// time, so a legacy executable path still reads as one.
func parseStartField(field string) (int64, bool) {
	if len(field) < 2 || !strings.HasPrefix(field, startTimeTag) {
		return 0, false
	}
	micros, err := strconv.ParseInt(field[len(startTimeTag):], 10, 64)
	if err != nil || micros <= 0 {
		return 0, false
	}
	return micros, true
}

// owner reports whether the process behind this claim is still this daemon:
// alive, and the same process the claim was written for. When the claim carries
// a start time and the platform reports one, the start time decides: a different
// start time is a different process, whatever its executable. Otherwise the
// recorded executable must match. A claim that resolves neither way is
// unverifiable, never wrong: signalling a recycled PID would kill a stranger.
func (c claim) owner() bool {
	if c.pid <= 0 || !proc.Alive(c.pid) {
		return false
	}
	if c.startMicros != 0 {
		if cur, err := proc.StartTime(c.pid); err == nil {
			return cur == c.startMicros
		}
	}
	if c.exePath == "" {
		return false
	}
	exe, err := proc.Executable(c.pid)
	if err != nil {
		return false
	}
	return exe == c.exePath || resolvedPath(exe) == resolvedPath(c.exePath)
}

// resolvedPath expands symlinks on whichever side the filesystem allows it; a
// path that no longer exists cannot be expanded on either, so it compares as
// itself. Windows reports an image as \Device\HarddiskVolume… and macOS reports
// a symlinked install location, neither of which is the form written to the pid
// file.
func resolvedPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}
