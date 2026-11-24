### fix(daemon): recognise the background gateway by process start time

- macOS 27 has no `kern.proc.pathname` sysctl, so the daemon's executable path could not be read, `status` reported a running gateway as absent and the pid file was cleared. On Darwin the process start time is now read from `kern.proc.pid` without cgo, and the pid file carries it (`PID t<microseconds> EXE`). A recycled PID has a different start time, so it never passes for the daemon, even with the same binary.
- Pid files written by earlier builds still parse. Where neither the start time nor the executable path can be read, the claim stays unverifiable and nothing is signalled on its word.
