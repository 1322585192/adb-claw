//go:build !unix && !windows

package pump

import "os/exec"

func detach(cmd *exec.Cmd) {}

func terminateProcess(pid int) {}
