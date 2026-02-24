package yole

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// ProcessCLIAdapter manages JVM/native process lifecycle.
type ProcessCLIAdapter struct {
	cmd     *exec.Cmd
	process *os.Process
}

// NewProcessCLIAdapter creates a new ProcessCLIAdapter.
func NewProcessCLIAdapter() *ProcessCLIAdapter {
	return &ProcessCLIAdapter{}
}

// LaunchJVM starts a JVM application from a JAR file.
func (p *ProcessCLIAdapter) LaunchJVM(
	ctx context.Context, jarPath string, args ...string,
) error {
	cmdArgs := []string{"-jar", jarPath}
	cmdArgs = append(cmdArgs, args...)

	p.cmd = exec.CommandContext(ctx, "java", cmdArgs...)
	p.cmd.Dir = filepath.Dir(jarPath)

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf(
			"failed to launch JVM app: %w", err,
		)
	}
	p.process = p.cmd.Process
	return nil
}

// IsRunning checks if the managed process is still alive.
func (p *ProcessCLIAdapter) IsRunning() bool {
	if p.process == nil {
		return false
	}
	err := p.process.Signal(syscall.Signal(0))
	return err == nil
}

// WaitForReady waits until the process is running or timeout.
func (p *ProcessCLIAdapter) WaitForReady(
	ctx context.Context, timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.IsRunning() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf(
		"process did not start within %v", timeout,
	)
}

// Stop gracefully terminates the process.
func (p *ProcessCLIAdapter) Stop() error {
	if p.process == nil {
		return nil
	}

	if err := p.process.Signal(
		syscall.SIGTERM,
	); err != nil {
		return nil
	}

	done := make(chan error, 1)
	go func() {
		_, err := p.process.Wait()
		done <- err
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return p.process.Kill()
	}
}
