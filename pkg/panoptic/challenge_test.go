package panoptic

import (
	"context"
	"testing"
	"time"

	"digital.vasic.challenges/pkg/assertion"
	"digital.vasic.challenges/pkg/challenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAdapter implements PanopticAdapter for testing.
type mockAdapter struct {
	available   bool
	runResult   *PanopticRunResult
	runErr      error
	versionStr  string
	versionErr  error
}

func (m *mockAdapter) Run(
	_ context.Context,
	_ string,
	_ ...RunOption,
) (*PanopticRunResult, error) {
	return m.runResult, m.runErr
}

func (m *mockAdapter) Version(
	_ context.Context,
) (string, error) {
	return m.versionStr, m.versionErr
}

func (m *mockAdapter) Available(_ context.Context) bool {
	return m.available
}

func TestNewPanopticChallenge(t *testing.T) {
	adapter := &mockAdapter{available: true}
	c := NewPanopticChallenge(
		"test-001", "Test", "A test", "ui",
		nil, adapter,
		[]challenge.AssertionDef{
			{Type: "all_apps_passed", Target: "all_apps_passed"},
		},
		WithConfigPath("/path/to/config.yaml"),
	)

	assert.Equal(t, challenge.ID("test-001"), c.ID())
	assert.Equal(t, "Test", c.Name())
	assert.Equal(t, "A test", c.Description())
	assert.Equal(t, "ui", c.Category())
	assert.Equal(t, "/path/to/config.yaml", c.configPath)
}

func TestPanopticChallenge_Validate_NoAdapter(t *testing.T) {
	c := NewPanopticChallenge(
		"test-001", "Test", "desc", "ui",
		nil, nil, nil,
		WithConfigPath("/path/config.yaml"),
	)

	cfg := challenge.NewConfig("test-001")
	cfg.ResultsDir = t.TempDir()
	cfg.LogsDir = t.TempDir()
	require.NoError(t, c.Configure(cfg))

	err := c.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestPanopticChallenge_Validate_NotAvailable(t *testing.T) {
	adapter := &mockAdapter{available: false}
	c := NewPanopticChallenge(
		"test-001", "Test", "desc", "ui",
		nil, adapter, nil,
		WithConfigPath("/path/config.yaml"),
	)

	cfg := challenge.NewConfig("test-001")
	cfg.ResultsDir = t.TempDir()
	cfg.LogsDir = t.TempDir()
	require.NoError(t, c.Configure(cfg))

	err := c.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestPanopticChallenge_Validate_NoConfig(t *testing.T) {
	adapter := &mockAdapter{available: true}
	c := NewPanopticChallenge(
		"test-001", "Test", "desc", "ui",
		nil, adapter, nil,
	)

	cfg := challenge.NewConfig("test-001")
	cfg.ResultsDir = t.TempDir()
	cfg.LogsDir = t.TempDir()
	require.NoError(t, c.Configure(cfg))

	err := c.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no config")
}

func TestPanopticChallenge_Execute_Success(t *testing.T) {
	adapter := &mockAdapter{
		available: true,
		runResult: &PanopticRunResult{
			ExitCode: 0,
			Apps: []AppResult{
				{
					Name:       "Admin",
					Success:    true,
					DurationMs: 5000,
				},
			},
			Screenshots: []string{"/tmp/a.png"},
			Duration:    5 * time.Second,
			Stdout:      "all passed",
		},
	}

	engine := assertion.NewEngine()
	require.NoError(t, RegisterEvaluators(engine))

	c := NewPanopticChallenge(
		"test-001", "Test", "desc", "ui",
		nil, adapter,
		[]challenge.AssertionDef{
			{
				Type:    "all_apps_passed",
				Target:  "all_apps_passed",
				Message: "all apps must pass",
			},
			{
				Type:    "app_count",
				Target:  "app_count",
				Value:   1,
				Message: "expected 1 app",
			},
		},
		WithConfigPath("/path/config.yaml"),
	)

	cfg := challenge.NewConfig("test-001")
	cfg.ResultsDir = t.TempDir()
	cfg.LogsDir = t.TempDir()
	require.NoError(t, c.Configure(cfg))
	c.SetAssertionEngine(NewEngineAdapter(engine))

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Len(t, result.Assertions, 2)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed, "assertion %s failed: %s",
			a.Target, a.Message,
		)
	}
	assert.Contains(t, result.Outputs, "exit_code")
	assert.Contains(t, result.Outputs, "stdout")
}

func TestPanopticChallenge_Execute_Failure(t *testing.T) {
	adapter := &mockAdapter{
		available: true,
		runResult: &PanopticRunResult{
			ExitCode: 1,
			Apps: []AppResult{
				{Name: "Admin", Success: false},
			},
			Duration: 3 * time.Second,
		},
	}

	engine := assertion.NewEngine()
	require.NoError(t, RegisterEvaluators(engine))

	c := NewPanopticChallenge(
		"test-002", "Fail Test", "desc", "ui",
		nil, adapter,
		[]challenge.AssertionDef{
			{
				Type:    "all_apps_passed",
				Target:  "all_apps_passed",
				Message: "all apps must pass",
			},
		},
		WithConfigPath("/path/config.yaml"),
	)

	cfg := challenge.NewConfig("test-002")
	cfg.ResultsDir = t.TempDir()
	cfg.LogsDir = t.TempDir()
	require.NoError(t, c.Configure(cfg))
	c.SetAssertionEngine(NewEngineAdapter(engine))

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
}

func TestPanopticChallenge_Execute_AdapterError(t *testing.T) {
	adapter := &mockAdapter{
		available: true,
		runErr:    assert.AnError,
	}

	c := NewPanopticChallenge(
		"test-003", "Error Test", "desc", "ui",
		nil, adapter, nil,
		WithConfigPath("/path/config.yaml"),
	)

	cfg := challenge.NewConfig("test-003")
	cfg.ResultsDir = t.TempDir()
	cfg.LogsDir = t.TempDir()
	require.NoError(t, c.Configure(cfg))

	result, err := c.Execute(context.Background())
	require.NoError(t, err) // Error is in result, not returned

	assert.Equal(t, challenge.StatusError, result.Status)
	assert.Contains(t, result.Error, "panoptic execution error")
}

func TestPanopticChallenge_Execute_WithConfigBuilder(t *testing.T) {
	adapter := &mockAdapter{
		available: true,
		runResult: &PanopticRunResult{
			ExitCode: 0,
			Duration: 1 * time.Second,
		},
	}

	builder := NewConfigBuilder("Generated", "./out")
	builder.AddWebApp("Admin", "http://localhost:3001", 60).
		Navigate("login", "http://localhost:3001/login").
		Done()

	c := NewPanopticChallenge(
		"test-004", "Builder Test", "desc", "ui",
		nil, adapter, nil,
		WithConfigBuilder(builder),
	)

	cfg := challenge.NewConfig("test-004")
	cfg.ResultsDir = t.TempDir()
	cfg.LogsDir = t.TempDir()
	require.NoError(t, c.Configure(cfg))

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	// Should not error since builder writes config.
	assert.NotEqual(t, challenge.StatusError, result.Status)
}

func TestPanopticChallenge_Execute_NoAssertions(t *testing.T) {
	adapter := &mockAdapter{
		available: true,
		runResult: &PanopticRunResult{
			ExitCode: 1,
			Duration: 1 * time.Second,
		},
	}

	c := NewPanopticChallenge(
		"test-005", "No Assertions", "desc", "ui",
		nil, adapter, nil,
		WithConfigPath("/path/config.yaml"),
	)

	cfg := challenge.NewConfig("test-005")
	cfg.ResultsDir = t.TempDir()
	cfg.LogsDir = t.TempDir()
	require.NoError(t, c.Configure(cfg))

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	// No assertions + non-zero exit = failed.
	assert.Equal(t, challenge.StatusFailed, result.Status)
}
