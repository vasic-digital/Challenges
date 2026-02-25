// Package container provides container verification utilities
// for the Challenges framework.
package container

import (
	"context"
	"testing"

	"digital.vasic.challenges/pkg/challenge"
)

// mockLogger is a simple mock for challenge.Logger.
type mockLogger struct {
	logs []string
}

func (m *mockLogger) Log(msg string) {
	m.logs = append(m.logs, msg)
}

func (m *mockLogger) Logf(format string, args ...interface{}) {
	m.logs = append(m.logs, format)
}

// TestNewVerifier tests the creation of a new Verifier.
func TestNewVerifier(t *testing.T) {
	logger := &mockLogger{}
	verifier := NewVerifier(logger)

	if verifier == nil {
		t.Fatal("NewVerifier returned nil")
	}

	if len(verifier.services) == 0 {
		t.Error("Expected default services to be set")
	}

	if verifier.logger != logger {
		t.Error("Expected logger to be set")
	}
}

// TestDefaultServices tests the default service configuration.
func TestDefaultServices(t *testing.T) {
	services := DefaultServices()

	if len(services) != 3 {
		t.Errorf("Expected 3 default services, got %d", len(services))
	}

	// Check for expected services
	serviceNames := make(map[string]bool)
	for _, svc := range services {
		serviceNames[svc.Name] = true
	}

	expectedServices := []string{"postgres", "backend", "freeswitch"}
	for _, expected := range expectedServices {
		if !serviceNames[expected] {
			t.Errorf("Expected service %s not found in defaults", expected)
		}
	}
}

// TestVerifierWithServices tests custom service configuration.
func TestVerifierWithServices(t *testing.T) {
	verifier := NewVerifier(nil)
	
	customServices := []ServiceConfig{
		{
			Type:    ServicePostgres,
			Name:    "custom-postgres",
			Host:    "127.0.0.1",
			Port:    5433,
			Timeout: 10 * 1000000000, // 10 seconds in nanoseconds
		},
	}

	verifier.WithServices(customServices)

	if len(verifier.services) != 1 {
		t.Errorf("Expected 1 custom service, got %d", len(verifier.services))
	}

	if verifier.services[0].Name != "custom-postgres" {
		t.Errorf("Expected service name 'custom-postgres', got '%s'", verifier.services[0].Name)
	}
}

// TestFindContainersDir tests the containers directory discovery.
func TestFindContainersDir(t *testing.T) {
	// This test may fail in CI environments where the directory doesn't exist
	dir := findContainersDir()
	
	// We can't assert the exact path, but we can verify it returns something
	// or an empty string if not found
	t.Logf("Found containers directory: %s", dir)
}

// TestPreConditionCheck tests the full pre-condition check.
func TestPreConditionCheck(t *testing.T) {
	logger := &mockLogger{}
	ctx := context.Background()

	// This test will likely fail if containers are not running
	// In a real test environment, you'd mock the container checks
	err := PreConditionCheck(ctx, logger)
	
	if err != nil {
		t.Logf("PreConditionCheck returned error (expected if containers not running): %v", err)
	}

	// Verify that logging occurred
	if len(logger.logs) == 0 {
		t.Error("Expected logging to occur during pre-condition check")
	}
}

// TestServiceConfig validates service configuration.
func TestServiceConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   ServiceConfig
		wantErr  bool
	}{
		{
			name: "valid postgres config",
			config: ServiceConfig{
				Type:    ServicePostgres,
				Name:    "postgres",
				Host:    "localhost",
				Port:    5432,
				Timeout: 30000000000,
			},
			wantErr: false,
		},
		{
			name: "valid backend config",
			config: ServiceConfig{
				Type:      ServiceBackend,
				Name:      "backend",
				Host:      "localhost",
				Port:      8090,
				HealthURL: "http://localhost:8090/health",
				Timeout:   30000000000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate configuration
			if tt.config.Port == 0 {
				t.Error("Port must not be zero")
			}
			if tt.config.Host == "" {
				t.Error("Host must not be empty")
			}
			if tt.config.Name == "" {
				t.Error("Name must not be empty")
			}
		})
	}
}

// Integration test that requires running containers
func TestIntegration_VerifyRunningContainers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := &mockLogger{}
	verifier := NewVerifier(logger)
	ctx := context.Background()

	err := verifier.Verify(ctx)
	if err != nil {
		t.Skipf("Integration test skipped (containers not running): %v", err)
	}
}
