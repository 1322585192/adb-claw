//go:build unix

package pump

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/llm-net/adb-claw/pkg/stream"
)

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func terminateProcess(pid int) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = process.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) && stream.PIDAlive(pid) {
		time.Sleep(30 * time.Millisecond)
	}
	if stream.PIDAlive(pid) {
		_ = process.Signal(syscall.SIGKILL)
	}
}
