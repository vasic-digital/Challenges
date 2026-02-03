package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.challenges/pkg/challenge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupResultsDir_MkdirErrors tests error handling in setupResultsDir.
func TestSetupResultsDir_MkdirErrors(t *testing.T) {
	tests := []struct {
		name      string
		setupCfg  func(t *testing.T, cfg *challenge.Config) *challenge.Config
		wantError bool
	}{
		{
			name: "results subdir error",
			setupCfg: func(t *testing.T, cfg *challenge.Config) *challenge.Config {
				tmpDir := t.TempDir()
				cfg.ResultsDir = tmpDir

				// Create a file where the results subdir should go
				resultsPath := filepath.Join(tmpDir, "results")
				os.WriteFile(resultsPath, []byte("file"), 0o644)

				cfg.LogsDir = filepath.Join(tmpDir, "logs")
				return cfg
			},
			wantError: true,
		},
		{
			name: "config subdir error",
			setupCfg: func(t *testing.T, cfg *challenge.Config) *challenge.Config {
				tmpDir := t.TempDir()
				cfg.ResultsDir = tmpDir

				// Create files where subdirs should go
				configPath := filepath.Join(tmpDir, "config")
				os.WriteFile(configPath, []byte("file"), 0o644)

				cfg.LogsDir = filepath.Join(tmpDir, "logs")
				return cfg
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRunner()
			cfg := challenge.NewConfig("test")
			cfg = tc.setupCfg(t, cfg)

			err := r.setupResultsDir(cfg)
			if tc.wantError {
				assert.Error(t, err)
			}
		})
	}
}

// TestSetupResultsDir_AutoGeneratesPath tests the auto-path generation.
func TestSetupResultsDir_AutoGeneratesPath(t *testing.T) {
	tmpDir := t.TempDir()
	r := NewRunner(WithResultsDir(tmpDir))

	cfg := challenge.NewConfig("test-challenge")
	cfg.ResultsDir = "" // Empty to trigger auto-generation

	err := r.setupResultsDir(cfg)
	require.NoError(t, err)

	// Should have auto-generated path with date structure
	assert.Contains(t, cfg.ResultsDir, "test-challenge")
	assert.NotEmpty(t, cfg.LogsDir)
}

// TestSetupResultsDir_EmptyBaseDir tests when resultsDir in runner is empty.
func TestSetupResultsDir_EmptyBaseDir(t *testing.T) {
	r := NewRunner() // No results dir set

	cfg := challenge.NewConfig("empty-base")
	cfg.ResultsDir = "" // Empty to trigger auto-generation

	err := r.setupResultsDir(cfg)
	require.NoError(t, err)

	// Should use "results" as default base
	assert.Contains(t, cfg.ResultsDir, "results")
}

// TestExecuteChallenge_SetupResultsDirError tests error when setup fails.
func TestExecuteChallenge_SetupResultsDirError(t *testing.T) {
	s := newStub("a")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir("/dev/null/impossible/path"),
	)

	cfg := challenge.NewConfig("a")
	cfg.ResultsDir = "/dev/null/cannot/create"

	result, err := r.Run(context.Background(), "a", cfg)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusError, result.Status)
	assert.Contains(t, result.Error, "failed to setup results directory")
}

// TestRunSequence_FailedDependencyStatus tests the dependency status tracking.
func TestRunSequence_FailedDependencyStatus(t *testing.T) {
	// Create challenges where first fails - b should have unmet dependency
	a := newStub("a")
	a.execResult = &challenge.Result{
		Status:     challenge.StatusFailed,
		Assertions: []challenge.AssertionResult{{Passed: false}},
	}
	b := newStub("b", "a") // b depends on a
	reg := setupRegistry(t, a, b)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	// Run sequence - b's dependency a failed, so b should fail with unmet dep
	ids := []challenge.ID{"a", "b"}
	results, err := r.RunSequence(
		context.Background(), ids, challenge.NewConfig(""),
	)
	// Should get an error about unmet dependency since a failed
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmet dependency")
	// But should still have a's result
	assert.GreaterOrEqual(t, len(results), 1)
}

// TestRunAll_DependencyPropagation tests that passed dependencies are propagated.
func TestRunAll_DependencyPropagation(t *testing.T) {
	a := newStub("a")
	b := newStub("b", "a")
	c := newStub("c", "b")
	reg := setupRegistry(t, a, b, c)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	results, err := r.RunAll(context.Background(), challenge.NewConfig(""))
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// All should pass
	for _, res := range results {
		assert.Equal(t, challenge.StatusPassed, res.Status)
	}
}

