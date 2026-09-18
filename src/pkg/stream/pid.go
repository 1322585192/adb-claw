package stream

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/llm-net/adb-claw/pkg/atomicfile"
)

// WritePID stores the pump process id.
func WritePID(key string, pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pump pid")
	}
	return atomicfile.Write(PIDPath(key), []byte(strconv.Itoa(pid)+"\n"), 0644)
}

// ReadPID returns the recorded pump pid.
func ReadPID(key string) (int, error) {
	data, err := os.ReadFile(PIDPath(key))
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid pump pid file")
	}
	return pid, nil
}

// RemovePID deletes the pid file.
func RemovePID(key string) {
	_ = os.Remove(PIDPath(key))
}

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

// Running reports whether this device's pump pid is alive.
func Running(key string) bool {
	pid, err := ReadPID(key)
	if err != nil {
		return false
	}
	if PIDAlive(pid) {
		return true
	}
	RemovePID(key)
	return false
}
