package userflow

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ BuildAdapter = (*RobolectricAdapter)(nil)

func TestNewRobolectricAdapter(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		adapter := NewRobolectricAdapter("/tmp/project")
		require.NotNil(t, adapter)
		assert.Equal(
			t, "/tmp/project", adapter.projectDir,
		)
		assert.Empty(t, adapter.gradleWrapper)
		assert.Empty(t, adapter.module)
		assert.Empty(t, adapter.testFilter)
		assert.Nil(t, adapter.jvmArgs)
	})

	t.Run("with_all_options", func(t *testing.T) {
		adapter := NewRobolectricAdapter(
			"/home/user/android-project",
			WithRobolectricGradleWrapper(
				"/opt/gradle/bin/gradle",
			),
			WithRobolectricModule(":app"),
			WithRobolectricTestFilter(
				"com.test.MyTest",
			),
			WithRobolectricJVMArgs(
				[]string{"-Xmx2g", "-Xms512m"},
			),
		)
		require.NotNil(t, adapter)
		assert.Equal(
			t,
			"/home/user/android-project",
			adapter.projectDir,
		)
		assert.Equal(
			t,
			"/opt/gradle/bin/gradle",
			adapter.gradleWrapper,
		)
		assert.Equal(t, ":app", adapter.module)
		assert.Equal(
			t, "com.test.MyTest", adapter.testFilter,
		)
		assert.Equal(
			t,
			[]string{"-Xmx2g", "-Xms512m"},
			adapter.jvmArgs,
		)
	})
}

func TestRobolectricAdapter_Available_NoGradle(
	t *testing.T,
) {
	adapter := NewRobolectricAdapter(
		"/nonexistent/project",
	)
	// Gradle wrapper does not exist at this path.
	available := adapter.Available(
		context.Background(),
	)
	assert.False(t, available)
}

func TestWithRobolectricModule(t *testing.T) {
	tests := []struct {
		name   string
		module string
	}{
		{
			name:   "app_module",
			module: ":app",
		},
		{
			name:   "library_module",
			module: ":core:data",
		},
		{
			name:   "empty_module",
			module: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewRobolectricAdapter(
				"/tmp",
				WithRobolectricModule(tt.module),
			)
			assert.Equal(
				t, tt.module, adapter.module,
			)
		})
	}
}

func TestWithRobolectricTestFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter string
	}{
		{
			name:   "class_filter",
			filter: "com.example.MyTest",
		},
		{
			name:   "method_filter",
			filter: "com.example.MyTest.testMethod",
		},
		{
			name:   "wildcard_filter",
			filter: "com.example.*",
		},
		{
			name:   "empty_filter",
			filter: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewRobolectricAdapter(
				"/tmp",
				WithRobolectricTestFilter(tt.filter),
			)
			assert.Equal(
				t, tt.filter, adapter.testFilter,
			)
		})
	}
}

func TestWithRobolectricJVMArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "memory_args",
			args: []string{"-Xmx4g", "-Xms1g"},
		},
		{
			name: "single_arg",
			args: []string{"-Xmx2g"},
		},
		{
			name: "empty_args",
			args: []string{},
		},
		{
			name: "nil_args",
			args: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewRobolectricAdapter(
				"/tmp",
				WithRobolectricJVMArgs(tt.args),
			)
			assert.Equal(
				t, tt.args, adapter.jvmArgs,
			)
		})
	}
}

func TestRobolectricAdapter_GradlePath(t *testing.T) {
	t.Run("default_path", func(t *testing.T) {
		adapter := NewRobolectricAdapter(
			"/home/user/project",
		)
		want := filepath.Join(
			"/home/user/project", "gradlew",
		)
		assert.Equal(t, want, adapter.gradlePath())
	})

	t.Run("custom_wrapper", func(t *testing.T) {
		adapter := NewRobolectricAdapter(
			"/home/user/project",
			WithRobolectricGradleWrapper(
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

func TestRobolectricAdapter_TaskName(t *testing.T) {
	tests := []struct {
		name     string
		module   string
		task     string
		wantTask string
	}{
		{
			name:     "no_module",
			module:   "",
			task:     "assembleDebug",
			wantTask: "assembleDebug",
		},
		{
			name:     "with_module",
			module:   ":app",
			task:     "assembleDebug",
			wantTask: ":app:assembleDebug",
		},
		{
			name:     "nested_module",
			module:   ":core:data",
			task:     "testDebugUnitTest",
			wantTask: ":core:data:testDebugUnitTest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewRobolectricAdapter(
				"/tmp",
				WithRobolectricModule(tt.module),
			)
			got := adapter.taskName(tt.task)
			assert.Equal(t, tt.wantTask, got)
		})
	}
}

func TestRobolectricAdapter_JVMArgFlags(t *testing.T) {
	tests := []struct {
		name     string
		jvmArgs  []string
		wantNil  bool
		wantFlag string
	}{
		{
			name:    "no_jvm_args",
			jvmArgs: nil,
			wantNil: true,
		},
		{
			name:    "empty_jvm_args",
			jvmArgs: []string{},
			wantNil: true,
		},
		{
			name:    "single_arg",
			jvmArgs: []string{"-Xmx2g"},
			wantFlag: "-Dorg.gradle.jvmargs=" +
				"-Xmx2g",
		},
		{
			name:    "multiple_args",
			jvmArgs: []string{"-Xmx4g", "-Xms1g"},
			wantFlag: "-Dorg.gradle.jvmargs=" +
				"-Xmx4g -Xms1g",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewRobolectricAdapter(
				"/tmp",
				WithRobolectricJVMArgs(tt.jvmArgs),
			)
			flags := adapter.jvmArgFlags()
			if tt.wantNil {
				assert.Nil(t, flags)
			} else {
				require.Len(t, flags, 1)
				assert.Equal(
					t, tt.wantFlag, flags[0],
				)
			}
		})
	}
}

func TestRobolectricAdapter_OptionsChaining(
	t *testing.T,
) {
	adapter := NewRobolectricAdapter(
		"/home/dev/android",
		WithRobolectricGradleWrapper(
			"/usr/local/bin/gradle",
		),
		WithRobolectricModule(":feature:login"),
		WithRobolectricTestFilter("*Integration*"),
		WithRobolectricJVMArgs(
			[]string{"-Xmx8g"},
		),
	)

	assert.Equal(
		t, "/home/dev/android", adapter.projectDir,
	)
	assert.Equal(
		t,
		"/usr/local/bin/gradle",
		adapter.gradleWrapper,
	)
	assert.Equal(
		t, ":feature:login", adapter.module,
	)
	assert.Equal(
		t, "*Integration*", adapter.testFilter,
	)
	assert.Equal(
		t, []string{"-Xmx8g"}, adapter.jvmArgs,
	)
}

func TestWithRobolectricGradleWrapper(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "absolute_path",
			path: "/opt/gradle/6.9/bin/gradle",
		},
		{
			name: "relative_path",
			path: "../gradlew",
		},
		{
			name: "empty_path",
			path: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewRobolectricAdapter(
				"/tmp",
				WithRobolectricGradleWrapper(tt.path),
			)
			assert.Equal(
				t, tt.path, adapter.gradleWrapper,
			)
		})
	}
}
