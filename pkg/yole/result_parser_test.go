package yole

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseGradleResultToValues_Basic(t *testing.T) {
	result := &GradleRunResult{
		Task:     ":shared:test",
		Success:  true,
		Duration: 5 * time.Second,
		Output:   "BUILD SUCCESSFUL",
		Suites: []JUnitTestSuite{
			{
				Name:     "FormatRegistryTests",
				Tests:    20,
				Failures: 0,
				Errors:   0,
			},
			{
				Name:     "MarkdownParserTests",
				Tests:    15,
				Failures: 2,
				Errors:   1,
			},
		},
	}

	values := ParseGradleResultToValues(result)

	assert.Equal(t, true, values["success"])
	assert.Equal(t, 5.0, values["duration"])
	assert.Equal(t, "BUILD SUCCESSFUL", values["output"])
	assert.Equal(t, ":shared:test", values["task"])
	assert.Equal(t, 35, values["total_tests"])
	assert.Equal(t, 2, values["total_failures"])
	assert.Equal(t, 1, values["total_errors"])
	assert.Equal(t, 2, values["suite_count"])
}

func TestParseGradleResultToValues_NoSuites(t *testing.T) {
	result := &GradleRunResult{
		Task:    ":shared:compileKotlinJvm",
		Success: true,
	}

	values := ParseGradleResultToValues(result)

	assert.Equal(t, true, values["success"])
	assert.Equal(t, 0, values["total_tests"])
	assert.Equal(t, 0, values["total_failures"])
	assert.Equal(t, 0, values["total_errors"])
	assert.Equal(t, 0, values["suite_count"])
}

func TestParseGradleResultToValues_Failed(t *testing.T) {
	result := &GradleRunResult{
		Task:     ":androidApp:assembleDebug",
		Success:  false,
		Duration: 30 * time.Second,
		Output:   "BUILD FAILED",
	}

	values := ParseGradleResultToValues(result)

	assert.Equal(t, false, values["success"])
	assert.Equal(t, 30.0, values["duration"])
	assert.Equal(t, "BUILD FAILED", values["output"])
}

func TestParseGradleResultToMetrics_Basic(t *testing.T) {
	result := &GradleRunResult{
		Task:     ":shared:test",
		Duration: 10 * time.Second,
		Suites: []JUnitTestSuite{
			{Tests: 30, Failures: 1, Errors: 2},
			{Tests: 20, Failures: 0, Errors: 0},
		},
	}

	metrics := ParseGradleResultToMetrics(result)

	assert.Equal(t, 10.0, metrics["duration"].Value)
	assert.Equal(t, "seconds", metrics["duration"].Unit)
	assert.Equal(t, float64(50), metrics["total_tests"].Value)
	assert.Equal(t, "count", metrics["total_tests"].Unit)
	assert.Equal(t, float64(3),
		metrics["total_failures"].Value,
	)
}

func TestParseGradleResultToMetrics_NoSuites(t *testing.T) {
	result := &GradleRunResult{
		Duration: 2 * time.Second,
	}

	metrics := ParseGradleResultToMetrics(result)

	assert.Equal(t, 2.0, metrics["duration"].Value)
	assert.Equal(t, float64(0), metrics["total_tests"].Value)
	assert.Equal(t, float64(0),
		metrics["total_failures"].Value,
	)
}

func TestParseGradleResultToMetrics_ZeroDuration(t *testing.T) {
	result := &GradleRunResult{}

	metrics := ParseGradleResultToMetrics(result)

	assert.Equal(t, 0.0, metrics["duration"].Value)
}
