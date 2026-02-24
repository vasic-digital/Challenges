package userflow

import "context"

// VisionAdapter detects UI elements from screenshots using
// computer vision. Implementations may use pixel-level
// heuristics, ML models, or external tools.
type VisionAdapter interface {
	// DetectElements analyzes the screenshot and returns all
	// detected UI elements.
	DetectElements(
		ctx context.Context, screenshot []byte,
	) ([]DetectedElement, error)

	// FindByType returns elements matching the given type
	// (e.g., "button", "textfield", "image", "link").
	FindByType(
		ctx context.Context, screenshot []byte,
		elemType string,
	) ([]DetectedElement, error)

	// FindByText returns elements whose text matches the
	// given string (case-insensitive substring match).
	FindByText(
		ctx context.Context, screenshot []byte,
		text string,
	) ([]DetectedElement, error)

	// Available reports whether the adapter can run.
	Available(ctx context.Context) bool
}

// DetectedElement represents a UI element found by
// computer vision.
type DetectedElement struct {
	Type       string  `json:"type"`
	Position   Point   `json:"position"`
	Size       Size    `json:"size"`
	Confidence float64 `json:"confidence"`
	Text       string  `json:"text"`
	Selector   string  `json:"selector"`
}

// Point is a pixel coordinate.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Size is a bounding box dimension.
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}
