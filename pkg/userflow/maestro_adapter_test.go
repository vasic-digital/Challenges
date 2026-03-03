package userflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ MobileAdapter = (*MaestroCLIAdapter)(nil)

func TestNewMaestroCLIAdapter(t *testing.T) {
	tests := []struct {
		name   string
		config MobileConfig
	}{
		{
			name: "full_config",
			config: MobileConfig{
				PackageName:  "com.example.app",
				ActivityName: ".MainActivity",
				DeviceSerial: "emulator-5554",
			},
		},
		{
			name: "minimal_config",
			config: MobileConfig{
				PackageName: "com.test",
			},
		},
		{
			name:   "empty_config",
			config: MobileConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewMaestroCLIAdapter(tt.config)
			require.NotNil(t, adapter)
			assert.Equal(
				t, tt.config, adapter.config,
			)
			assert.Empty(t, adapter.tempDir)
		})
	}
}

func TestMaestroCLIAdapter_Available(t *testing.T) {
	adapter := NewMaestroCLIAdapter(MobileConfig{
		PackageName: "com.test.app",
	})
	// Returns false if maestro is not in PATH.
	// This is a graceful check.
	result := adapter.Available(context.Background())
	assert.IsType(t, true, result)
}

func TestMaestroCLIAdapter_Close_NoTempDir(
	t *testing.T,
) {
	adapter := NewMaestroCLIAdapter(MobileConfig{})
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, adapter.tempDir)
}

func TestMaestroCLIAdapter_Close_WithTempDir(
	t *testing.T,
) {
	adapter := NewMaestroCLIAdapter(MobileConfig{})
	// Simulate ensureTempDir having been called.
	adapter.tempDir = "/tmp/maestro-test-cleanup"

	err := adapter.Close(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, adapter.tempDir)
}

func TestMaestroCLIAdapter_RunInstrumentedTests_Unsupported(
	t *testing.T,
) {
	adapter := NewMaestroCLIAdapter(MobileConfig{})
	result, err := adapter.RunInstrumentedTests(
		context.Background(), "com.test.SomeTest",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
	assert.Nil(t, result)
}

func TestMaestroCLIAdapter_WaitForApp_ContextCancel(
	t *testing.T,
) {
	adapter := NewMaestroCLIAdapter(MobileConfig{
		PackageName: "com.test.app",
	})

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel() // Cancel immediately.

	err := adapter.WaitForApp(ctx, 5*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wait for app")
}

func TestMaestroCLIAdapter_EnsureTempDir(
	t *testing.T,
) {
	adapter := NewMaestroCLIAdapter(MobileConfig{})
	assert.Empty(t, adapter.tempDir)

	err := adapter.ensureTempDir()
	require.NoError(t, err)
	assert.NotEmpty(t, adapter.tempDir)

	// Second call should be idempotent.
	originalDir := adapter.tempDir
	err = adapter.ensureTempDir()
	require.NoError(t, err)
	assert.Equal(t, originalDir, adapter.tempDir)

	// Clean up.
	adapter.Close(context.Background())
}

func TestMaestroFlow_ToYAML(t *testing.T) {
	tests := []struct {
		name     string
		flow     MaestroFlow
		contains []string
	}{
		{
			name: "with_app_id_and_commands",
			flow: MaestroFlow{
				AppID: "com.example.app",
				Commands: []string{
					"launchApp: com.example.app",
					"tapOn: Login",
				},
			},
			contains: []string{
				"appId: com.example.app",
				"---",
				"- launchApp: com.example.app",
				"- tapOn: Login",
			},
		},
		{
			name: "no_app_id",
			flow: MaestroFlow{
				Commands: []string{
					"pressKey: back",
				},
			},
			contains: []string{
				"---",
				"- pressKey: back",
			},
		},
		{
			name: "empty_commands",
			flow: MaestroFlow{
				AppID:    "com.test",
				Commands: []string{},
			},
			contains: []string{
				"appId: com.test",
				"---",
			},
		},
		{
			name:     "empty_flow",
			flow:     MaestroFlow{},
			contains: []string{"---"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := tt.flow.toYAML()
			for _, s := range tt.contains {
				assert.Contains(t, yaml, s)
			}
		})
	}
}

func TestMaestroFlow_ToYAML_NoAppID(t *testing.T) {
	flow := MaestroFlow{
		Commands: []string{
			"inputText: hello",
		},
	}
	yaml := flow.toYAML()
	assert.NotContains(t, yaml, "appId:")
	assert.Contains(t, yaml, "---")
	assert.Contains(t, yaml, "- inputText: hello")
}

func TestEscapeYAMLString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no_special_chars",
			in:   "simple",
			want: "simple",
		},
		{
			name: "with_colon",
			in:   "key: value",
			want: `"key: value"`,
		},
		{
			name: "with_hash",
			in:   "text#comment",
			want: `"text#comment"`,
		},
		{
			name: "with_single_quote",
			in:   "it's here",
			want: `"it's here"`,
		},
		{
			name: "with_double_quote",
			in:   `say "hello"`,
			want: `"say \"hello\""`,
		},
		{
			name: "with_backslash",
			in:   `path\to\file`,
			want: `"path\\to\\file"`,
		},
		{
			name: "with_at_sign",
			in:   "user@host",
			want: `"user@host"`,
		},
		{
			name: "empty_string",
			in:   "",
			want: "",
		},
		{
			name: "path_no_escaping",
			in:   "/usr/local/bin/app",
			want: "/usr/local/bin/app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeYAMLString(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMaestroCLIAdapter_ConfigPreserved(
	t *testing.T,
) {
	config := MobileConfig{
		PackageName:  "com.vasic.app",
		ActivityName: ".ui.SplashActivity",
		DeviceSerial: "pixel-6a",
	}
	adapter := NewMaestroCLIAdapter(config)

	assert.Equal(
		t, "com.vasic.app", adapter.config.PackageName,
	)
	assert.Equal(
		t,
		".ui.SplashActivity",
		adapter.config.ActivityName,
	)
	assert.Equal(
		t, "pixel-6a", adapter.config.DeviceSerial,
	)
}
