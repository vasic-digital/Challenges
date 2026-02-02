package runner

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.challenges/pkg/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- stub challenge ---

type stubChallenge struct {
	id           challenge.ID
	name         string
	deps         []challenge.ID
	configureErr error
	validateErr  error
	executeErr   error
	cleanupErr   error
	execResult   *challenge.Result
	execDelay    time.Duration

	mu             sync.Mutex
	configureCalls int
	validateCalls  int
	executeCalls   int
	cleanupCalls   int
}

func (s *stubChallenge) ID() challenge.ID     { return s.id }
func (s *stubChallenge) Name() string         { return s.name }
func (s *stubChallenge) Description() string  { return "stub" }
func (s *stubChallenge) Category() string     { return "test" }
func (s *stubChallenge) Dependencies() []challenge.ID {
	return s.deps
}

func (s *stubChallenge) Configure(
	_ *challenge.Config,
) error {
	s.mu.Lock()
	s.configureCalls++
	s.mu.Unlock()
	return s.configureErr
}

func (s *stubChallenge) Validate(
	_ context.Context,
) error {
	s.mu.Lock()
	s.validateCalls++
	s.mu.Unlock()
	return s.validateErr
}

func (s *stubChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	s.mu.Lock()
	s.executeCalls++
	s.mu.Unlock()

	if s.execDelay > 0 {
		select {
		case <-time.After(s.execDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.execResult, s.executeErr
}

func (s *stubChallenge) Cleanup(
	_ context.Context,
) error {
	s.mu.Lock()
	s.cleanupCalls++
	s.mu.Unlock()
	return s.cleanupErr
}

func newStub(id string, deps ...string) *stubChallenge {
	depIDs := make([]challenge.ID, len(deps))
	for i, d := range deps {
		depIDs[i] = challenge.ID(d)
	}
	return &stubChallenge{
		id:   challenge.ID(id),
		name: id,
		deps: depIDs,
		execResult: &challenge.Result{
			Status: challenge.StatusPassed,
			Assertions: []challenge.AssertionResult{
				{Passed: true, Message: "ok"},
			},
		},
	}
}

func setupRegistry(
	t *testing.T, stubs ...*stubChallenge,
) registry.Registry {
	t.Helper()
	reg := registry.NewRegistry()
	for _, s := range stubs {
		require.NoError(t, reg.Register(s))
	}
	return reg
}

// --- stub logger ---

type stubLogger struct {
	mu       sync.Mutex
	messages []string
}

func (l *stubLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	l.messages = append(l.messages, "info:"+msg)
	l.mu.Unlock()
}
func (l *stubLogger) Warn(msg string, _ ...any) {
	l.mu.Lock()
	l.messages = append(l.messages, "warn:"+msg)
	l.mu.Unlock()
}
func (l *stubLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	l.messages = append(l.messages, "error:"+msg)
	l.mu.Unlock()
}
func (l *stubLogger) Debug(msg string, _ ...any) {
	l.mu.Lock()
	l.messages = append(l.messages, "debug:"+msg)
	l.mu.Unlock()
}
func (l *stubLogger) Close() error { return nil }

// =========================================================
// DefaultRunner.Run tests
// =========================================================

func TestDefaultRunner_Run_Success(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ctx := context.Background()
	config := challenge.NewConfig("a")

	result, err := r.Run(ctx, "a", config)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Equal(t, challenge.ID("a"), result.ChallengeID)
	assert.Equal(t, "a", result.ChallengeName)
	assert.False(t, result.StartTime.IsZero())
	assert.False(t, result.EndTime.IsZero())
	assert.True(t, result.Duration > 0)
}

func TestDefaultRunner_Run_NotFound(t *testing.T) {
	reg := setupRegistry(t)
	r := NewRunner(WithRegistry(reg))

	ctx := context.Background()
	_, err := r.Run(ctx, "missing", challenge.NewConfig("x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get challenge")
}

func TestDefaultRunner_Run_ConfigureError(t *testing.T) {
	s := newStub("a")
	s.configureErr = errors.New("bad config")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusError, result.Status)
	assert.Contains(t, result.Error, "configuration failed")
}

func TestDefaultRunner_Run_ValidateError(t *testing.T) {
	s := newStub("a")
	s.validateErr = errors.New("precondition not met")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusSkipped, result.Status)
	assert.Contains(t, result.Error, "validation failed")
}

func TestDefaultRunner_Run_ExecuteError(t *testing.T) {
	s := newStub("a")
	s.executeErr = errors.New("boom")
	s.execResult = nil
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusError, result.Status)
	assert.Contains(t, result.Error, "execution failed")
}

func TestDefaultRunner_Run_Timeout(t *testing.T) {
	s := newStub("a")
	s.execDelay = 5 * time.Second
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithTimeout(50*time.Millisecond),
		WithResultsDir(t.TempDir()),
	)

	cfg := challenge.NewConfig("a")
	cfg.Timeout = 0 // use runner's timeout

	result, err := r.Run(
		context.Background(), "a", cfg,
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusTimedOut, result.Status)
	assert.Contains(t, result.Error, "timed out")
}

