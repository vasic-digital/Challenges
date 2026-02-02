package challenge

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChallenge implements Challenge for testing.
type mockChallenge struct {
	id           ID
	name         string
	description  string
	category     string
	dependencies []ID
	configErr    error
	validateErr  error
	executeErr   error
	cleanupErr   error
	result       *Result
}

func (m *mockChallenge) ID() ID               { return m.id }
func (m *mockChallenge) Name() string          { return m.name }
func (m *mockChallenge) Description() string   { return m.description }
func (m *mockChallenge) Category() string      { return m.category }
func (m *mockChallenge) Dependencies() []ID    { return m.dependencies }
func (m *mockChallenge) Configure(_ *Config) error {
	return m.configErr
}
func (m *mockChallenge) Validate(_ context.Context) error {
	return m.validateErr
}
func (m *mockChallenge) Execute(
	_ context.Context,
) (*Result, error) {
	return m.result, m.executeErr
}
func (m *mockChallenge) Cleanup(_ context.Context) error {
	return m.cleanupErr
}

func TestChallenge_Interface(t *testing.T) {
	var c Challenge = &mockChallenge{
		id:          "test-001",
		name:        "Test Challenge",
		description: "A test challenge",
		category:    "unit",
		dependencies: []ID{"dep-001"},
	}

	assert.Equal(t, ID("test-001"), c.ID())
	assert.Equal(t, "Test Challenge", c.Name())
	assert.Equal(t, "A test challenge", c.Description())
	assert.Equal(t, "unit", c.Category())
	assert.Equal(t, []ID{"dep-001"}, c.Dependencies())
}

func TestChallenge_Lifecycle(t *testing.T) {
	result := &Result{
		ChallengeID:   "test-001",
		ChallengeName: "Test",
		Status:        StatusPassed,
		StartTime:     time.Now(),
		EndTime:       time.Now(),
		Duration:      100 * time.Millisecond,
	}

	c := &mockChallenge{
		id:     "test-001",
		name:   "Test",
		result: result,
	}

	ctx := context.Background()

	require.NoError(t, c.Configure(NewConfig("test-001")))
	require.NoError(t, c.Validate(ctx))

	r, err := c.Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, StatusPassed, r.Status)

	require.NoError(t, c.Cleanup(ctx))
}

func TestChallenge_NoDependencies(t *testing.T) {
	c := &mockChallenge{
		id:           "standalone",
		dependencies: nil,
	}
	assert.Nil(t, c.Dependencies())
}

func TestChallenge_MultipleDependencies(t *testing.T) {
	deps := []ID{"dep-a", "dep-b", "dep-c"}
	c := &mockChallenge{
		id:           "multi-dep",
		dependencies: deps,
	}
	assert.Len(t, c.Dependencies(), 3)
	assert.Equal(t, deps, c.Dependencies())
}

// mockLogger implements Logger for testing.
type mockLogger struct {
	infos  []string
	warns  []string
	errors []string
	debugs []string
	closed bool
}

func (m *mockLogger) Info(msg string, _ ...any) {
	m.infos = append(m.infos, msg)
}
func (m *mockLogger) Warn(msg string, _ ...any) {
	m.warns = append(m.warns, msg)
}
func (m *mockLogger) Error(msg string, _ ...any) {
	m.errors = append(m.errors, msg)
}
func (m *mockLogger) Debug(msg string, _ ...any) {
	m.debugs = append(m.debugs, msg)
}
func (m *mockLogger) Close() error {
	m.closed = true
	return nil
}

func TestLogger_Interface(t *testing.T) {
	var l Logger = &mockLogger{}
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")
	l.Debug("debug message")
	require.NoError(t, l.Close())

	ml := l.(*mockLogger)
	assert.Equal(t, []string{"info message"}, ml.infos)
	assert.Equal(t, []string{"warn message"}, ml.warns)
	assert.Equal(t, []string{"error message"}, ml.errors)
	assert.Equal(t, []string{"debug message"}, ml.debugs)
	assert.True(t, ml.closed)
}

// mockAssertionEngine implements AssertionEngine for testing.
type mockAssertionEngine struct {
	results []AssertionResult
}

func (m *mockAssertionEngine) Evaluate(
	a AssertionDef, _ any,
) AssertionResult {
	return AssertionResult{
		Type:   a.Type,
		Target: a.Target,
		Passed: true,
	}
}

