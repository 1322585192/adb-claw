//go:build unix

package stream

import (
	"os"
	"syscall"
)

// PIDAlive reports whether pid is still a running process.
func PIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
