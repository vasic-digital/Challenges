// Package panoptic provides integration between the Challenges
// framework and the Panoptic UI testing tool. It wraps Panoptic
// as a subprocess adapter, parses results, and provides custom
// assertion evaluators for UI testing outcomes.
package panoptic

import "time"

// PanopticRunResult captures the complete output of a Panoptic
// execution, including per-app results, artifacts, and reports.
type PanopticRunResult struct {
	// ExitCode is the process exit code (0 = success).
	ExitCode int `json:"exit_code"`

	// Apps holds per-app test results.
	Apps []AppResult `json:"apps"`

	// Screenshots lists absolute paths to captured screenshots.
	Screenshots []string `json:"screenshots"`

	// Videos lists absolute paths to recorded videos.
	Videos []string `json:"videos"`

	// ReportHTML is the path to the HTML report, if generated.
	ReportHTML string `json:"report_html"`

	// ReportJSON is the path to the JSON report, if generated.
	ReportJSON string `json:"report_json"`

	// AIErrorReport is the path to the AI error detection
	// report, if generated.
	AIErrorReport string `json:"ai_error_report"`

	// AIGeneratedTests is the path to AI-generated test
	// definitions, if generated.
	AIGeneratedTests string `json:"ai_generated_tests"`

	// VisionReport is the path to the computer vision analysis
	// report, if generated.
	VisionReport string `json:"vision_report"`

	// Stdout is the captured standard output from Panoptic.
	Stdout string `json:"stdout"`

	// Stderr is the captured standard error from Panoptic.
	Stderr string `json:"stderr"`

	// Duration is the total execution time.
	Duration time.Duration `json:"duration"`
}

// AppResult captures the outcome of testing a single application.
type AppResult struct {
	// Name is the application name from the config.
	Name string `json:"app_name"`

	// Type is the application type (web, desktop, mobile).
	Type string `json:"app_type"`

	// Success indicates whether the app's tests passed.
	Success bool `json:"success"`

	// Duration is the per-app execution time.
	Duration time.Duration `json:"duration"`

	// DurationMs is the duration in milliseconds.
	DurationMs int64 `json:"duration_ms"`

	// Screenshots lists screenshot paths for this app.
	Screenshots []string `json:"screenshots"`

	// Videos lists video paths for this app.
	Videos []string `json:"videos"`

	// Error is the error message, if the app failed.
	Error string `json:"error,omitempty"`
}

// PanopticConfig mirrors the Panoptic configuration structure
// for programmatic generation without importing Panoptic.
type PanopticConfig struct {
	Name     string            `yaml:"name"`
	Output   string            `yaml:"output"`
	Apps     []PanopticApp     `yaml:"apps"`
	Actions  []PanopticAction  `yaml:"actions,omitempty"`
	Settings PanopticSettings  `yaml:"settings"`
}

// PanopticApp defines a single application to test.
type PanopticApp struct {
	Name        string                 `yaml:"name"`
	Type        string                 `yaml:"type"`
	URL         string                 `yaml:"url,omitempty"`
	Path        string                 `yaml:"path,omitempty"`
	Platform    string                 `yaml:"platform,omitempty"`
	Timeout     int                    `yaml:"timeout"`
	Environment map[string]string      `yaml:"environment,omitempty"`
	Actions     []PanopticAction       `yaml:"actions"`
}

// PanopticAction defines a single test action.
type PanopticAction struct {
	Name       string                 `yaml:"name"`
	Type       string                 `yaml:"type"`
	URL        string                 `yaml:"url,omitempty"`
	Target     string                 `yaml:"target,omitempty"`
	Value      string                 `yaml:"value,omitempty"`
	Selector   string                 `yaml:"selector,omitempty"`
	WaitTime   int                    `yaml:"wait_time,omitempty"`
	Duration   int                    `yaml:"duration,omitempty"`
	Parameters map[string]interface{} `yaml:"parameters,omitempty"`
}

// PanopticSettings holds Panoptic execution settings.
type PanopticSettings struct {
	ScreenshotFormat string                 `yaml:"screenshot_format,omitempty"`
	VideoFormat      string                 `yaml:"video_format,omitempty"`
	Quality          int                    `yaml:"quality,omitempty"`
	Headless         bool                   `yaml:"headless"`
	WindowWidth      int                    `yaml:"window_width,omitempty"`
	WindowHeight     int                    `yaml:"window_height,omitempty"`
	EnableMetrics    bool                   `yaml:"enable_metrics,omitempty"`
	LogLevel         string                 `yaml:"log_level,omitempty"`
	AITesting        *AITestingSettings     `yaml:"ai_testing,omitempty"`
	Cloud            map[string]interface{} `yaml:"cloud,omitempty"`
	Enterprise       map[string]interface{} `yaml:"enterprise,omitempty"`
}

// AITestingSettings configures Panoptic's AI testing features.
type AITestingSettings struct {
	EnableErrorDetection bool    `yaml:"enable_error_detection"`
	EnableTestGeneration bool    `yaml:"enable_test_generation"`
	EnableVisionAnalysis bool    `yaml:"enable_vision_analysis"`
	ConfidenceThreshold  float64 `yaml:"confidence_threshold,omitempty"`
}
