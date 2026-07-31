package piruntime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/Tencent/Xelora/internal/workspace"
)

var (
	managerMu sync.RWMutex
	manager   *Manager
)

// Manager owns the Personal sidecar process and shared Facade.
type Manager struct {
	client  *Client
	facade  *Facade
	command string
	args    []string
}

// DefaultFacade returns the process-wide Facade (may be nil).
func DefaultFacade() *Facade {
	managerMu.RLock()
	defer managerMu.RUnlock()
	if manager == nil {
		return nil
	}
	return manager.facade
}

// StartFromPersonalBundle starts the sidecar when Personal local folders are enabled.
// Looks for sidecars next to the executable, then under repo-relative paths for dev.
func StartFromPersonalBundle(ctx context.Context) error {
	if !workspace.PersonalLocalFoldersEnabled() {
		return nil
	}
	cmd, args, err := resolveSidecarLaunch()
	if err != nil {
		return err
	}
	return StartManager(ctx, cmd, args)
}

// StartManager starts (or restarts) the global manager.
func StartManager(ctx context.Context, command string, args []string) error {
	managerMu.Lock()
	defer managerMu.Unlock()
	if manager != nil {
		_ = manager.client.Close()
		manager = nil
	}
	cli, err := Start(ctx, StartOptions{Command: command, Args: args})
	if err != nil {
		return err
	}
	if err := cli.Handshake(ctx); err != nil {
		_ = cli.Close()
		return err
	}
	m := &Manager{
		client:  cli,
		command: command,
		args:    args,
	}
	m.facade = NewFacade(cli, nil)
	manager = m
	return nil
}

// Stop shuts down the global sidecar.
func Stop(ctx context.Context) error {
	_ = ctx
	managerMu.Lock()
	defer managerMu.Unlock()
	if manager == nil {
		return nil
	}
	err := manager.client.Close()
	manager = nil
	return err
}

func resolveSidecarLaunch() (string, []string, error) {
	if override := os.Getenv("XELORA_PI_SIDECAR"); override != "" {
		return override, nil, nil
	}
	if nodeScript := os.Getenv("XELORA_PI_SIDECAR_SCRIPT"); nodeScript != "" {
		node, err := lookNode()
		if err != nil {
			return "", nil, err
		}
		return node, []string{nodeScript}, nil
	}

	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}
	exeDir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(exeDir, "pi-workspace", "dist", "main.js"),
		filepath.Join(exeDir, "sidecars", "pi-workspace", "dist", "main.js"),
	}
	// Dev: walk up from cwd for repo root sidecars/
	cwd, _ := os.Getwd()
	for dir := cwd; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		candidates = append(candidates, filepath.Join(dir, "sidecars", "pi-workspace", "dist", "main.js"))
	}

	var script string
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			script = c
			break
		}
	}
	if script == "" {
		return "", nil, fmt.Errorf("%s: pi-workspace sidecar not found", ErrRuntimeUnavailable)
	}
	node, err := lookNode()
	if err != nil {
		return "", nil, err
	}
	return node, []string{script}, nil
}

func lookNode() (string, error) {
	if p := os.Getenv("XELORA_NODE"); p != "" {
		return p, nil
	}
	path, err := exec.LookPath("node")
	if err != nil {
		return "", fmt.Errorf("%s: node not found (install Node >= 22 or set XELORA_NODE)", ErrRuntimeUnavailable)
	}
	return path, nil
}
