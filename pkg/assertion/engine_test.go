package assertion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine_RegistersAllBuiltins(t *testing.T) {
	e := NewEngine()

	builtins := []string{
		"not_empty", "not_mock", "contains",
		"contains_any", "min_length", "quality_score",
		"reasoning_present", "code_valid", "min_count",
		"exact_count", "max_latency", "all_valid",
		"no_duplicates", "all_pass", "no_mock_responses",
		"min_score",
	}

	for _, name := range builtins {
		assert.True(t, e.HasEvaluator(name),
			"missing built-in evaluator: %s", name)
	}
}

func TestDefaultEngine_Register_Success(t *testing.T) {
	e := NewEngine()

	err := e.Register("custom", func(
		_ Definition, _ any,
	) (bool, string) {
		return true, "custom ok"
	})

	require.NoError(t, err)
	assert.True(t, e.HasEvaluator("custom"))
}

func TestDefaultEngine_Register_Duplicate(t *testing.T) {
	e := NewEngine()

	err := e.Register("not_empty", func(
		_ Definition, _ any,
	) (bool, string) {
		return true, "dup"
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestDefaultEngine_Evaluate_UnknownType(t *testing.T) {
	e := NewEngine()

	r := e.Evaluate(Definition{
		Type:   "nonexistent",
		Target: "x",
	}, "hello")

	assert.False(t, r.Passed)
	assert.Contains(t, r.Message, "unknown assertion type")
}

func TestDefaultEngine_Evaluate_SetsFields(t *testing.T) {
	e := NewEngine()

	r := e.Evaluate(Definition{
		Type:   "not_empty",
		Target: "response",
		Value:  nil,
	}, "hello world")

	assert.True(t, r.Passed)
	assert.Equal(t, "not_empty", r.Type)
	assert.Equal(t, "response", r.Target)
	assert.Equal(t, "hello world", r.Actual)
}

func TestDefaultEngine_EvaluateAll_MissingTarget(t *testing.T) {
	e := NewEngine()

	results := e.EvaluateAll(
		[]Definition{
			{Type: "not_empty", Target: "missing"},
		},
		map[string]any{"other": "value"},
	)

	require.Len(t, results, 1)
	assert.False(t, results[0].Passed)
	assert.Contains(t, results[0].Message, "target not found")
}

func TestDefaultEngine_EvaluateAll_MultipleAssertions(t *testing.T) {
	e := NewEngine()

	results := e.EvaluateAll(
		[]Definition{
			{Type: "not_empty", Target: "a"},
			{Type: "contains", Target: "a", Value: "hello"},
			{Type: "min_length", Target: "a", Value: 3},
		},
		map[string]any{"a": "hello world"},
	)

	require.Len(t, results, 3)
	for _, r := range results {
		assert.True(t, r.Passed, "assertion %s failed", r.Type)
	}
}

func TestDefaultEngine_HasEvaluator(t *testing.T) {
	e := NewEngine()

	assert.True(t, e.HasEvaluator("contains"))
	assert.False(t, e.HasEvaluator("does_not_exist"))
}
