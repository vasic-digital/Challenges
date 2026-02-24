package userflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// PanopticErrorAnalyzerAdapter implements ErrorAnalyzerAdapter
// by invoking the Panoptic CLI's errors subcommand.
type PanopticErrorAnalyzerAdapter struct {
	binaryPath string
}

// Compile-time interface check.
var _ ErrorAnalyzerAdapter = (*PanopticErrorAnalyzerAdapter)(nil)

// NewPanopticErrorAnalyzerAdapter creates an adapter that
// invokes the Panoptic binary at the given path for error
// analysis.
func NewPanopticErrorAnalyzerAdapter(
	binaryPath string,
) *PanopticErrorAnalyzerAdapter {
	return &PanopticErrorAnalyzerAdapter{
		binaryPath: binaryPath,
	}
}

// panopticErrorRecommendation is the JSON structure returned
// by the panoptic errors analyze CLI for a recommendation.
type panopticErrorRecommendation struct {
	Type     string `json:"type"`
	Priority string `json:"priority"`
	Message  string `json:"message"`
	Impact   string `json:"impact"`
}

// panopticErrorAnalysis is the JSON structure returned by the
// panoptic errors analyze CLI command.
type panopticErrorAnalysis struct {
	TotalErrors     int                           `json:"total_errors"`
	Categories      map[string]int                `json:"categories"`
	Severity        map[string]int                `json:"severity"`
	Recommendations []panopticErrorRecommendation `json:"recommendations"`
}

// toErrorAnalysis converts the CLI JSON struct into an
// ErrorAnalysis.
func (p panopticErrorAnalysis) toErrorAnalysis() *ErrorAnalysis {
	recs := make(
		[]ErrorRecommendation, len(p.Recommendations),
	)
	for i, r := range p.Recommendations {
		recs[i] = ErrorRecommendation{
			Type:     r.Type,
			Priority: r.Priority,
			Message:  r.Message,
			Impact:   r.Impact,
		}
	}
	return &ErrorAnalysis{
		TotalErrors:     p.TotalErrors,
		Categories:      p.Categories,
		Severity:        p.Severity,
		Recommendations: recs,
	}
}

// AnalyzeErrors writes the log content to a temp file and
// invokes panoptic errors analyze to detect error patterns.
func (a *PanopticErrorAnalyzerAdapter) AnalyzeErrors(
	ctx context.Context, logContent string,
) (*ErrorAnalysis, error) {
	tmpFile, err := os.CreateTemp("", "errors-*.log")
	if err != nil {
		return nil, fmt.Errorf(
			"create temp log file: %w", err,
		)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(logContent); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf(
			"write log content: %w", err,
		)
	}
	tmpFile.Close()

	cmd := exec.CommandContext(
		ctx, a.binaryPath,
		"errors", "analyze",
		"--input", tmpFile.Name(),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"panoptic errors analyze: %w\noutput: %s",
			err, string(out),
		)
	}

	var raw panopticErrorAnalysis
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf(
			"parse error analysis output: %w", err,
		)
	}

	return raw.toErrorAnalysis(), nil
}

// Available reports whether the Panoptic binary exists at the
// configured path.
func (a *PanopticErrorAnalyzerAdapter) Available(
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
