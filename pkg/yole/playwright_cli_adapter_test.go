package yole

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlaywrightCLIAdapter(t *testing.T) {
	adapter := NewPlaywrightCLIAdapter()
	assert.Equal(t, "chromium", adapter.browserType)
	assert.NotNil(t, adapter)
}

func TestPlaywrightCLIAdapter_Available_NpxMissing(
	t *testing.T,
) {
	origCmd := playwrightCommandFunc
	defer func() { playwrightCommandFunc = origCmd }()

	playwrightCommandFunc = func(
		ctx context.Context,
		name string,
		args ...string,
	) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}

	adapter := NewPlaywrightCLIAdapter()
	// Available checks exec.LookPath first, then runs npx.
	// This test verifies the function doesn't panic.
	_ = adapter.Available(context.Background())
}

func TestPlaywrightCLIAdapter_Initialize(t *testing.T) {
	adapter := NewPlaywrightCLIAdapter()

	err := adapter.Initialize(
		context.Background(), "firefox",
	)
	require.NoError(t, err)
	defer adapter.Close(context.Background())

	assert.Equal(t, "firefox", adapter.browserType)
	assert.DirExists(t, adapter.scriptDir)
}

func TestPlaywrightCLIAdapter_Initialize_DefaultBrowser(
	t *testing.T,
) {
	adapter := NewPlaywrightCLIAdapter()

	err := adapter.Initialize(context.Background(), "")
	require.NoError(t, err)
	defer adapter.Close(context.Background())

	assert.Equal(t, "chromium", adapter.browserType)
}

func TestPlaywrightCLIAdapter_Navigate_NoURL(t *testing.T) {
	origCmd := playwrightCommandFunc
	defer func() { playwrightCommandFunc = origCmd }()

	playwrightCommandFunc = func(
		ctx context.Context,
		name string,
		args ...string,
	) *exec.Cmd {
		return exec.CommandContext(
			ctx, "echo", "NAVIGATE_OK",
		)
	}

	adapter := NewPlaywrightCLIAdapter()
	require.NoError(t, adapter.Initialize(
		context.Background(), "",
	))
	defer adapter.Close(context.Background())

	err := adapter.Navigate(
		context.Background(), "http://localhost:8080",
	)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", adapter.url)
}

