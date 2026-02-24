package userflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTestResultToValues(t *testing.T) {
	tests := []struct {
		name   string
		result *TestResult
		checks func(t *testing.T, vals map[string]any)
	}{
		{
			name:   "nil result",
			result: nil,
			checks: func(t *testing.T, vals map[string]any) {
				assert.Empty(t, vals)
			},
		},
		{
			name: "all passing",
			result: &TestResult{
				TotalTests:  10,
				TotalFailed: 0,
				TotalErrors: 0,
				Duration:    5 * time.Second,
				Suites:      []TestSuite{{Name: "s1"}},
				Output:      "ok",
			},
			checks: func(t *testing.T, vals map[string]any) {
				assert.Equal(t, 10, vals["total_tests"])
				assert.Equal(t, 0, vals["total_failed"])
				assert.Equal(t, 0, vals["total_errors"])
				assert.Equal(t, 5000, vals["duration_ms"])
				assert.Equal(t, 1, vals["suite_count"])
				assert.Equal(t, "ok", vals["output"])
				assert.Equal(t, true, vals["all_tests_pass"])
			},
		},
		{
			name: "with failures",
			result: &TestResult{
				TotalTests:  10,
				TotalFailed: 2,
				TotalErrors: 1,
				Duration:    3 * time.Second,
			},
			checks: func(t *testing.T, vals map[string]any) {
				assert.Equal(t, 2, vals["total_failed"])
				assert.Equal(t, 1, vals["total_errors"])
				assert.Equal(
					t, false, vals["all_tests_pass"],
				)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals := ParseTestResultToValues(tt.result)
			tt.checks(t, vals)
		})
	}
}

func TestParseTestResultToMetrics(t *testing.T) {
	tests := []struct {
		name   string
		result *TestResult
		checks func(
			t *testing.T,
			m map[string]any,
		)
	}{
		{
			name:   "nil result",
			result: nil,
			checks: func(
				t *testing.T, m map[string]any,
			) {
				assert.Empty(t, m)
			},
		},
		{
			name: "valid result",
			result: &TestResult{
				TotalTests:   20,
				TotalFailed:  1,
				TotalErrors:  0,
				TotalSkipped: 2,
				Duration:     10 * time.Second,
			},
			checks: func(
				t *testing.T, m map[string]any,
			) {
				assert.NotEmpty(t, m)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := ParseTestResultToMetrics(tt.result)
			if tt.result == nil {
				assert.Empty(t, metrics)
				return
			}
			assert.Equal(
				t, float64(20),
				metrics["total_tests"].Value,
			)
			assert.Equal(
				t, float64(1),
				metrics["total_failed"].Value,
			)
			assert.Equal(
				t, float64(0),
				metrics["total_errors"].Value,
			)
			assert.Equal(
				t, float64(2),
				metrics["total_skipped"].Value,
			)
			assert.Equal(
				t, 10.0,
				metrics["duration"].Value,
			)
			assert.Equal(
				t, "count",
				metrics["total_tests"].Unit,
			)
			assert.Equal(
				t, "s",
				metrics["duration"].Unit,
			)
		})
	}
}

func TestParseBuildResultToValues(t *testing.T) {
	tests := []struct {
		name   string
		result *BuildResult
		checks func(t *testing.T, vals map[string]any)
	}{
		{
			name:   "nil result",
			result: nil,
			checks: func(t *testing.T, vals map[string]any) {
				assert.Empty(t, vals)
			},
		},
		{
			name: "successful build",
			result: &BuildResult{
				Target:   "api",
				Success:  true,
				Duration: 30 * time.Second,
				Output:   "ok",
				Artifacts: []string{
					"bin/api", "bin/api.exe",
				},
			},
			checks: func(t *testing.T, vals map[string]any) {
				assert.Equal(t, "api", vals["target"])
				assert.Equal(t, true, vals["success"])
				assert.Equal(t, 30000, vals["duration_ms"])
				assert.Equal(t, "ok", vals["output"])
				assert.Equal(
					t, 2, vals["artifact_count"],
				)
			},
		},
		{
			name: "failed build",
			result: &BuildResult{
				Target:    "web",
				Success:   false,
				Duration:  5 * time.Second,
				Output:    "error",
				Artifacts: nil,
			},
			checks: func(t *testing.T, vals map[string]any) {
				assert.Equal(t, false, vals["success"])
				assert.Equal(
					t, 0, vals["artifact_count"],
				)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals := ParseBuildResultToValues(tt.result)
			tt.checks(t, vals)
		})
	}
}

func TestParseBuildResultToMetrics(t *testing.T) {
	tests := []struct {
		name   string
		result *BuildResult
		checks func(t *testing.T, empty bool)
	}{
		{
			name:   "nil result",
			result: nil,
			checks: func(t *testing.T, empty bool) {
				assert.True(t, empty)
			},
		},
		{
			name: "successful build",
			result: &BuildResult{
				Target:   "api",
				Success:  true,
				Duration: 30 * time.Second,
				Artifacts: []string{
					"bin/api",
				},
			},
			checks: func(t *testing.T, empty bool) {
				assert.False(t, empty)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := ParseBuildResultToMetrics(tt.result)
			if tt.result == nil {
				assert.Empty(t, metrics)
				tt.checks(t, true)
				return
			}
			tt.checks(t, false)
			assert.Equal(
				t, 1.0,
				metrics["build_success"].Value,
			)
			assert.Equal(
				t, 30.0,
				metrics["build_duration"].Value,
			)
			assert.Equal(
				t, float64(1),
				metrics["artifact_count"].Value,
			)
			assert.Equal(
				t, "bool",
				metrics["build_success"].Unit,
			)
			assert.Equal(
				t, "s",
				metrics["build_duration"].Unit,
			)
		})
	}
}
