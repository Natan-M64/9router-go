//go:build darwin && cgo

package app

/*
#include <launch.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"net"
	"os"
	"unsafe"
)

func listenPlatformSocket() (net.Listener, string, bool, error) {
	name := C.CString("Listeners")
	defer C.free(unsafe.Pointer(name))

	var fds *C.int
	var cnt C.size_t
	rc := C.launch_activate_socket(name, &fds, &cnt)
	if rc != 0 || cnt == 0 || fds == nil {
		return nil, "", false, nil
	}
	defer C.free(unsafe.Pointer(fds))

	firstFD := *(*C.int)(unsafe.Pointer(fds))
	f := os.NewFile(uintptr(firstFD), "launchd-socket")
	if f == nil {
		return nil, "", false, fmt.Errorf("launchd socket fd %d invalid", int(firstFD))
	}
	ln, err := net.FileListener(f)
	_ = f.Close()
	if err != nil {
		return nil, "", false, fmt.Errorf("launchd FileListener: %w", err)
	}
	return ln, "launchd", true, nil
}
