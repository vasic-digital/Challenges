package userflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

func TestVisionFlowChallenge_Execute_WithSelector(
	t *testing.T,
) {
	browser := newMockBrowserAdapter()
	vision := &mockVisionAdapter{
		elements:  nil,
		available: true,
	}

	flow := BrowserFlow{
		Name:     "selector-flow",
		StartURL: "http://localhost:3000/login",
		Config: BrowserConfig{
			BrowserType: "chromium",
			Headless:    true,
		},
		Steps: []BrowserStep{
			{
				Name:     "fill email",
				Action:   "fill",
				Selector: "#email",
				Value:    "user@example.com",
			},
			{
				Name:     "click login",
				Action:   "click",
				Selector: "#login-btn",
			},
		},
	}

	ch := NewVisionFlowChallenge(
		"VISION-001", "Selector Flow",
		"Steps with selectors use normal browser",
		nil, browser, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 2)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed, "failed: %s", a.Message)
	}

	// Verify browser adapter was used directly.
	require.Len(t, browser.filledSels, 1)
	assert.Equal(t, "#email", browser.filledSels[0])
	assert.Equal(
		t, "user@example.com", browser.filledVals[0],
	)
	require.Len(t, browser.clickedSels, 1)
	assert.Equal(t, "#login-btn", browser.clickedSels[0])

	// Vision should not have been invoked (no screenshots
	// for detection).
	assert.Equal(t, "0", result.Outputs["vision_detections"])
	assert.True(t, browser.closed)
}

func TestVisionFlowChallenge_Execute_WithVision(
	t *testing.T,
) {
	browser := newMockBrowserAdapter()
	browser.evaluateResults["document.elementFromPoint(150,50).click()"] = ""

	vision := &mockVisionAdapter{
		elements: []DetectedElement{
			{
				Type:       "button",
				Position:   Point{X: 150, Y: 50},
				Size:       Size{Width: 100, Height: 40},
				Confidence: 0.93,
				Text:       "Submit",
			},
			{
				Type:       "button",
				Position:   Point{X: 300, Y: 50},
				Size:       Size{Width: 100, Height: 40},
				Confidence: 0.88,
				Text:       "Cancel",
			},
		},
		available: true,
	}

	flow := BrowserFlow{
		Name:     "vision-flow",
		StartURL: "http://localhost:3000/form",
		Config: BrowserConfig{
			BrowserType: "chromium",
			Headless:    true,
		},
		Steps: []BrowserStep{
			{
				Name:   "click submit via vision",
				Action: "click",
				// No Selector: triggers vision path.
				Value: "button:Submit",
			},
		},
	}

	ch := NewVisionFlowChallenge(
		"VISION-002", "Vision Flow",
		"Steps without selectors use vision",
		nil, browser, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.True(t, result.Assertions[0].Passed)

	// Screenshot was taken for vision detection.
	assert.GreaterOrEqual(t, browser.screenshotCount, 1)

	// Vision detected elements.
	assert.Equal(t, "1", result.Outputs["vision_detections"])

	// Confidence metric recorded.
	confMetric, ok := result.Metrics["vision_confidence_click submit via vision"]
	assert.True(t, ok)
	assert.InDelta(t, 0.93, confMetric.Value, 0.001)

	// Vision elements detected metric recorded.
	veMetric, ok := result.Metrics["vision_elements_detected"]
	assert.True(t, ok)
	assert.Equal(t, 1.0, veMetric.Value)

	assert.True(t, browser.closed)
}

func TestVisionFlowChallenge_Execute_BrowserUnavailable(
	t *testing.T,
) {
	browser := &mockBrowserUnavailable{}
	vision := &mockVisionAdapter{
		available: true,
	}

	flow := BrowserFlow{
		Name:     "unavailable-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:   "click",
				Action: "click",
				Value:  "button:OK",
			},
		},
	}

	ch := NewVisionFlowChallenge(
		"VISION-003", "Unavailable",
		"Browser unavailable - skipped",
		nil, browser, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.Contains(
		t, result.Assertions[0].Message,
		"Browser not available",
	)
}

