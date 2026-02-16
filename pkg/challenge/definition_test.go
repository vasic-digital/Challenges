package challenge

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinition_JSONRoundTrip(t *testing.T) {
	def := Definition{
		ID:                "challenge-1",
		Name:              "API Integration Test",
		Description:       "Tests API endpoints",
		Category:          "integration",
		Dependencies:      []ID{"setup-db", "start-server"},
		EstimatedDuration: "30s",
		Inputs: []Input{
			{Name: "api_url", Source: "env", Required: true},
			{Name: "auth_token", Source: "config", Required: false},
		},
		Outputs: []Output{
			{Name: "response_body", Type: "json", Description: "API response"},
			{Name: "status_code", Type: "string", Description: "HTTP status code"},
		},
		Assertions: []AssertionDef{
			{Type: "not_empty", Target: "response_body", Message: "response should not be empty"},
			{Type: "contains", Target: "response_body", Value: "success", Message: "should contain success"},
		},
		Metrics:       []string{"latency_ms", "bytes_received"},
		Configuration: json.RawMessage(`{"retry_count": 3}`),
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var decoded Definition
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, def.ID, decoded.ID)
	assert.Equal(t, def.Name, decoded.Name)
	assert.Equal(t, def.Description, decoded.Description)
	assert.Equal(t, def.Category, decoded.Category)
	assert.Equal(t, def.Dependencies, decoded.Dependencies)
	assert.Equal(t, def.EstimatedDuration, decoded.EstimatedDuration)
	assert.Len(t, decoded.Inputs, 2)
	assert.Len(t, decoded.Outputs, 2)
	assert.Len(t, decoded.Assertions, 2)
	assert.Len(t, decoded.Metrics, 2)
}

func TestInput_Fields(t *testing.T) {
	input := Input{
		Name:     "api_key",
		Source:   "env",
		Required: true,
	}
	assert.Equal(t, "api_key", input.Name)
	assert.Equal(t, "env", input.Source)
	assert.True(t, input.Required)
}

func TestOutput_Fields(t *testing.T) {
	output := Output{
		Name:        "logs",
		Type:        "file",
		Description: "challenge execution logs",
	}
	assert.Equal(t, "logs", output.Name)
	assert.Equal(t, "file", output.Type)
	assert.Equal(t, "challenge execution logs", output.Description)
}

func TestAssertionDef_Fields(t *testing.T) {
	def := AssertionDef{
		Type:    "contains_any",
		Target:  "output",
		Value:   nil,
		Values:  []any{"foo", "bar", "baz"},
		Message: "output should contain one of the values",
	}
	assert.Equal(t, "contains_any", def.Type)
	assert.Equal(t, "output", def.Target)
	assert.Nil(t, def.Value)
	assert.Len(t, def.Values, 3)
}

func TestAssertionDef_JSONOmitEmpty(t *testing.T) {
	def := AssertionDef{
		Type:    "not_empty",
		Target:  "response",
		Message: "response must not be empty",
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasValue := raw["value"]
	assert.False(t, hasValue, "value should be omitted when nil")
}

func TestDefinition_EmptyDependencies(t *testing.T) {
	def := Definition{
		ID:           "standalone",
		Dependencies: nil,
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var decoded Definition
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.Dependencies)
}
