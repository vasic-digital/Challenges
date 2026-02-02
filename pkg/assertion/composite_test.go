package assertion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllPassComposite_AllPass(t *testing.T) {
	e := NewEngine()

	assertions := []Definition{
		{Type: "not_empty", Target: "response"},
		{Type: "contains", Target: "response", Value: "hello"},
	}
	values := map[string]any{
		"response": "hello world",
	}

	r := AllPassComposite(e, assertions, values)
	assert.True(t, r.Passed)
	assert.Equal(t, "all_pass", r.Type)
	assert.Contains(t, r.Message, "2 assertions passed")
}

func TestAllPassComposite_OneFails(t *testing.T) {
	e := NewEngine()

	assertions := []Definition{
		{Type: "not_empty", Target: "response"},
		{Type: "contains", Target: "response", Value: "xyz"},
	}
	values := map[string]any{
		"response": "hello world",
	}

	r := AllPassComposite(e, assertions, values)
	assert.False(t, r.Passed)
	assert.Equal(t, "all_pass", r.Type)
	assert.Contains(t, r.Message, "failed")
}

func TestAnyPassComposite_OneMatches(t *testing.T) {
	e := NewEngine()

	assertions := []Definition{
		{Type: "contains", Target: "response", Value: "xyz"},
		{Type: "contains", Target: "response", Value: "hello"},
	}
	values := map[string]any{
		"response": "hello world",
	}

	r := AnyPassComposite(e, assertions, values)
	assert.True(t, r.Passed)
	assert.Equal(t, "any_pass", r.Type)
}

func TestAnyPassComposite_NoneMatch(t *testing.T) {
	e := NewEngine()

	assertions := []Definition{
		{Type: "contains", Target: "response", Value: "xyz"},
		{Type: "contains", Target: "response", Value: "abc"},
	}
	values := map[string]any{
		"response": "hello world",
	}

	r := AnyPassComposite(e, assertions, values)
	assert.False(t, r.Passed)
	assert.Equal(t, "any_pass", r.Type)
	assert.Contains(t, r.Message, "none of")
}

func TestCompositeAllPass_Evaluator(t *testing.T) {
	e := NewEngine()

	sub := []Definition{
		{Type: "not_empty", Target: "val"},
		{Type: "min_length", Target: "val", Value: 3},
	}

	ev := CompositeAllPass(e, sub)
	ok, msg := ev(Definition{}, "hello world")
	assert.True(t, ok)
	assert.Contains(t, msg, "passed")
}

func TestCompositeAnyPass_Evaluator(t *testing.T) {
	e := NewEngine()

	sub := []Definition{
		{Type: "contains", Target: "val", Value: "xyz"},
		{Type: "not_empty", Target: "val"},
	}

	ev := CompositeAnyPass(e, sub)
	ok, msg := ev(Definition{}, "hello")
	assert.True(t, ok)
	assert.Contains(t, msg, "passed")
}

func TestCompositeAnyPass_Evaluator_AllFail(t *testing.T) {
	e := NewEngine()

	sub := []Definition{
		{Type: "contains", Target: "val", Value: "xyz"},
		{Type: "contains", Target: "val", Value: "abc"},
	}

	ev := CompositeAnyPass(e, sub)
	ok, _ := ev(Definition{}, "hello")
	assert.False(t, ok)
}

func TestParseAssertionString(t *testing.T) {
	tests := []struct {
		input    string
		wantType string
		wantVal  any
	}{
		{"contains:func", "contains", "func"},
		{"not_empty", "not_empty", nil},
		{"min_length:100", "min_length", "100"},
		{"contains:a:b:c", "contains", "a:b:c"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			typ, val := ParseAssertionString(tt.input)
			assert.Equal(t, tt.wantType, typ)
			assert.Equal(t, tt.wantVal, val)
		})
	}
}
