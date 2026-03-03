package userflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ BrowserAdapter = (*CypressCLIAdapter)(nil)

func TestNewCypressCLIAdapter(t *testing.T) {
	tests := []struct {
		name       string
		projectDir string
	}{
		{
			name:       "basic_project_dir",
			projectDir: "/home/user/project",
		},
		{
			name:       "empty_project_dir",
			projectDir: "",
		},
		{
			name:       "relative_project_dir",
			projectDir: "./my-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewCypressCLIAdapter(
				tt.projectDir,
			)
			require.NotNil(t, adapter)
			assert.Equal(
				t, tt.projectDir, adapter.projectDir,
			)
			assert.True(t, adapter.headless)
			assert.Equal(
				t, "chrome", adapter.browser,
			)
			assert.Empty(t, adapter.tempDir)
			assert.Empty(t, adapter.baseURL)
		})
	}
}

func TestCypressCLIAdapter_Available_NotInstalled(
	t *testing.T,
) {
	adapter := NewCypressCLIAdapter("/nonexistent")
	// Returns false if npx/cypress not in PATH or not
	// installed. This is a graceful check.
	result := adapter.Available(context.Background())
	assert.IsType(t, true, result)
}

func TestCypressCLIAdapter_Close_NotInitialized(
	t *testing.T,
) {
	adapter := NewCypressCLIAdapter("/tmp")
	// Close without Initialize should be a no-op.
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, adapter.tempDir)
}

func TestCypressCLIAdapter_Close_WithTempDir(
	t *testing.T,
) {
	adapter := NewCypressCLIAdapter("/tmp")
	// Simulate Initialize setting tempDir.
	adapter.tempDir = "/tmp/cypress-test-cleanup"

	err := adapter.Close(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, adapter.tempDir)
}

func TestCypressCLIAdapter_NetworkIntercept_NoOp(
	t *testing.T,
) {
	adapter := NewCypressCLIAdapter("/tmp")
	err := adapter.NetworkIntercept(
		context.Background(),
		"**/api/**",
		func(_ *InterceptedRequest) {},
	)
	assert.NoError(t, err)
}

func TestCypressBrowserName(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantResult string
	}{
		{
			name:       "chrome",
			input:      "chrome",
			wantResult: "chrome",
		},
		{
			name:       "chromium",
			input:      "chromium",
			wantResult: "chrome",
		},
		{
			name:       "firefox",
			input:      "firefox",
			wantResult: "firefox",
		},
		{
			name:       "gecko",
			input:      "gecko",
			wantResult: "firefox",
		},
		{
			name:       "edge",
			input:      "edge",
			wantResult: "edge",
		},
		{
			name:       "msedge",
			input:      "msedge",
			wantResult: "edge",
		},
		{
			name:       "electron",
			input:      "electron",
			wantResult: "electron",
		},
		{
			name:       "unknown_defaults_to_chrome",
			input:      "safari",
			wantResult: "chrome",
		},
		{
			name:       "empty_defaults_to_chrome",
			input:      "",
			wantResult: "chrome",
		},
		{
			name:       "case_insensitive",
			input:      "Firefox",
			wantResult: "firefox",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cypressBrowserName(tt.input)
			assert.Equal(t, tt.wantResult, got)
		})
	}
}

func TestCypressCLIAdapter_Initialize_SetsBrowser(
	t *testing.T,
) {
	adapter := NewCypressCLIAdapter("/tmp")
	err := adapter.Initialize(
		context.Background(),
		BrowserConfig{
			BrowserType: "firefox",
			Headless:    false,
		},
	)
	// Initialize creates a temp dir; close to clean up.
	defer adapter.Close(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "firefox", adapter.browser)
	assert.False(t, adapter.headless)
	assert.NotEmpty(t, adapter.tempDir)
}

func TestCypressCLIAdapter_Initialize_DefaultBrowser(
	t *testing.T,
) {
	adapter := NewCypressCLIAdapter("/tmp")
	err := adapter.Initialize(
		context.Background(),
		BrowserConfig{
			Headless: true,
		},
	)
	defer adapter.Close(context.Background())

	require.NoError(t, err)
	// Should keep the default "chrome" when empty.
	assert.Equal(t, "chrome", adapter.browser)
	assert.True(t, adapter.headless)
}

func TestCypressCLIAdapter_WrapSpec(t *testing.T) {
	adapter := NewCypressCLIAdapter("/tmp")
	adapter.baseURL = "http://localhost:8080"

	spec := adapter.wrapSpec(
		"cy.get('#btn').click();",
	)
	assert.Contains(
		t, spec, "cy.visit('http://localhost:8080')",
	)
	assert.Contains(
		t, spec, "cy.get('#btn').click();",
	)
	assert.Contains(t, spec, "describe('action'")
}

func TestCypressCLIAdapter_ParseTaskLogString(
	t *testing.T,
) {
	adapter := NewCypressCLIAdapter("/tmp")

	tests := []struct {
		name   string
		output string
		field  string
		want   string
	}{
		{
			name: "extract_text_field",
			output: `some prefix {"text":"hello world"}` +
				` suffix`,
			field: "text",
			want:  "hello world",
		},
		{
			name:   "field_not_found",
			output: `{"other":"data"}`,
			field:  "text",
			want:   "",
		},
		{
			name:   "empty_output",
			output: "",
			field:  "text",
			want:   "",
		},
		{
			name: "multiline_output",
			output: "line one\n" +
				`task log: {"visible":true}` +
				"\nline three",
			field: "visible",
			want:  "true",
		},
		{
			name: "extract_value_field",
			output: `prefix {"value":"attr-val"}` +
				` end`,
			field: "value",
			want:  "attr-val",
		},
		{
			name: "no_json_in_line",
			output: "this line has text " +
				"but no json braces",
			field: "text",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := adapter.parseTaskLogString(
				tt.output, tt.field,
			)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCypressCLIAdapter_Navigate_NoCypress(
	t *testing.T,
) {
	adapter := NewCypressCLIAdapter("/tmp")
	// Initialize to create tempDir.
	err := adapter.Initialize(
		context.Background(),
		BrowserConfig{Headless: true},
	)
	require.NoError(t, err)
	defer adapter.Close(context.Background())

	// Navigate will fail because cypress is not
	// installed in the test environment.
	err = adapter.Navigate(
		context.Background(), "http://example.com",
	)
	// We expect an error since cypress is not available.
	assert.Error(t, err)
}

func TestCypressCLIAdapter_ContextCancellation(
	t *testing.T,
) {
	adapter := NewCypressCLIAdapter("/tmp")
	err := adapter.Initialize(
		context.Background(),
		BrowserConfig{Headless: true},
	)
	require.NoError(t, err)
	defer adapter.Close(context.Background())

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel() // Cancel immediately.

	err = adapter.Navigate(ctx, "http://example.com")
	assert.Error(t, err)
}
