package app

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// listenForServer returns a socket-activated listener when launched by systemd
// (LISTEN_FDS) or macOS launchd (launch_activate_socket), falling back to a
// normal TCP listener on addr.
func listenForServer(addr string) (net.Listener, string, error) {
	if ln, src, ok, err := listenSystemdSocket(); ok || err != nil {
		return ln, src, err
	}
	if ln, src, ok, err := listenPlatformSocket(); ok || err != nil {
		return ln, src, err
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", err
	}
	return ln, "tcp", nil
}

func listenSystemdSocket() (net.Listener, string, bool, error) {
	raw := strings.TrimSpace(os.Getenv("LISTEN_FDS"))
	if raw == "" {
		return nil, "", false, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return nil, "", false, nil
	}
	// systemd passes the first activated socket on FD 3.
	f := os.NewFile(3, "systemd-socket")
	if f == nil {
		return nil, "", false, fmt.Errorf("systemd LISTEN_FDS=%d: fd 3 invalid", n)
	}
	ln, err := net.FileListener(f)
	_ = f.Close()
	if err != nil {
		return nil, "", false, fmt.Errorf("systemd FileListener: %w", err)
	}
	return ln, "systemd", true, nil
}