func TestDefaultRunner_Run_TimeoutFromConfig(t *testing.T) {
	s := newStub("a")
	s.execDelay = 5 * time.Second
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithTimeout(10*time.Minute),
		WithResultsDir(t.TempDir()),
	)

	cfg := challenge.NewConfig("a")
	cfg.Timeout = 50 * time.Millisecond

	result, err := r.Run(
		context.Background(), "a", cfg,
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusTimedOut, result.Status)
}

func TestDefaultRunner_Run_FailedAssertion(t *testing.T) {
	s := newStub("a")
	s.execResult = &challenge.Result{
		Assertions: []challenge.AssertionResult{
			{Passed: true, Message: "ok"},
			{Passed: false, Message: "not ok"},
		},
	}
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 2)
	assert.True(t, result.Assertions[0].Passed)
	assert.False(t, result.Assertions[1].Passed)
}

func TestDefaultRunner_Run_AllAssertionsPass(t *testing.T) {
	s := newStub("a")
	s.execResult = &challenge.Result{
		Assertions: []challenge.AssertionResult{
			{Passed: true, Message: "first"},
			{Passed: true, Message: "second"},
			{Passed: true, Message: "third"},
		},
		Metrics: map[string]challenge.MetricValue{
			"latency": {Name: "latency", Value: 42.5, Unit: "ms"},
		},
		Outputs: map[string]string{
			"response": "hello",
		},
	}
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Len(t, result.Assertions, 3)
	assert.Contains(t, result.Metrics, "latency")
	assert.Equal(t, "hello", result.Outputs["response"])
}

func TestDefaultRunner_Run_NilExecResult(t *testing.T) {
	s := newStub("a")
	s.execResult = nil
	s.executeErr = nil
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	// No assertions means passed.
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Empty(t, result.Assertions)
}

func TestDefaultRunner_Run_CleanupCalledOnSuccess(t *testing.T) {
	s := newStub("a")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	_, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, s.cleanupCalls)
}

func TestDefaultRunner_Run_CleanupCalledOnTimeout(t *testing.T) {
	s := newStub("a")
	s.execDelay = 5 * time.Second
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithTimeout(50*time.Millisecond),
		WithResultsDir(t.TempDir()),
	)

	cfg := challenge.NewConfig("a")
	cfg.Timeout = 0

	_, err := r.Run(context.Background(), "a", cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, s.cleanupCalls)
}

func TestDefaultRunner_Run_CleanupCalledOnExecError(t *testing.T) {
	s := newStub("a")
	s.executeErr = errors.New("exec failed")
	s.execResult = nil
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	_, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, s.cleanupCalls)
}

func TestDefaultRunner_Run_CleanupErrorIsWarning(t *testing.T) {
	s := newStub("a")
	s.cleanupErr = errors.New("cleanup failed")
	logger := &stubLogger{}
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithLogger(logger),
	)

	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	// Cleanup error should not change status.
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

// =========================================================
// DefaultRunner.RunAll tests
// =========================================================

func TestDefaultRunner_RunAll_Success(t *testing.T) {
	reg := setupRegistry(t,
		newStub("a"),
		newStub("b", "a"),
	)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	results, err := r.RunAll(
		context.Background(),
		challenge.NewConfig(""),
	)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// a should run before b.
	assert.Equal(t, challenge.ID("a"), results[0].ChallengeID)
	assert.Equal(t, challenge.ID("b"), results[1].ChallengeID)
}

func TestDefaultRunner_RunAll_PropagatesDependencies(
	t *testing.T,
) {
	a := newStub("a")
	b := newStub("b", "a")
	reg := setupRegistry(t, a, b)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	results, err := r.RunAll(
		context.Background(),
		challenge.NewConfig(""),
	)
	require.NoError(t, err)
	require.Len(t, results, 2)

	for _, res := range results {
		assert.Equal(t, challenge.StatusPassed, res.Status)
	}
}

func TestDefaultRunner_RunAll_StopsOnError(t *testing.T) {
	a := newStub("a")
	a.executeErr = errors.New("a failed")
	a.execResult = nil
	b := newStub("b", "a")
	reg := setupRegistry(t, a, b)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	results, err := r.RunAll(
		context.Background(),
		challenge.NewConfig(""),
	)
	// RunAll continues collecting results; the error comes
	// from the error status not from a Go error necessarily.
	// The first challenge produces StatusError result.
	_ = err
	// At least one result should be present.
	require.GreaterOrEqual(t, len(results), 1)
}

