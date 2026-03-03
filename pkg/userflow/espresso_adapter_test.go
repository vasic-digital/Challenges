package userflow

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ MobileAdapter = (*EspressoAdapter)(nil)

func TestNewEspressoAdapter(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		config := MobileConfig{
			PackageName:  "com.example.app",
			ActivityName: ".MainActivity",
			DeviceSerial: "emulator-5554",
		}
		adapter := NewEspressoAdapter(
			"/tmp/project", config,
		)
		require.NotNil(t, adapter)
		assert.Equal(
			t, "/tmp/project", adapter.projectDir,
		)
		assert.Equal(
			t, config, adapter.config,
		)
		assert.Empty(t, adapter.gradleWrapper)
		assert.Empty(t, adapter.module)
		assert.Empty(t, adapter.testRunner)
		assert.Nil(t, adapter.instrumentArgs)
	})

	t.Run("with_all_options", func(t *testing.T) {
		config := MobileConfig{
			PackageName:  "com.test.app",
			ActivityName: ".TestActivity",
		}
		instrumentArgs := map[string]string{
			"size":          "medium",
			"annotation":    "com.test.Smoke",
			"clearPackage":  "true",
		}
		adapter := NewEspressoAdapter(
			"/home/user/android",
			config,
			WithEspressoGradleWrapper(
				"/opt/gradle/gradle",
			),
			WithEspressoModule(":app"),
			WithEspressoTestRunner(
				"com.custom.TestRunner",
			),
			WithEspressoInstrumentationArgs(
				instrumentArgs,
			),
		)
		require.NotNil(t, adapter)
		assert.Equal(
			t,
			"/home/user/android",
			adapter.projectDir,
		)
		assert.Equal(
			t,
			"/opt/gradle/gradle",
			adapter.gradleWrapper,
		)
		assert.Equal(t, ":app", adapter.module)
		assert.Equal(
			t,
			"com.custom.TestRunner",
			adapter.testRunner,
		)
		assert.Equal(
			t, instrumentArgs, adapter.instrumentArgs,
		)
	})
}

func TestEspressoAdapter_Available_NoADB(
	t *testing.T,
) {
	adapter := NewEspressoAdapter(
		"/nonexistent/project",
		MobileConfig{},
	)
	// adb may or may not be in PATH; gradle wrapper
	// definitely does not exist at this path.
	available := adapter.Available(
		context.Background(),
	)
	assert.False(t, available)
}

func TestEspressoAdapter_Close(t *testing.T) {
	adapter := NewEspressoAdapter(
		"/tmp", MobileConfig{},
	)
	// Close is a no-op for Espresso.
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
}

func TestEspressoAdapter_Close_MultipleCalls(
	t *testing.T,
) {
	adapter := NewEspressoAdapter(
		"/tmp", MobileConfig{},
	)
	// Multiple close calls should all succeed.
	for i := 0; i < 5; i++ {
		err := adapter.Close(context.Background())
		assert.NoError(t, err)
	}
}

func TestWithEspressoModule(t *testing.T) {
	tests := []struct {
		name   string
		module string
	}{
		{
			name:   "app_module",
			module: ":app",
		},
		{
			name:   "nested_module",
			module: ":features:login",
		},
		{
			name:   "empty_module",
			module: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewEspressoAdapter(
				"/tmp", MobileConfig{},
				WithEspressoModule(tt.module),
			)
			assert.Equal(
				t, tt.module, adapter.module,
			)
		})
	}
}

func TestWithEspressoTestRunner(t *testing.T) {
	tests := []struct {
		name   string
		runner string
	}{
		{
			name: "default_junit_runner",
			runner: "androidx.test.runner" +
				".AndroidJUnitRunner",
		},
		{
			name: "custom_runner",
			runner: "com.company.test" +
				".CustomRunner",
		},
		{
			name:   "empty_runner",
			runner: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewEspressoAdapter(
				"/tmp", MobileConfig{},
				WithEspressoTestRunner(tt.runner),
			)
			assert.Equal(
				t, tt.runner, adapter.testRunner,
			)
		})
	}
}

func TestWithEspressoInstrumentationArgs(
	t *testing.T,
) {
	tests := []struct {
		name string
		args map[string]string
	}{
		{
			name: "size_filter",
			args: map[string]string{
				"size": "small",
			},
		},
		{
			name: "multiple_args",
			args: map[string]string{
				"size":       "medium",
				"annotation": "com.test.Smoke",
				"debug":      "true",
			},
		},
		{
			name: "empty_args",
			args: map[string]string{},
		},
		{
			name: "nil_args",
			args: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewEspressoAdapter(
				"/tmp", MobileConfig{},
				WithEspressoInstrumentationArgs(
					tt.args,
				),
			)
			assert.Equal(
				t, tt.args, adapter.instrumentArgs,
			)
		})
	}
}

func TestWithEspressoGradleWrapper(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "absolute_path",
			path: "/opt/gradle/7.0/bin/gradle",
		},
		{
			name: "relative_path",
			path: "./gradlew",
		},
		{
			name: "empty_path",
			path: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewEspressoAdapter(
				"/tmp", MobileConfig{},
				WithEspressoGradleWrapper(tt.path),
			)
			assert.Equal(
				t, tt.path, adapter.gradleWrapper,
			)
		})
	}
}

