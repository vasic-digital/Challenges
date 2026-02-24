package userflow

import "context"

// BuildAdapter defines the interface for build, test, and lint
// operations. Implementations may wrap Go, npm, Gradle, Cargo,
// or other build toolchains.
type BuildAdapter interface {
	// Build executes a build target and returns the result.
	Build(
		ctx context.Context, target BuildTarget,
	) (*BuildResult, error)

	// RunTests executes a test target and returns the result.
	RunTests(
		ctx context.Context, target TestTarget,
	) (*TestResult, error)

	// Lint executes a lint target and returns the result.
	Lint(
		ctx context.Context, target LintTarget,
	) (*LintResult, error)

	// Available returns true if the build toolchain is
	// installed and usable.
	Available(ctx context.Context) bool
}
