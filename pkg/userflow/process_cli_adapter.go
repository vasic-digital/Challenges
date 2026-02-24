package userflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ProcessCLIAdapter implements ProcessAdapter by managing
// a subprocess via os/exec. It supports graceful shutdown
// (SIGTERM then SIGKILL) and readiness polling.
type ProcessCLIAdapter struct {
	mu   sync.Mutex
	cmd  *exec.Cmd
	done chan struct{}
}

// Compile-time interface check.
var _ ProcessAdapter = (*ProcessCLIAdapter)(nil)

// NewProcessCLIAdapter creates a new ProcessCLIAdapter.
func NewProcessCLIAdapter() *ProcessCLIAdapter {
	return &ProcessCLIAdapter{}
}

// Launch starts the process described by config. The process
// inherits the given environment variables and working directory.
func (a *ProcessCLIAdapter) Launch(
	ctx context.Context, config ProcessConfig,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cmd != nil && a.cmd.Process != nil {
		return fmt.Errorf(
			"process already running (pid %d)",
			a.cmd.Process.Pid,
		)
	}

	cmd := exec.CommandContext(ctx, config.Command, config.Args...)
	if config.WorkDir != "" {
		cmd.Dir = config.WorkDir
	}
	if len(config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range config.Env {
			cmd.Env = append(
				cmd.Env, fmt.Sprintf("%s=%s", k, v),
			)
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch process: %w", err)
	}

	a.cmd = cmd
	a.done = make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(a.done)
	}()

	return nil
}

// IsRunning returns true if the process is still alive,
// checked via a signal(0) probe.
func (a *ProcessCLIAdapter) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cmd == nil || a.cmd.Process == nil {
		return false
	}

	select {
	case <-a.done:
		return false
	default:
	}

	err := a.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// WaitForReady polls IsRunning every 200ms until the process
// is detected as running, or the timeout expires.
func (a *ProcessCLIAdapter) WaitForReady(
	ctx context.Context, timeout time.Duration,
) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		if a.IsRunning() {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"wait for ready: %w", ctx.Err(),
			)
		case <-deadline:
			return fmt.Errorf(
				"wait for ready: timed out after %s",
				timeout,
			)
		case <-ticker.C:
			// continue polling
		}
	}
}

// Stop sends SIGTERM, waits up to 5 seconds, then sends
// SIGKILL if the process has not exited.
func (a *ProcessCLIAdapter) Stop() error {
	a.mu.Lock()
	cmd := a.cmd
	done := a.done
	a.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Send SIGTERM for graceful shutdown.
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited.
		if cmd.ProcessState != nil {
			return nil
		}
		return fmt.Errorf("send SIGTERM: %w", err)
	}

	// Wait up to 5 seconds for graceful exit.
	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
	}

	// Force kill.
	if err := cmd.Process.Kill(); err != nil {
		if cmd.ProcessState != nil {
			return nil
		}
		return fmt.Errorf("send SIGKILL: %w", err)
	}

	<-done
	return nil
}
