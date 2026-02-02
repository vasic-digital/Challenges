package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short key", "abc", "***"},
		{"exact 8", "12345678", "********"},
		{"normal key", "sk-ant-api-key-123456", "sk-a*************3456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, RedactAPIKey(tt.input))
		})
	}
}

func TestRedactURL(t *testing.T) {
	result := RedactURL("https://user:secretpassword123@example.com/path")
	assert.NotContains(t, result, "secretpassword123")
	assert.Contains(t, result, "user")

	// Invalid URL returns as-is
	assert.Equal(t, "not a url :", RedactURL("not a url :"))
}

func TestRedactHeaders(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer sk-ant-test-key-very-long",
		"Content-Type":  "application/json",
		"X-Api-Key":     "secret-api-key-12345678",
	}
	redacted := RedactHeaders(headers)
	assert.Equal(t, "application/json", redacted["Content-Type"])
	assert.NotEqual(t, headers["Authorization"], redacted["Authorization"])
	assert.NotEqual(t, headers["X-Api-Key"], redacted["X-Api-Key"])
}

func TestValidateAPIKeyFormat(t *testing.T) {
	assert.True(t, ValidateAPIKeyFormat("sk-ant-api03-very-long-key-here"))
	assert.True(t, ValidateAPIKeyFormat("sk-proj-1234567890abcdef1234"))
	assert.True(t, ValidateAPIKeyFormat("xai-some-long-api-key-value"))
	assert.True(t, ValidateAPIKeyFormat("some-random-long-enough-key-here"))
	assert.False(t, ValidateAPIKeyFormat(""))
	assert.False(t, ValidateAPIKeyFormat("short"))
}
