package challenge

import "context"

// ID uniquely identifies a challenge.
type ID string

// Challenge defines the interface that all challenges must implement.
// Each challenge goes through a lifecycle: Configure -> Validate ->
// Execute -> Cleanup. Dependencies between challenges are expressed
// via ID references and resolved by the runner.
type Challenge interface {
	// ID returns the unique identifier for this challenge.
	ID() ID

	// Name returns the human-readable name of this challenge.
	Name() string

	// Description returns a detailed description of what
	// this challenge validates.
	Description() string

	// Category returns the category grouping for this challenge
	// (e.g., "integration", "e2e", "security").
	Category() string

	// Dependencies returns the IDs of challenges that must
	// complete successfully before this challenge can execute.
	Dependencies() []ID

	// Configure applies runtime configuration to the challenge.
	// Must be called before Validate or Execute.
	Configure(config *Config) error

	// Validate checks that all preconditions are met for
	// execution (e.g., required services are available,
	// dependencies have passed).
	Validate(ctx context.Context) error

	// Execute runs the challenge and returns its result.
	Execute(ctx context.Context) (*Result, error)

	// Cleanup releases any resources allocated during
	// Configure or Execute.
	Cleanup(ctx context.Context) error
}

// Logger defines the minimal logging interface used by challenges.
// Implementations should be provided by the logging package.
type Logger interface {
	// Info logs an informational message.
	Info(msg string, args ...any)

	// Warn logs a warning message.
	Warn(msg string, args ...any)

	// Error logs an error message.
	Error(msg string, args ...any)

	// Debug logs a debug-level message.
	Debug(msg string, args ...any)

	// Close flushes and closes the logger.
	Close() error
}

// AssertionEngine evaluates assertions against actual values.
type AssertionEngine interface {
	// Evaluate checks a single assertion against the given value.
	Evaluate(assertion AssertionDef, value any) AssertionResult

	// EvaluateAll checks multiple assertions against a map of
	// named values. Each assertion's Target field is used as the
	// key into the values map.
	EvaluateAll(
		assertions []AssertionDef,
		values map[string]any,
	) []AssertionResult
}
