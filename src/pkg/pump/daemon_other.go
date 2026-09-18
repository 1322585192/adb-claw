//go:build !unix

package pump

import "os/exec"

func detach(cmd *exec.Cmd) {}
