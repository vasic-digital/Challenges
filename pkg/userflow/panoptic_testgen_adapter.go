package userflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// PanopticTestGenAdapter implements TestGenAdapter by invoking
// the Panoptic CLI's testgen subcommand.
type PanopticTestGenAdapter struct {
	binaryPath string
}

// Compile-time interface check.
var _ TestGenAdapter = (*PanopticTestGenAdapter)(nil)

// NewPanopticTestGenAdapter creates an adapter that invokes the
// Panoptic binary at the given path for test generation.
func NewPanopticTestGenAdapter(
	binaryPath string,
) *PanopticTestGenAdapter {
	return &PanopticTestGenAdapter{binaryPath: binaryPath}
}

// panopticTestStep is the flat JSON structure returned by the
// panoptic testgen generate CLI command for a single step.
type panopticTestStep struct {
	Action     string            `json:"action"`
	Target     string            `json:"target"`
	Value      string            `json:"value,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// panopticGeneratedTest is the JSON structure returned by the
// panoptic testgen generate CLI command.
type panopticGeneratedTest struct {
	Name       string             `json:"name"`
	Category   string             `json:"category"`
	Priority   string             `json:"priority"`
	Confidence float64            `json:"confidence"`
	Steps      []panopticTestStep `json:"steps"`
}

// toGeneratedTest converts the CLI JSON struct into a
// GeneratedTest with map[string]any parameters.
func (p panopticGeneratedTest) toGeneratedTest() GeneratedTest {
	steps := make([]TestStep, len(p.Steps))
	for i, s := range p.Steps {
		var params map[string]any
		if len(s.Parameters) > 0 {
			params = make(map[string]any, len(s.Parameters))
			for k, v := range s.Parameters {
				params[k] = v
			}
		}
		steps[i] = TestStep{
			Action:     s.Action,
			Target:     s.Target,
			Value:      s.Value,
			Parameters: params,
		}
	}
	return GeneratedTest{
		Name:       p.Name,
		Category:   p.Category,
		Priority:   p.Priority,
		Confidence: p.Confidence,
		Steps:      steps,
	}
}

// GenerateTests writes the screenshot to a temp file and
// invokes panoptic testgen generate to produce test cases.
func (a *PanopticTestGenAdapter) GenerateTests(
	ctx context.Context, screenshot []byte,
) ([]GeneratedTest, error) {
	tmpFile, err := os.CreateTemp("", "testgen-*.png")
	if err != nil {
		return nil, fmt.Errorf(
			"create temp screenshot: %w", err,
		)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(screenshot); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf(
			"write screenshot: %w", err,
		)
	}
	tmpFile.Close()

	cmd := exec.CommandContext(
		ctx, a.binaryPath,
		"testgen", "generate",
		"--screenshot", tmpFile.Name(),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"panoptic testgen generate: %w\noutput: %s",
			err, string(out),
		)
	}

	var raw []panopticGeneratedTest
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf(
			"parse testgen output: %w", err,
		)
	}

	tests := make([]GeneratedTest, len(raw))
	for i, r := range raw {
		tests[i] = r.toGeneratedTest()
	}
	return tests, nil
}

// GenerateReport writes the screenshot to a temp file and
// invokes panoptic testgen report to produce a markdown
// analysis report.
func (a *PanopticTestGenAdapter) GenerateReport(
	ctx context.Context, screenshot []byte,
) (string, error) {
	tmpFile, err := os.CreateTemp("", "testgen-*.png")
	if err != nil {
		return "", fmt.Errorf(
			"create temp screenshot: %w", err,
		)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(screenshot); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf(
			"write screenshot: %w", err,
		)
	}
	tmpFile.Close()

	tmpReport, err := os.CreateTemp("", "report-*.md")
	if err != nil {
		return "", fmt.Errorf(
			"create temp report: %w", err,
		)
	}
	defer os.Remove(tmpReport.Name())
	tmpReport.Close()

	cmd := exec.CommandContext(
		ctx, a.binaryPath,
		"testgen", "report",
		"--screenshot", tmpFile.Name(),
		"--output", tmpReport.Name(),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"panoptic testgen report: %w\noutput: %s",
			err, string(out),
		)
	}

	report, err := os.ReadFile(tmpReport.Name())
	if err != nil {
		return "", fmt.Errorf(
			"read report output: %w", err,
		)
	}
	return string(report), nil
}

// Available reports whether the Panoptic binary exists at the
// configured path.
func (a *PanopticTestGenAdapter) Available(
	_ context.Context,
) bool {
	if _, err := os.Stat(a.binaryPath); err == nil {
		return true
	}
	// Fall back to PATH lookup.
	if _, err := exec.LookPath(a.binaryPath); err == nil {
		return true
	}
	return false
}
