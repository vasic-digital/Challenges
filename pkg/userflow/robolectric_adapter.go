package userflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RobolectricAdapter implements BuildAdapter for running
// Android unit tests via Robolectric. Robolectric executes
// Android tests on the JVM without requiring an emulator or
// physical device. All commands are delegated to Gradle.
type RobolectricAdapter struct {
	projectDir    string
	gradleWrapper string
	module        string
	testFilter    string
	jvmArgs       []string
}

// Compile-time interface check.
var _ BuildAdapter = (*RobolectricAdapter)(nil)

// RobolectricOption configures a RobolectricAdapter.
type RobolectricOption func(*RobolectricAdapter)

// WithRobolectricGradleWrapper sets a custom path to the
// Gradle wrapper script. Defaults to ./gradlew in the
// project directory.
func WithRobolectricGradleWrapper(
	path string,
) RobolectricOption {
	return func(a *RobolectricAdapter) {
		a.gradleWrapper = path
	}
}

// WithRobolectricModule sets the Gradle module prefix
// (e.g., ":app") for multi-module projects.
func WithRobolectricModule(
	module string,
) RobolectricOption {
	return func(a *RobolectricAdapter) {
		a.module = module
	}
}

// WithRobolectricTestFilter sets a default test filter
// applied to all test invocations (e.g., a class name or
// method pattern).
func WithRobolectricTestFilter(
	filter string,
) RobolectricOption {
	return func(a *RobolectricAdapter) {
		a.testFilter = filter
	}
}

// WithRobolectricJVMArgs sets additional JVM arguments
// passed to the Gradle daemon (e.g., "-Xmx2g").
func WithRobolectricJVMArgs(
	args []string,
) RobolectricOption {
	return func(a *RobolectricAdapter) {
		a.jvmArgs = args
	}
}

