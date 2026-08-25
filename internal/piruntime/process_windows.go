//go:build windows

package piruntime

import (
	"os/exec"
	"syscall"
)

func prepareSidecarCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
