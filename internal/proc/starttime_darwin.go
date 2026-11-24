//go:build darwin

package proc

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"
)

// startTime reads the process start time from the kinfo_proc that
// kern.proc.pid returns. The struct begins with extern_proc, whose p_starttime
// is a timeval at offset 0: tv_sec (8 bytes on 64-bit Darwin) then tv_usec
// (4 bytes, padded to 8). Darwin runs little-endian on both amd64 and arm64.
//
// This replaces the kern.proc.pathname lookup, which macOS 27 no longer
// provides: the executable path cannot be read there, but the start time can.
func startTime(pid int) (int64, error) {
	raw, err := unix.SysctlRaw("kern.proc.pid", pid)
	if err != nil {
		return 0, fmt.Errorf("sysctl kern.proc.pid for pid %d: %w", pid, err)
	}
	if len(raw) < 12 {
		return 0, fmt.Errorf("sysctl kern.proc.pid for pid %d: short kinfo_proc (%d bytes)", pid, len(raw))
	}
	sec := int64(binary.LittleEndian.Uint64(raw[0:8]))
	usec := int64(int32(binary.LittleEndian.Uint32(raw[8:12])))
	return sec*1_000_000 + usec, nil
}
