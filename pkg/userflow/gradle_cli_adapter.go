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

// GradleCLIAdapter implements BuildAdapter for Gradle-based
// projects by shelling out to ./gradlew commands.
type GradleCLIAdapter struct {
	projectRoot  string
	useContainer bool
}

// Compile-time interface check.
var _ BuildAdapter = (*GradleCLIAdapter)(nil)

// NewGradleCLIAdapter creates a GradleCLIAdapter rooted at
// projectRoot. If useContainer is true, commands are prefixed
// with `podman-compose run --rm build`.
func NewGradleCLIAdapter(
	projectRoot string, useContainer bool,
) *GradleCLIAdapter {
	return &GradleCLIAdapter{
		projectRoot:  projectRoot,
		useContainer: useContainer,
	}
}

// Build executes a Gradle build task and returns the result.
func (a *GradleCLIAdapter) Build(
	ctx context.Context, target BuildTarget,
) (*BuildResult, error) {
	args := []string{target.Task}
	args = append(args, target.Args...)

	start := time.Now()
	output, err := a.runGradle(ctx, args...)
	elapsed := time.Since(start)

	result := &BuildResult{
		Target:   target.Name,
		Success:  err == nil,
		Duration: elapsed,
		Output:   output,
	}

	return result, err
}

// RunTests executes a Gradle test task, optionally filtering
// by test class, then parses JUnit XML results.
func (a *GradleCLIAdapter) RunTests(
	ctx context.Context, target TestTarget,
) (*TestResult, error) {
	args := []string{target.Task}
	if target.Filter != "" {
		args = append(args, "--tests", target.Filter)
	}

	start := time.Now()
	output, runErr := a.runGradle(ctx, args...)
	elapsed := time.Since(start)

	// Search for JUnit XML in standard paths.
	xmlPaths := []string{
		"build/test-results",
		"app/build/test-results",
	}
	var allSuites []JUnitTestSuite
	for _, base := range xmlPaths {
		searchDir := filepath.Join(a.projectRoot, base)
		matches, err := filepath.Glob(
			filepath.Join(searchDir, "**", "*.xml"),
		)
		if err != nil || len(matches) == 0 {
			// Also try one level of nesting.
			matches, _ = filepath.Glob(
				filepath.Join(searchDir, "*", "*.xml"),
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

	if len(allSuites) > 0 {
		result := JUnitToTestResult(
			allSuites, elapsed, output,
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

// Lint executes a Gradle lint task and returns the result.
func (a *GradleCLIAdapter) Lint(
	ctx context.Context, target LintTarget,
) (*LintResult, error) {
	args := []string{target.Task}
	args = append(args, target.Args...)

	start := time.Now()
	output, err := a.runGradle(ctx, args...)
	elapsed := time.Since(start)

	return &LintResult{
		Tool:     "gradle:" + target.Task,
		Success:  err == nil,
		Duration: elapsed,
		Output:   output,
	}, err
}

// Available returns true if gradlew exists in the project root,
// Java is in PATH, and the Android SDK is configured (when the
// project is an Android project).
func (a *GradleCLIAdapter) Available(
	_ context.Context,
) bool {
	_, err := os.Stat(
		filepath.Join(a.projectRoot, "gradlew"),
	)
	if err != nil {
		return false
	}
	// Check that Java is available (required for Gradle).
	_, javaErr := exec.LookPath("java")
	if javaErr != nil {
		return false
	}
	// Check that Android SDK is configured if this is an
	// Android project.
	buildGradle := filepath.Join(
		a.projectRoot, "build.gradle",
	)
	if _, err := os.Stat(buildGradle); err == nil {
		// Android project — verify SDK is set.
		if os.Getenv("ANDROID_HOME") == "" &&
			os.Getenv("ANDROID_SDK_ROOT") == "" {
			// Check local.properties for sdk.dir.
			lpPath := filepath.Join(
				a.projectRoot, "local.properties",
			)
			data, err := os.ReadFile(lpPath)
			if err != nil ||
				!strings.Contains(
					string(data), "sdk.dir",
				) {
				return false
			}
		}
	}
	return true
}

// runGradle executes a gradlew command, optionally inside a
// container, and returns the combined output.
func (a *GradleCLIAdapter) runGradle(
	ctx context.Context, args ...string,
) (string, error) {
	var cmd *exec.Cmd
	if a.useContainer {
		containerArgs := []string{
			"run", "--rm", "build", "./gradlew",
		}
		containerArgs = append(containerArgs, args...)
		cmd = exec.CommandContext(
			ctx, "podman-compose", containerArgs...,
		)
	} else {
		cmd = exec.CommandContext(
			ctx, "./gradlew", args...,
		)
	}
	cmd.Dir = a.projectRoot

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf(
			"gradle %v: %w", args, err,
		)
	}
	return string(out), nil
}
