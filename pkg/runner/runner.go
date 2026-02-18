// Package runner provides the challenge execution engine. It
// supports single, sequential, and parallel execution modes
// with configurable timeouts and lifecycle hooks.
package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.challenges/pkg/registry"
)

// Runner defines the interface for challenge execution.
type Runner interface {
	// Run executes a single challenge by ID.
	Run(
		ctx context.Context,
		id challenge.ID,
		config *challenge.Config,
	) (*challenge.Result, error)

	// RunAll executes all challenges in dependency order.
	RunAll(
		ctx context.Context,
		config *challenge.Config,
	) ([]*challenge.Result, error)

	// RunSequence executes the given challenges in order,
	// checking that dependencies have been met.
	RunSequence(
		ctx context.Context,
		ids []challenge.ID,
		config *challenge.Config,
	) ([]*challenge.Result, error)

	// RunParallel executes independent challenges
	// concurrently with the given concurrency limit.
	RunParallel(
		ctx context.Context,
		ids []challenge.ID,
		config *challenge.Config,
		maxConcurrency int,
	) ([]*challenge.Result, error)
}

// ExecuteHook allows testing of error paths in executeChallenge.
// It is called after executeChallenge completes and can override
// the returned error. This is only intended for testing.
type ExecuteHook func(
	c challenge.Challenge,
	result *challenge.Result,
	err error,
) (*challenge.Result, error)

// DefaultRunner is the standard Runner implementation.
type DefaultRunner struct {
	registry       registry.Registry
	logger         challenge.Logger
	timeout        time.Duration
	staleThreshold time.Duration
	resultsDir     string
	preHooks       []Hook
	postHooks      []Hook
	executeHook    ExecuteHook // test hook for executeChallenge errors
}

// Hook is a function invoked before or after challenge
// execution. It receives the challenge and its config.
type Hook func(
	ctx context.Context,
	c challenge.Challenge,
	cfg *challenge.Config,
) error

// NewRunner creates a DefaultRunner with the supplied options.
func NewRunner(opts ...RunnerOption) *DefaultRunner {
	r := &DefaultRunner{
		registry: registry.Default,
		timeout:  10 * time.Minute,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run executes a single challenge by ID.
func (r *DefaultRunner) Run(
	ctx context.Context,
	id challenge.ID,
	config *challenge.Config,
) (*challenge.Result, error) {
	c, err := r.registry.Get(id)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get challenge: %w", err,
		)
	}
	return r.executeChallenge(ctx, c, config)
}

// RunAll executes all challenges in dependency order. If a
// challenge passes, its results directory is propagated to
// downstream dependents.
func (r *DefaultRunner) RunAll(
	ctx context.Context,
	config *challenge.Config,
) ([]*challenge.Result, error) {
	ordered, err := r.registry.GetDependencyOrder()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get dependency order: %w", err,
		)
	}

	var results []*challenge.Result
	depResults := make(map[challenge.ID]string)

	for _, c := range ordered {
		cfg := *config
		cfg.ChallengeID = c.ID()
		cfg.Dependencies = depResults

		result, execErr := r.executeChallenge(ctx, c, &cfg)
		if execErr != nil {
			return results, fmt.Errorf(
				"challenge %s failed: %w",
				c.ID(), execErr,
			)
		}

		results = append(results, result)

		if result.Status == challenge.StatusPassed {
			depResults[c.ID()] = cfg.ResultsDir
		}
	}

	return results, nil
}

// RunSequence executes challenges in the given order, verifying
// that each challenge's dependencies have already been executed
// and passed within this sequence.
func (r *DefaultRunner) RunSequence(
	ctx context.Context,
	ids []challenge.ID,
	config *challenge.Config,
) ([]*challenge.Result, error) {
	var results []*challenge.Result
	depResults := make(map[challenge.ID]string)

	for _, id := range ids {
		c, err := r.registry.Get(id)
		if err != nil {
			return results, fmt.Errorf(
				"failed to get challenge %s: %w", id, err,
			)
		}

		for _, dep := range c.Dependencies() {
			if _, exists := depResults[dep]; !exists {
				return results, fmt.Errorf(
					"challenge %s has unmet dependency: %s",
					id, dep,
				)
			}
		}

		cfg := *config
		cfg.ChallengeID = id
		cfg.Dependencies = depResults

		result, execErr := r.executeChallenge(ctx, c, &cfg)
		if execErr != nil {
			return results, fmt.Errorf(
				"challenge %s failed: %w", id, execErr,
			)
		}

		results = append(results, result)

		if result.Status == challenge.StatusPassed {
			depResults[id] = cfg.ResultsDir
		}
	}

	return results, nil
}

