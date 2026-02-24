package userflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface check.
var _ DesktopAdapter = (*TauriCLIAdapter)(nil)

func TestTauriCLIAdapter_Constructor(t *testing.T) {
	adapter := NewTauriCLIAdapter("/usr/bin/myapp")
	assert.NotNil(t, adapter)
	assert.Equal(t, "/usr/bin/myapp", adapter.binaryPath)
	assert.Empty(t, adapter.sessionID)
	assert.Nil(t, adapter.cmd)
}

func TestTauriCLIAdapter_Available_NotExists(
	t *testing.T,
) {
	adapter := NewTauriCLIAdapter(
		"/nonexistent/path/binary",
	)
	assert.False(t, adapter.Available(context.Background()))
}

func TestTauriCLIAdapter_Available_Exists(t *testing.T) {
	// Use a binary that exists on the system.
	adapter := NewTauriCLIAdapter("/bin/sh")
	assert.True(t, adapter.Available(context.Background()))
}

func TestTauriCLIAdapter_IsAppRunning_NotStarted(
	t *testing.T,
) {
	adapter := NewTauriCLIAdapter("/bin/sh")
	running, err := adapter.IsAppRunning(
		context.Background(),
	)
	assert.NoError(t, err)
	assert.False(t, running)
}

func TestTauriCLIAdapter_Close_NotStarted(t *testing.T) {
	adapter := NewTauriCLIAdapter("/bin/sh")
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
}

func TestTauriCLIAdapter_FindFreePort(t *testing.T) {
	port, err := findFreePort()
	assert.NoError(t, err)
	assert.Greater(t, port, 0)
	assert.Less(t, port, 65536)
}

func TestTauriCLIAdapter_ConfigVariants(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
	}{
		{
			name:       "absolute_path",
			binaryPath: "/usr/local/bin/app",
		},
		{
			name:       "relative_path",
			binaryPath: "./target/release/app",
		},
		{
			name:       "home_path",
			binaryPath: "/home/user/app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewTauriCLIAdapter(tt.binaryPath)
			assert.Equal(
				t, tt.binaryPath, adapter.binaryPath,
			)
		})
	}
}

func TestTauriCLIAdapter_LaunchApp_AlreadyRunning(
	t *testing.T,
) {
	adapter := NewTauriCLIAdapter("/bin/sleep")
	ctx := context.Background()

	// Launch first time.
	err := adapter.LaunchApp(ctx, DesktopAppConfig{
		BinaryPath: "/bin/sleep",
		Args:       []string{"60"},
	})
	assert.NoError(t, err)
	defer func() { _ = adapter.Close(ctx) }()

	// Launch again should fail.
	err = adapter.LaunchApp(ctx, DesktopAppConfig{
		BinaryPath: "/bin/sleep",
		Args:       []string{"60"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}
