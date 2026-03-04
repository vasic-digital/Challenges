package userflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPCCommand_ZeroValue(t *testing.T) {
	var cmd IPCCommand
	assert.Equal(t, "", cmd.Name)
	assert.Equal(t, "", cmd.Command)
	assert.Nil(t, cmd.Args)
	assert.Equal(t, "", cmd.ExpectedResult)
	assert.Nil(t, cmd.Assertions)
}

func TestIPCCommand_FieldAssignment(t *testing.T) {
	cmd := IPCCommand{
		Name:           "greet_user",
		Command:        "greet",
		Args:           []string{"Alice"},
		ExpectedResult: "Hello Alice",
		Assertions: []StepAssertion{
			{
				Type:    "contains",
				Target:  "body",
				Value:   "Hello",
				Message: "should contain greeting",
			},
		},
	}

	assert.Equal(t, "greet_user", cmd.Name)
	assert.Equal(t, "greet", cmd.Command)
	assert.Equal(t, []string{"Alice"}, cmd.Args)
	assert.Equal(t, "Hello Alice", cmd.ExpectedResult)
	require.Len(t, cmd.Assertions, 1)
	assert.Equal(t, "contains", cmd.Assertions[0].Type)
	assert.Equal(t, "body", cmd.Assertions[0].Target)
	assert.Equal(t, "Hello", cmd.Assertions[0].Value)
	assert.Equal(
		t,
		"should contain greeting",
		cmd.Assertions[0].Message,
	)
}

func TestIPCCommand_JSONMarshal_Full(t *testing.T) {
	cmd := IPCCommand{
		Name:           "get_version",
		Command:        "app_version",
		Args:           []string{"--verbose"},
		ExpectedResult: "1.0.0",
		Assertions: []StepAssertion{
			{
				Type:    "not_empty",
				Target:  "result",
				Message: "version must not be empty",
			},
		},
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "get_version", decoded["name"])
	assert.Equal(t, "app_version", decoded["command"])
	assert.Equal(t, "1.0.0", decoded["expected_result"])

	args, ok := decoded["args"].([]interface{})
	require.True(t, ok)
	assert.Len(t, args, 1)
	assert.Equal(t, "--verbose", args[0])

	assertions, ok := decoded["assertions"].([]interface{})
	require.True(t, ok)
	assert.Len(t, assertions, 1)
}

func TestIPCCommand_JSONMarshal_OmitsEmptyFields(
	t *testing.T,
) {
	cmd := IPCCommand{
		Name:    "ping",
		Command: "health_check",
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// name and command always present.
	assert.Equal(t, "ping", decoded["name"])
	assert.Equal(t, "health_check", decoded["command"])

	// omitempty fields should be absent.
	_, hasArgs := decoded["args"]
	assert.False(t, hasArgs, "args should be omitted")

	_, hasExpected := decoded["expected_result"]
	assert.False(
		t, hasExpected,
		"expected_result should be omitted",
	)

	_, hasAssertions := decoded["assertions"]
	assert.False(
		t, hasAssertions,
		"assertions should be omitted",
	)
}

func TestIPCCommand_JSONUnmarshal_Full(t *testing.T) {
	raw := `{
		"name": "save_file",
		"command": "file_save",
		"args": ["doc.txt", "/tmp"],
		"expected_result": "saved",
		"assertions": [
			{
				"type": "contains",
				"target": "response",
				"value": "saved",
				"message": "should confirm save"
			}
		]
	}`

	var cmd IPCCommand
	err := json.Unmarshal([]byte(raw), &cmd)
	require.NoError(t, err)

	assert.Equal(t, "save_file", cmd.Name)
	assert.Equal(t, "file_save", cmd.Command)
	assert.Equal(
		t, []string{"doc.txt", "/tmp"}, cmd.Args,
	)
	assert.Equal(t, "saved", cmd.ExpectedResult)
	require.Len(t, cmd.Assertions, 1)
	assert.Equal(
		t, "contains", cmd.Assertions[0].Type,
	)
}

func TestIPCCommand_JSONUnmarshal_MinimalFields(
	t *testing.T,
) {
	raw := `{"name":"x","command":"y"}`

	var cmd IPCCommand
	err := json.Unmarshal([]byte(raw), &cmd)
	require.NoError(t, err)

	assert.Equal(t, "x", cmd.Name)
	assert.Equal(t, "y", cmd.Command)
	assert.Nil(t, cmd.Args)
	assert.Equal(t, "", cmd.ExpectedResult)
	assert.Nil(t, cmd.Assertions)
}

func TestIPCCommand_JSONRoundTrip(t *testing.T) {
	original := IPCCommand{
		Name:           "round_trip",
		Command:        "echo",
		Args:           []string{"hello", "world"},
		ExpectedResult: "hello world",
		Assertions: []StepAssertion{
			{
				Type:    "not_empty",
				Target:  "stdout",
				Message: "must have output",
			},
			{
				Type:    "contains",
				Target:  "stdout",
				Value:   "hello",
				Message: "must contain hello",
			},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored IPCCommand
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, original.Name, restored.Name)
	assert.Equal(t, original.Command, restored.Command)
	assert.Equal(t, original.Args, restored.Args)
	assert.Equal(
		t, original.ExpectedResult,
		restored.ExpectedResult,
	)
	require.Len(t, restored.Assertions, 2)
	assert.Equal(
		t,
		original.Assertions[0].Type,
		restored.Assertions[0].Type,
	)
	assert.Equal(
		t,
		original.Assertions[1].Value,
		restored.Assertions[1].Value,
	)
}

func TestIPCCommand_MultipleArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no_args",
			args: nil,
		},
		{
			name: "single_arg",
			args: []string{"one"},
		},
		{
			name: "multiple_args",
			args: []string{"a", "b", "c", "d"},
		},
		{
			name: "empty_string_arg",
			args: []string{""},
		},
		{
			name: "args_with_spaces",
			args: []string{"hello world", "foo bar"},
		},
		{
			name: "args_with_special_chars",
			args: []string{"--flag=val", "-v", "a=b&c=d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := IPCCommand{
				Name:    "test",
				Command: "cmd",
				Args:    tt.args,
			}

			data, err := json.Marshal(cmd)
			require.NoError(t, err)

			var restored IPCCommand
			err = json.Unmarshal(data, &restored)
			require.NoError(t, err)

			if tt.args == nil {
				assert.Nil(t, restored.Args)
			} else {
				assert.Equal(
					t, tt.args, restored.Args,
				)
			}
		})
	}
}

