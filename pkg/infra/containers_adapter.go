package infra

import (
	"context"
	"fmt"
)

// ContainersAdapter is the default InfraProvider implementation that
// bridges to the digital.vasic.containers module. Users provide
// the actual container lifecycle operations through callback functions
// to avoid hard-coding the containers module dependency.
type ContainersAdapter struct {
	ensureFunc func(ctx context.Context, name string) error
	releaseFunc func(ctx context.Context, name string) error
	healthFunc  func(ctx context.Context, name string) error
	shutdownFunc func(ctx context.Context) error
}

// ContainersAdapterOption configures the ContainersAdapter.
type ContainersAdapterOption func(*ContainersAdapter)

// WithEnsureFunc sets the function called to ensure a service is running.
func WithEnsureFunc(fn func(ctx context.Context, name string) error) ContainersAdapterOption {
	return func(a *ContainersAdapter) { a.ensureFunc = fn }
}

// WithReleaseFunc sets the function called to release a service.
func WithReleaseFunc(fn func(ctx context.Context, name string) error) ContainersAdapterOption {
	return func(a *ContainersAdapter) { a.releaseFunc = fn }
}

// WithHealthFunc sets the function called to health check a service.
func WithHealthFunc(fn func(ctx context.Context, name string) error) ContainersAdapterOption {
	return func(a *ContainersAdapter) { a.healthFunc = fn }
}

// WithShutdownFunc sets the function called to shutdown all services.
func WithShutdownFunc(fn func(ctx context.Context) error) ContainersAdapterOption {
	return func(a *ContainersAdapter) { a.shutdownFunc = fn }
}

// NewContainersAdapter creates a new ContainersAdapter with the given options.
func NewContainersAdapter(opts ...ContainersAdapterOption) *ContainersAdapter {
	a := &ContainersAdapter{}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *ContainersAdapter) EnsureRunning(ctx context.Context, serviceName string) error {
	if a.ensureFunc == nil {
		return fmt.Errorf("ensure function not configured for service %s", serviceName)
	}
	return a.ensureFunc(ctx, serviceName)
}

func (a *ContainersAdapter) Release(ctx context.Context, serviceName string) error {
	if a.releaseFunc == nil {
		return nil // Release is optional
	}
	return a.releaseFunc(ctx, serviceName)
}

func (a *ContainersAdapter) HealthCheck(ctx context.Context, serviceName string) error {
	if a.healthFunc == nil {
		return fmt.Errorf("health check function not configured for service %s", serviceName)
	}
	return a.healthFunc(ctx, serviceName)
}

func (a *ContainersAdapter) Shutdown(ctx context.Context) error {
	if a.shutdownFunc == nil {
		return nil // Shutdown is optional
	}
	return a.shutdownFunc(ctx)
}
