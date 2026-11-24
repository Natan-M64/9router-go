//go:build !darwin || !cgo

package app

import "net"

func listenPlatformSocket() (net.Listener, string, bool, error) {
	return nil, "", false, nil
}
