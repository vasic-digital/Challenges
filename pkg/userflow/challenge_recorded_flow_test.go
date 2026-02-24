package userflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

// mockBrowserForRecording implements BrowserAdapter for
// recorded flow tests.
type mockBrowserForRecording struct {
	initialized bool
	navigated   string
	clicked     []string
	filled      map[string]string
	closed      bool
	available   bool
	screenshot  []byte

	initErr     error
	navigateErr error
	clickErr    error
	fillErr     error

	visibleResults  map[string]bool
	textResults     map[string]string
	evaluateResults map[string]string
}

func newMockBrowserForRecording() *mockBrowserForRecording {
	return &mockBrowserForRecording{
		available:       true,
		filled:          make(map[string]string),
		screenshot:      []byte{0x89, 0x50, 0x4E, 0x47},
		visibleResults:  make(map[string]bool),
		textResults:     make(map[string]string),
		evaluateResults: make(map[string]string),
	}
}

func (m *mockBrowserForRecording) Initialize(
	_ context.Context, _ BrowserConfig,
) error {
	m.initialized = true
	return m.initErr
}

func (m *mockBrowserForRecording) Navigate(
	_ context.Context, url string,
) error {
	m.navigated = url
	return m.navigateErr
}

func (m *mockBrowserForRecording) Click(
	_ context.Context, selector string,
) error {
	m.clicked = append(m.clicked, selector)
	return m.clickErr
}

func (m *mockBrowserForRecording) Fill(
	_ context.Context, selector, value string,
) error {
	m.filled[selector] = value
	return m.fillErr
}

func (m *mockBrowserForRecording) SelectOption(
	_ context.Context, _, _ string,
) error {
	return nil
}

func (m *mockBrowserForRecording) IsVisible(
	_ context.Context, selector string,
) (bool, error) {
	if v, ok := m.visibleResults[selector]; ok {
		return v, nil
	}
	return true, nil
}

func (m *mockBrowserForRecording) WaitForSelector(
	_ context.Context, _ string, _ time.Duration,
) error {
	return nil
}

func (m *mockBrowserForRecording) GetText(
	_ context.Context, selector string,
) (string, error) {
	if t, ok := m.textResults[selector]; ok {
		return t, nil
	}
	return "", nil
}

func (m *mockBrowserForRecording) GetAttribute(
	_ context.Context, _, _ string,
) (string, error) {
	return "", nil
}

func (m *mockBrowserForRecording) Screenshot(
	_ context.Context,
) ([]byte, error) {
	return m.screenshot, nil
}

func (m *mockBrowserForRecording) EvaluateJS(
	_ context.Context, script string,
) (string, error) {
	if r, ok := m.evaluateResults[script]; ok {
		return r, nil
	}
	return "", nil
}

func (m *mockBrowserForRecording) NetworkIntercept(
	_ context.Context,
	_ string,
	_ func(req *InterceptedRequest),
) error {
	return nil
}

func (m *mockBrowserForRecording) Close(
	_ context.Context,
) error {
	m.closed = true
	return nil
}

func (m *mockBrowserForRecording) Available(
	_ context.Context,
) bool {
	return m.available
}

// mockRecorderForFlow implements RecorderAdapter for
// recorded flow tests.
type mockRecorderForFlow struct {
	started   bool
	stopped   bool
	available bool
	config    RecordingConfig
	result    *RecordingResult
	startErr  error
	stopErr   error
	recording bool
}

func newMockRecorderForFlow() *mockRecorderForFlow {
	return &mockRecorderForFlow{
		available: true,
		result: &RecordingResult{
			FilePath:   "/tmp/recording.webm",
			Duration:   5 * time.Second,
			FrameCount: 150,
			FileSize:   1024000,
		},
	}
}

func (m *mockRecorderForFlow) StartRecording(
	_ context.Context, config RecordingConfig,
) error {
	m.config = config
	m.started = true
	m.recording = true
	return m.startErr
}

func (m *mockRecorderForFlow) StopRecording(
	_ context.Context,
) (*RecordingResult, error) {
	m.stopped = true
	m.recording = false
	return m.result, m.stopErr
}

func (m *mockRecorderForFlow) IsRecording() bool {
	return m.recording
}

func (m *mockRecorderForFlow) Available(
	_ context.Context,
) bool {
	return m.available
}

// --- RecordedBrowserFlowChallenge tests ---

func TestNewRecordedBrowserFlowChallenge_Constructor(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	flow := BrowserFlow{
		Name:     "test-flow",
		StartURL: "http://localhost:3000",
		Config: BrowserConfig{
			BrowserType: "chromium",
			Headless:    true,
		},
	}
	deps := []challenge.ID{"SETUP-001", "HEALTH-001"}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-001", "Recorded Flow",
		"Recorded browser flow test",
		deps, browser, recorder, flow,
	)

	assert.Equal(
		t, challenge.ID("REC-001"), ch.ID(),
	)
	assert.Equal(t, "Recorded Flow", ch.Name())
	assert.Equal(
		t, "Recorded browser flow test",
		ch.Description(),
	)
	assert.Equal(t, deps, ch.Dependencies())
}

