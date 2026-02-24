package userflow

import (
	"digital.vasic.challenges/pkg/challenge"
)

// ParseTestResultToValues converts a TestResult into a map of
// named values suitable for assertion evaluation.
func ParseTestResultToValues(
	r *TestResult,
) map[string]any {
	if r == nil {
		return map[string]any{}
	}
	return map[string]any{
		"total_tests":   r.TotalTests,
		"total_failed":  r.TotalFailed,
		"total_errors":  r.TotalErrors,
		"total_skipped": r.TotalSkipped,
		"duration_ms":   int(r.Duration.Milliseconds()),
		"suite_count":   len(r.Suites),
		"output":        r.Output,
		"all_tests_pass": r.TotalFailed == 0 &&
			r.TotalErrors == 0,
	}
}

// ParseTestResultToMetrics converts a TestResult into a map
// of challenge MetricValue entries.
func ParseTestResultToMetrics(
	r *TestResult,
) map[string]challenge.MetricValue {
	if r == nil {
		return map[string]challenge.MetricValue{}
	}
	return map[string]challenge.MetricValue{
		"total_tests": {
			Name:  "total_tests",
			Value: float64(r.TotalTests),
			Unit:  "count",
		},
		"total_failed": {
			Name:  "total_failed",
			Value: float64(r.TotalFailed),
			Unit:  "count",
		},
		"total_errors": {
			Name:  "total_errors",
			Value: float64(r.TotalErrors),
			Unit:  "count",
		},
		"total_skipped": {
			Name:  "total_skipped",
			Value: float64(r.TotalSkipped),
			Unit:  "count",
		},
		"duration": {
			Name:  "duration",
			Value: r.Duration.Seconds(),
			Unit:  "s",
		},
	}
}

// ParseBuildResultToValues converts a BuildResult into a map
// of named values suitable for assertion evaluation.
func ParseBuildResultToValues(
	r *BuildResult,
) map[string]any {
	if r == nil {
		return map[string]any{}
	}
	return map[string]any{
		"target":         r.Target,
		"success":        r.Success,
		"duration_ms":    int(r.Duration.Milliseconds()),
		"output":         r.Output,
		"artifact_count": len(r.Artifacts),
	}
}

// ParseBuildResultToMetrics converts a BuildResult into a map
// of challenge MetricValue entries.
func ParseBuildResultToMetrics(
	r *BuildResult,
) map[string]challenge.MetricValue {
	if r == nil {
		return map[string]challenge.MetricValue{}
	}
	successVal := 0.0
	if r.Success {
		successVal = 1.0
	}
	return map[string]challenge.MetricValue{
		"build_success": {
			Name:  "build_success",
			Value: successVal,
			Unit:  "bool",
		},
		"build_duration": {
			Name:  "build_duration",
			Value: r.Duration.Seconds(),
			Unit:  "s",
		},
		"artifact_count": {
			Name:  "artifact_count",
			Value: float64(len(r.Artifacts)),
			Unit:  "count",
		},
	}
}
