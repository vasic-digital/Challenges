package panoptic

import (
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.challenges/pkg/assertion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterEvaluators(t *testing.T) {
	engine := assertion.NewEngine()
	err := RegisterEvaluators(engine)
	require.NoError(t, err)

	evaluatorNames := []string{
		"screenshot_exists", "video_exists",
		"no_ui_errors", "ai_confidence_above",
		"all_apps_passed", "max_duration",
		"report_exists", "app_count",
	}
	for _, name := range evaluatorNames {
		assert.True(t, engine.HasEvaluator(name),
			"evaluator %s not registered", name,
		)
	}
}

func TestRegisterEvaluators_Duplicate(t *testing.T) {
	engine := assertion.NewEngine()
	err := RegisterEvaluators(engine)
	require.NoError(t, err)

	// Second registration should fail.
	err = RegisterEvaluators(engine)
	assert.Error(t, err)
}

func TestEvaluateScreenshotExists(t *testing.T) {
	tests := []struct {
		name     string
		def      assertion.Definition
		value    any
		wantPass bool
	}{
		{
			name:     "enough screenshots as count",
			def:      assertion.Definition{Value: 2},
			value:    3,
			wantPass: true,
		},
		{
			name:     "not enough screenshots",
			def:      assertion.Definition{Value: 5},
			value:    2,
			wantPass: false,
		},
		{
			name:     "screenshots as slice",
			def:      assertion.Definition{Value: 1},
			value:    []any{"a.png", "b.png"},
			wantPass: true,
		},
		{
			name:     "default min count",
			def:      assertion.Definition{},
			value:    1,
			wantPass: true,
		},
		{
			name:     "zero screenshots",
			def:      assertion.Definition{},
			value:    0,
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateScreenshotExists(
				tt.def, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateVideoExists(t *testing.T) {
	tests := []struct {
		name     string
		def      assertion.Definition
		value    any
		wantPass bool
	}{
		{
			name:     "has video",
			def:      assertion.Definition{Value: 1},
			value:    []any{"v.mp4"},
			wantPass: true,
		},
		{
			name:     "no videos",
			def:      assertion.Definition{Value: 1},
			value:    []any{},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateVideoExists(
				tt.def, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateNoUIErrors(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantPass bool
	}{
		{
			name:     "empty string",
			value:    "",
			wantPass: true,
		},
		{
			name:     "non-string",
			value:    42,
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateNoUIErrors(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateNoUIErrors_WithFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Clean error report.
	cleanPath := filepath.Join(tmpDir, "clean.json")
	require.NoError(t, os.WriteFile(cleanPath, []byte(
		`{"error_count": 0, "status": "clean"}`,
	), 0o644))

	// Restore original readFileFunc after test.
	origRead := readFileFunc
	defer func() { readFileFunc = origRead }()
	readFileFunc = func(path string) ([]byte, error) {
		return os.ReadFile(path)
	}

	passed, _ := evaluateNoUIErrors(
		assertion.Definition{}, cleanPath,
	)
	assert.True(t, passed)

	// Error report with errors.
	errorPath := filepath.Join(tmpDir, "errors.json")
	require.NoError(t, os.WriteFile(errorPath, []byte(
		`{"errors": [{"msg": "button missing"}], "error_count": 1}`,
	), 0o644))

	passed, _ = evaluateNoUIErrors(
		assertion.Definition{}, errorPath,
	)
	assert.False(t, passed)
}

func TestEvaluateAIConfidenceAbove(t *testing.T) {
	tests := []struct {
		name     string
		def      assertion.Definition
		value    any
		wantPass bool
	}{
		{
			name:     "above threshold",
			def:      assertion.Definition{Value: 0.8},
			value:    0.95,
			wantPass: true,
		},
		{
			name:     "below threshold",
			def:      assertion.Definition{Value: 0.9},
			value:    0.5,
			wantPass: false,
		},
		{
			name:     "default threshold",
			def:      assertion.Definition{},
			value:    0.80,
			wantPass: true,
		},
		{
			name:     "not a number",
			def:      assertion.Definition{Value: 0.8},
			value:    "not a number",
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateAIConfidenceAbove(
				tt.def, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateAllAppsPassed(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantPass bool
	}{
		{
			name:     "all passed",
			value:    true,
			wantPass: true,
		},
		{
			name:     "some failed",
			value:    false,
			wantPass: false,
		},
		{
			name:     "not boolean",
			value:    "yes",
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateAllAppsPassed(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateMaxDuration(t *testing.T) {
	tests := []struct {
		name     string
		def      assertion.Definition
		value    any
		wantPass bool
	}{
		{
			name:     "within limit",
			def:      assertion.Definition{Value: int64(10000)},
			value:    int64(5000),
			wantPass: true,
		},
		{
			name:     "exceeds limit",
			def:      assertion.Definition{Value: int64(5000)},
			value:    int64(10000),
			wantPass: false,
		},
		{
			name:     "at limit",
			def:      assertion.Definition{Value: int64(5000)},
			value:    int64(5000),
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateMaxDuration(
				tt.def, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateReportExists(t *testing.T) {
	passed, _ := evaluateReportExists(
		assertion.Definition{}, true,
	)
	assert.True(t, passed)

	passed, _ = evaluateReportExists(
		assertion.Definition{}, false,
	)
	assert.False(t, passed)

	passed, _ = evaluateReportExists(
		assertion.Definition{}, "not bool",
	)
	assert.False(t, passed)
}

func TestEvaluateAppCount(t *testing.T) {
	tests := []struct {
		name     string
		def      assertion.Definition
		value    any
		wantPass bool
	}{
		{
			name:     "correct count",
			def:      assertion.Definition{Value: 2},
			value:    2,
			wantPass: true,
		},
		{
			name:     "wrong count",
			def:      assertion.Definition{Value: 3},
			value:    2,
			wantPass: false,
		},
		{
			name:     "not a number",
			def:      assertion.Definition{Value: 2},
			value:    "two",
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateAppCount(
				tt.def, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}
