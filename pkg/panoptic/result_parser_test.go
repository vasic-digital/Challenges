package panoptic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseResultToAssertionValues_Nil(t *testing.T) {
	values := ParseResultToAssertionValues(nil)
	assert.Empty(t, values)
}

func TestParseResultToAssertionValues_Basic(t *testing.T) {
	result := &PanopticRunResult{
		ExitCode: 0,
		Apps: []AppResult{
			{
				Name:       "Admin",
				Success:    true,
				DurationMs: 5000,
			},
			{
				Name:       "Web",
				Success:    true,
				DurationMs: 3000,
			},
		},
		Screenshots: []string{"/tmp/a.png", "/tmp/b.png"},
		Videos:      []string{"/tmp/v.mp4"},
		Duration:    8 * time.Second,
		Stdout:      "test output",
	}

	values := ParseResultToAssertionValues(result)

	assert.Equal(t, 0, values["exit_code"])
	assert.Equal(t, true, values["all_apps_passed"])
	assert.Equal(t, 2, values["app_count"])
	assert.Equal(t, 2, values["passed_count"])
	assert.Equal(t, 0, values["failed_count"])
	assert.Equal(t, 2, values["total_screenshots"])
	assert.Equal(t, 1, values["total_videos"])
	assert.Equal(t, int64(8000), values["total_duration_ms"])
	assert.Equal(t, int64(5000), values["max_duration_ms"])
	assert.Equal(t, "test output", values["stdout"])
}

func TestParseResultToAssertionValues_FailedApp(t *testing.T) {
	result := &PanopticRunResult{
		ExitCode: 1,
		Apps: []AppResult{
			{Name: "Admin", Success: true, DurationMs: 2000},
			{Name: "Web", Success: false, DurationMs: 1000},
		},
		Duration: 3 * time.Second,
	}

	values := ParseResultToAssertionValues(result)

	assert.Equal(t, false, values["all_apps_passed"])
	assert.Equal(t, 1, values["passed_count"])
	assert.Equal(t, 1, values["failed_count"])
}

func TestParseResultToAssertionValues_WithArtifacts(t *testing.T) {
	result := &PanopticRunResult{
		ExitCode:         0,
		AIErrorReport:    "/tmp/ai_errors.json",
		AIGeneratedTests: "/tmp/ai_tests.yaml",
		VisionReport:     "/tmp/vision.json",
		Duration:         1 * time.Second,
	}

	values := ParseResultToAssertionValues(result)

	assert.Equal(t,
		"/tmp/ai_errors.json",
		values["ai_error_report"],
	)
	assert.Equal(t,
		"/tmp/ai_tests.yaml",
		values["ai_generated_tests"],
	)
	assert.Equal(t,
		"/tmp/vision.json",
		values["vision_report"],
	)
}

func TestParseResultToMetrics_Nil(t *testing.T) {
	metrics := ParseResultToMetrics(nil)
	assert.Empty(t, metrics)
}

func TestParseResultToMetrics_Basic(t *testing.T) {
	result := &PanopticRunResult{
		ExitCode: 0,
		Apps: []AppResult{
			{
				Name:       "Admin",
				Success:    true,
				DurationMs: 5000,
			},
		},
		Screenshots: []string{"/tmp/a.png"},
		Videos:      []string{"/tmp/v.mp4"},
		Duration:    5 * time.Second,
	}

	metrics := ParseResultToMetrics(result)

	assert.Equal(t, float64(5000),
		metrics["total_duration_ms"].Value,
	)
	assert.Equal(t, "ms",
		metrics["total_duration_ms"].Unit,
	)
	assert.Equal(t, float64(1),
		metrics["app_count"].Value,
	)
	assert.Equal(t, float64(1),
		metrics["screenshot_count"].Value,
	)
	assert.Equal(t, float64(1),
		metrics["video_count"].Value,
	)
	assert.Equal(t, float64(1),
		metrics["passed_count"].Value,
	)
	assert.Equal(t, float64(0),
		metrics["failed_count"].Value,
	)
}

func TestParseResultToMetrics_PerAppDuration(t *testing.T) {
	result := &PanopticRunResult{
		Apps: []AppResult{
			{DurationMs: 3000},
			{DurationMs: 5000},
		},
		Duration: 8 * time.Second,
	}

	metrics := ParseResultToMetrics(result)

	assert.Equal(t, float64(3000),
		metrics["app_0_duration_ms"].Value,
	)
	assert.Equal(t, float64(5000),
		metrics["app_1_duration_ms"].Value,
	)
}

func Test_allAppsPassed(t *testing.T) {
	tests := []struct {
		name   string
		result *PanopticRunResult
		want   bool
	}{
		{
			name: "all pass",
			result: &PanopticRunResult{
				Apps: []AppResult{
					{Success: true},
					{Success: true},
				},
			},
			want: true,
		},
		{
			name: "one fails",
			result: &PanopticRunResult{
				Apps: []AppResult{
					{Success: true},
					{Success: false},
				},
			},
			want: false,
		},
		{
			name: "no apps exit 0",
			result: &PanopticRunResult{
				ExitCode: 0,
			},
			want: true,
		},
		{
			name: "no apps exit 1",
			result: &PanopticRunResult{
				ExitCode: 1,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, allAppsPassed(tt.result))
		})
	}
}