func TestRecordedBrowserFlowChallenge_Category(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	flow := BrowserFlow{
		Name:     "cat-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-002", "Category Test", "Check category",
		nil, browser, recorder, flow,
	)

	assert.Equal(t, "browser", ch.Category())
}

func TestRecordedBrowserFlowChallenge_Execute_Success(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	flow := BrowserFlow{
		Name:     "login-flow",
		StartURL: "http://localhost:3000/login",
		Config: BrowserConfig{
			BrowserType: "chromium",
			Headless:    true,
		},
		Steps: []BrowserStep{
			{
				Name:     "fill username",
				Action:   "fill",
				Selector: "#username",
				Value:    "admin",
			},
			{
				Name:     "fill password",
				Action:   "fill",
				Selector: "#password",
				Value:    "secret",
			},
			{
				Name:     "click login",
				Action:   "click",
				Selector: "#login-btn",
			},
		},
	}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-003", "Recorded Login",
		"Login with recording",
		nil, browser, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)

	// Verify browser interactions.
	assert.True(t, browser.initialized)
	assert.Equal(
		t, "http://localhost:3000/login",
		browser.navigated,
	)
	assert.Equal(t, "admin", browser.filled["#username"])
	assert.Equal(t, "secret", browser.filled["#password"])
	require.Len(t, browser.clicked, 1)
	assert.Equal(t, "#login-btn", browser.clicked[0])
	assert.True(t, browser.closed)

	// Verify recorder interactions.
	assert.True(t, recorder.started)
	assert.True(t, recorder.stopped)
	assert.Equal(
		t, "http://localhost:3000/login",
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

	// Verify standard metrics.
	_, ok = result.Metrics["total_duration"]
	assert.True(t, ok)
	steps := result.Metrics["steps_executed"]
	assert.Equal(t, 3.0, steps.Value)

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

func TestRecordedBrowserFlowChallenge_Execute_BrowserUnavailable(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	browser.available = false
	recorder := newMockRecorderForFlow()
	flow := BrowserFlow{
		Name:     "unavail-browser",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-004", "Browser Unavail",
		"Browser not available",
		nil, browser, recorder, flow,
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

func TestRecordedBrowserFlowChallenge_Execute_RecorderUnavailable(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	recorder.available = false
	flow := BrowserFlow{
		Name:     "unavail-recorder",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-005", "Recorder Unavail",
		"Recorder not available",
		nil, browser, recorder, flow,
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

func TestRecordedBrowserFlowChallenge_Execute_InitFailure(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	browser.initErr = fmt.Errorf("chromium not found")
	recorder := newMockRecorderForFlow()
	flow := BrowserFlow{
		Name:     "init-fail",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-006", "Init Fail",
		"Browser init fails",
		nil, browser, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.Contains(
		t, result.Assertions[0].Message,
		"chromium not found",
	)

	// Recorder should not have been started.
	assert.False(t, recorder.started)
}

func TestRecordedBrowserFlowChallenge_Execute_RecordingStartFailure(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	recorder.startErr = fmt.Errorf("ffmpeg not installed")
	flow := BrowserFlow{
		Name:     "rec-start-fail",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-007", "Rec Start Fail",
		"Recording start fails",
		nil, browser, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Error, "ffmpeg not installed",
	)
}

func TestRecordedBrowserFlowChallenge_Execute_StepFailure(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	browser.clickErr = fmt.Errorf("element not found")
	recorder := newMockRecorderForFlow()
	flow := BrowserFlow{
		Name:     "step-fail",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:     "click missing",
				Action:   "click",
				Selector: "#missing",
			},
		},
	}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-008", "Step Fail",
		"Step fails but recording continues",
		nil, browser, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)

	// Recording should still have been stopped.
	assert.True(t, recorder.stopped)

	// Recording metrics should still be present.
	_, ok := result.Metrics["video_duration"]
	assert.True(t, ok)
}

func TestRecordedBrowserFlowChallenge_Execute_RecordingStopFailure(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	recorder.result = nil
	recorder.stopErr = fmt.Errorf("recording corrupted")
	flow := BrowserFlow{
		Name:     "rec-stop-fail",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-009", "Rec Stop Fail",
		"Recording stop fails",
		nil, browser, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Error, "recording corrupted",
	)

	// Should have a video_recorded assertion that failed.
	hasRecorded := false
	for _, a := range result.Assertions {
		if a.Target == "video_recorded" {
			hasRecorded = true
			assert.False(t, a.Passed)
		}
	}
	assert.True(
		t, hasRecorded,
		"missing video_recorded assertion",
	)
}

func TestRecordedBrowserFlowChallenge_Execute_ZeroIntegrity(
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
	flow := BrowserFlow{
		Name:     "zero-integrity",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewRecordedBrowserFlowChallenge(
		"REC-010", "Zero Integrity",
		"Recording with zero integrity",
		nil, browser, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)

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
