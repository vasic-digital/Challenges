package userflow

import (
	"context"
	"time"
)

// ProcessAdapter defines the interface for managing long-running
// processes (servers, services, background workers). Used to
// start and stop services needed for integration testing.
type ProcessAdapter interface {
	// Launch starts the process with the given configuration.
	Launch(
		ctx context.Context, config ProcessConfig,
	) error

	// IsRunning returns true if the process is currently
	// running.
	IsRunning() bool

	// WaitForReady blocks until the process signals it is
	// ready to accept connections, up to the given timeout.
	WaitForReady(
		ctx context.Context, timeout time.Duration,
	) error

	// Stop terminates the running process gracefully.
	Stop() error
}