func TestPlaywrightCLIAdapter_Click_NoPage(t *testing.T) {
	adapter := NewPlaywrightCLIAdapter()

	err := adapter.Click(
		context.Background(), "#button",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Navigate first")
}

func TestPlaywrightCLIAdapter_ClickByText_NoPage(
	t *testing.T,
) {
	adapter := NewPlaywrightCLIAdapter()

	err := adapter.ClickByText(
		context.Background(), "Submit",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Navigate first")
}

func TestPlaywrightCLIAdapter_IsVisible_NoPage(
	t *testing.T,
) {
	adapter := NewPlaywrightCLIAdapter()

	visible, err := adapter.IsVisible(
		context.Background(), "#element",
	)
	assert.Error(t, err)
	assert.False(t, visible)
}

func TestPlaywrightCLIAdapter_Screenshot_NoPage(
	t *testing.T,
) {
	adapter := NewPlaywrightCLIAdapter()

	data, err := adapter.Screenshot(context.Background())
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestPlaywrightCLIAdapter_Close(t *testing.T) {
	adapter := NewPlaywrightCLIAdapter()
	require.NoError(t, adapter.Initialize(
		context.Background(), "",
	))

	dir := adapter.scriptDir
	assert.DirExists(t, dir)

	require.NoError(t, adapter.Close(context.Background()))
	assert.NoDirExists(t, dir)
}

func TestPlaywrightCLIAdapter_Close_NoInit(t *testing.T) {
	adapter := NewPlaywrightCLIAdapter()
	assert.NoError(t, adapter.Close(context.Background()))
}

func TestPlaywrightCLIAdapter_RunScript(t *testing.T) {
	origCmd := playwrightCommandFunc
	defer func() { playwrightCommandFunc = origCmd }()

	playwrightCommandFunc = func(
		ctx context.Context,
		name string,
		args ...string,
	) *exec.Cmd {
		// Run cat on the script to verify it was written.
		return exec.CommandContext(ctx, "echo", "SCRIPT_OK")
	}

	adapter := NewPlaywrightCLIAdapter()
	require.NoError(t, adapter.Initialize(
		context.Background(), "",
	))
	defer adapter.Close(context.Background())

	out, err := adapter.runScript(
		context.Background(), "console.log('test');",
	)
	require.NoError(t, err)
	assert.Contains(t, out, "SCRIPT_OK")
}

func TestPlaywrightCLIAdapter_RunScript_WritesFile(
	t *testing.T,
) {
	origCmd := playwrightCommandFunc
	defer func() { playwrightCommandFunc = origCmd }()

	var capturedArgs []string
	playwrightCommandFunc = func(
		ctx context.Context,
		name string,
		args ...string,
	) *exec.Cmd {
		capturedArgs = args
		return exec.CommandContext(ctx, "echo", "ok")
	}

	adapter := NewPlaywrightCLIAdapter()
	require.NoError(t, adapter.Initialize(
		context.Background(), "",
	))
	defer adapter.Close(context.Background())

	_, err := adapter.runScript(
		context.Background(), "console.log('hello');",
	)
	require.NoError(t, err)

	// Verify script file was passed to node.
	require.Len(t, capturedArgs, 1)
	scriptPath := capturedArgs[0]
	assert.Equal(t,
		filepath.Join(adapter.scriptDir, "pw_script.js"),
		scriptPath,
	)

	// Verify file contents.
	data, readErr := os.ReadFile(scriptPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "console.log('hello');")
}

func TestPlaywrightCLIAdapter_RunScript_Failure(
	t *testing.T,
) {
	origCmd := playwrightCommandFunc
	defer func() { playwrightCommandFunc = origCmd }()

	playwrightCommandFunc = func(
		ctx context.Context,
		name string,
		args ...string,
	) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}

	adapter := NewPlaywrightCLIAdapter()
	require.NoError(t, adapter.Initialize(
		context.Background(), "",
	))
	defer adapter.Close(context.Background())

	_, err := adapter.runScript(
		context.Background(), "throw new Error('fail');",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "playwright script failed")
}

func TestPlaywrightCLIAdapter_IsVisible_WithMock(
	t *testing.T,
) {
	origCmd := playwrightCommandFunc
	defer func() { playwrightCommandFunc = origCmd }()

	playwrightCommandFunc = func(
		ctx context.Context,
		name string,
		args ...string,
	) *exec.Cmd {
		return exec.CommandContext(
			ctx, "echo", "VISIBLE_TRUE",
		)
	}

	adapter := NewPlaywrightCLIAdapter()
	require.NoError(t, adapter.Initialize(
		context.Background(), "",
	))
	defer adapter.Close(context.Background())
	adapter.url = "http://localhost:8080"

	visible, err := adapter.IsVisible(
		context.Background(), "#content",
	)
	require.NoError(t, err)
	assert.True(t, visible)
}

func TestPlaywrightCLIAdapter_IsVisible_False(t *testing.T) {
	origCmd := playwrightCommandFunc
	defer func() { playwrightCommandFunc = origCmd }()

	playwrightCommandFunc = func(
		ctx context.Context,
		name string,
		args ...string,
	) *exec.Cmd {
		return exec.CommandContext(
			ctx, "echo", "VISIBLE_FALSE",
		)
	}

	adapter := NewPlaywrightCLIAdapter()
	require.NoError(t, adapter.Initialize(
		context.Background(), "",
	))
	defer adapter.Close(context.Background())
	adapter.url = "http://localhost:8080"

	visible, err := adapter.IsVisible(
		context.Background(), "#missing",
	)
	require.NoError(t, err)
	assert.False(t, visible)
}