func TestEspressoAdapter_GradlePath(t *testing.T) {
	t.Run("default_path", func(t *testing.T) {
		adapter := NewEspressoAdapter(
			"/home/user/project", MobileConfig{},
		)
		want := filepath.Join(
			"/home/user/project", "gradlew",
		)
		assert.Equal(t, want, adapter.gradlePath())
	})

	t.Run("custom_wrapper", func(t *testing.T) {
		adapter := NewEspressoAdapter(
			"/home/user/project",
			MobileConfig{},
			WithEspressoGradleWrapper(
				"/opt/gradle/gradle",
			),
		)
		assert.Equal(
			t,
			"/opt/gradle/gradle",
			adapter.gradlePath(),
		)
	})
}

func TestEspressoAdapter_TaskName(t *testing.T) {
	tests := []struct {
		name     string
		module   string
		task     string
		wantTask string
	}{
		{
			name:     "no_module",
			module:   "",
			task:     "installDebug",
			wantTask: "installDebug",
		},
		{
			name:   "with_module",
			module: ":app",
			task:   "connectedDebugAndroidTest",
			wantTask: ":app:" +
				"connectedDebugAndroidTest",
		},
		{
			name:     "nested_module",
			module:   ":core:ui",
			task:     "installDebug",
			wantTask: ":core:ui:installDebug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewEspressoAdapter(
				"/tmp", MobileConfig{},
				WithEspressoModule(tt.module),
			)
			got := adapter.taskName(tt.task)
			assert.Equal(t, tt.wantTask, got)
		})
	}
}

func TestEspressoAdapter_ResolvedRunner(t *testing.T) {
	t.Run("default_runner", func(t *testing.T) {
		adapter := NewEspressoAdapter(
			"/tmp", MobileConfig{},
		)
		want := "androidx.test.runner" +
			".AndroidJUnitRunner"
		assert.Equal(
			t, want, adapter.resolvedRunner(),
		)
	})

	t.Run("custom_runner", func(t *testing.T) {
		adapter := NewEspressoAdapter(
			"/tmp", MobileConfig{},
			WithEspressoTestRunner(
				"com.custom.Runner",
			),
		)
		assert.Equal(
			t,
			"com.custom.Runner",
			adapter.resolvedRunner(),
		)
	})
}

func TestEspressoAdapter_DeviceArgs(t *testing.T) {
	tests := []struct {
		name   string
		serial string
		input  []string
		want   []string
	}{
		{
			name:   "with_serial",
			serial: "device-123",
			input:  []string{"shell", "ls"},
			want: []string{
				"-s", "device-123", "shell", "ls",
			},
		},
		{
			name:   "no_serial",
			serial: "",
			input:  []string{"shell", "ls"},
			want:   []string{"shell", "ls"},
		},
		{
			name:   "with_serial_empty_args",
			serial: "emulator-5554",
			input:  []string{},
			want: []string{
				"-s", "emulator-5554",
			},
		},
		{
			name:   "no_serial_single_arg",
			serial: "",
			input:  []string{"devices"},
			want:   []string{"devices"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewEspressoAdapter(
				"/tmp",
				MobileConfig{
					DeviceSerial: tt.serial,
				},
			)
			got := adapter.deviceArgs(tt.input...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEspressoAdapter_OptionsChaining(
	t *testing.T,
) {
	config := MobileConfig{
		PackageName:  "com.vasic.app",
		ActivityName: ".SplashActivity",
		DeviceSerial: "pixel-6",
	}
	instrumentArgs := map[string]string{
		"size": "large",
	}

	adapter := NewEspressoAdapter(
		"/home/dev/project",
		config,
		WithEspressoGradleWrapper("/usr/bin/gradle"),
		WithEspressoModule(":app"),
		WithEspressoTestRunner("com.custom.Runner"),
		WithEspressoInstrumentationArgs(
			instrumentArgs,
		),
	)

	assert.Equal(
		t,
		"/home/dev/project",
		adapter.projectDir,
	)
	assert.Equal(t, config, adapter.config)
	assert.Equal(
		t, "/usr/bin/gradle", adapter.gradleWrapper,
	)
	assert.Equal(t, ":app", adapter.module)
	assert.Equal(
		t, "com.custom.Runner", adapter.testRunner,
	)
	assert.Equal(
		t, instrumentArgs, adapter.instrumentArgs,
	)
}

func TestEspressoAdapter_ConfigPreserved(
	t *testing.T,
) {
	configs := []MobileConfig{
		{
			PackageName:  "com.example.app",
			ActivityName: ".Main",
			DeviceSerial: "emulator-5554",
		},
		{
			PackageName:  "com.minimal",
			ActivityName: ".Activity",
		},
		{},
	}

	for _, cfg := range configs {
		adapter := NewEspressoAdapter("/tmp", cfg)
		assert.Equal(
			t,
			cfg.PackageName,
			adapter.config.PackageName,
		)
		assert.Equal(
			t,
			cfg.ActivityName,
			adapter.config.ActivityName,
		)
		assert.Equal(
			t,
			cfg.DeviceSerial,
			adapter.config.DeviceSerial,
		)
	}
}
