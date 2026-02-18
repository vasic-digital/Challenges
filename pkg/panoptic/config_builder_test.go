package panoptic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewConfigBuilder(t *testing.T) {
	b := NewConfigBuilder("Test Suite", "./output")
	config := b.Build()

	assert.Equal(t, "Test Suite", config.Name)
	assert.Equal(t, "./output", config.Output)
	assert.True(t, config.Settings.Headless)
	assert.Equal(t, 1920, config.Settings.WindowWidth)
	assert.Equal(t, 1080, config.Settings.WindowHeight)
	assert.Equal(t, 90, config.Settings.Quality)
	assert.Equal(t, "png", config.Settings.ScreenshotFormat)
	assert.Equal(t, "mp4", config.Settings.VideoFormat)
}

func TestConfigBuilder_AddWebApp(t *testing.T) {
	b := NewConfigBuilder("Test", "./out")
	b.AddWebApp("Admin", "http://localhost:3001", 60).
		Navigate("login", "http://localhost:3001/login").
		Fill("user", "input[name='username']", "admin").
		Click("submit", "button[type='submit']").
		Wait("load", 3).
		Screenshot("dash", "dashboard.png").
		Done()

	config := b.Build()

	require.Len(t, config.Apps, 1)
	app := config.Apps[0]
	assert.Equal(t, "Admin", app.Name)
	assert.Equal(t, "web", app.Type)
	assert.Equal(t, "http://localhost:3001", app.URL)
	assert.Equal(t, 60, app.Timeout)
	assert.Len(t, app.Actions, 5)

	assert.Equal(t, "navigate", app.Actions[0].Type)
	assert.Equal(t, "fill", app.Actions[1].Type)
	assert.Equal(t, "click", app.Actions[2].Type)
	assert.Equal(t, "wait", app.Actions[3].Type)
	assert.Equal(t, 3, app.Actions[3].WaitTime)
	assert.Equal(t, "screenshot", app.Actions[4].Type)
}

func TestConfigBuilder_AddDesktopApp(t *testing.T) {
	b := NewConfigBuilder("Desktop", "./out")
	b.AddDesktopApp("App", "/usr/bin/app", "linux", 30).Done()

	config := b.Build()

	require.Len(t, config.Apps, 1)
	assert.Equal(t, "desktop", config.Apps[0].Type)
	assert.Equal(t, "/usr/bin/app", config.Apps[0].Path)
	assert.Equal(t, "linux", config.Apps[0].Platform)
}

func TestConfigBuilder_AddMobileApp(t *testing.T) {
	b := NewConfigBuilder("Mobile", "./out")
	b.AddMobileApp("App", "android", 60).Done()

	config := b.Build()

	require.Len(t, config.Apps, 1)
	assert.Equal(t, "mobile", config.Apps[0].Type)
	assert.Equal(t, "android", config.Apps[0].Platform)
}

func TestConfigBuilder_MultipleApps(t *testing.T) {
	b := NewConfigBuilder("Multi", "./out")
	b.AddWebApp("Admin", "http://localhost:3001", 60).Done()
	b.AddWebApp("Web", "http://localhost:3000", 60).Done()

	config := b.Build()
	assert.Len(t, config.Apps, 2)
}

func TestConfigBuilder_Settings(t *testing.T) {
	b := NewConfigBuilder("Test", "./out")
	b.SetHeadless(false).
		SetQuality(80).
		SetWindowSize(1280, 720).
		SetLogLevel("debug")

	config := b.Build()

	assert.False(t, config.Settings.Headless)
	assert.Equal(t, 80, config.Settings.Quality)
	assert.Equal(t, 1280, config.Settings.WindowWidth)
	assert.Equal(t, 720, config.Settings.WindowHeight)
	assert.Equal(t, "debug", config.Settings.LogLevel)
}

func TestConfigBuilder_AITesting(t *testing.T) {
	b := NewConfigBuilder("AI", "./out")
	b.EnableAITesting(AITestingOpts{
		ErrorDetection:      true,
		TestGeneration:      true,
		VisionAnalysis:      true,
		ConfidenceThreshold: 0.85,
	})

	config := b.Build()

	require.NotNil(t, config.Settings.AITesting)
	assert.True(t, config.Settings.AITesting.EnableErrorDetection)
	assert.True(t, config.Settings.AITesting.EnableTestGeneration)
	assert.True(t, config.Settings.AITesting.EnableVisionAnalysis)
	assert.InDelta(t, 0.85,
		config.Settings.AITesting.ConfidenceThreshold, 0.001,
	)
}

func TestConfigBuilder_Cloud(t *testing.T) {
	b := NewConfigBuilder("Cloud", "./out")
	b.EnableCloud(CloudOpts{
		Provider:   "aws",
		Bucket:     "test-bucket",
		EnableSync: true,
	})

	config := b.Build()

	require.NotNil(t, config.Settings.Cloud)
	assert.Equal(t, "aws", config.Settings.Cloud["provider"])
	assert.Equal(t, "test-bucket", config.Settings.Cloud["bucket"])
	assert.Equal(t, true, config.Settings.Cloud["enable_sync"])
}

func TestConfigBuilder_Enterprise(t *testing.T) {
	b := NewConfigBuilder("Enterprise", "./out")
	b.EnableEnterprise(EnterpriseOpts{
		ConfigPath: "/etc/enterprise.yaml",
	})

	config := b.Build()

	require.NotNil(t, config.Settings.Enterprise)
	assert.Equal(t, "/etc/enterprise.yaml",
		config.Settings.Enterprise["config_path"],
	)
}

func TestConfigBuilder_AppActions(t *testing.T) {
	b := NewConfigBuilder("Actions", "./out")
	b.AddWebApp("Admin", "http://localhost:3001", 60).
		Record("session", "recording.mp4", 120).
		AIErrorDetection("errors", "ai_errors.json").
		AITestGeneration("tests", "ai_tests.yaml").
		VisionReport("vision", "vision.json").
		Submit("form", "form#main").
		Done()

	config := b.Build()
	actions := config.Apps[0].Actions

	assert.Equal(t, "record", actions[0].Type)
	assert.Equal(t, 120, actions[0].Duration)
	assert.Equal(t, "smart_error_detection", actions[1].Type)
	assert.Equal(t, "ai_test_generation", actions[2].Type)
	assert.Equal(t, "vision_report", actions[3].Type)
	assert.Equal(t, "submit", actions[4].Type)
}

func TestConfigBuilder_WriteYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_config.yaml")

	b := NewConfigBuilder("YAML Test", "./out")
	b.AddWebApp("Admin", "http://localhost:3001", 60).
		Navigate("login", "http://localhost:3001/login").
		Screenshot("page", "page.png").
		Done()

	err := b.WriteYAML(path)
	require.NoError(t, err)

	// Verify file exists and is valid YAML.
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var config PanopticConfig
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err)

	assert.Equal(t, "YAML Test", config.Name)
	assert.Equal(t, "./out", config.Output)
	require.Len(t, config.Apps, 1)
	assert.Equal(t, "Admin", config.Apps[0].Name)
	assert.Len(t, config.Apps[0].Actions, 2)
}

func TestConfigBuilder_WriteYAML_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sub", "dir", "config.yaml")

	b := NewConfigBuilder("Nested", "./out")
	err := b.WriteYAML(path)
	require.NoError(t, err)

	assert.FileExists(t, path)
}