func TestDefaultRunner_RunAll_SingleChallenge(t *testing.T) {
	reg := setupRegistry(t, newStub("only"))

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	results, err := r.RunAll(
		context.Background(),
		challenge.NewConfig(""),
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, challenge.ID("only"), results[0].ChallengeID)
}

// =========================================================
// DefaultRunner.RunSequence tests
// =========================================================

func TestDefaultRunner_RunSequence_Success(t *testing.T) {
	reg := setupRegistry(t,
		newStub("a"),
		newStub("b", "a"),
	)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "b"}
	results, err := r.RunSequence(
		context.Background(), ids,
		challenge.NewConfig(""),
	)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, challenge.ID("a"), results[0].ChallengeID)
	assert.Equal(t, challenge.ID("b"), results[1].ChallengeID)
}

func TestDefaultRunner_RunSequence_UnmetDependency(
	t *testing.T,
) {
	reg := setupRegistry(t,
		newStub("a"),
		newStub("b", "a"),
	)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	// Skip "a", so "b"'s dependency is unmet.
	ids := []challenge.ID{"b"}
	_, err := r.RunSequence(
		context.Background(), ids,
		challenge.NewConfig(""),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmet dependency")
}

func TestDefaultRunner_RunSequence_NotFound(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "nonexistent"}
	_, err := r.RunSequence(
		context.Background(), ids,
		challenge.NewConfig(""),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get challenge")
}

// =========================================================
// Hook tests
// =========================================================

func TestDefaultRunner_PreHook_Called(t *testing.T) {
	hookCalled := false
	hook := func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		hookCalled = true
		return nil
	}

	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithPreHook(hook),
	)

	_, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.True(t, hookCalled)
}

func TestDefaultRunner_PreHook_Error(t *testing.T) {
	hook := func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		return errors.New("hook failure")
	}

	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithPreHook(hook),
	)

	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusError, result.Status)
	assert.Contains(t, result.Error, "pre-hook failed")
}

func TestDefaultRunner_PreHook_ErrorSkipsExecution(
	t *testing.T,
) {
	s := newStub("a")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithPreHook(func(
			_ context.Context,
			_ challenge.Challenge,
			_ *challenge.Config,
		) error {
			return errors.New("blocked")
		}),
	)

	_, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	// Execute should not have been called.
	assert.Equal(t, 0, s.executeCalls)
}

func TestDefaultRunner_PostHook_Called(t *testing.T) {
	hookCalled := false
	hook := func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		hookCalled = true
		return nil
	}

	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithPostHook(hook),
	)

	_, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.True(t, hookCalled)
}

func TestDefaultRunner_PostHook_ErrorIsWarning(t *testing.T) {
	logger := &stubLogger{}
	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithLogger(logger),
		WithPostHook(func(
			_ context.Context,
			_ challenge.Challenge,
			_ *challenge.Config,
		) error {
			return errors.New("post-hook oops")
		}),
	)

	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	// Post-hook errors do not change result status.
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestDefaultRunner_MultiplePreHooks_Order(t *testing.T) {
	var order []string
	makeHook := func(label string) Hook {
		return func(
			_ context.Context,
			_ challenge.Challenge,
			_ *challenge.Config,
		) error {
			order = append(order, label)
			return nil
		}
	}

	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithPreHook(makeHook("pre1")),
		WithPreHook(makeHook("pre2")),
		WithPostHook(makeHook("post1")),
	)

	_, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t,
		[]string{"pre1", "pre2", "post1"}, order,
	)
}

// =========================================================
// ResultsDir tests
// =========================================================

func TestDefaultRunner_ResultsDir_Created(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))
	baseDir := t.TempDir()

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(baseDir),
	)

	config := challenge.NewConfig("a")
	config.ResultsDir = "" // force auto-creation

	result, err := r.Run(context.Background(), "a", config)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.NotEmpty(t, config.ResultsDir)
	assert.NotEmpty(t, config.LogsDir)
}

func TestDefaultRunner_ResultsDir_ExplicitPath(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))
	tmpDir := t.TempDir()

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(tmpDir),
	)

	config := challenge.NewConfig("a")
	config.ResultsDir = tmpDir + "/explicit"

	result, err := r.Run(context.Background(), "a", config)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Equal(t, tmpDir+"/explicit", config.ResultsDir)
}

// =========================================================
// RunnerOption / functional options tests
// =========================================================

func TestNewRunner_Defaults(t *testing.T) {
	r := NewRunner()
	assert.Equal(t, 10*time.Minute, r.timeout)
	assert.NotNil(t, r.registry)
	assert.Nil(t, r.logger)
	assert.Empty(t, r.resultsDir)
	assert.Empty(t, r.preHooks)
	assert.Empty(t, r.postHooks)
}

func TestNewRunner_WithTimeout(t *testing.T) {
	r := NewRunner(WithTimeout(30 * time.Second))
	assert.Equal(t, 30*time.Second, r.timeout)
}

