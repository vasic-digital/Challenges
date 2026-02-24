package userflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/assertion"
)

func TestRegisterEvaluators_Success(t *testing.T) {
	engine := assertion.NewEngine()
	err := RegisterEvaluators(engine)
	require.NoError(t, err)

	evaluatorNames := []string{
		"build_succeeds", "all_tests_pass", "lint_passes",
		"app_launches", "app_stable", "status_code",
		"response_contains", "response_not_empty",
		"json_field_equals", "screenshot_exists",
		"flow_completes", "within_duration",
	}
	for _, name := range evaluatorNames {
		assert.True(
			t, engine.HasEvaluator(name),
			"evaluator %s should be registered", name,
		)
	}
}

func TestRegisterEvaluators_DuplicateError(t *testing.T) {
	engine := assertion.NewEngine()
	err := RegisterEvaluators(engine)
	require.NoError(t, err)

	err = RegisterEvaluators(engine)
	assert.Error(t, err)
}

func TestEvaluateBuildSucceeds(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"true", true, true},
		{"false", false, false},
		{"wrong type", "yes", false},
		{"int type", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateBuildSucceeds(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateAllTestsPass(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"zero failures", 0, true},
		{"some failures", 3, false},
		{"float zero", float64(0), true},
		{"float nonzero", float64(2), false},
		{"wrong type", "0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateAllTestsPass(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateLintPasses(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"true", true, true},
		{"false", false, false},
		{"wrong type", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateLintPasses(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateAppLaunches(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"true", true, true},
		{"false", false, false},
		{"wrong type", "true", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateAppLaunches(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateAppStable(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"true", true, true},
		{"false", false, false},
		{"wrong type", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateAppStable(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		defValue any
		value    any
		passed   bool
	}{
		{"match int", 200, 200, true},
		{"mismatch int", 200, 404, false},
		{"match float", float64(200), float64(200), true},
		{"mismatch float", float64(200), float64(500), false},
		{"wrong value type", 200, "200", false},
		{"wrong def type", "200", 200, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := assertion.Definition{
				Value: tt.defValue,
			}
			passed, msg := evaluateStatusCode(
				def, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateResponseContains(t *testing.T) {
	tests := []struct {
		name     string
		defValue any
		value    any
		passed   bool
	}{
		{
			"contains",
			"success",
			"operation success",
			true,
		},
		{
			"not contains",
			"error",
			"all good",
			false,
		},
		{
			"wrong value type",
			"x",
			123,
			false,
		},
		{
			"wrong def type",
			123,
			"hello",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := assertion.Definition{
				Value: tt.defValue,
			}
			passed, msg := evaluateResponseContains(
				def, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateResponseNotEmpty(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"non-empty string", "hello", true},
		{"empty string", "", false},
		{"non-empty bytes", []byte{1, 2}, true},
		{"empty bytes", []byte{}, false},
		{"wrong type", 42, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateResponseNotEmpty(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateJSONFieldEquals(t *testing.T) {
	tests := []struct {
		name     string
		defValue any
		value    any
		passed   bool
	}{
		{"match string", "admin", "admin", true},
		{"mismatch string", "admin", "user", false},
		{"match int", 42, 42, true},
		{"mismatch int", 42, 43, false},
		{"match via sprintf", "true", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := assertion.Definition{
				Value: tt.defValue,
			}
			passed, msg := evaluateJSONFieldEquals(
				def, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateScreenshotExists(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{
			"non-empty bytes",
			[]byte{0x89, 0x50, 0x4E, 0x47},
			true,
		},
		{"empty bytes", []byte{}, false},
		{"wrong type", "png", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateScreenshotExists(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateFlowCompletes(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"true", true, true},
		{"false", false, false},
		{"wrong type", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateFlowCompletes(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestEvaluateWithinDuration(t *testing.T) {
	tests := []struct {
		name     string
		defValue any
		value    any
		passed   bool
	}{
		{"within limit", 1000, 500, true},
		{"at limit", 1000, 1000, true},
		{"exceeds limit", 1000, 1500, false},
		{"wrong value type", 1000, "500", false},
		{"wrong def type", "1000", 500, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := assertion.Definition{
				Value: tt.defValue,
			}
			passed, msg := evaluateWithinDuration(
				def, tt.value,
			)
			assert.Equal(t, tt.passed, passed)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestToIntVal(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		expect int
		ok     bool
	}{
		{"int", 42, 42, true},
		{"int64", int64(100), 100, true},
		{"float64", float64(3.14), 3, true},
		{"float32", float32(2.7), 2, true},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := toIntVal(tt.value)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.expect, val)
			}
		})
	}
}
