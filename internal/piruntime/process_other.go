//go:build !windows

package piruntime

import "os/exec"

func prepareSidecarCommand(cmd *exec.Cmd) {}
