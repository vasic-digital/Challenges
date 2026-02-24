package yole

import (
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
		"build_succeeds", "all_tests_pass",
		"lint_passes", "app_launches",
		"app_stable", "format_renders",
		"test_count_above", "no_test_failures",
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

func TestEvaluateBuildSucceeds(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantPass bool
		wantMsg  string
	}{
		{
			name:     "success",
			value:    true,
			wantPass: true,
			wantMsg:  "build succeeded",
		},
		{
			name:     "failure",
			value:    false,
			wantPass: false,
			wantMsg:  "build failed",
		},
		{
			name:     "not boolean",
			value:    "yes",
			wantPass: false,
		},
		{
			name:     "int type",
			value:    42,
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateBuildSucceeds(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
			if tt.wantMsg != "" {
				assert.Contains(t, msg, tt.wantMsg)
			}
		})
	}
}

func TestEvaluateAllTestsPass(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantPass bool
	}{
		{
			name:     "zero failures int",
			value:    0,
			wantPass: true,
		},
		{
			name:     "zero failures float",
			value:    0.0,
			wantPass: true,
		},
		{
			name:     "some failures int",
			value:    3,
			wantPass: false,
		},
		{
			name:     "some failures float",
			value:    2.0,
			wantPass: false,
		},
		{
			name:     "non-numeric",
			value:    "none",
			wantPass: true, // toIntVal returns 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateAllTestsPass(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateLintPasses(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantPass bool
	}{
		{
			name:     "passes",
			value:    true,
			wantPass: true,
		},
		{
			name:     "fails",
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
			passed, _ := evaluateLintPasses(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateAppLaunches(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantPass bool
	}{
		{
			name:     "launched",
			value:    true,
			wantPass: true,
		},
		{
			name:     "not launched",
			value:    false,
			wantPass: false,
		},
		{
			name:     "not boolean",
			value:    1,
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateAppLaunches(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateAppStable(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantPass bool
		wantMsg  string
	}{
		{
			name:     "stable",
			value:    true,
			wantPass: true,
			wantMsg:  "stable",
		},
		{
			name:     "crashed",
			value:    false,
			wantPass: false,
			wantMsg:  "crashed",
		},
		{
			name:     "not boolean",
			value:    "running",
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, msg := evaluateAppStable(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
			if tt.wantMsg != "" {
				assert.Contains(t, msg, tt.wantMsg)
			}
		})
	}
}

func TestEvaluateFormatRenders(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantPass bool
	}{
		{
			name:     "positive length int",
			value:    100,
			wantPass: true,
		},
		{
			name:     "positive length float",
			value:    42.0,
			wantPass: true,
		},
		{
			name:     "zero length",
			value:    0,
			wantPass: false,
		},
		{
			name:     "negative length",
			value:    -1,
			wantPass: false,
		},
		{
			name:     "non-numeric",
			value:    "abc",
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateFormatRenders(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateTestCountAbove(t *testing.T) {
	tests := []struct {
		name     string
		def      assertion.Definition
		value    any
		wantPass bool
	}{
		{
			name:     "above minimum",
			def:      assertion.Definition{Value: 10},
			value:    15,
			wantPass: true,
		},
		{
			name:     "at minimum",
			def:      assertion.Definition{Value: 10},
			value:    10,
			wantPass: true,
		},
		{
			name:     "below minimum",
			def:      assertion.Definition{Value: 10},
			value:    5,
			wantPass: false,
		},
		{
			name:     "nil min count",
			def:      assertion.Definition{},
			value:    5,
			wantPass: true,
		},
		{
			name:     "zero count zero min",
			def:      assertion.Definition{},
			value:    0,
			wantPass: true,
		},
		{
			name:     "float values",
			def:      assertion.Definition{Value: 5.0},
			value:    10.0,
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateTestCountAbove(
				tt.def, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestEvaluateNoTestFailures(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantPass bool
	}{
		{
			name:     "no failures",
			value:    0,
			wantPass: true,
		},
		{
			name:     "has failures",
			value:    3,
			wantPass: false,
		},
		{
			name:     "float zero",
			value:    0.0,
			wantPass: true,
		},
		{
			name:     "float failures",
			value:    2.0,
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := evaluateNoTestFailures(
				assertion.Definition{}, tt.value,
			)
			assert.Equal(t, tt.wantPass, passed)
		})
	}
}

func TestToIntVal(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int
	}{
		{"int", 42, 42},
		{"int64", int64(99), 99},
		{"float64", float64(7.8), 7},
		{"string", "hello", 0},
		{"nil", nil, 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, toIntVal(tt.val))
		})
	}
}
