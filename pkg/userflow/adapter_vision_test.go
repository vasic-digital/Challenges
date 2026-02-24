package userflow

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface check.
var _ VisionAdapter = (*PanopticVisionAdapter)(nil)

func TestDetectedElement_Fields(t *testing.T) {
	elem := DetectedElement{
		Type:       "button",
		Position:   Point{X: 100, Y: 200},
		Size:       Size{Width: 80, Height: 40},
		Confidence: 0.95,
		Text:       "Submit",
		Selector:   "button.submit",
	}

	assert.Equal(t, "button", elem.Type)
	assert.Equal(t, 100, elem.Position.X)
	assert.Equal(t, 200, elem.Position.Y)
	assert.Equal(t, 80, elem.Size.Width)
	assert.Equal(t, 40, elem.Size.Height)
	assert.InDelta(t, 0.95, elem.Confidence, 0.001)
	assert.Equal(t, "Submit", elem.Text)
	assert.Equal(t, "button.submit", elem.Selector)
}

func TestPoint_ZeroValue(t *testing.T) {
	var p Point
	assert.Equal(t, 0, p.X)
	assert.Equal(t, 0, p.Y)
}

func TestSize_ZeroValue(t *testing.T) {
	var s Size
	assert.Equal(t, 0, s.Width)
	assert.Equal(t, 0, s.Height)
}

func TestPanopticVisionAdapter_Constructor(t *testing.T) {
	adapter := NewPanopticVisionAdapter("/usr/bin/panoptic")
	assert.NotNil(t, adapter)
	assert.Equal(t, "/usr/bin/panoptic", adapter.binaryPath)
}

func TestPanopticVisionAdapter_Available_NotFound(
	t *testing.T,
) {
	adapter := NewPanopticVisionAdapter(
		"/nonexistent/path/to/panoptic-binary-xyz",
	)
	assert.False(
		t, adapter.Available(context.Background()),
	)
}

func TestPanopticVisionAdapter_Available_ExistingBinary(
	t *testing.T,
) {
	// /bin/sh exists on virtually all systems.
	adapter := NewPanopticVisionAdapter("/bin/sh")
	assert.True(
		t, adapter.Available(context.Background()),
	)
}

func TestPanopticElement_ToDetectedElement(t *testing.T) {
	raw := panopticElement{
		Type:       "textfield",
		X:          10,
		Y:          20,
		Width:      300,
		Height:     50,
		Confidence: 0.87,
		Text:       "Username",
		Selector:   "input#username",
	}

	elem := raw.toDetectedElement()
	assert.Equal(t, "textfield", elem.Type)
	assert.Equal(t, Point{X: 10, Y: 20}, elem.Position)
	assert.Equal(t, Size{Width: 300, Height: 50}, elem.Size)
	assert.InDelta(t, 0.87, elem.Confidence, 0.001)
	assert.Equal(t, "Username", elem.Text)
	assert.Equal(t, "input#username", elem.Selector)
}

// mockVisionAdapter is a test double implementing
// VisionAdapter with configurable element lists.
type mockVisionAdapter struct {
	elements  []DetectedElement
	available bool
}

var _ VisionAdapter = (*mockVisionAdapter)(nil)

func (m *mockVisionAdapter) DetectElements(
	_ context.Context, _ []byte,
) ([]DetectedElement, error) {
	return m.elements, nil
}

func (m *mockVisionAdapter) FindByType(
	_ context.Context, _ []byte, elemType string,
) ([]DetectedElement, error) {
	lower := strings.ToLower(elemType)
	var matched []DetectedElement
	for _, e := range m.elements {
		if strings.ToLower(e.Type) == lower {
			matched = append(matched, e)
		}
	}
	return matched, nil
}

func (m *mockVisionAdapter) FindByText(
	_ context.Context, _ []byte, text string,
) ([]DetectedElement, error) {
	lower := strings.ToLower(text)
	var matched []DetectedElement
	for _, e := range m.elements {
		if strings.Contains(
			strings.ToLower(e.Text), lower,
		) {
			matched = append(matched, e)
		}
	}
	return matched, nil
}