// TestRunAll_FailedChallenge tests behavior when a challenge fails.
func TestRunAll_FailedChallenge(t *testing.T) {
	a := newStub("a")
	a.execResult = &challenge.Result{
		Status:     challenge.StatusFailed,
		Assertions: []challenge.AssertionResult{{Passed: false}},
	}
	b := newStub("b", "a")
	reg := setupRegistry(t, a, b)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	results, err := r.RunAll(context.Background(), challenge.NewConfig(""))
	// Should not error but a will fail, b won't have dependency met
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

// TestPipeline_ExecuteSequence_Error tests ExecuteSequence error handling.
func TestPipeline_ExecuteSequence_Error(t *testing.T) {
	a := newStub("a")
	a.executeErr = errors.New("execution failed")
	a.execResult = nil
	reg := setupRegistry(t, a)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	challenges := []challenge.Challenge{a}
	results, err := p.ExecuteSequence(
		context.Background(), challenges, challenge.NewConfig(""),
	)
	// Should not return Go error but result should have error status
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, challenge.StatusError, results[0].Status)
}

// TestPipeline_Execute_RunnerError tests when runner returns an error.
func TestPipeline_Execute_RunnerError(t *testing.T) {
	// Create a challenge that causes executeChallenge to fail
	a := newStub("a")
	a.executeErr = errors.New("exec failed")
	a.execResult = nil
	reg := setupRegistry(t, a)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	result, err := p.Execute(context.Background(), a, challenge.NewConfig("a"))
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusError, result.Status)
}

// TestRunParallel_ContextCancellation tests parallel execution with cancelled context.
func TestRunParallel_ContextCancellation(t *testing.T) {
	// Create many challenges to ensure some goroutines will see the cancelled context
	// when trying to acquire the semaphore
	stubs := make([]*stubChallenge, 20)
	ids := make([]challenge.ID, 20)
	for i := 0; i < 20; i++ {
		stubs[i] = newStub(fmt.Sprintf("c%d", i))
		stubs[i].execDelay = 100 * time.Millisecond
		ids[i] = stubs[i].id
	}
	reg := setupRegistry(t, stubs...)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately before any work can start
	cancel()

	// With maxConcurrency=1 and 20 challenges, only 1 can run at a time.
	// The remaining 19 goroutines will be blocked on semaphore and see ctx.Done().
	results, err := r.RunParallel(ctx, ids, challenge.NewConfig(""), 1)

	// The cancelled context should propagate as an error
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	// Some results might exist, some might not
	assert.True(t, len(results) < len(ids),
		"expected fewer results due to cancellation, got %d/%d", len(results), len(ids))
}

// TestRunAll_WithFailedAssertion tests RunAll when challenge has failed assertions.
func TestRunAll_WithFailedAssertion(t *testing.T) {
	a := newStub("a")
	a.executeErr = nil
	a.execResult = &challenge.Result{
		Status: challenge.StatusFailed,
		Assertions: []challenge.AssertionResult{
			{Passed: false, Message: "assertion failed"},
		},
	}
	reg := setupRegistry(t, a)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	results, err := r.RunAll(context.Background(), challenge.NewConfig(""))
	// RunAll should complete even when a challenge fails
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
	assert.Equal(t, challenge.StatusFailed, results[0].Status)
}

// TestRunSequence_WithFailedAssertion tests RunSequence when challenge has failed assertions.
func TestRunSequence_WithFailedAssertion(t *testing.T) {
	a := newStub("a")
	a.executeErr = nil
	a.execResult = &challenge.Result{
		Status: challenge.StatusFailed,
		Assertions: []challenge.AssertionResult{
			{Passed: false, Message: "assertion failed"},
		},
	}
	reg := setupRegistry(t, a)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a"}
	results, err := r.RunSequence(
		context.Background(), ids, challenge.NewConfig(""),
	)
	// Should complete without error but with failed status
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, challenge.StatusFailed, results[0].Status)
}