func (m *mockAssertionEngine) EvaluateAll(
	assertions []AssertionDef, _ map[string]any,
) []AssertionResult {
	if m.results != nil {
		return m.results
	}
	results := make([]AssertionResult, len(assertions))
	for i, a := range assertions {
		results[i] = AssertionResult{
			Type:   a.Type,
			Target: a.Target,
			Passed: true,
		}
	}
	return results
}

func TestAssertionEngine_Interface(t *testing.T) {
	var e AssertionEngine = &mockAssertionEngine{}

	r := e.Evaluate(
		AssertionDef{Type: "equals", Target: "status"},
		"ok",
	)
	assert.True(t, r.Passed)
	assert.Equal(t, "equals", r.Type)

	results := e.EvaluateAll(
		[]AssertionDef{
			{Type: "equals", Target: "a"},
			{Type: "not_empty", Target: "b"},
		},
		map[string]any{"a": 1, "b": "hello"},
	)
	assert.Len(t, results, 2)
	for _, ar := range results {
		assert.True(t, ar.Passed)
	}
}

func TestResult_AllPassed(t *testing.T) {
	tests := []struct {
		name       string
		assertions []AssertionResult
		want       bool
	}{
		{
			name:       "empty assertions",
			assertions: nil,
			want:       true,
		},
		{
			name: "all passed",
			assertions: []AssertionResult{
				{Passed: true},
				{Passed: true},
			},
			want: true,
		},
		{
			name: "one failed",
			assertions: []AssertionResult{
				{Passed: true},
				{Passed: false},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{Assertions: tt.assertions}
			assert.Equal(t, tt.want, r.AllPassed())
		})
	}
}

func TestResult_IsFinal(t *testing.T) {
	finals := []string{
		StatusPassed, StatusFailed, StatusSkipped,
		StatusTimedOut, StatusError,
	}
	nonFinals := []string{StatusPending, StatusRunning}

	for _, s := range finals {
		r := &Result{Status: s}
		assert.True(t, r.IsFinal(), "expected %s to be final", s)
	}
	for _, s := range nonFinals {
		r := &Result{Status: s}
		assert.False(
			t, r.IsFinal(),
			"expected %s to be non-final", s,
		)
	}
}

func TestConfig_NewConfig(t *testing.T) {
	cfg := NewConfig("my-challenge")
	assert.Equal(t, ID("my-challenge"), cfg.ChallengeID)
	assert.Equal(t, "results", cfg.ResultsDir)
	assert.Equal(t, "logs", cfg.LogsDir)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.NotNil(t, cfg.Environment)
	assert.NotNil(t, cfg.Dependencies)
}

func TestConfig_GetEnv(t *testing.T) {
	cfg := &Config{
		Environment: map[string]string{
			"MY_VAR": "my_value",
		},
	}

	assert.Equal(t, "my_value", cfg.GetEnv("MY_VAR", "default"))
	assert.Equal(t, "default", cfg.GetEnv("MISSING", "default"))

	nilCfg := &Config{}
	assert.Equal(t, "fb", nilCfg.GetEnv("ANY", "fb"))
}

func TestDefinition_Fields(t *testing.T) {
	def := Definition{
		ID:                "def-001",
		Name:              "Sample",
		Description:       "A sample definition",
		Category:          "integration",
		Dependencies:      []ID{"dep-x"},
		EstimatedDuration: "5m",
		Inputs: []Input{
			{Name: "url", Source: "env", Required: true},
		},
		Outputs: []Output{
			{
				Name:        "response",
				Type:        "json",
				Description: "API response",
			},
		},
		Assertions: []AssertionDef{
			{
				Type:    "equals",
				Target:  "status",
				Value:   200,
				Message: "status should be 200",
			},
		},
		Metrics: []string{"latency_ms", "throughput"},
	}

	assert.Equal(t, ID("def-001"), def.ID)
	assert.Len(t, def.Inputs, 1)
	assert.True(t, def.Inputs[0].Required)
	assert.Len(t, def.Outputs, 1)
	assert.Len(t, def.Assertions, 1)
	assert.Equal(t, "equals", def.Assertions[0].Type)
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", StatusPending)
	assert.Equal(t, "running", StatusRunning)
	assert.Equal(t, "passed", StatusPassed)
	assert.Equal(t, "failed", StatusFailed)
	assert.Equal(t, "skipped", StatusSkipped)
	assert.Equal(t, "timed_out", StatusTimedOut)
	assert.Equal(t, "error", StatusError)
}