func TestIPCCommand_MultipleAssertions(t *testing.T) {
	cmd := IPCCommand{
		Name:    "multi_assert",
		Command: "compute",
		Assertions: []StepAssertion{
			{
				Type:    "not_empty",
				Target:  "result",
				Message: "result required",
			},
			{
				Type:    "contains",
				Target:  "result",
				Value:   "42",
				Message: "must contain answer",
			},
			{
				Type:    "min_length",
				Target:  "result",
				Value:   10,
				Message: "must be at least 10 chars",
			},
		},
	}

	require.Len(t, cmd.Assertions, 3)
	assert.Equal(t, "not_empty", cmd.Assertions[0].Type)
	assert.Equal(t, "contains", cmd.Assertions[1].Type)
	assert.Equal(
		t, "min_length", cmd.Assertions[2].Type,
	)
	// Value can be any type.
	assert.Equal(t, 10, cmd.Assertions[2].Value)
}

func TestIPCCommand_EmptyStrings(t *testing.T) {
	cmd := IPCCommand{
		Name:           "",
		Command:        "",
		Args:           []string{},
		ExpectedResult: "",
		Assertions:     []StepAssertion{},
	}

	assert.Equal(t, "", cmd.Name)
	assert.Equal(t, "", cmd.Command)
	assert.Empty(t, cmd.Args)
	assert.Equal(t, "", cmd.ExpectedResult)
	assert.Empty(t, cmd.Assertions)
}

func TestIPCCommand_JSONTags(t *testing.T) {
	cmd := IPCCommand{
		Name:           "tag_test",
		Command:        "verify_tags",
		Args:           []string{"a"},
		ExpectedResult: "ok",
		Assertions: []StepAssertion{
			{Type: "not_empty", Target: "x"},
		},
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	raw := string(data)
	// Verify JSON keys match the json tags.
	assert.Contains(t, raw, `"name"`)
	assert.Contains(t, raw, `"command"`)
	assert.Contains(t, raw, `"args"`)
	assert.Contains(t, raw, `"expected_result"`)
	assert.Contains(t, raw, `"assertions"`)
}

func TestIPCCommand_AssertionsWithVariousValueTypes(
	t *testing.T,
) {
	// StepAssertion.Value is `any`, so verify
	// JSON round-trip with various types.
	cmd := IPCCommand{
		Name:    "typed_values",
		Command: "check",
		Assertions: []StepAssertion{
			{
				Type:   "exact",
				Target: "count",
				Value:  float64(42),
			},
			{
				Type:   "contains",
				Target: "body",
				Value:  "hello",
			},
			{
				Type:   "flag",
				Target: "active",
				Value:  true,
			},
		},
	}

	data, err := json.Marshal(cmd)
	require.NoError(t, err)

	var restored IPCCommand
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	require.Len(t, restored.Assertions, 3)
	// JSON numbers unmarshal as float64.
	assert.Equal(
		t, float64(42),
		restored.Assertions[0].Value,
	)
	assert.Equal(
		t, "hello", restored.Assertions[1].Value,
	)
	assert.Equal(
		t, true, restored.Assertions[2].Value,
	)
}
