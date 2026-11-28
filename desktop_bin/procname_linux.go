//go:build linux && !android

package main

import (
	"syscall"
	"unsafe"
)

func setWorkerProcessName(name string) {
	if name == "" {
		return
	}
	// PR_SET_NAME (15) accepts at most 15 bytes plus trailing NUL and updates
	// /proc/self/comm so pgrep/pkill -x OneXrayCore ignore this worker.
	var buf [16]byte
	copy(buf[:15], name)
	_, _, _ = syscall.Syscall6(
		syscall.SYS_PRCTL,
		15,
		uintptr(unsafe.Pointer(&buf[0])),
		0,
		0,
		0,
		0,
	)
}
