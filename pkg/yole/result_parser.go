package yole

import (
	"digital.vasic.challenges/pkg/challenge"
)

// ParseGradleResultToValues converts a GradleRunResult into
// a map suitable for assertion evaluation.
func ParseGradleResultToValues(
	result *GradleRunResult,
) map[string]any {
	values := map[string]any{
		"success":  result.Success,
		"duration": result.Duration.Seconds(),
		"output":   result.Output,
		"task":     result.Task,
	}

	totalTests := 0
	totalFailures := 0
	totalErrors := 0

	for _, suite := range result.Suites {
		totalTests += suite.Tests
		totalFailures += suite.Failures
		totalErrors += suite.Errors
	}

	values["total_tests"] = totalTests
	values["total_failures"] = totalFailures
	values["total_errors"] = totalErrors
	values["suite_count"] = len(result.Suites)

	return values
}

// ParseGradleResultToMetrics converts a GradleRunResult into
// challenge MetricValue entries.
func ParseGradleResultToMetrics(
	result *GradleRunResult,
) map[string]challenge.MetricValue {
	metrics := map[string]challenge.MetricValue{
		"duration": {
			Name:  "duration",
			Value: result.Duration.Seconds(),
			Unit:  "seconds",
		},
	}

	totalTests := 0
	totalFailures := 0

	for _, suite := range result.Suites {
		totalTests += suite.Tests
		totalFailures += suite.Failures + suite.Errors
	}

	metrics["total_tests"] = challenge.MetricValue{
		Name:  "total_tests",
		Value: float64(totalTests),
		Unit:  "count",
	}
	metrics["total_failures"] = challenge.MetricValue{
		Name:  "total_failures",
		Value: float64(totalFailures),
		Unit:  "count",
	}

	return metrics
}
