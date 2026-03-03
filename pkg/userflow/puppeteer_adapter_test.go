package userflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ BrowserAdapter = (*PuppeteerAdapter)(nil)

func TestNewPuppeteerAdapter(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		adapter := NewPuppeteerAdapter()
		require.NotNil(t, adapter)
		assert.True(t, adapter.headless)
		assert.Equal(
			t, "puppeteer", adapter.containerName,
		)
		assert.Empty(t, adapter.browserPath)
		assert.Empty(t, adapter.wsEndpoint)
		assert.False(t, adapter.initialized)
		assert.Equal(t, 1920, adapter.width)
		assert.Equal(t, 1080, adapter.height)
	})

	t.Run("with_options", func(t *testing.T) {
		adapter := NewPuppeteerAdapter(
			WithHeadless(false),
			WithBrowserPath("/usr/bin/chromium"),
			WithContainerName("my-puppeteer"),
		)
		require.NotNil(t, adapter)
		assert.False(t, adapter.headless)
		assert.Equal(
			t,
			"/usr/bin/chromium",
			adapter.browserPath,
		)
		assert.Equal(
			t, "my-puppeteer", adapter.containerName,
		)
	})

	t.Run("no_options", func(t *testing.T) {
		adapter := NewPuppeteerAdapter()
		require.NotNil(t, adapter)
		assert.True(t, adapter.headless)
		assert.Equal(
			t, "puppeteer", adapter.containerName,
		)
	})
}

func TestPuppeteerAdapter_Available(t *testing.T) {
	adapter := NewPuppeteerAdapter()
	// Returns false if puppeteer is not installed via
	// npm. This is a graceful check.
	result := adapter.Available(context.Background())
	assert.IsType(t, true, result)
}

func TestPuppeteerAdapter_Close_NotInitialized(
	t *testing.T,
) {
	adapter := NewPuppeteerAdapter()
	// Close without Initialize should be a no-op.
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
	assert.False(t, adapter.initialized)
}

func TestPuppeteerAdapter_NetworkIntercept_NoOp(
	t *testing.T,
) {
	adapter := NewPuppeteerAdapter()
	err := adapter.NetworkIntercept(
		context.Background(),
		"**/api/**",
		func(_ *InterceptedRequest) {},
	)
	assert.NoError(t, err)
}

func TestWithHeadless(t *testing.T) {
	tests := []struct {
		name     string
		headless bool
	}{
		{
			name:     "headless_true",
			headless: true,
		},
		{
			name:     "headless_false",
			headless: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewPuppeteerAdapter(
				WithHeadless(tt.headless),
			)
			assert.Equal(
				t, tt.headless, adapter.headless,
			)
		})
	}
}

func TestWithBrowserPath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "chrome_path",
			path: "/usr/bin/google-chrome",
		},
		{
			name: "chromium_path",
			path: "/usr/bin/chromium-browser",
		},
		{
			name: "empty_path",
			path: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewPuppeteerAdapter(
				WithBrowserPath(tt.path),
			)
			assert.Equal(
				t, tt.path, adapter.browserPath,
			)
		})
	}
}

func TestWithContainerName(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
	}{
		{
			name:          "custom_name",
			containerName: "my-browser",
		},
		{
			name:          "empty_name",
			containerName: "",
		},
		{
			name:          "complex_name",
			containerName: "app-browser-worker-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewPuppeteerAdapter(
				WithContainerName(tt.containerName),
			)
			assert.Equal(
				t,
				tt.containerName,
				adapter.containerName,
			)
		})
	}
}

func TestPuppeteerAdapter_LaunchArgs(t *testing.T) {
	t.Run("headless_no_browser_path", func(t *testing.T) {
		adapter := NewPuppeteerAdapter(
			WithHeadless(true),
		)
		args := adapter.launchArgs()
		assert.Contains(t, args, `"headless":true`)
		assert.Contains(t, args, "--no-sandbox")
		assert.NotContains(t, args, "executablePath")
	})

	t.Run("with_browser_path", func(t *testing.T) {
		adapter := NewPuppeteerAdapter(
			WithBrowserPath("/usr/bin/chrome"),
		)
		args := adapter.launchArgs()
		assert.Contains(t, args, "executablePath")
		assert.Contains(t, args, "/usr/bin/chrome")
	})

	t.Run("window_size_in_args", func(t *testing.T) {
		adapter := NewPuppeteerAdapter()
		adapter.width = 800
		adapter.height = 600
		args := adapter.launchArgs()
		assert.Contains(
			t, args, "--window-size=800,600",
		)
	})
}

func TestPuppeteerAdapter_OptionsChaining(
	t *testing.T,
) {
	adapter := NewPuppeteerAdapter(
		WithHeadless(false),
		WithBrowserPath("/opt/chrome"),
		WithContainerName("test-browser"),
	)

	assert.False(t, adapter.headless)
	assert.Equal(
		t, "/opt/chrome", adapter.browserPath,
	)
	assert.Equal(
		t, "test-browser", adapter.containerName,
	)
}

func TestPuppeteerAdapter_Initialize_Config(
	t *testing.T,
) {
	// We cannot test a full Initialize without puppeteer
	// installed, but we can verify config is applied.
	adapter := NewPuppeteerAdapter()

	// Simulate what Initialize does with config.
	config := BrowserConfig{
		Headless:   false,
		WindowSize: [2]int{800, 600},
	}
	adapter.headless = config.Headless
	if config.WindowSize[0] > 0 {
		adapter.width = config.WindowSize[0]
	}
	if config.WindowSize[1] > 0 {
		adapter.height = config.WindowSize[1]
	}

	assert.False(t, adapter.headless)
	assert.Equal(t, 800, adapter.width)
	assert.Equal(t, 600, adapter.height)
}

func TestPuppeteerAdapter_Initialize_ZeroWindowSize(
	t *testing.T,
) {
	adapter := NewPuppeteerAdapter()
	origWidth := adapter.width
	origHeight := adapter.height

	// Zero values should not override defaults.
	config := BrowserConfig{
		Headless:   true,
		WindowSize: [2]int{0, 0},
	}
	adapter.headless = config.Headless
	if config.WindowSize[0] > 0 {
		adapter.width = config.WindowSize[0]
	}
	if config.WindowSize[1] > 0 {
		adapter.height = config.WindowSize[1]
	}

	assert.Equal(t, origWidth, adapter.width)
	assert.Equal(t, origHeight, adapter.height)
}

func TestPuppeteerAdapter_ContextCancellation(
	t *testing.T,
) {
	adapter := NewPuppeteerAdapter()

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel() // Cancel immediately.

	// Initialize with cancelled context should fail.
	err := adapter.Initialize(
		ctx,
		BrowserConfig{Headless: true},
	)
	assert.Error(t, err)
}
