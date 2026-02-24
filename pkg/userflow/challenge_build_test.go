package userflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

// mockBuildAdapter implements BuildAdapter for testing.
type mockBuildAdapter struct {
	buildResults map[string]*BuildResult
	buildErrors  map[string]error
	testResults  map[string]*TestResult
	testErrors   map[string]error
	lintResults  map[string]*LintResult
	lintErrors   map[string]error
}

func newMockBuildAdapter() *mockBuildAdapter {
	return &mockBuildAdapter{
		buildResults: make(map[string]*BuildResult),
		buildErrors:  make(map[string]error),
		testResults:  make(map[string]*TestResult),
		testErrors:   make(map[string]error),
		lintResults:  make(map[string]*LintResult),
		lintErrors:   make(map[string]error),
	}
}

func (m *mockBuildAdapter) Build(
	_ context.Context, target BuildTarget,
) (*BuildResult, error) {
	if err, ok := m.buildErrors[target.Name]; ok {
		return nil, err
	}
	if r, ok := m.buildResults[target.Name]; ok {
		return r, nil
	}
	return &BuildResult{
		Target:   target.Name,
		Success:  true,
		Duration: 100 * time.Millisecond,
	}, nil
}

func (m *mockBuildAdapter) RunTests(
	_ context.Context, target TestTarget,
) (*TestResult, error) {
	if err, ok := m.testErrors[target.Name]; ok {
		return nil, err
	}
	if r, ok := m.testResults[target.Name]; ok {
		return r, nil
	}
	return &TestResult{
		TotalTests: 10,
	}, nil
}

func (m *mockBuildAdapter) Lint(
	_ context.Context, target LintTarget,
) (*LintResult, error) {
	if err, ok := m.lintErrors[target.Name]; ok {
		return nil, err
	}
	if r, ok := m.lintResults[target.Name]; ok {
		return r, nil
	}
	return &LintResult{
		Tool:    target.Name,
		Success: true,
	}, nil
}

func (m *mockBuildAdapter) Available(
	_ context.Context,
) bool {
	return true
}

// --- BuildChallenge tests ---

func TestNewBuildChallenge(t *testing.T) {
	adapter := newMockBuildAdapter()
	targets := []BuildTarget{
		{Name: "backend", Task: "build"},
	}
	ch := NewBuildChallenge(
		"BUILD-001", "Build All", "Build all targets",
		nil, adapter, targets,
	)

	assert.Equal(
		t, challenge.ID("BUILD-001"), ch.ID(),
	)
	assert.Equal(t, "Build All", ch.Name())
	assert.Equal(t, "build", ch.Category())
	assert.Empty(t, ch.Dependencies())
}

func TestBuildChallenge_Execute_AllPass(t *testing.T) {
	adapter := newMockBuildAdapter()
	adapter.buildResults["backend"] = &BuildResult{
		Target: "backend", Success: true,
		Duration: 2 * time.Second,
	}
	adapter.buildResults["frontend"] = &BuildResult{
		Target: "frontend", Success: true,
		Duration: 3 * time.Second,
	}

	targets := []BuildTarget{
		{Name: "backend", Task: "build"},
		{Name: "frontend", Task: "build"},
	}
	ch := NewBuildChallenge(
		"BUILD-002", "Build", "Build targets",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 2)
	assert.True(t, result.Assertions[0].Passed)
	assert.True(t, result.Assertions[1].Passed)
	assert.Contains(
		t, result.Assertions[0].Message, "succeeded",
	)

	durKey := "backend_build_duration"
	dur, ok := result.Metrics[durKey]
	require.True(t, ok)
	assert.Equal(t, 2.0, dur.Value)
	assert.Equal(t, "s", dur.Unit)
}

func TestBuildChallenge_Execute_OneFails(t *testing.T) {
	adapter := newMockBuildAdapter()
	adapter.buildResults["backend"] = &BuildResult{
		Target: "backend", Success: true,
		Duration: 1 * time.Second,
	}
	adapter.buildErrors["frontend"] = fmt.Errorf(
		"npm build failed",
	)

	targets := []BuildTarget{
		{Name: "backend", Task: "build"},
		{Name: "frontend", Task: "build"},
	}
	ch := NewBuildChallenge(
		"BUILD-003", "Build", "Build targets",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 2)
	assert.True(t, result.Assertions[0].Passed)
	assert.False(t, result.Assertions[1].Passed)
	assert.Contains(
		t, result.Assertions[1].Message,
		"npm build failed",
	)
}

