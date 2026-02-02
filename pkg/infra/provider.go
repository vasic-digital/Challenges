package infra

import "context"

// InfraProvider defines the interface for infrastructure management.
// This bridges the challenge framework to container orchestration.
type InfraProvider interface {
	// EnsureRunning ensures a service is running and healthy.
	EnsureRunning(ctx context.Context, serviceName string) error
	// Release releases a service (decrements usage, may trigger idle shutdown).
	Release(ctx context.Context, serviceName string) error
	// HealthCheck checks if a service is healthy.
	HealthCheck(ctx context.Context, serviceName string) error
	// Shutdown shuts down all managed services.
	Shutdown(ctx context.Context) error
}
