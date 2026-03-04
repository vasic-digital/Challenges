package userflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface check.
var _ BrowserAdapter = (*PlaywrightCLIAdapter)(nil)

func TestPlaywrightCLIAdapter_Constructor(t *testing.T) {
	adapter := NewPlaywrightCLIAdapter(
		"ws://localhost:9222",
	)
	assert.NotNil(t, adapter)
	// cdpToHTTP converts ws:// to http://.
	assert.Equal(
		t, "http://localhost:9222", adapter.cdpEndpoint,
	)
	assert.Equal(
		t, "playwright", adapter.containerName,
	)
	assert.False(t, adapter.initialized)
}

func TestPlaywrightCLIAdapter_Available_NoEndpoint(
	t *testing.T,
) {
	adapter := NewPlaywrightCLIAdapter(
		"ws://localhost:19999",
	)
	// No CDP server running on this port.
	assert.False(t, adapter.Available(context.Background()))
}

func TestPlaywrightCLIAdapter_Close_NotInitialized(
	t *testing.T,
) {
	adapter := NewPlaywrightCLIAdapter(
		"ws://localhost:9222",
	)
	// Close without Initialize should be a no-op.
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
}

func TestPlaywrightCLIAdapter_NetworkIntercept_NoOp(
	t *testing.T,
) {
	adapter := NewPlaywrightCLIAdapter(
		"ws://localhost:9222",
	)
	err := adapter.NetworkIntercept(
		context.Background(),
		"**/api/**",
		func(_ *InterceptedRequest) {},
	)
	assert.NoError(t, err)
}

func TestPlaywrightCLIAdapter_EscapeJS(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no_escaping",
			in:   "button.submit",
			want: "button.submit",
		},
		{
			name: "single_quote",
			in:   "button[data-id='foo']",
			want: "button[data-id=\\'foo\\']",
		},
		{
			name: "backslash",
			in:   "div\\class",
			want: "div\\\\class",
		},
		{
			name: "mixed",
			in:   "a[href='/test']",
			want: "a[href=\\'/test\\']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeJS(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlaywrightCLIAdapter_ConfigVariants(
	t *testing.T,
) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "ws_endpoint",
			endpoint: "ws://localhost:9222",
			expected: "http://localhost:9222",
		},
		{
			name:     "wss_endpoint",
			endpoint: "wss://localhost:9222",
			expected: "https://localhost:9222",
		},
		{
			name:     "custom_port",
			endpoint: "ws://browser-host:3000/ws",
			expected: "http://browser-host:3000/ws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewPlaywrightCLIAdapter(
				tt.endpoint,
			)
			assert.Equal(
				t, tt.expected, adapter.cdpEndpoint,
			)
		})
	}
}