func (m *mockVisionAdapter) Available(
	_ context.Context,
) bool {
	return m.available
}

func TestMockVisionAdapter_FindByType_Filtering(
	t *testing.T,
) {
	mock := &mockVisionAdapter{
		elements: []DetectedElement{
			{
				Type:     "button",
				Position: Point{X: 10, Y: 20},
				Size:     Size{Width: 100, Height: 40},
				Text:     "Submit",
			},
			{
				Type:     "textfield",
				Position: Point{X: 10, Y: 80},
				Size:     Size{Width: 200, Height: 30},
				Text:     "Email",
			},
			{
				Type:     "button",
				Position: Point{X: 120, Y: 20},
				Size:     Size{Width: 100, Height: 40},
				Text:     "Cancel",
			},
			{
				Type:     "image",
				Position: Point{X: 0, Y: 0},
				Size:     Size{Width: 50, Height: 50},
				Text:     "",
			},
		},
		available: true,
	}

	ctx := context.Background()

	buttons, err := mock.FindByType(
		ctx, nil, "button",
	)
	assert.NoError(t, err)
	assert.Len(t, buttons, 2)
	assert.Equal(t, "Submit", buttons[0].Text)
	assert.Equal(t, "Cancel", buttons[1].Text)

	textfields, err := mock.FindByType(
		ctx, nil, "textfield",
	)
	assert.NoError(t, err)
	assert.Len(t, textfields, 1)
	assert.Equal(t, "Email", textfields[0].Text)

	images, err := mock.FindByType(
		ctx, nil, "image",
	)
	assert.NoError(t, err)
	assert.Len(t, images, 1)

	links, err := mock.FindByType(
		ctx, nil, "link",
	)
	assert.NoError(t, err)
	assert.Empty(t, links)
}

func TestMockVisionAdapter_FindByType_CaseInsensitive(
	t *testing.T,
) {
	mock := &mockVisionAdapter{
		elements: []DetectedElement{
			{Type: "Button", Text: "OK"},
			{Type: "BUTTON", Text: "Cancel"},
		},
	}

	ctx := context.Background()
	results, err := mock.FindByType(ctx, nil, "button")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestMockVisionAdapter_FindByText_Filtering(
	t *testing.T,
) {
	mock := &mockVisionAdapter{
		elements: []DetectedElement{
			{Type: "button", Text: "Submit Form"},
			{Type: "button", Text: "Cancel"},
			{Type: "label", Text: "Please submit your data"},
			{Type: "link", Text: "Back to home"},
		},
		available: true,
	}

	ctx := context.Background()

	results, err := mock.FindByText(ctx, nil, "submit")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Submit Form", results[0].Text)
	assert.Equal(
		t,
		"Please submit your data",
		results[1].Text,
	)

	results, err = mock.FindByText(ctx, nil, "home")
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Back to home", results[0].Text)

	results, err = mock.FindByText(
		ctx, nil, "nonexistent",
	)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestMockVisionAdapter_FindByText_CaseInsensitive(
	t *testing.T,
) {
	mock := &mockVisionAdapter{
		elements: []DetectedElement{
			{Type: "button", Text: "SUBMIT"},
			{Type: "label", Text: "Submit Here"},
		},
	}

	ctx := context.Background()
	results, err := mock.FindByText(ctx, nil, "submit")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestMockVisionAdapter_DetectElements_ReturnsAll(
	t *testing.T,
) {
	elems := []DetectedElement{
		{Type: "button", Text: "A"},
		{Type: "link", Text: "B"},
		{Type: "image", Text: "C"},
	}
	mock := &mockVisionAdapter{elements: elems}

	ctx := context.Background()
	results, err := mock.DetectElements(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, elems, results)
}

func TestMockVisionAdapter_Available(t *testing.T) {
	mock := &mockVisionAdapter{available: true}
	assert.True(
		t, mock.Available(context.Background()),
	)

	mock.available = false
	assert.False(
		t, mock.Available(context.Background()),
	)
}
