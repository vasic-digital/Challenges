package runner

import (
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.challenges/pkg/registry"
)

// RunnerOption configures a DefaultRunner.
type RunnerOption func(*DefaultRunner)

// WithRegistry sets the challenge registry used by the runner.
func WithRegistry(reg registry.Registry) RunnerOption {
	return func(r *DefaultRunner) {
		r.registry = reg
	}
}

// WithLogger sets the logger used by the runner.
func WithLogger(logger challenge.Logger) RunnerOption {
	return func(r *DefaultRunner) {
		r.logger = logger
	}
}

// WithTimeout sets the default execution timeout for
// challenges that do not specify their own.
func WithTimeout(timeout time.Duration) RunnerOption {
	return func(r *DefaultRunner) {
		r.timeout = timeout
	}
}

// WithResultsDir sets the base directory where challenge
// results are written.
func WithResultsDir(dir string) RunnerOption {
	return func(r *DefaultRunner) {
		r.resultsDir = dir
	}
}

// WithPreHook adds a pre-execution hook to the runner.
func WithPreHook(h Hook) RunnerOption {
	return func(r *DefaultRunner) {
		r.preHooks = append(r.preHooks, h)
	}
}

// WithPostHook adds a post-execution hook to the runner.
func WithPostHook(h Hook) RunnerOption {
	return func(r *DefaultRunner) {
		r.postHooks = append(r.postHooks, h)
	}
}

// WithExecuteHook sets a test hook that is called after
// executeChallenge completes. It can override the result
// and error for testing error handling paths.
// This is intended for testing only.
func WithExecuteHook(h ExecuteHook) RunnerOption {
	return func(r *DefaultRunner) {
		r.executeHook = h
	}
}
