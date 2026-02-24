package userflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// NPMCLIAdapter implements BuildAdapter for Node.js projects
// by shelling out to npm/npx commands.
type NPMCLIAdapter struct {
	projectRoot string
}

// Compile-time interface check.
var _ BuildAdapter = (*NPMCLIAdapter)(nil)

// NewNPMCLIAdapter creates an NPMCLIAdapter rooted at
// projectRoot.
func NewNPMCLIAdapter(projectRoot string) *NPMCLIAdapter {
	return &NPMCLIAdapter{projectRoot: projectRoot}
}

// Build runs `npm run <task>` in the project root.
func (a *NPMCLIAdapter) Build(
	ctx context.Context, target BuildTarget,
) (*BuildResult, error) {
	args := []string{"run", target.Task}
	args = append(args, target.Args...)

	start := time.Now()
	output, err := a.runNPM(ctx, args...)
	elapsed := time.Since(start)

	return &BuildResult{
		Target:   target.Name,
		Success:  err == nil,
		Duration: elapsed,
		Output:   output,
	}, err
}

// RunTests runs vitest via npx with JUnit output, parses the
// results, and returns a TestResult.
func (a *NPMCLIAdapter) RunTests(
	ctx context.Context, target TestTarget,
) (*TestResult, error) {
	tmpFile := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("vitest-junit-%d.xml", time.Now().UnixNano()),
	)
	defer os.Remove(tmpFile)

	args := []string{
		"vitest", "run",
		"--reporter=junit",
		"--outputFile=" + tmpFile,
	}
	if target.Filter != "" {
		args = append(args, "-t", target.Filter)
	}

	start := time.Now()
	output, runErr := a.runNPX(ctx, args...)
	elapsed := time.Since(start)

	// Try to parse JUnit XML output.
	data, err := os.ReadFile(tmpFile)
	if err == nil && len(data) > 0 {
		suites, err := ParseJUnitXML(data)
		if err == nil {
			result := JUnitToTestResult(
				suites, elapsed, output,
			)
			return result, runErr
		}
	}

	// Fallback when no JUnit XML is available.
	result := &TestResult{
		Duration: elapsed,
		Output:   output,
	}
	if runErr != nil {
		result.TotalFailed = 1
	}
	return result, runErr
}

// eslintMessage is used to parse ESLint JSON output entries.
type eslintMessage struct {
	Severity int `json:"severity"`
}

// eslintResult is used to parse a single ESLint file result.
type eslintResult struct {
	Messages     []eslintMessage `json:"messages"`
	ErrorCount   int             `json:"errorCount"`
	WarningCount int             `json:"warningCount"`
}

// Lint runs ESLint via npx and parses the JSON output for
// error and warning counts.
func (a *NPMCLIAdapter) Lint(
	ctx context.Context, target LintTarget,
) (*LintResult, error) {
	args := []string{
		"eslint", ".", "--format=json",
	}
	args = append(args, target.Args...)

	start := time.Now()
	output, runErr := a.runNPX(ctx, args...)
	elapsed := time.Since(start)

	var warnings, errors int
	var results []eslintResult
	if err := json.Unmarshal(
		[]byte(output), &results,
	); err == nil {
		for _, r := range results {
			errors += r.ErrorCount
			warnings += r.WarningCount
		}
	}

	return &LintResult{
		Tool:     "eslint",
		Success:  runErr == nil && errors == 0,
		Duration: elapsed,
		Warnings: warnings,
		Errors:   errors,
		Output:   output,
	}, runErr
}

// Available returns true if package.json exists in the project
// root.
func (a *NPMCLIAdapter) Available(
	_ context.Context,
) bool {
	_, err := os.Stat(
		filepath.Join(a.projectRoot, "package.json"),
	)
	return err == nil
}

// runNPM executes an npm command in the project root and
// returns combined output.
func (a *NPMCLIAdapter) runNPM(
	ctx context.Context, args ...string,
) (string, error) {
	cmd := exec.CommandContext(ctx, "npm", args...)
	cmd.Dir = a.projectRoot

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf(
			"npm %v: %w", args, err,
		)
	}
	return string(out), nil
}

// runNPX executes an npx command in the project root and
// returns combined output.
func (a *NPMCLIAdapter) runNPX(
	ctx context.Context, args ...string,
) (string, error) {
	cmd := exec.CommandContext(ctx, "npx", args...)
	cmd.Dir = a.projectRoot

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf(
			"npx %v: %w", args, err,
		)
	}
	return string(out), nil
}