func TestNewVisionFlowChallenge_Constructor(t *testing.T) {
	browser := newMockBrowserAdapter()
	vision := &mockVisionAdapter{available: true}
	flow := BrowserFlow{
		Name:     "test-flow",
		StartURL: "http://localhost:3000",
		Config: BrowserConfig{
			BrowserType: "chromium",
			Headless:    true,
		},
	}
	deps := []challenge.ID{"ENV-SETUP", "API-HEALTH"}

	ch := NewVisionFlowChallenge(
		"VISION-004", "Vision Test",
		"Test vision challenge",
		deps, browser, vision, flow,
	)

	assert.Equal(
		t, challenge.ID("VISION-004"), ch.ID(),
	)
	assert.Equal(t, "Vision Test", ch.Name())
	assert.Equal(t, "browser", ch.Category())
	assert.Equal(t, deps, ch.Dependencies())
	assert.NotNil(t, ch.browser)
	assert.NotNil(t, ch.vision)
	assert.Equal(t, "test-flow", ch.flow.Name)
}

func TestVisionFlowChallenge_Execute_VisionFindByText(
	t *testing.T,
) {
	browser := newMockBrowserAdapter()
	browser.evaluateResults["document.elementFromPoint(200,100).click()"] = ""

	vision := &mockVisionAdapter{
		elements: []DetectedElement{
			{
				Type:       "link",
				Position:   Point{X: 200, Y: 100},
				Size:       Size{Width: 80, Height: 20},
				Confidence: 0.91,
				Text:       "Learn More",
			},
		},
		available: true,
	}

	flow := BrowserFlow{
		Name:     "find-by-text-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:   "click learn more",
				Action: "click",
				// No ":" means FindByText is used.
				Value: "Learn More",
			},
		},
	}

	ch := NewVisionFlowChallenge(
		"VISION-005", "FindByText",
		"Vision uses FindByText when no colon",
		nil, browser, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Equal(t, "1", result.Outputs["vision_detections"])
}

func TestVisionFlowChallenge_Execute_VisionNotFound(
	t *testing.T,
) {
	browser := newMockBrowserAdapter()
	vision := &mockVisionAdapter{
		elements:  []DetectedElement{},
		available: true,
	}

	flow := BrowserFlow{
		Name:     "not-found-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:   "click missing",
				Action: "click",
				Value:  "button:NonExistent",
			},
		},
	}

	ch := NewVisionFlowChallenge(
		"VISION-006", "Not Found",
		"Vision finds no element",
		nil, browser, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message,
		"no element found",
	)
}

// mockBrowserUnavailable is a BrowserAdapter that reports
// itself as unavailable.
type mockBrowserUnavailable struct{}

func (m *mockBrowserUnavailable) Initialize(
	_ context.Context, _ BrowserConfig,
) error {
	return nil
}

func (m *mockBrowserUnavailable) Navigate(
	_ context.Context, _ string,
) error {
	return nil
}

func (m *mockBrowserUnavailable) Click(
	_ context.Context, _ string,
) error {
	return nil
}

func (m *mockBrowserUnavailable) Fill(
	_ context.Context, _, _ string,
) error {
	return nil
}

func (m *mockBrowserUnavailable) SelectOption(
	_ context.Context, _, _ string,
) error {
	return nil
}

func (m *mockBrowserUnavailable) IsVisible(
	_ context.Context, _ string,
) (bool, error) {
	return false, nil
}

func (m *mockBrowserUnavailable) WaitForSelector(
	_ context.Context, _ string, _ time.Duration,
) error {
	return nil
}

func (m *mockBrowserUnavailable) GetText(
	_ context.Context, _ string,
) (string, error) {
	return "", nil
}

func (m *mockBrowserUnavailable) GetAttribute(
	_ context.Context, _, _ string,
) (string, error) {
	return "", nil
}

func (m *mockBrowserUnavailable) Screenshot(
	_ context.Context,
) ([]byte, error) {
	return nil, nil
}

func (m *mockBrowserUnavailable) EvaluateJS(
	_ context.Context, _ string,
) (string, error) {
	return "", nil
}

func (m *mockBrowserUnavailable) NetworkIntercept(
	_ context.Context,
	_ string,
	_ func(req *InterceptedRequest),
) error {
	return nil
}

func (m *mockBrowserUnavailable) Close(
	_ context.Context,
) error {
	return nil
}

func (m *mockBrowserUnavailable) Available(
	_ context.Context,
) bool {
	return false
}