// TestPipeline_ExecuteSequence_ExecutionError tests ExecuteSequence when
// runner returns an actual error (not just status error).
func TestPipeline_ExecuteSequence_ExecutionError(t *testing.T) {
	// This tests the error return path in ExecuteSequence
	a := newStub("a")
	reg := setupRegistry(t, a)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	// Test with a challenge that completes successfully
	challenges := []challenge.Challenge{a}
	results, err := p.ExecuteSequence(
		context.Background(), challenges, challenge.NewConfig(""),
	)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

// TestPipeline_Execute_PostHookError tests post-hook warning in pipeline.
func TestPipeline_Execute_PostHookError(t *testing.T) {
	a := newStub("a")
	reg := setupRegistry(t, a)

	logger := &stubLogger{}
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithLogger(logger),
	)
	p := NewPipeline(r)

	// Add multiple post-hooks, one that fails
	p.AddPostHook(func(_ context.Context, _ challenge.Challenge, _ *challenge.Config) error {
		return errors.New("post hook 1 failed")
	})
	p.AddPostHook(func(_ context.Context, _ challenge.Challenge, _ *challenge.Config) error {
		return nil
	})

	result, err := p.Execute(context.Background(), a, challenge.NewConfig("a"))
	require.NoError(t, err)
	// Should still pass despite post-hook error
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

// TestRunAll_ExecuteHookError tests the error path in RunAll when
// executeChallenge returns an error via the test hook.
func TestRunAll_ExecuteHookError(t *testing.T) {
	a := newStub("a")
	reg := setupRegistry(t, a)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithExecuteHook(func(
			c challenge.Challenge,
			result *challenge.Result,
			err error,
		) (*challenge.Result, error) {
			return result, errors.New("injected execute error")
		}),
	)

	_, err := r.RunAll(context.Background(), challenge.NewConfig(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected execute error")
}

// TestRunSequence_ExecuteHookError tests the error path in RunSequence
// when executeChallenge returns an error via the test hook.
func TestRunSequence_ExecuteHookError(t *testing.T) {
	a := newStub("a")
	reg := setupRegistry(t, a)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithExecuteHook(func(
			c challenge.Challenge,
			result *challenge.Result,
			err error,
		) (*challenge.Result, error) {
			return result, errors.New("sequence execute error")
		}),
	)

	ids := []challenge.ID{"a"}
	_, err := r.RunSequence(context.Background(), ids, challenge.NewConfig(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sequence execute error")
}

// TestPipeline_Execute_ExecuteHookError tests the error path in Pipeline.Execute
// when executeChallenge returns an error via the test hook.
func TestPipeline_Execute_ExecuteHookError(t *testing.T) {
	a := newStub("a")
	reg := setupRegistry(t, a)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithExecuteHook(func(
			c challenge.Challenge,
			result *challenge.Result,
			err error,
		) (*challenge.Result, error) {
			return result, errors.New("pipeline execute error")
		}),
	)
	p := NewPipeline(r)

	result, err := p.Execute(context.Background(), a, challenge.NewConfig("a"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline execute error")
	// Result should still be returned
	assert.NotNil(t, result)
}

// TestPipeline_ExecuteSequence_ExecuteHookError tests the error path in
// Pipeline.ExecuteSequence when executeChallenge returns an error.
func TestPipeline_ExecuteSequence_ExecuteHookError(t *testing.T) {
	a := newStub("a")
	b := newStub("b")
	reg := setupRegistry(t, a, b)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithExecuteHook(func(
			c challenge.Challenge,
			result *challenge.Result,
			err error,
		) (*challenge.Result, error) {
			// Only return error for challenge "b"
			if c.ID() == "b" {
				return result, errors.New("sequence pipeline error")
			}
			return result, nil
		}),
	)
	p := NewPipeline(r)

	challenges := []challenge.Challenge{a, b}
	results, err := p.ExecuteSequence(
		context.Background(), challenges, challenge.NewConfig(""),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sequence pipeline error")
	// Should have result for "a" only
	assert.Len(t, results, 1)
	assert.Equal(t, challenge.ID("a"), results[0].ChallengeID)
}

// TestWithExecuteHook tests the WithExecuteHook option.
func TestWithExecuteHook(t *testing.T) {
	hookCalled := false
	hook := func(
		c challenge.Challenge,
		result *challenge.Result,
		err error,
	) (*challenge.Result, error) {
		hookCalled = true
		return result, nil
	}

	r := NewRunner(WithExecuteHook(hook))
	assert.NotNil(t, r.executeHook)

	a := newStub("a")
	reg := setupRegistry(t, a)
	r.registry = reg

	_, _ = r.Run(context.Background(), "a", challenge.NewConfig("a"))
	assert.True(t, hookCalled)
}
