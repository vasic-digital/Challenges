package userflow

import "context"

// ErrorAnalyzerAdapter analyzes log output for error
// patterns. Implementations may use regex matching, ML
// classification, or external tools.
type ErrorAnalyzerAdapter interface {
	// AnalyzeErrors processes log content and returns an
	// analysis of detected errors.
	AnalyzeErrors(
		ctx context.Context, logContent string,
	) (*ErrorAnalysis, error)

	// Available reports whether the adapter can run.
	Available(ctx context.Context) bool
}

// ErrorAnalysis contains the results of error analysis.
type ErrorAnalysis struct {
	TotalErrors     int                   `json:"total_errors"`
	Categories      map[string]int        `json:"categories"`
	Severity        map[string]int        `json:"severity"`
	Recommendations []ErrorRecommendation `json:"recommendations"`
}

// ErrorRecommendation is a suggested fix from error analysis.
type ErrorRecommendation struct {
	Type     string `json:"type"`
	Priority string `json:"priority"`
	Message  string `json:"message"`
	Impact   string `json:"impact"`
}
