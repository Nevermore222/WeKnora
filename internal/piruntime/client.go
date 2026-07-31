package piruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Caller is the Facade dependency for sidecar RPC.
type Caller interface {
	Handshake(ctx context.Context) error
	Call(ctx context.Context, req Request) (Response, error)
	Close() error
}

// StartOptions configures the sidecar child process.
type StartOptions struct {
	Command string
	Args    []string
	Env     []string
	Dir     string
}

// Client talks NDJSON over stdio to a sidecar process.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
}

// Start launches the sidecar process.
func Start(ctx context.Context, opts StartOptions) (*Client, error) {
	if opts.Command == "" {
		return nil, fmt.Errorf("%s: empty command", ErrRuntimeUnavailable)
	}
	cmd := exec.CommandContext(ctx, opts.Command, opts.Args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("%s: %w", ErrRuntimeUnavailable, err)
	}
	sc := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 4*1024*1024)
	return &Client{cmd: cmd, stdin: stdin, stdout: sc}, nil
}

// Handshake verifies protocol compatibility.
func (c *Client) Handshake(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.writeLocked(ctx, HandshakeIn{Type: "handshake", ProtocolVersion: ProtocolVersion}); err != nil {
		return err
	}
	line, err := c.readLineLocked(ctx)
	if err != nil {
		return err
	}
	var out HandshakeOut
	if err := json.Unmarshal(line, &out); err != nil {
		return fmt.Errorf("%s: bad handshake json: %w", ErrRuntimeUnavailable, err)
	}
	if !out.OK || out.Type != "handshake_ok" {
		code := out.ErrorCode
		if code == "" {
			code = ErrRuntimeUnavailable
		}
		return fmt.Errorf("%s: handshake rejected", code)
	}
	if out.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("%s: sidecar=%d client=%d", ErrProtocolMismatch, out.ProtocolVersion, ProtocolVersion)
	}
	return nil
}

// Call sends one tool request and waits for one response.
func (c *Client) Call(ctx context.Context, req Request) (Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if req.ProtocolVersion == 0 {
		req.ProtocolVersion = ProtocolVersion
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 60_000
	}
	if err := c.writeLocked(ctx, req); err != nil {
		return Response{}, err
	}
	line, err := c.readLineLocked(ctx)
	if err != nil {
		return Response{}, err
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return Response{}, fmt.Errorf("%s: bad response json: %w", ErrRuntimeUnavailable, err)
	}
	return resp, nil
}

// Close terminates the sidecar process.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stdin != nil {
		_ = c.stdin.Close()
		c.stdin = nil
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_, _ = c.cmd.Process.Wait()
	}
	return nil
}

func (c *Client) writeLocked(ctx context.Context, v any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	done := make(chan error, 1)
	go func() {
		_, err := c.stdin.Write(b)
		done <- err
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (c *Client) readLineLocked(ctx context.Context) ([]byte, error) {
	type result struct {
		line []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		if !c.stdout.Scan() {
			if err := c.stdout.Err(); err != nil {
				ch <- result{err: err}
				return
			}
			ch <- result{err: io.EOF}
			return
		}
		// Copy because Scanner reuses the buffer.
		line := append([]byte(nil), c.stdout.Bytes()...)
		ch <- result{line: line}
	}()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%s: %w", ErrTimeout, ctx.Err())
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("%s: %w", ErrRuntimeUnavailable, r.err)
		}
		return r.line, nil
	case <-time.After(2 * time.Minute):
		// Safety net if caller forgot a deadline.
		return nil, fmt.Errorf("%s: read stalled", ErrTimeout)
	}
}
