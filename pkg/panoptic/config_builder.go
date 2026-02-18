package panoptic

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigBuilder provides a fluent API for constructing Panoptic
// configuration files programmatically.
type ConfigBuilder struct {
	config PanopticConfig
	apps   []*AppBuilder
}

// AppBuilder is a sub-builder for a single application config.
type AppBuilder struct {
	parent  *ConfigBuilder
	app     PanopticApp
}

// NewConfigBuilder creates a ConfigBuilder with the given name
// and output directory.
func NewConfigBuilder(name, outputDir string) *ConfigBuilder {
	return &ConfigBuilder{
		config: PanopticConfig{
			Name:   name,
			Output: outputDir,
			Settings: PanopticSettings{
				ScreenshotFormat: "png",
				VideoFormat:      "mp4",
				Quality:          90,
				Headless:         true,
				WindowWidth:      1920,
				WindowHeight:     1080,
				EnableMetrics:    true,
				LogLevel:         "info",
			},
		},
	}
}

// AddWebApp adds a web application target and returns an
// AppBuilder for chaining actions.
func (b *ConfigBuilder) AddWebApp(
	name, url string, timeout int,
) *AppBuilder {
	ab := &AppBuilder{
		parent: b,
		app: PanopticApp{
			Name:    name,
			Type:    "web",
			URL:     url,
			Timeout: timeout,
		},
	}
	b.apps = append(b.apps, ab)
	return ab
}

// AddDesktopApp adds a desktop application target.
func (b *ConfigBuilder) AddDesktopApp(
	name, path, platform string, timeout int,
) *AppBuilder {
	ab := &AppBuilder{
		parent: b,
		app: PanopticApp{
			Name:     name,
			Type:     "desktop",
			Path:     path,
			Platform: platform,
			Timeout:  timeout,
		},
	}
	b.apps = append(b.apps, ab)
	return ab
}

// AddMobileApp adds a mobile application target.
func (b *ConfigBuilder) AddMobileApp(
	name, platform string, timeout int,
) *AppBuilder {
	ab := &AppBuilder{
		parent: b,
		app: PanopticApp{
			Name:     name,
			Type:     "mobile",
			Platform: platform,
			Timeout:  timeout,
		},
	}
	b.apps = append(b.apps, ab)
	return ab
}

// SetHeadless sets whether to run in headless mode.
func (b *ConfigBuilder) SetHeadless(headless bool) *ConfigBuilder {
	b.config.Settings.Headless = headless
	return b
}

// SetQuality sets the screenshot/video quality (1-100).
func (b *ConfigBuilder) SetQuality(quality int) *ConfigBuilder {
	b.config.Settings.Quality = quality
	return b
}

// SetWindowSize sets the browser window dimensions.
func (b *ConfigBuilder) SetWindowSize(
	width, height int,
) *ConfigBuilder {
	b.config.Settings.WindowWidth = width
	b.config.Settings.WindowHeight = height
	return b
}

// SetLogLevel sets the logging level.
func (b *ConfigBuilder) SetLogLevel(
	level string,
) *ConfigBuilder {
	b.config.Settings.LogLevel = level
	return b
}

// EnableAITesting configures AI testing features.
func (b *ConfigBuilder) EnableAITesting(
	opts AITestingOpts,
) *ConfigBuilder {
	b.config.Settings.AITesting = &AITestingSettings{
		EnableErrorDetection: opts.ErrorDetection,
		EnableTestGeneration: opts.TestGeneration,
		EnableVisionAnalysis: opts.VisionAnalysis,
		ConfidenceThreshold:  opts.ConfidenceThreshold,
	}
	return b
}

// EnableCloud configures cloud integration.
func (b *ConfigBuilder) EnableCloud(
	opts CloudOpts,
) *ConfigBuilder {
	b.config.Settings.Cloud = map[string]interface{}{
		"provider":    opts.Provider,
		"bucket":      opts.Bucket,
		"enable_sync": opts.EnableSync,
	}
	return b
}

// EnableEnterprise configures enterprise features.
func (b *ConfigBuilder) EnableEnterprise(
	opts EnterpriseOpts,
) *ConfigBuilder {
	b.config.Settings.Enterprise = map[string]interface{}{
		"config_path": opts.ConfigPath,
	}
	return b
}

// Build produces the final PanopticConfig.
func (b *ConfigBuilder) Build() PanopticConfig {
	config := b.config
	config.Apps = make([]PanopticApp, len(b.apps))
	for i, ab := range b.apps {
		config.Apps[i] = ab.app
	}
	return config
}

// WriteYAML marshals the config to YAML and writes it to the
// given file path.
func (b *ConfigBuilder) WriteYAML(path string) error {
	config := b.Build()

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config YAML: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}

	return nil
}

// --- AppBuilder methods ---

// Navigate adds a navigation action.
func (ab *AppBuilder) Navigate(
	name, url string,
) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name: name,
		Type: "navigate",
		URL:  url,
	})
	return ab
}

// Fill adds a form fill action.
func (ab *AppBuilder) Fill(
	name, selector, value string,
) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name:     name,
		Type:     "fill",
		Selector: selector,
		Value:    value,
	})
	return ab
}

// Click adds a click action.
func (ab *AppBuilder) Click(
	name, selector string,
) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name:     name,
		Type:     "click",
		Selector: selector,
	})
	return ab
}

// Wait adds a wait action.
func (ab *AppBuilder) Wait(name string, seconds int) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name:     name,
		Type:     "wait",
		WaitTime: seconds,
	})
	return ab
}

// Screenshot adds a screenshot action.
func (ab *AppBuilder) Screenshot(
	name, filename string,
) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name: name,
		Type: "screenshot",
		Parameters: map[string]interface{}{
			"filename": filename,
		},
	})
	return ab
}

// Record adds a video recording action.
func (ab *AppBuilder) Record(
	name, filename string, durationSec int,
) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name:     name,
		Type:     "record",
		Duration: durationSec,
		Parameters: map[string]interface{}{
			"filename": filename,
		},
	})
	return ab
}

// AIErrorDetection adds an AI error detection action.
func (ab *AppBuilder) AIErrorDetection(
	name, output string,
) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name: name,
		Type: "smart_error_detection",
		Parameters: map[string]interface{}{
			"output": output,
		},
	})
	return ab
}

// AITestGeneration adds an AI test generation action.
func (ab *AppBuilder) AITestGeneration(
	name, output string,
) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name: name,
		Type: "ai_test_generation",
		Parameters: map[string]interface{}{
			"output": output,
		},
	})
	return ab
}

// VisionReport adds a computer vision analysis action.
func (ab *AppBuilder) VisionReport(
	name, output string,
) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name: name,
		Type: "vision_report",
		Parameters: map[string]interface{}{
			"output": output,
		},
	})
	return ab
}

// Submit adds a form submit action.
func (ab *AppBuilder) Submit(
	name, selector string,
) *AppBuilder {
	ab.app.Actions = append(ab.app.Actions, PanopticAction{
		Name:     name,
		Type:     "submit",
		Selector: selector,
	})
	return ab
}

// Done returns the parent ConfigBuilder.
func (ab *AppBuilder) Done() *ConfigBuilder {
	return ab.parent
}