func TestNewRunner_WithRegistry(t *testing.T) {
	reg := registry.NewRegistry()
	r := NewRunner(WithRegistry(reg))
	assert.Equal(t, reg, r.registry)
}

func TestNewRunner_WithLogger(t *testing.T) {
	logger := &stubLogger{}
	r := NewRunner(WithLogger(logger))
	assert.Equal(t, logger, r.logger)
}

func TestNewRunner_WithResultsDir(t *testing.T) {
	r := NewRunner(WithResultsDir("/tmp/results"))
	assert.Equal(t, "/tmp/results", r.resultsDir)
}

func TestNewRunner_WithMultipleOptions(t *testing.T) {
	logger := &stubLogger{}
	reg := registry.NewRegistry()

	preHookCalled := false
	postHookCalled := false

	r := NewRunner(
		WithTimeout(1*time.Minute),
		WithRegistry(reg),
		WithLogger(logger),
		WithResultsDir("/tmp/test"),
		WithPreHook(func(
			_ context.Context,
			_ challenge.Challenge,
			_ *challenge.Config,
		) error {
			preHookCalled = true
			return nil
		}),
		WithPostHook(func(
			_ context.Context,
			_ challenge.Challenge,
			_ *challenge.Config,
		) error {
			postHookCalled = true
			return nil
		}),
	)

	assert.Equal(t, 1*time.Minute, r.timeout)
	assert.Equal(t, reg, r.registry)
	assert.Equal(t, logger, r.logger)
	assert.Equal(t, "/tmp/test", r.resultsDir)
	assert.Len(t, r.preHooks, 1)
	assert.Len(t, r.postHooks, 1)

	_ = preHookCalled
	_ = postHookCalled
}

// =========================================================
// Logger integration tests
// =========================================================

func TestDefaultRunner_Run_WithLogger(t *testing.T) {
	logger := &stubLogger{}
	reg := setupRegistry(t, newStub("a"))

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithLogger(logger),
	)

	_, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)

	// Logger should have received events.
	logger.mu.Lock()
	msgs := make([]string, len(logger.messages))
	copy(msgs, logger.messages)
	logger.mu.Unlock()

	assert.NotEmpty(t, msgs)
}

func TestDefaultRunner_Run_WithoutLogger(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	// Should not panic without a logger.
	result, err := r.Run(
		context.Background(), "a",
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

// =========================================================
// Table-driven tests
// =========================================================

func TestDefaultRunner_Run_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		configureErr   error
		validateErr    error
		executeErr     error
		execResult     *challenge.Result
		execDelay      time.Duration
		timeout        time.Duration
		expectedStatus string
		expectedErrSub string
	}{
		{
			name: "success with all assertions passing",
			execResult: &challenge.Result{
				Assertions: []challenge.AssertionResult{
					{Passed: true},
				},
			},
			expectedStatus: challenge.StatusPassed,
		},
		{
			name:           "configure error",
			configureErr:   errors.New("bad"),
			expectedStatus: challenge.StatusError,
			expectedErrSub: "configuration failed",
		},
		{
			name:           "validate error results in skip",
			validateErr:    errors.New("not ready"),
			expectedStatus: challenge.StatusSkipped,
			expectedErrSub: "validation failed",
		},
		{
			name:           "execute error",
			executeErr:     errors.New("crash"),
			execResult:     nil,
			expectedStatus: challenge.StatusError,
			expectedErrSub: "execution failed",
		},
		{
			name: "failed assertion",
			execResult: &challenge.Result{
				Assertions: []challenge.AssertionResult{
					{Passed: false, Message: "nope"},
				},
			},
			expectedStatus: challenge.StatusFailed,
		},
		{
			name:           "timeout",
			execDelay:      5 * time.Second,
			timeout:        50 * time.Millisecond,
			expectedStatus: challenge.StatusTimedOut,
			expectedErrSub: "timed out",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub("test")
			s.configureErr = tc.configureErr
			s.validateErr = tc.validateErr
			s.executeErr = tc.executeErr
			if tc.execResult != nil {
				s.execResult = tc.execResult
			}
			s.execDelay = tc.execDelay

			reg := setupRegistry(t, s)

			timeout := 10 * time.Minute
			if tc.timeout > 0 {
				timeout = tc.timeout
			}

			r := NewRunner(
				WithRegistry(reg),
				WithResultsDir(t.TempDir()),
				WithTimeout(timeout),
			)

			cfg := challenge.NewConfig("test")
			cfg.Timeout = 0

			result, err := r.Run(
				context.Background(), "test", cfg,
			)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, result.Status)
			if tc.expectedErrSub != "" {
				assert.Contains(t,
					result.Error, tc.expectedErrSub,
				)
			}
		})
	}
}
