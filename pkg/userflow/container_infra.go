package userflow

import (
	"context"
	"fmt"
	"time"

	containers_compose "digital.vasic.containers/pkg/compose"
	containers_event "digital.vasic.containers/pkg/event"
	containers_health "digital.vasic.containers/pkg/health"
	containers_logging "digital.vasic.containers/pkg/logging"
	containers_runtime "digital.vasic.containers/pkg/runtime"
	containers_serviceregistry "digital.vasic.containers/pkg/serviceregistry"
)

// TestEnvironment manages containerized test infrastructure.
// It composes Containers module subsystems for lifecycle
// management of platform-specific service groups.
type TestEnvironment struct {
	runtime     containers_runtime.ContainerRuntime
	compose     containers_compose.ComposeOrchestrator
	health      containers_health.HealthChecker
	registry    *containers_serviceregistry.ServiceRegistry
	eventBus    containers_event.EventBus
	logger      containers_logging.Logger
	groups      []PlatformGroup
	projectName string
	composeFile string
}

// PlatformGroup defines a set of containers that run together
// within a resource budget.
type PlatformGroup struct {
	Name          string                           `json:"name"`
	Services      []string                         `json:"services"`
	CPULimit      float64                          `json:"cpu_limit"`
	MemoryMB      int                              `json:"memory_mb"`
	HealthTargets []containers_health.HealthTarget `json:"-"`
	ComposeFile   string                           `json:"compose_file,omitempty"`
}

// TestEnvironmentOption configures a TestEnvironment.
type TestEnvironmentOption func(*TestEnvironment)

// WithComposeFile sets the docker-compose file path.
func WithComposeFile(path string) TestEnvironmentOption {
	return func(te *TestEnvironment) {
		te.composeFile = path
	}
}

// WithProjectName sets the compose project name.
func WithProjectName(name string) TestEnvironmentOption {
	return func(te *TestEnvironment) {
		te.projectName = name
	}
}

// WithPlatformGroups sets the platform groups.
func WithPlatformGroups(
	groups []PlatformGroup,
) TestEnvironmentOption {
	return func(te *TestEnvironment) {
		te.groups = groups
	}
}

// WithLogger sets the logger for the test environment.
func WithLogger(
	l containers_logging.Logger,
) TestEnvironmentOption {
	return func(te *TestEnvironment) {
		te.logger = l
	}
}

// NewTestEnvironment creates a new TestEnvironment with
// auto-detected runtime (Podman-first) and configured
// subsystems from the Containers module.
func NewTestEnvironment(
	opts ...TestEnvironmentOption,
) (*TestEnvironment, error) {
	// Use a 10-second timeout to avoid hanging when no container
	// runtime is available (e.g., podman/docker exec blocks).
	ctx, cancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	defer cancel()

	// 1. Auto-detect container runtime (Podman-first).
	rt, err := containers_runtime.AutoDetect(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"auto-detect runtime: %w", err,
		)
	}

	te := &TestEnvironment{
		runtime:     rt,
		logger:      containers_logging.NoopLogger{},
		projectName: "userflow-test",
		composeFile: "docker-compose.test.yml",
	}

	// Apply options before creating subsystems that depend
	// on configured values.
	for _, opt := range opts {
		opt(te)
	}

	// 2. Create compose orchestrator.
	orch, err := containers_compose.NewDefaultOrchestrator(
		".", te.logger,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create compose orchestrator: %w", err,
		)
	}
	te.compose = orch

	// 3. Create health checker with built-in TCP/HTTP/gRPC.
	te.health = containers_health.NewDefaultChecker()

	// 4. Create service registry.
	te.registry = containers_serviceregistry.New()

	// 5. Create event bus with buffered delivery.
	te.eventBus = containers_event.NewEventBus(64)

	return te, nil
}

// Setup starts containers for a given platform group, waits
// for health checks, and registers services in the registry.
func (te *TestEnvironment) Setup(
	ctx context.Context,
	group PlatformGroup,
) error {
	composeFile := te.composeFile
	if group.ComposeFile != "" {
		composeFile = group.ComposeFile
	}

	project := containers_compose.ComposeProject{
		Name:     te.projectName,
		File:     composeFile,
		Services: group.Services,
	}

	te.logger.Info(
		"starting platform group %s with %d services",
		group.Name, len(group.Services),
	)

	if err := te.compose.Up(ctx, project); err != nil {
		return fmt.Errorf(
			"compose up for group %s: %w",
			group.Name, err,
		)
	}

	// Run health checks on all targets.
	if len(group.HealthTargets) > 0 {
		results := te.health.CheckAll(
			ctx, group.HealthTargets,
		)
		for _, r := range results {
			if !r.Healthy {
				return fmt.Errorf(
					"health check failed for %s: %s",
					r.Target, r.Error,
				)
			}
		}
	}

	te.logger.Info(
		"platform group %s is ready", group.Name,
	)
	return nil
}

// Teardown stops containers for a platform group and
// unregisters services from the registry.
func (te *TestEnvironment) Teardown(
	ctx context.Context,
	group PlatformGroup,
) error {
	composeFile := te.composeFile
	if group.ComposeFile != "" {
		composeFile = group.ComposeFile
	}

	project := containers_compose.ComposeProject{
		Name:     te.projectName,
		File:     composeFile,
		Services: group.Services,
	}

	te.logger.Info(
		"tearing down platform group %s", group.Name,
	)

	if err := te.compose.Down(ctx, project); err != nil {
		return fmt.Errorf(
			"compose down for group %s: %w",
			group.Name, err,
		)
	}

	// Unregister services from the registry.
	for _, svc := range group.Services {
		te.registry.Unregister(svc)
	}

	return nil
}

// SetupAll starts all platform groups sequentially.
func (te *TestEnvironment) SetupAll(
	ctx context.Context,
) error {
	for _, group := range te.groups {
		if err := te.Setup(ctx, group); err != nil {
			return fmt.Errorf(
				"setup group %s: %w", group.Name, err,
			)
		}
	}
	return nil
}

// TeardownAll stops all platform groups in reverse order.
func (te *TestEnvironment) TeardownAll(
	ctx context.Context,
) error {
	var firstErr error
	for i := len(te.groups) - 1; i >= 0; i-- {
		if err := te.Teardown(
			ctx, te.groups[i],
		); err != nil {
			te.logger.Error(
				"teardown group %s failed: %v",
				te.groups[i].Name, err,
			)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// Runtime returns the container runtime.
func (te *TestEnvironment) Runtime() containers_runtime.ContainerRuntime {
	return te.runtime
}

// Registry returns the service registry.
func (te *TestEnvironment) Registry() *containers_serviceregistry.ServiceRegistry {
	return te.registry
}

// EventBus returns the event bus.
func (te *TestEnvironment) EventBus() containers_event.EventBus {
	return te.eventBus
}
