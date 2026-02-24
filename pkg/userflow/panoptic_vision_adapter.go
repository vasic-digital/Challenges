package userflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PanopticVisionAdapter implements VisionAdapter by invoking
// the Panoptic CLI's vision subcommand.
type PanopticVisionAdapter struct {
	binaryPath string
}

// Compile-time interface check.
var _ VisionAdapter = (*PanopticVisionAdapter)(nil)

// NewPanopticVisionAdapter creates an adapter that invokes the
// Panoptic binary at the given path for vision detection.
func NewPanopticVisionAdapter(
	binaryPath string,
) *PanopticVisionAdapter {
	return &PanopticVisionAdapter{binaryPath: binaryPath}
}

// panopticElement is the flat JSON structure returned by the
// panoptic vision detect CLI command.
type panopticElement struct {
	Type       string  `json:"type"`
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Confidence float64 `json:"confidence"`
	Text       string  `json:"text"`
	Selector   string  `json:"selector"`
}

// toDetectedElement converts the flat CLI JSON struct into a
// DetectedElement with nested Point and Size.
func (e panopticElement) toDetectedElement() DetectedElement {
	return DetectedElement{
		Type:       e.Type,
		Position:   Point{X: e.X, Y: e.Y},
		Size:       Size{Width: e.Width, Height: e.Height},
		Confidence: e.Confidence,
		Text:       e.Text,
		Selector:   e.Selector,
	}
}

// DetectElements writes the screenshot to a temp file and
// invokes panoptic vision detect to find all UI elements.
func (a *PanopticVisionAdapter) DetectElements(
	ctx context.Context, screenshot []byte,
) ([]DetectedElement, error) {
	tmpFile, err := os.CreateTemp("", "vision-*.png")
	if err != nil {
		return nil, fmt.Errorf(
			"create temp screenshot: %w", err,
		)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(screenshot); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf(
			"write screenshot: %w", err,
		)
	}
	tmpFile.Close()

	cmd := exec.CommandContext(
		ctx, a.binaryPath,
		"vision", "detect",
		"--screenshot", tmpFile.Name(),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"panoptic vision detect: %w\noutput: %s",
			err, string(out),
		)
	}

	var raw []panopticElement
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf(
			"parse vision output: %w", err,
		)
	}

	elements := make([]DetectedElement, len(raw))
	for i, r := range raw {
		elements[i] = r.toDetectedElement()
	}
	return elements, nil
}

// FindByType calls DetectElements and filters results by the
// element Type field (case-insensitive match).
func (a *PanopticVisionAdapter) FindByType(
	ctx context.Context, screenshot []byte,
	elemType string,
) ([]DetectedElement, error) {
	all, err := a.DetectElements(ctx, screenshot)
	if err != nil {
		return nil, err
	}

	lower := strings.ToLower(elemType)
	var matched []DetectedElement
	for _, e := range all {
		if strings.ToLower(e.Type) == lower {
			matched = append(matched, e)
		}
	}
	return matched, nil
}

// FindByText calls DetectElements and filters results by a
// case-insensitive substring match on the Text field.
func (a *PanopticVisionAdapter) FindByText(
	ctx context.Context, screenshot []byte,
	text string,
) ([]DetectedElement, error) {
	all, err := a.DetectElements(ctx, screenshot)
	if err != nil {
		return nil, err
	}

	lower := strings.ToLower(text)
	var matched []DetectedElement
	for _, e := range all {
		if strings.Contains(
			strings.ToLower(e.Text), lower,
		) {
			matched = append(matched, e)
		}
	}
	return matched, nil
}

// Available reports whether the Panoptic binary exists at the
// configured path.
func (a *PanopticVisionAdapter) Available(
	_ context.Context,
) bool {
	if _, err := os.Stat(a.binaryPath); err == nil {
		return true
	}
	// Fall back to PATH lookup.
	if _, err := exec.LookPath(a.binaryPath); err == nil {
		return true
	}
	return false
}
