package assertion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateNotEmpty(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"nil value", nil, false},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"non-empty string", "hello", true},
		{"empty slice", []any{}, false},
		{"non-empty slice", []any{1}, true},
		{"empty map", map[string]any{}, false},
		{"non-empty map", map[string]any{"k": "v"}, true},
		{"integer", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := evaluateNotEmpty(Definition{}, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateNotMock(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"real response", "Here is a thoughtful answer", true},
		{"lorem ipsum", "Lorem Ipsum dolor sit", false},
		{"placeholder", "This is a placeholder", false},
		{"mock response", "mock response data", false},
		{"TODO", "TODO: implement later", false},
		{"not implemented", "not implemented yet", false},
		{"MOCK tag", "[MOCK] fake data", false},
		{"test response", "test response value", false},
		{"dummy", "dummy output here", false},
		{"sample output", "sample output text", false},
		{"non-string", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := evaluateNotMock(Definition{}, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateContains(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected any
		passed   bool
	}{
		{
			"found case-insensitive",
			"Hello World", "hello", true,
		},
		{
			"not found", "Hello World", "xyz", false,
		},
		{
			"non-string value", 42, "42", false,
		},
		{
			"non-string expected",
			"hello", 42, false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Definition{Value: tt.expected}
			ok, _ := evaluateContains(d, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateContainsAny(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		def    Definition
		passed bool
	}{
		{
			"csv string match",
			"Go is great",
			Definition{Value: "python,go,rust"},
			true,
		},
		{
			"csv string no match",
			"Java is fine",
			Definition{Value: "python,go,rust"},
			false,
		},
		{
			"slice of any match",
			"Python rocks",
			Definition{Value: []any{"go", "python", "rust"}},
			true,
		},
		{
			"values field match",
			"Use Rust here",
			Definition{Values: []any{"go", "rust"}},
			true,
		},
		{
			"string slice match",
			"Use Go here",
			Definition{Value: []string{"go", "rust"}},
			true,
		},
		{
			"non-string value",
			42,
			Definition{Value: "42"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := evaluateContainsAny(tt.def, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateMinLength(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		min    any
		passed bool
	}{
		{"meets minimum", "hello", 5, true},
		{"exceeds minimum", "hello world", 5, true},
		{"below minimum", "hi", 5, false},
		{"float64 minimum", "hello", float64(3), true},
		{"non-string value", 42, 1, false},
		{"non-number minimum", "hello", "five", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Definition{Value: tt.min}
			ok, _ := evaluateMinLength(d, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateQualityScore(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		min    any
		passed bool
	}{
		{"above threshold", float64(8.5), float64(7.0), true},
		{"equal threshold", float64(7.0), float64(7.0), true},
		{"below threshold", float64(6.0), float64(7.0), false},
		{"int min", float64(8.0), 7, true},
		{"non-number value", "high", float64(7.0), false},
		{"non-number min", float64(8.0), "seven", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Definition{Value: tt.min}
			ok, _ := evaluateQualityScore(d, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateReasoningPresent(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{
			"has because",
			"This works because of X", true,
		},
		{
			"has step",
			"Step 1: do this", true,
		},
		{
			"no indicators",
			"The answer is 42", false,
		},
		{
			"non-string", 42, false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := evaluateReasoningPresent(
				Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateCodeValid(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{
			"go function",
			"func main() {}", true,
		},
		{
			"python function",
			"def hello():", true,
		},
		{
			"code block",
			"```\ncode here\n```", true,
		},
		{
			"plain text",
			"This is just text.", false,
		},
		{
			"import statement",
			"import fmt", true,
		},
		{
			"non-string", 42, false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := evaluateCodeValid(
				Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateMinCount(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		min    any
		passed bool
	}{
		{"int meets", 5, 3, true},
		{"int below", 2, 3, false},
		{"float64 meets", float64(5), 3, true},
		{"slice meets", []any{1, 2, 3}, 2, true},
		{"slice below", []any{1}, 2, false},
		{"map meets", map[string]any{"a": 1, "b": 2}, 2, true},
		{"non-countable", "hello", 1, false},
		{"non-number min", 5, "three", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Definition{Value: tt.min}
			ok, _ := evaluateMinCount(d, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateExactCount(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected any
		passed   bool
	}{
		{"int match", 3, 3, true},
		{"int mismatch", 3, 5, false},
		{"slice match", []any{1, 2}, 2, true},
		{"slice mismatch", []any{1, 2}, 3, false},
		{"non-countable", "hi", 2, false},
		{"non-number expected", 3, "three", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Definition{Value: tt.expected}
			ok, _ := evaluateExactCount(d, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateMaxLatency(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		max    any
		passed bool
	}{
		{"within limit", int64(100), int64(200), true},
		{"at limit", int64(200), int64(200), true},
		{"over limit", int64(300), int64(200), false},
		{"int value", 100, 200, true},
		{"float64 value", float64(100), float64(200), true},
		{"non-number value", "fast", 200, false},
		{"non-number max", 100, "fast", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Definition{Value: tt.max}
			ok, _ := evaluateMaxLatency(d, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateAllValid(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"all valid", []any{"a", "b", 1}, true},
		{"contains nil", []any{"a", nil}, false},
		{"contains empty", []any{"a", ""}, false},
		{"non-array", "hello", false},
		{"empty array", []any{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := evaluateAllValid(Definition{}, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateNoDuplicates(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{"unique", []any{"a", "b", "c"}, true},
		{"duplicate", []any{"a", "b", "a"}, false},
		{"empty", []any{}, true},
		{"non-array", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := evaluateNoDuplicates(
				Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateAllPass(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{
			"all results passed",
			[]Result{
				{Passed: true},
				{Passed: true},
			},
			true,
		},
		{
			"one result failed",
			[]Result{
				{Passed: true},
				{Passed: false, Type: "x", Message: "fail"},
			},
			false,
		},
		{
			"map items all passed",
			[]any{
				map[string]any{"passed": true},
				map[string]any{"passed": true},
			},
			true,
		},
		{
			"map items one failed",
			[]any{
				map[string]any{"passed": true},
				map[string]any{"passed": false},
			},
			false,
		},
		{
			"non-array", "hello", false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := evaluateAllPass(Definition{}, tt.value)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateNoMockResponses(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		passed bool
	}{
		{
			"all real",
			[]any{"Real answer one", "Real answer two"},
			true,
		},
		{
			"one mocked",
			[]any{"Real answer", "lorem ipsum dolor"},
			false,
		},
		{
			"single real",
			"This is a real answer",
			true,
		},
		{
			"single mock",
			"placeholder data",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := evaluateNoMockResponses(
				Definition{}, tt.value,
			)
			assert.Equal(t, tt.passed, ok)
		})
	}
}

func TestEvaluateMinScore(t *testing.T) {
	d := Definition{Value: float64(7.0)}

	ok, _ := evaluateMinScore(d, float64(8.0))
	assert.True(t, ok)

	ok, _ = evaluateMinScore(d, float64(5.0))
	assert.False(t, ok)
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int
		ok   bool
	}{
		{"int", 5, 5, true},
		{"float64", float64(5.7), 5, true},
		{"int64", int64(5), 5, true},
		{"string", "5", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toInt(tt.val)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want float64
		ok   bool
	}{
		{"float64", float64(3.14), 3.14, true},
		{"int", 3, 3.0, true},
		{"int64", int64(3), 3.0, true},
		{"string", "3.14", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toFloat64(tt.val)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.InDelta(t, tt.want, got, 0.001)
			}
		})
	}
}

func TestToCount(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int
		ok   bool
	}{
		{"int", 5, 5, true},
		{"float64", float64(5.7), 5, true},
		{"int64", int64(5), 5, true},
		{"slice", []any{1, 2, 3}, 3, true},
		{"map", map[string]any{"a": 1, "b": 2}, 2, true},
		{"string", "hello", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toCount(tt.val)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
