package userflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

// mockVisionForRecording implements VisionAdapter for
// recorded vision flow tests with per-query result maps.
type mockVisionForRecording struct {
	available         bool
	findByTextResults map[string][]DetectedElement
	findByTypeResults map[string][]DetectedElement
}

func newMockVisionForRecording() *mockVisionForRecording {
	return &mockVisionForRecording{
		available: true,
		findByTextResults: make(
			map[string][]DetectedElement,
		),
		findByTypeResults: make(
			map[string][]DetectedElement,
		),
	}
}

func (m *mockVisionForRecording) DetectElements(
	_ context.Context, _ []byte,
) ([]DetectedElement, error) {
	var all []DetectedElement
	for _, elems := range m.findByTextResults {
		all = append(all, elems...)
	}
	return all, nil
}

func (m *mockVisionForRecording) FindByType(
	_ context.Context, _ []byte, elemType string,
) ([]DetectedElement, error) {
	if elems, ok := m.findByTypeResults[elemType]; ok {
		return elems, nil
	}
	return nil, nil
}

func (m *mockVisionForRecording) FindByText(
	_ context.Context, _ []byte, text string,
) ([]DetectedElement, error) {
	if elems, ok := m.findByTextResults[text]; ok {
		return elems, nil
	}
	return nil, nil
}

func (m *mockVisionForRecording) Available(
	_ context.Context,
) bool {
	return m.available
}

// --- RecordedVisionFlowChallenge tests ---

func TestNewRecordedVisionFlowChallenge_Constructor(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	vision := newMockVisionForRecording()
	flow := BrowserFlow{
		Name:     "test-flow",
		StartURL: "http://localhost:3000",
		Config: BrowserConfig{
			BrowserType: "chromium",
			Headless:    true,
		},
	}
	deps := []challenge.ID{"SETUP-001", "HEALTH-001"}

	ch := NewRecordedVisionFlowChallenge(
		"RECVIS-001", "Recorded Vision Flow",
		"Recorded vision flow test",
		deps, browser, recorder, vision, flow,
	)

	assert.Equal(
		t, challenge.ID("RECVIS-001"), ch.ID(),
	)
	assert.Equal(
		t, "Recorded Vision Flow", ch.Name(),
	)
	assert.Equal(
		t, "Recorded vision flow test",
		ch.Description(),
	)
	assert.Equal(t, deps, ch.Dependencies())
}

func TestRecordedVisionFlowChallenge_Category(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	vision := newMockVisionForRecording()
	flow := BrowserFlow{
		Name:     "cat-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedVisionFlowChallenge(
		"RECVIS-002", "Category Test",
		"Check category",
		nil, browser, recorder, vision, flow,
	)

	assert.Equal(t, "browser", ch.Category())
}

func TestRecordedVisionFlowChallenge_Execute_Success(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	browser.evaluateResults["document.elementFromPoint(150,50).click()"] = ""
	recorder := newMockRecorderForFlow()
	vision := newMockVisionForRecording()
	vision.findByTypeResults["button"] = []DetectedElement{
		{
			Type:       "button",
			Position:   Point{X: 150, Y: 50},
			Size:       Size{Width: 100, Height: 40},
			Confidence: 0.95,
			Text:       "Submit",
		},
	}

	flow := BrowserFlow{
		Name:     "mixed-flow",
		StartURL: "http://localhost:3000/form",
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
				Name:   "click submit via vision",
				Action: "click",
				// No Selector: triggers vision path.
				Value: "button:Submit",
			},
		},
	}

	ch := NewRecordedVisionFlowChallenge(
		"RECVIS-003", "Recorded Mixed Flow",
		"Mixed CSS and vision with recording",
		nil, browser, recorder, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusPassed, result.Status,
	)

	// Verify browser interactions.
	assert.True(t, browser.initialized)
	assert.Equal(
		t, "http://localhost:3000/form",
		browser.navigated,
	)
	assert.Equal(
		t, "user@example.com",
		browser.filled["#email"],
	)
	assert.True(t, browser.closed)

	// Verify recorder interactions.
	assert.True(t, recorder.started)
	assert.True(t, recorder.stopped)
	assert.Equal(
		t, "http://localhost:3000/form",
		recorder.config.URL,
	)

	// Verify recording metrics are present.
	vidDur, ok := result.Metrics["video_duration"]
	assert.True(t, ok)
	assert.Equal(t, 5.0, vidDur.Value)

	vidFrames, ok := result.Metrics["video_frame_count"]
	assert.True(t, ok)
	assert.Equal(t, 150.0, vidFrames.Value)

	vidSize, ok := result.Metrics["video_file_size"]
	assert.True(t, ok)
	assert.Equal(t, 1024000.0, vidSize.Value)

	// Verify recording outputs.
	assert.Equal(
		t, "/tmp/recording.webm",
		result.Outputs["video_path"],
	)

	// Verify vision metrics.
	veMetric, ok := result.Metrics["vision_elements_detected"]
	assert.True(t, ok)
	assert.Equal(t, 1.0, veMetric.Value)

	// Verify vision outputs.
	assert.Equal(
		t, "1", result.Outputs["vision_detections"],
	)

	// Verify standard metrics.
	_, ok = result.Metrics["total_duration"]
	assert.True(t, ok)
	steps := result.Metrics["steps_executed"]
	assert.Equal(t, 2.0, steps.Value)

	// Verify assertions include recording start,
	// step results, and recording integrity.
	hasRecStart := false
	hasRecIntegrity := false
	for _, a := range result.Assertions {
		if a.Target == "start_recording" {
			hasRecStart = true
			assert.True(t, a.Passed)
		}
		if a.Target == "video_integrity" {
			hasRecIntegrity = true
			assert.True(t, a.Passed)
		}
	}
	assert.True(
		t, hasRecStart,
		"missing recording start assertion",
	)
	assert.True(
		t, hasRecIntegrity,
		"missing recording integrity assertion",
	)
}

