package assertion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAssertionString_TypeAndValue(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType string
		expectedVal  any
	}{
		{
			name:         "contains with value",
			input:        "contains:func",
			expectedType: "contains",
			expectedVal:  "func",
		},
		{
			name:         "min_length with numeric value",
			input:        "min_length:100",
			expectedType: "min_length",
			expectedVal:  "100",
		},
		{
			name:         "not_empty without value",
			input:        "not_empty",
			expectedType: "not_empty",
			expectedVal:  nil,
		},
		{
			name:         "quality_score with decimal",
			input:        "quality_score:0.8",
			expectedType: "quality_score",
			expectedVal:  "0.8",
		},
		{
			name:         "contains_any with CSV",
			input:        "contains_any:foo,bar,baz",
			expectedType: "contains_any",
			expectedVal:  "foo,bar,baz",
		},
		{
			name:         "value with colons",
			input:        "contains:http://example.com",
			expectedType: "contains",
			expectedVal:  "http://example.com",
		},
		{
			name:         "empty string",
			input:        "",
			expectedType: "",
			expectedVal:  nil,
		},
		{
			name:         "type only with trailing colon",
			input:        "not_empty:",
			expectedType: "not_empty",
			expectedVal:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aType, aValue := ParseAssertionString(tt.input)
			assert.Equal(t, tt.expectedType, aType)
			assert.Equal(t, tt.expectedVal, aValue)
		})
	}
}