func TestBuildChallenge_Execute_BuildNotSuccess(
	t *testing.T,
) {
	adapter := newMockBuildAdapter()
	adapter.buildResults["app"] = &BuildResult{
		Target:   "app",
		Success:  false,
		Duration: 500 * time.Millisecond,
	}

	targets := []BuildTarget{
		{Name: "app", Task: "build"},
	}
	ch := NewBuildChallenge(
		"BUILD-004", "Build", "Build app",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
}

func TestBuildChallenge_Execute_WithDeps(t *testing.T) {
	adapter := newMockBuildAdapter()
	deps := []challenge.ID{"ENV-SETUP"}
	targets := []BuildTarget{
		{Name: "app", Task: "build"},
	}
	ch := NewBuildChallenge(
		"BUILD-005", "Build", "Build app",
		deps, adapter, targets,
	)

	assert.Equal(
		t, []challenge.ID{"ENV-SETUP"},
		ch.Dependencies(),
	)
}

// --- UnitTestChallenge tests ---

func TestNewUnitTestChallenge(t *testing.T) {
	adapter := newMockBuildAdapter()
	targets := []TestTarget{
		{Name: "go-tests", Task: "test"},
	}
	ch := NewUnitTestChallenge(
		"TEST-001", "Unit Tests", "Run all unit tests",
		nil, adapter, targets,
	)

	assert.Equal(
		t, challenge.ID("TEST-001"), ch.ID(),
	)
	assert.Equal(t, "Unit Tests", ch.Name())
	assert.Equal(t, "test", ch.Category())
}

func TestUnitTestChallenge_Execute_AllPass(t *testing.T) {
	adapter := newMockBuildAdapter()
	adapter.testResults["go-tests"] = &TestResult{
		TotalTests: 42, TotalFailed: 0,
		TotalErrors: 0,
	}
	adapter.testResults["js-tests"] = &TestResult{
		TotalTests: 100, TotalFailed: 0,
		TotalErrors: 0,
	}

	targets := []TestTarget{
		{Name: "go-tests", Task: "test"},
		{Name: "js-tests", Task: "test"},
	}
	ch := NewUnitTestChallenge(
		"TEST-002", "Tests", "Run tests",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 2)
	assert.True(t, result.Assertions[0].Passed)
	assert.True(t, result.Assertions[1].Passed)

	total := result.Metrics["total_tests"]
	assert.Equal(t, 142.0, total.Value)
	assert.Equal(t, "tests", total.Unit)

	failures := result.Metrics["total_failures"]
	assert.Equal(t, 0.0, failures.Value)
}

func TestUnitTestChallenge_Execute_SomeFailures(
	t *testing.T,
) {
	adapter := newMockBuildAdapter()
	adapter.testResults["go-tests"] = &TestResult{
		TotalTests: 42, TotalFailed: 3,
		TotalErrors: 0,
	}

	targets := []TestTarget{
		{Name: "go-tests", Task: "test"},
	}
	ch := NewUnitTestChallenge(
		"TEST-003", "Tests", "Run tests",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Actual, "3 failures",
	)

	failures := result.Metrics["total_failures"]
	assert.Equal(t, 3.0, failures.Value)
}

func TestUnitTestChallenge_Execute_Error(t *testing.T) {
	adapter := newMockBuildAdapter()
	adapter.testErrors["go-tests"] = fmt.Errorf(
		"compilation error",
	)

	targets := []TestTarget{
		{Name: "go-tests", Task: "test"},
	}
	ch := NewUnitTestChallenge(
		"TEST-004", "Tests", "Run tests",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Message,
		"compilation error",
	)
}

func TestUnitTestChallenge_Execute_WithErrors(
	t *testing.T,
) {
	adapter := newMockBuildAdapter()
	adapter.testResults["tests"] = &TestResult{
		TotalTests: 10, TotalFailed: 0,
		TotalErrors: 2,
	}

	targets := []TestTarget{
		{Name: "tests", Task: "test"},
	}
	ch := NewUnitTestChallenge(
		"TEST-005", "Tests", "Run tests",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Actual, "2 errors",
	)
}

// --- LintChallenge tests ---

func TestNewLintChallenge(t *testing.T) {
	adapter := newMockBuildAdapter()
	targets := []LintTarget{
		{Name: "golangci-lint", Task: "lint"},
	}
	ch := NewLintChallenge(
		"LINT-001", "Lint", "Run linters",
		nil, adapter, targets,
	)

	assert.Equal(
		t, challenge.ID("LINT-001"), ch.ID(),
	)
	assert.Equal(t, "Lint", ch.Name())
	assert.Equal(t, "lint", ch.Category())
}

func TestLintChallenge_Execute_AllPass(t *testing.T) {
	adapter := newMockBuildAdapter()
	adapter.lintResults["golangci-lint"] = &LintResult{
		Tool: "golangci-lint", Success: true,
		Warnings: 0, Errors: 0,
	}
	adapter.lintResults["eslint"] = &LintResult{
		Tool: "eslint", Success: true,
		Warnings: 2, Errors: 0,
	}

	targets := []LintTarget{
		{Name: "golangci-lint", Task: "lint"},
		{Name: "eslint", Task: "lint"},
	}
	ch := NewLintChallenge(
		"LINT-002", "Lint All", "Run all linters",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 2)
	assert.True(t, result.Assertions[0].Passed)
	assert.True(t, result.Assertions[1].Passed)

	eslintWarnings := result.Metrics["eslint_warnings"]
	assert.Equal(t, 2.0, eslintWarnings.Value)
	assert.Equal(t, "warnings", eslintWarnings.Unit)
}

func TestLintChallenge_Execute_OneFails(t *testing.T) {
	adapter := newMockBuildAdapter()
	adapter.lintResults["golangci-lint"] = &LintResult{
		Tool: "golangci-lint", Success: false,
		Warnings: 0, Errors: 5,
	}

	targets := []LintTarget{
		{Name: "golangci-lint", Task: "lint"},
	}
	ch := NewLintChallenge(
		"LINT-003", "Lint", "Run linters",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)

	errMetric := result.Metrics["golangci-lint_errors"]
	assert.Equal(t, 5.0, errMetric.Value)
}

func TestLintChallenge_Execute_Error(t *testing.T) {
	adapter := newMockBuildAdapter()
	adapter.lintErrors["eslint"] = fmt.Errorf(
		"eslint not found",
	)

	targets := []LintTarget{
		{Name: "eslint", Task: "lint"},
	}
	ch := NewLintChallenge(
		"LINT-004", "Lint", "Run linters",
		nil, adapter, targets,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Message,
		"eslint not found",
	)
}