func TestRecordedVisionFlowChallenge_Execute_BrowserUnavailable(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	browser.available = false
	recorder := newMockRecorderForFlow()
	vision := newMockVisionForRecording()
	flow := BrowserFlow{
		Name:     "unavail-browser",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedVisionFlowChallenge(
		"RECVIS-004", "Browser Unavail",
		"Browser not available",
		nil, browser, recorder, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error, "browser not available",
	)

	// Recorder should not have been started.
	assert.False(t, recorder.started)
	assert.False(t, browser.initialized)
}

func TestRecordedVisionFlowChallenge_Execute_RecorderUnavailable(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	recorder.available = false
	vision := newMockVisionForRecording()
	flow := BrowserFlow{
		Name:     "unavail-recorder",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedVisionFlowChallenge(
		"RECVIS-005", "Recorder Unavail",
		"Recorder not available",
		nil, browser, recorder, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error, "recorder not available",
	)

	// Browser should not have been initialized.
	assert.False(t, browser.initialized)
	assert.False(t, recorder.started)
}

func TestRecordedVisionFlowChallenge_Execute_VisionUnavailable(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	vision := newMockVisionForRecording()
	vision.available = false
	flow := BrowserFlow{
		Name:     "unavail-vision",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedVisionFlowChallenge(
		"RECVIS-006", "Vision Unavail",
		"Vision not available",
		nil, browser, recorder, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error, "vision not available",
	)

	// Browser and recorder should not have been started.
	assert.False(t, browser.initialized)
	assert.False(t, recorder.started)
}

func TestRecordedVisionFlowChallenge_Execute_RecordingStartFailure(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	recorder.startErr = fmt.Errorf("ffmpeg not installed")
	vision := newMockVisionForRecording()
	flow := BrowserFlow{
		Name:     "rec-start-fail",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedVisionFlowChallenge(
		"RECVIS-007", "Rec Start Fail",
		"Recording start fails",
		nil, browser, recorder, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusFailed, result.Status,
	)
	assert.Contains(
		t, result.Error, "ffmpeg not installed",
	)
}

func TestRecordedVisionFlowChallenge_Execute_VisionStepSuccess(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	browser.evaluateResults["document.elementFromPoint(200,100).click()"] = ""
	recorder := newMockRecorderForFlow()
	vision := newMockVisionForRecording()
	vision.findByTextResults["Learn More"] = []DetectedElement{
		{
			Type:       "link",
			Position:   Point{X: 200, Y: 100},
			Size:       Size{Width: 80, Height: 20},
			Confidence: 0.91,
			Text:       "Learn More",
		},
	}

	flow := BrowserFlow{
		Name:     "vision-step-flow",
		StartURL: "http://localhost:3000",
		Config: BrowserConfig{
			BrowserType: "chromium",
			Headless:    true,
		},
		Steps: []BrowserStep{
			{
				Name:   "click learn more",
				Action: "click",
				// No Selector and no ":" triggers
				// FindByText path.
				Value: "Learn More",
			},
		},
	}

	ch := NewRecordedVisionFlowChallenge(
		"RECVIS-008", "Vision Step Success",
		"Vision step with recording",
		nil, browser, recorder, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusPassed, result.Status,
	)

	// Vision detections recorded.
	assert.Equal(
		t, "1", result.Outputs["vision_detections"],
	)
	veMetric, ok := result.Metrics["vision_elements_detected"]
	assert.True(t, ok)
	assert.Equal(t, 1.0, veMetric.Value)

	// Confidence metric recorded.
	confMetric, ok := result.Metrics["vision_confidence_click learn more"]
	assert.True(t, ok)
	assert.InDelta(t, 0.91, confMetric.Value, 0.001)

	// Recording metrics present.
	vidDur, ok := result.Metrics["video_duration"]
	assert.True(t, ok)
	assert.Equal(t, 5.0, vidDur.Value)

	// Recording was started and stopped.
	assert.True(t, recorder.started)
	assert.True(t, recorder.stopped)
}

func TestRecordedVisionFlowChallenge_Execute_ZeroIntegrity(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	recorder.result = &RecordingResult{
		FilePath:   "/tmp/empty.webm",
		Duration:   0,
		FrameCount: 0,
		FileSize:   0,
	}
	vision := newMockVisionForRecording()
	flow := BrowserFlow{
		Name:     "zero-integrity",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedVisionFlowChallenge(
		"RECVIS-009", "Zero Integrity",
		"Recording with zero integrity",
		nil, browser, recorder, vision, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusFailed, result.Status,
	)

	// Integrity assertion should fail.
	hasIntegrity := false
	for _, a := range result.Assertions {
		if a.Target == "video_integrity" {
			hasIntegrity = true
			assert.False(t, a.Passed)
		}
	}
	assert.True(
		t, hasIntegrity,
		"missing video_integrity assertion",
	)
}
