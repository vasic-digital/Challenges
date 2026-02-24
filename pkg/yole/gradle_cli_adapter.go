package yole

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GradleCLIAdapter executes Gradle tasks via subprocess.
type GradleCLIAdapter struct {
	ProjectRoot string
	UseDocker   bool
}

// NewGradleCLIAdapter creates a GradleCLIAdapter.
func NewGradleCLIAdapter(
	projectRoot string, useDocker bool,
) *GradleCLIAdapter {
	return &GradleCLIAdapter{
		ProjectRoot: projectRoot,
		UseDocker:   useDocker,
	}
}

// RunTask executes a Gradle task and returns the result.
func (g *GradleCLIAdapter) RunTask(
	ctx context.Context, task string, args ...string,
) (*GradleRunResult, error) {
	start := time.Now()

	cmdArgs := []string{task}
	cmdArgs = append(cmdArgs, args...)

	var cmd *exec.Cmd
	if g.UseDocker {
		dockerArgs := []string{
			"compose", "run", "--rm", "build",
			"./gradlew",
		}
		dockerArgs = append(dockerArgs, cmdArgs...)
		cmd = exec.CommandContext(ctx, "docker", dockerArgs...)
	} else {
		cmd = exec.CommandContext(
			ctx,
			filepath.Join(g.ProjectRoot, "gradlew"),
			cmdArgs...,
		)
	}
	cmd.Dir = g.ProjectRoot

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := &GradleRunResult{
		Task:     task,
		Success:  err == nil,
		Duration: duration,
		Output:   string(output),
	}

	return result, err
}

// RunTests executes a Gradle test task and parses JUnit XML.
func (g *GradleCLIAdapter) RunTests(
	ctx context.Context, task string, testFilter string,
) (*GradleRunResult, error) {
	args := []string{}
	if testFilter != "" {
		args = append(args, "--tests", testFilter)
	}

	result, err := g.RunTask(ctx, task, args...)
	if err != nil && result == nil {
		return nil, fmt.Errorf(
			"gradle task failed: %w", err,
		)
	}

	suites, parseErr := g.ParseJUnitResults()
	if parseErr == nil {
		result.Suites = suites
	}

	return result, err
}

// Available checks if gradlew exists.
func (g *GradleCLIAdapter) Available(
	ctx context.Context,
) bool {
	path := filepath.Join(g.ProjectRoot, "gradlew")
	_, err := os.Stat(path)
	return err == nil
}

// ParseJUnitResults finds and parses JUnit XML files from
// test output.
func (g *GradleCLIAdapter) ParseJUnitResults() (
	[]JUnitTestSuite, error,
) {
	var allSuites []JUnitTestSuite

	searchPaths := []string{
		filepath.Join(
			g.ProjectRoot, "shared", "build",
			"test-results",
		),
		filepath.Join(
			g.ProjectRoot, "androidApp", "build",
			"test-results",
		),
		filepath.Join(
			g.ProjectRoot, "desktopApp", "build",
			"test-results",
		),
		filepath.Join(
			g.ProjectRoot, "webApp", "build",
			"test-results",
		),
	}

	for _, searchPath := range searchPaths {
		_ = filepath.Walk(
			searchPath,
			func(
				path string, info os.FileInfo, err error,
			) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() &&
					strings.HasSuffix(path, ".xml") {
					suites, parseErr := parseJUnitXML(path)
					if parseErr == nil {
						allSuites = append(
							allSuites, suites...,
						)
					}
				}
				return nil
			},
		)
	}

	return allSuites, nil
}

func parseJUnitXML(path string) ([]JUnitTestSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var suites JUnitTestSuites
	if err := xml.Unmarshal(data, &suites); err == nil &&
		len(suites.TestSuites) > 0 {
		return suites.TestSuites, nil
	}

	var suite JUnitTestSuite
	if err := xml.Unmarshal(data, &suite); err == nil {
		return []JUnitTestSuite{suite}, nil
	}

	return nil, fmt.Errorf(
		"unable to parse JUnit XML: %s", path,
	)
}
