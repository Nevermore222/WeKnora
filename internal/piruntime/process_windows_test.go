//go:build windows

package piruntime

import (
	"os/exec"
	"testing"
)

func TestPrepareSidecarCommandHidesWindow(t *testing.T) {
	cmd := exec.Command("node")
	prepareSidecarCommand(cmd)

	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("sidecar command must hide its Windows console window")
	}
}
