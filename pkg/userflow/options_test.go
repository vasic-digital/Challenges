package userflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveChallengeConfig_Defaults(t *testing.T) {
	cfg := resolveChallengeConfig(nil)
	require.NotNil(t, cfg)
	assert.False(t, cfg.containerized)
	assert.Equal(t, ".", cfg.projectRoot)
	assert.Equal(t, "podman", cfg.runtimeName)
}

func TestResolveChallengeConfig_EmptySlice(t *testing.T) {
	cfg := resolveChallengeConfig([]ChallengeOption{})
	require.NotNil(t, cfg)
	assert.False(t, cfg.containerized)
	assert.Equal(t, ".", cfg.projectRoot)
	assert.Equal(t, "podman", cfg.runtimeName)
}

func TestResolveChallengeConfig_AllOptions(t *testing.T) {
	cfg := resolveChallengeConfig([]ChallengeOption{
		WithContainerized(true),
		WithProjectRoot("/opt/project"),
		WithRuntimeName("docker"),
	})
	require.NotNil(t, cfg)
	assert.True(t, cfg.containerized)
	assert.Equal(t, "/opt/project", cfg.projectRoot)
	assert.Equal(t, "docker", cfg.runtimeName)
}

func TestResolveChallengeConfig_LastOptionWins(
	t *testing.T,
) {
	cfg := resolveChallengeConfig([]ChallengeOption{
		WithContainerized(true),
		WithContainerized(false),
		WithProjectRoot("/first"),
		WithProjectRoot("/second"),
		WithRuntimeName("docker"),
		WithRuntimeName("podman"),
	})
	require.NotNil(t, cfg)
	assert.False(t, cfg.containerized)
	assert.Equal(t, "/second", cfg.projectRoot)
	assert.Equal(t, "podman", cfg.runtimeName)
}

func TestWithContainerized_True(t *testing.T) {
	cfg := &challengeConfig{}
	opt := WithContainerized(true)
	opt(cfg)
	assert.True(t, cfg.containerized)
}

func TestWithContainerized_False(t *testing.T) {
	cfg := &challengeConfig{containerized: true}
	opt := WithContainerized(false)
	opt(cfg)
	assert.False(t, cfg.containerized)
}

func TestWithProjectRoot_ValidPath(t *testing.T) {
	tests := []struct {
		name string
		root string
	}{
		{
			name: "absolute_path",
			root: "/home/user/project",
		},
		{
			name: "relative_path",
			root: "./subdir",
		},
		{
			name: "empty_string",
			root: "",
		},
		{
			name: "root_directory",
			root: "/",
		},
		{
			name: "nested_path",
			root: "/a/b/c/d/e",
		},
		{
			name: "path_with_spaces",
			root: "/path with spaces/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &challengeConfig{}
			opt := WithProjectRoot(tt.root)
			opt(cfg)
			assert.Equal(t, tt.root, cfg.projectRoot)
		})
	}
}

func TestWithRuntimeName_Variants(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
	}{
		{name: "docker", runtime: "docker"},
		{name: "podman", runtime: "podman"},
		{name: "empty", runtime: ""},
		{name: "custom", runtime: "containerd"},
		{name: "nerdctl", runtime: "nerdctl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &challengeConfig{}
			opt := WithRuntimeName(tt.runtime)
			opt(cfg)
			assert.Equal(
				t, tt.runtime, cfg.runtimeName,
			)
		})
	}
}

func TestResolveChallengeConfig_SingleOption(t *testing.T) {
	tests := []struct {
		name      string
		opt       ChallengeOption
		checkFn   func(t *testing.T, cfg *challengeConfig)
	}{
		{
			name: "only_containerized",
			opt:  WithContainerized(true),
			checkFn: func(
				t *testing.T, cfg *challengeConfig,
			) {
				assert.True(t, cfg.containerized)
				// Others stay default.
				assert.Equal(
					t, ".", cfg.projectRoot,
				)
				assert.Equal(
					t, "podman", cfg.runtimeName,
				)
			},
		},
		{
			name: "only_project_root",
			opt:  WithProjectRoot("/custom"),
			checkFn: func(
				t *testing.T, cfg *challengeConfig,
			) {
				assert.False(t, cfg.containerized)
				assert.Equal(
					t, "/custom", cfg.projectRoot,
				)
				assert.Equal(
					t, "podman", cfg.runtimeName,
				)
			},
		},
		{
			name: "only_runtime",
			opt:  WithRuntimeName("docker"),
			checkFn: func(
				t *testing.T, cfg *challengeConfig,
			) {
				assert.False(t, cfg.containerized)
				assert.Equal(
					t, ".", cfg.projectRoot,
				)
				assert.Equal(
					t, "docker", cfg.runtimeName,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := resolveChallengeConfig(
				[]ChallengeOption{tt.opt},
			)
			require.NotNil(t, cfg)
			tt.checkFn(t, cfg)
		})
	}
}

func TestChallengeOption_FunctionType(t *testing.T) {
	// Verify ChallengeOption is a function type
	// that accepts *challengeConfig.
	var opt ChallengeOption = func(c *challengeConfig) {
		c.containerized = true
		c.projectRoot = "/test"
		c.runtimeName = "custom"
	}

	cfg := &challengeConfig{}
	opt(cfg)
	assert.True(t, cfg.containerized)
	assert.Equal(t, "/test", cfg.projectRoot)
	assert.Equal(t, "custom", cfg.runtimeName)
}
