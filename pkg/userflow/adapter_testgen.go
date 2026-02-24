package userflow

import "context"

// TestGenAdapter generates test cases from screenshots using
// AI-powered analysis. Implementations may use visual element
// detection, ML models, or external tools.
type TestGenAdapter interface {
	// GenerateTests produces test cases from the screenshot.
	GenerateTests(
		ctx context.Context, screenshot []byte,
	) ([]GeneratedTest, error)

	// GenerateReport produces a markdown report from the
	// screenshot analysis.
	GenerateReport(
		ctx context.Context, screenshot []byte,
	) (string, error)

	// Available reports whether the adapter can run.
	Available(ctx context.Context) bool
}

// GeneratedTest is a test case produced by AI analysis.
type GeneratedTest struct {
	Name       string     `json:"name"`
	Category   string     `json:"category"`
	Priority   string     `json:"priority"`
	Confidence float64    `json:"confidence"`
	Steps      []TestStep `json:"steps"`
}

// TestStep is a single action within a generated test.
type TestStep struct {
	Action     string         `json:"action"`
	Target     string         `json:"target"`
	Value      string         `json:"value,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty"`
}