// NewRobolectricAdapter creates a RobolectricAdapter rooted
// at projectDir. Options may override the Gradle wrapper
// path, module, test filter, and JVM arguments.
func NewRobolectricAdapter(
	projectDir string, opts ...RobolectricOption,
) *RobolectricAdapter {
	a := &RobolectricAdapter{
		projectDir: projectDir,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// gradlePath returns the resolved path to the Gradle wrapper.
func (a *RobolectricAdapter) gradlePath() string {
	if a.gradleWrapper != "" {
		return a.gradleWrapper
	}
	return filepath.Join(a.projectDir, "gradlew")
}

// taskName prepends the module prefix to a task name when
// a module is configured (e.g., ":app:assembleDebug").
func (a *RobolectricAdapter) taskName(
	task string,
) string {
	if a.module != "" {
		return a.module + ":" + task
	}
	return task
}

// Build executes the Gradle assembleDebug task (or the task
// specified in the BuildTarget) and returns the result.
// Commands are resource-limited with nice and ionice.
func (a *RobolectricAdapter) Build(
	ctx context.Context, target BuildTarget,
) (*BuildResult, error) {
	task := target.Task
	if task == "" {
		task = "assembleDebug"
	}

	args := []string{a.taskName(task)}
	args = append(args, a.jvmArgFlags()...)
	args = append(args, target.Args...)

	start := time.Now()
	output, err := a.runGradle(ctx, args...)
	elapsed := time.Since(start)

	return &BuildResult{
		Target:   target.Name,
		Success:  err == nil,
		Duration: elapsed,
		Output:   output,
	}, err
}

// RunTests executes the Gradle testDebugUnitTest task and
// parses JUnit XML results from the standard Robolectric
// output directory. An optional filter is applied via the
// --tests flag.
func (a *RobolectricAdapter) RunTests(
	ctx context.Context, target TestTarget,
) (*TestResult, error) {
	task := target.Task
	if task == "" {
		task = "testDebugUnitTest"
	}

	args := []string{a.taskName(task)}
	args = append(args, a.jvmArgFlags()...)

	// Apply test filter from target or default.
	filter := target.Filter
	if filter == "" {
		filter = a.testFilter
	}
	if filter != "" {
		args = append(args, "--tests", filter)
	}

	start := time.Now()
	output, runErr := a.runGradle(ctx, args...)
	elapsed := time.Since(start)

	// Parse JUnit XML from standard result directories.
	suites := a.collectJUnitResults()
	if len(suites) > 0 {
		result := JUnitToTestResult(
			suites, elapsed, output,
		)
		return result, runErr
	}

	// No JUnit XML found; return basic result.
	result := &TestResult{
		Duration: elapsed,
		Output:   output,
	}
	if runErr != nil {
		result.TotalFailed = 1
	}
	return result, runErr
}

// collectJUnitResults searches standard Robolectric JUnit
// XML output directories and parses all found XML files.
func (a *RobolectricAdapter) collectJUnitResults() []JUnitTestSuite {
	searchDirs := []string{
		"build/test-results/testDebugUnitTest",
		"app/build/test-results/testDebugUnitTest",
	}
	if a.module != "" {
		mod := strings.TrimPrefix(a.module, ":")
		searchDirs = append(
			searchDirs,
			filepath.Join(
				mod,
				"build/test-results/testDebugUnitTest",
			),
		)
	}

	var allSuites []JUnitTestSuite
	for _, base := range searchDirs {
		dir := filepath.Join(a.projectDir, base)
		// Try direct XML files first.
		matches, err := filepath.Glob(
			filepath.Join(dir, "*.xml"),
		)
		if err != nil || len(matches) == 0 {
			// Try one level of nesting.
			matches, _ = filepath.Glob(
				filepath.Join(dir, "*", "*.xml"),
			)
		}
		for _, m := range matches {
			data, err := os.ReadFile(m)
			if err != nil {
				continue
			}
			suites, err := ParseJUnitXML(data)
			if err != nil {
				continue
			}
			allSuites = append(allSuites, suites...)
		}
	}
	return allSuites
}

// Lint executes the Gradle lintDebug task (or the task
// specified in the LintTarget) and returns the result.
func (a *RobolectricAdapter) Lint(
	ctx context.Context, target LintTarget,
) (*LintResult, error) {
	task := target.Task
	if task == "" {
		task = "lintDebug"
	}

	args := []string{a.taskName(task)}
	args = append(args, target.Args...)

	start := time.Now()
	output, err := a.runGradle(ctx, args...)
	elapsed := time.Since(start)

	return &LintResult{
		Tool:     "gradle:" + task,
		Success:  err == nil,
		Duration: elapsed,
		Output:   output,
	}, err
}

// Available returns true if the Gradle wrapper exists,
// Java is in PATH, and `./gradlew --version` succeeds.
func (a *RobolectricAdapter) Available(
	ctx context.Context,
) bool {
	wrapper := a.gradlePath()
	if _, err := os.Stat(wrapper); err != nil {
		return false
	}
	if _, err := exec.LookPath("java"); err != nil {
		return false
	}
	cmd := exec.CommandContext(
		ctx, wrapper, "--version",
	)
	cmd.Dir = a.projectDir
	return cmd.Run() == nil
}

// jvmArgFlags converts JVM arguments into Gradle-compatible
// -Dorg.gradle.jvmargs flags.
func (a *RobolectricAdapter) jvmArgFlags() []string {
	if len(a.jvmArgs) == 0 {
		return nil
	}
	joined := strings.Join(a.jvmArgs, " ")
	return []string{
		"-Dorg.gradle.jvmargs=" + joined,
	}
}

// runGradle executes a Gradle wrapper command with resource
// limits (nice -n 19, ionice -c 3) and returns the combined
// output.
func (a *RobolectricAdapter) runGradle(
	ctx context.Context, args ...string,
) (string, error) {
	wrapper := a.gradlePath()

	// Build resource-limited command:
	// nice -n 19 ionice -c 3 ./gradlew <args>
	cmdArgs := []string{
		"-n", "19",
		"ionice", "-c", "3",
		wrapper,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(
		ctx, "nice", cmdArgs...,
	)
	cmd.Dir = a.projectDir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf(
			"robolectric gradle %v: %w", args, err,
		)
	}
	return string(out), nil
}
