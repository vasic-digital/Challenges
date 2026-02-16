package assertion

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinition_JSONRoundTrip(t *testing.T) {
	def := Definition{
		Type:    "contains",
		Target:  "output",
		Value:   "expected_value",
		Values:  []any{"val1", "val2"},
		Message: "should contain expected value",
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var decoded Definition
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, def.Type, decoded.Type)
	assert.Equal(t, def.Target, decoded.Target)
	assert.Equal(t, def.Message, decoded.Message)
}

func TestDefinition_JSONOmitEmpty(t *testing.T) {
	def := Definition{
		Type:    "not_empty",
		Target:  "response",
		Message: "response should not be empty",
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasValue := raw["value"]
	assert.False(t, hasValue, "value should be omitted when empty")
}

func TestResult_Fields(t *testing.T) {
	result := Result{
		Type:     "contains",
		Target:   "output",
		Expected: "foo",
		Actual:   "foobar",
		Passed:   true,
		Message:  "output contains foo",
	}

	assert.Equal(t, "contains", result.Type)
	assert.Equal(t, "output", result.Target)
	assert.Equal(t, "foo", result.Expected)
	assert.Equal(t, "foobar", result.Actual)
	assert.True(t, result.Passed)
	assert.Equal(t, "output contains foo", result.Message)
}

func TestResult_JSONRoundTrip(t *testing.T) {
	result := Result{
		Type:     "min_length",
		Target:   "response",
		Expected: 100,
		Actual:   "a long response text",
		Passed:   false,
		Message:  "response too short",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded Result
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.Type, decoded.Type)
	assert.Equal(t, result.Target, decoded.Target)
	assert.Equal(t, result.Passed, decoded.Passed)
	assert.Equal(t, result.Message, decoded.Message)
}