// RunParallel executes the given challenges concurrently using
// at most maxConcurrency goroutines. It delegates to the
// parallel runner implementation.
func (r *DefaultRunner) RunParallel(
	ctx context.Context,
	ids []challenge.ID,
	config *challenge.Config,
	maxConcurrency int,
) ([]*challenge.Result, error) {
	return runParallel(ctx, r, ids, config, maxConcurrency)
}

// executeChallenge runs a single challenge through its full
// lifecycle: setup dir -> pre-hooks -> configure -> validate ->
// execute with timeout -> evaluate assertions -> post-hooks ->
// cleanup.
func (r *DefaultRunner) executeChallenge(
	ctx context.Context,
	c challenge.Challenge,
	config *challenge.Config,
) (*challenge.Result, error) {
	result := &challenge.Result{
		ChallengeID:   c.ID(),
		ChallengeName: c.Name(),
		Status:        challenge.StatusRunning,
		StartTime:     time.Now(),
		Metrics:       make(map[string]challenge.MetricValue),
		Outputs:       make(map[string]string),
	}

	// Setup results directory.
	if err := r.setupResultsDir(config); err != nil {
		result.Status = challenge.StatusError
		result.Error = fmt.Sprintf(
			"failed to setup results directory: %v", err,
		)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, nil
	}

	result.Logs = challenge.LogPaths{
		ChallengeLog: filepath.Join(
			config.LogsDir, "challenge.log",
		),
		OutputLog: filepath.Join(
			config.LogsDir, "output.log",
		),
	}

	r.logEvent("challenge_started", map[string]any{
		"challenge_id":   c.ID(),
		"challenge_name": c.Name(),
	})

	// Pre-hooks.
	for _, hook := range r.preHooks {
		if err := hook(ctx, c, config); err != nil {
			result.Status = challenge.StatusError
			result.Error = fmt.Sprintf(
				"pre-hook failed: %v", err,
			)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(
				result.StartTime,
			)
			return result, nil
		}
	}

	// Configure.
	if err := c.Configure(config); err != nil {
		result.Status = challenge.StatusError
		result.Error = fmt.Sprintf(
			"configuration failed: %v", err,
		)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		r.logEvent("challenge_error", map[string]any{
			"challenge_id": c.ID(),
			"error":        result.Error,
		})
		return result, nil
	}

	// Validate.
	if err := c.Validate(ctx); err != nil {
		result.Status = challenge.StatusSkipped
		result.Error = fmt.Sprintf(
			"validation failed: %v", err,
		)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		r.logEvent("challenge_skipped", map[string]any{
			"challenge_id": c.ID(),
			"reason":       result.Error,
		})
		return result, nil
	}

	// Setup progress-based liveness detection. If the
	// challenge supports progress reporting, attach a
	// ProgressReporter so the liveness monitor can track
	// forward progress. This allows long-running challenges
	// (hours) while detecting stuck ones (no progress).
	var progress *challenge.ProgressReporter
	type progressAware interface {
		SetProgressReporter(*challenge.ProgressReporter)
	}
	if pa, ok := c.(progressAware); ok {
		progress = challenge.NewProgressReporter()
		pa.SetProgressReporter(progress)
		defer progress.Close()
	}

	// Determine stale threshold: per-challenge config
	// overrides the runner default.
	staleThreshold := config.StaleThreshold
	if staleThreshold == 0 {
		staleThreshold = r.staleThreshold
	}

	// Execute with timeout. The timeout is a hard upper
	// bound; the liveness monitor provides a softer
	// progress-based check within that window.
	timeout := config.Timeout
	if timeout == 0 {
		timeout = r.timeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Start liveness monitor before Execute. It watches
	// the progress channel and cancels execCtx if no
	// progress is reported within the stale threshold.
	stopLiveness, stuckCh := startLivenessMonitor(
		progress, staleThreshold, cancel,
		r.logger, c.ID(),
	)
	defer stopLiveness()

	execResult, execErr := c.Execute(execCtx)

	// Stop liveness monitor immediately after Execute
	// returns to prevent false stuck detection during
	// post-processing.
	stopLiveness()

	// Check if the challenge was killed due to no
	// progress (stuck) vs hard timeout vs normal error.
	wasStuck := false
	if stuckCh != nil {
		select {
		case <-stuckCh:
			wasStuck = true
		default:
		}
	}

	// Handle stuck challenge (no progress within stale
	// threshold). This takes priority over timeout since
	// the liveness monitor cancelled the context.
	if wasStuck {
		result.Status = challenge.StatusStuck
		result.Error = fmt.Sprintf(
			"challenge stuck: no progress reported "+
				"within %v", staleThreshold,
		)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(
			result.StartTime,
		)
		r.logEvent("challenge_stuck", map[string]any{
			"challenge_id":           c.ID(),
			"stale_threshold_seconds": staleThreshold.Seconds(),
		})
		_ = c.Cleanup(ctx)
		return result, nil
	}

	// Handle timeout.
	if execCtx.Err() == context.DeadlineExceeded {
		result.Status = challenge.StatusTimedOut
		result.Error = "challenge execution timed out"
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		r.logEvent("challenge_timeout", map[string]any{
			"challenge_id":    c.ID(),
			"timeout_seconds": timeout.Seconds(),
		})
		_ = c.Cleanup(ctx)
		return result, nil
	}

	// Handle execution error.
	if execErr != nil {
		result.Status = challenge.StatusError
		result.Error = fmt.Sprintf(
			"execution failed: %v", execErr,
		)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		r.logEvent("challenge_error", map[string]any{
			"challenge_id": c.ID(),
			"error":        result.Error,
		})
		_ = c.Cleanup(ctx)
		return result, nil
	}

	// Merge execution result.
	if execResult != nil {
		result.Assertions = execResult.Assertions
		result.Metrics = execResult.Metrics
		result.Outputs = execResult.Outputs
	}

	// Determine final status from assertions.
	result.Status = challenge.StatusPassed
	for _, a := range result.Assertions {
		if !a.Passed {
			result.Status = challenge.StatusFailed
			break
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Post-hooks.
	for _, hook := range r.postHooks {
		if err := hook(ctx, c, config); err != nil {
			r.logEvent("post_hook_warning", map[string]any{
				"challenge_id": c.ID(),
				"warning":      err.Error(),
			})
		}
	}

	r.logEvent("challenge_completed", map[string]any{
		"challenge_id":     c.ID(),
		"status":           result.Status,
		"duration_seconds": result.Duration.Seconds(),
	})

	// Cleanup.
	if err := c.Cleanup(ctx); err != nil {
		r.logEvent("cleanup_warning", map[string]any{
			"challenge_id": c.ID(),
			"warning":      err.Error(),
		})
	}

	// Apply test hook if set.
	if r.executeHook != nil {
		return r.executeHook(c, result, nil)
	}

	return result, nil
}

// setupResultsDir creates the results directory structure.
func (r *DefaultRunner) setupResultsDir(
	config *challenge.Config,
) error {
	if config.ResultsDir == "" {
		now := time.Now()
		baseDir := r.resultsDir
		if baseDir == "" {
			baseDir = "results"
		}

		config.ResultsDir = filepath.Join(
			baseDir,
			string(config.ChallengeID),
			now.Format("2006"),
			now.Format("01"),
			now.Format("02"),
			now.Format("20060102_150405"),
		)
	}

	config.LogsDir = filepath.Join(
		config.ResultsDir, "logs",
	)

	if err := os.MkdirAll(config.LogsDir, 0755); err != nil {
		return err
	}

	resultsDir := filepath.Join(
		config.ResultsDir, "results",
	)
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return err
	}

	configDir := filepath.Join(
		config.ResultsDir, "config",
	)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	return nil
}

// logEvent emits a structured log entry if a logger is
// configured.
func (r *DefaultRunner) logEvent(
	event string,
	data map[string]any,
) {
	if r.logger == nil {
		return
	}

	parts := make([]any, 0, len(data)*2)
	for k, v := range data {
		parts = append(parts, k, v)
	}
	r.logger.Info(event, parts...)
}
