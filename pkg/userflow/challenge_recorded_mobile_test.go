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

// mockMobileForRecording implements MobileAdapter for
// recorded mobile challenge tests.
type mockMobileForRecording struct {
	installErr  error
	launchErr   error
	stopErr     error
	tapErr      error
	sendKeysErr error
	pressKeyErr error
	waitErr     error
	closeErr    error

	running        bool
	runningErr     error
	screenshotData []byte
	screenshotErr  error

	testResults map[string]*TestResult
	testErrors  map[string]error

	installed bool
	launched  bool
	stopped   bool
	available bool

	tappedCoords [][2]int
	sentKeys     []string
	pressedKeys  []string
}

func newMockMobileForRecording() *mockMobileForRecording {
	return &mockMobileForRecording{
		available:      true,
		running:        true,
		screenshotData: []byte{0x89, 0x50, 0x4E, 0x47},
		testResults:    make(map[string]*TestResult),
		testErrors:     make(map[string]error),
	}
}

func (m *mockMobileForRecording) IsDeviceAvailable(
	_ context.Context,
) (bool, error) {
	return true, nil
}

func (m *mockMobileForRecording) InstallApp(
	_ context.Context, _ string,
) error {
	m.installed = true
	return m.installErr
}

func (m *mockMobileForRecording) LaunchApp(
	_ context.Context,
) error {
	m.launched = true
	return m.launchErr
}

func (m *mockMobileForRecording) StopApp(
	_ context.Context,
) error {
	m.stopped = true
	return m.stopErr
}

func (m *mockMobileForRecording) IsAppRunning(
	_ context.Context,
) (bool, error) {
	return m.running, m.runningErr
}

func (m *mockMobileForRecording) TakeScreenshot(
	_ context.Context,
) ([]byte, error) {
	return m.screenshotData, m.screenshotErr
}

func (m *mockMobileForRecording) Tap(
	_ context.Context, x, y int,
) error {
	m.tappedCoords = append(
		m.tappedCoords, [2]int{x, y},
	)
	return m.tapErr
}

func (m *mockMobileForRecording) SendKeys(
	_ context.Context, text string,
) error {
	m.sentKeys = append(m.sentKeys, text)
	return m.sendKeysErr
}

func (m *mockMobileForRecording) PressKey(
	_ context.Context, keycode string,
) error {
	m.pressedKeys = append(m.pressedKeys, keycode)
	return m.pressKeyErr
}

func (m *mockMobileForRecording) WaitForApp(
	_ context.Context, _ time.Duration,
) error {
	return m.waitErr
}

func (m *mockMobileForRecording) RunInstrumentedTests(
	_ context.Context, testClass string,
) (*TestResult, error) {
	if err, ok := m.testErrors[testClass]; ok {
		return nil, err
	}
	if r, ok := m.testResults[testClass]; ok {
		return r, nil
	}
	return &TestResult{TotalTests: 5}, nil
}

func (m *mockMobileForRecording) Close(
	_ context.Context,
) error {
	return m.closeErr
}

func (m *mockMobileForRecording) Available(
	_ context.Context,
) bool {
	return m.available
}

// --- RecordedMobileLaunchChallenge tests ---

func TestNewRecordedMobileLaunchChallenge_Constructor(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()
	deps := []challenge.ID{"SETUP-001", "HEALTH-001"}

	ch := NewRecordedMobileLaunchChallenge(
		"RMOB-001", "Recorded Launch",
		"Launch with recording",
		deps, adapter, recorder,
		"/tmp/app.apk",
		100*time.Millisecond,
	)

	assert.Equal(
		t, challenge.ID("RMOB-001"), ch.ID(),
	)
	assert.Equal(t, "Recorded Launch", ch.Name())
	assert.Equal(
		t, "Launch with recording",
		ch.Description(),
	)
	assert.Equal(t, deps, ch.Dependencies())
}

func TestRecordedMobileLaunchChallenge_Category(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()

	ch := NewRecordedMobileLaunchChallenge(
		"RMOB-002", "Category Test",
		"Check category",
		nil, adapter, recorder,
		"/tmp/app.apk",
		50*time.Millisecond,
	)

	assert.Equal(t, "mobile", ch.Category())
}

func TestRecordedMobileLaunchChallenge_Execute_Success(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()

	ch := NewRecordedMobileLaunchChallenge(
		"RMOB-003", "Recorded Launch",
		"Full recorded launch flow",
		nil, adapter, recorder,
		"/tmp/app.apk",
		50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusPassed, result.Status,
	)

	// Verify adapter interactions.
	assert.True(t, adapter.installed)
	assert.True(t, adapter.launched)
	assert.True(t, adapter.stopped)

	// Verify recorder interactions.
	assert.True(t, recorder.started)
	assert.True(t, recorder.stopped)
	assert.Equal(
		t, "mobile://launch", recorder.config.URL,
	)
	assert.Equal(
		t, "mobile", recorder.config.OutputDir,
	)

	// Verify recording metrics.
	vidDur, ok := result.Metrics["video_duration"]
	assert.True(t, ok)
	assert.Equal(t, 5.0, vidDur.Value)

	vidFrames, ok := result.Metrics["video_frame_count"]
	assert.True(t, ok)
	assert.Equal(t, 150.0, vidFrames.Value)

	vidSize, ok := result.Metrics["video_file_size"]
	assert.True(t, ok)
	assert.Equal(t, 1024000.0, vidSize.Value)

	// Verify launch duration metric.
	launchDur, ok := result.Metrics["launch_duration"]
	assert.True(t, ok)
	assert.Equal(t, "s", launchDur.Unit)

	// Verify outputs.
	assert.Equal(
		t, "/tmp/recording.webm",
		result.Outputs["video_path"],
	)
	assert.NotEmpty(
		t, result.Outputs["screenshot_size"],
	)

	// Verify assertions include recording start,
	// install, launch, stability, and integrity.
	hasRecStart := false
	hasRecIntegrity := false
	hasInstall := false
	hasLaunch := false
	hasStability := false
	for _, a := range result.Assertions {
		switch a.Target {
		case "start_recording":
			hasRecStart = true
			assert.True(t, a.Passed)
		case "video_integrity":
			hasRecIntegrity = true
			assert.True(t, a.Passed)
		case "app_install":
			hasInstall = true
			assert.True(t, a.Passed)
		case "app_launch":
			hasLaunch = true
			assert.True(t, a.Passed)
		case "app_stable":
			hasStability = true
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
	assert.True(
		t, hasInstall,
		"missing install assertion",
	)
	assert.True(
		t, hasLaunch,
		"missing launch assertion",
	)
	assert.True(
		t, hasStability,
		"missing stability assertion",
	)
}

func TestRecordedMobileLaunchChallenge_Execute_AdapterUnavailable(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	adapter.available = false
	recorder := newMockRecorderForFlow()

	ch := NewRecordedMobileLaunchChallenge(
		"RMOB-004", "Adapter Unavail",
		"Adapter not available",
		nil, adapter, recorder,
		"/tmp/app.apk",
		50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error,
		"mobile adapter not available",
	)

	// Recorder should not have been started.
	assert.False(t, recorder.started)
	assert.False(t, adapter.installed)
}

func TestRecordedMobileLaunchChallenge_Execute_RecorderUnavailable(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()
	recorder.available = false

	ch := NewRecordedMobileLaunchChallenge(
		"RMOB-005", "Recorder Unavail",
		"Recorder not available",
		nil, adapter, recorder,
		"/tmp/app.apk",
		50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error, "recorder not available",
	)

	// Adapter should not have been used.
	assert.False(t, adapter.installed)
	assert.False(t, recorder.started)
}

func TestRecordedMobileLaunchChallenge_Execute_RecordingStartFailure(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()
	recorder.startErr = fmt.Errorf("ffmpeg not installed")

	ch := NewRecordedMobileLaunchChallenge(
		"RMOB-006", "Rec Start Fail",
		"Recording start fails",
		nil, adapter, recorder,
		"/tmp/app.apk",
		50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusFailed, result.Status,
	)
	assert.Contains(
		t, result.Error, "ffmpeg not installed",
	)

	// App should not have been installed.
	assert.False(t, adapter.installed)
}

func TestRecordedMobileLaunchChallenge_Execute_ZeroIntegrity(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()
	recorder.result = &RecordingResult{
		FilePath:   "/tmp/empty.webm",
		Duration:   0,
		FrameCount: 0,
		FileSize:   0,
	}

	ch := NewRecordedMobileLaunchChallenge(
		"RMOB-007", "Zero Integrity",
		"Recording with zero integrity",
		nil, adapter, recorder,
		"/tmp/app.apk",
		50*time.Millisecond,
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

// --- RecordedMobileFlowChallenge tests ---

func TestNewRecordedMobileFlowChallenge_Constructor(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()
	flow := MobileFlow{
		Name: "test-flow",
		Steps: []MobileStep{
			{Name: "tap", Action: "tap",
				X: 100, Y: 200},
		},
	}
	deps := []challenge.ID{"SETUP-001"}

	ch := NewRecordedMobileFlowChallenge(
		"RMOB-FLOW-001", "Recorded Flow",
		"Recorded mobile flow test",
		deps, adapter, recorder, flow,
	)

	assert.Equal(
		t, challenge.ID("RMOB-FLOW-001"), ch.ID(),
	)
	assert.Equal(t, "Recorded Flow", ch.Name())
	assert.Equal(
		t, "Recorded mobile flow test",
		ch.Description(),
	)
	assert.Equal(t, deps, ch.Dependencies())
}

func TestRecordedMobileFlowChallenge_Category(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()
	flow := MobileFlow{Name: "cat-flow"}

	ch := NewRecordedMobileFlowChallenge(
		"RMOB-FLOW-002", "Category Test",
		"Check category",
		nil, adapter, recorder, flow,
	)

	assert.Equal(t, "mobile", ch.Category())
}

func TestRecordedMobileFlowChallenge_Execute_Success(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()
	flow := MobileFlow{
		Name: "login-flow",
		Steps: []MobileStep{
			{Name: "launch", Action: "launch"},
			{Name: "tap login", Action: "tap",
				X: 540, Y: 960},
			{Name: "enter user", Action: "send_keys",
				Value: "admin"},
			{Name: "press back", Action: "press_key",
				Value: "KEYCODE_BACK"},
			{Name: "wait", Action: "wait"},
			{Name: "check running",
				Action: "assert_running"},
			{Name: "screenshot",
				Action: "screenshot"},
			{Name: "stop", Action: "stop"},
		},
	}

	ch := NewRecordedMobileFlowChallenge(
		"RMOB-FLOW-003", "Recorded Login",
		"Login with recording",
		nil, adapter, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusPassed, result.Status,
	)

	// Verify adapter interactions.
	assert.True(t, adapter.launched)
	require.Len(t, adapter.tappedCoords, 1)
	assert.Equal(
		t, [2]int{540, 960},
		adapter.tappedCoords[0],
	)
	require.Len(t, adapter.sentKeys, 1)
	assert.Equal(t, "admin", adapter.sentKeys[0])
	require.Len(t, adapter.pressedKeys, 1)
	assert.Equal(
		t, "KEYCODE_BACK", adapter.pressedKeys[0],
	)

	// Verify recorder interactions.
	assert.True(t, recorder.started)
	assert.True(t, recorder.stopped)
	assert.Equal(
		t, "mobile://flow", recorder.config.URL,
	)
	assert.Equal(
		t, "mobile", recorder.config.OutputDir,
	)

	// Verify recording metrics.
	vidDur, ok := result.Metrics["video_duration"]
	assert.True(t, ok)
	assert.Equal(t, 5.0, vidDur.Value)

	vidFrames, ok := result.Metrics["video_frame_count"]
	assert.True(t, ok)
	assert.Equal(t, 150.0, vidFrames.Value)

	vidSize, ok := result.Metrics["video_file_size"]
	assert.True(t, ok)
	assert.Equal(t, 1024000.0, vidSize.Value)

	// Verify standard metrics.
	steps := result.Metrics["steps_executed"]
	assert.Equal(t, 8.0, steps.Value)

	// Verify outputs.
	assert.Equal(
		t, "/tmp/recording.webm",
		result.Outputs["video_path"],
	)

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

func TestRecordedMobileFlowChallenge_Execute_AdapterUnavailable(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	adapter.available = false
	recorder := newMockRecorderForFlow()
	flow := MobileFlow{
		Name: "unavail-adapter",
		Steps: []MobileStep{
			{Name: "tap", Action: "tap",
				X: 100, Y: 200},
		},
	}

	ch := NewRecordedMobileFlowChallenge(
		"RMOB-FLOW-004", "Adapter Unavail",
		"Adapter not available",
		nil, adapter, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error,
		"mobile adapter not available",
	)

	// Recorder should not have been started.
	assert.False(t, recorder.started)
}

func TestRecordedMobileFlowChallenge_Execute_RecorderUnavailable(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	recorder := newMockRecorderForFlow()
	recorder.available = false
	flow := MobileFlow{
		Name: "unavail-recorder",
		Steps: []MobileStep{
			{Name: "tap", Action: "tap",
				X: 100, Y: 200},
		},
	}

	ch := NewRecordedMobileFlowChallenge(
		"RMOB-FLOW-005", "Recorder Unavail",
		"Recorder not available",
		nil, adapter, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error, "recorder not available",
	)

	// Adapter should not have been used.
	assert.False(t, recorder.started)
}

func TestRecordedMobileFlowChallenge_Execute_StepFailure(
	t *testing.T,
) {
	adapter := newMockMobileForRecording()
	adapter.tapErr = fmt.Errorf("device disconnected")
	recorder := newMockRecorderForFlow()
	flow := MobileFlow{
		Name: "step-fail",
		Steps: []MobileStep{
			{Name: "tap btn", Action: "tap",
				X: 100, Y: 200},
		},
	}

	ch := NewRecordedMobileFlowChallenge(
		"RMOB-FLOW-006", "Step Fail",
		"Step fails but recording continues",
		nil, adapter, recorder, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusFailed, result.Status,
	)

	// Recording should still have been stopped.
	assert.True(t, recorder.stopped)

	// Recording metrics should still be present.
	_, ok := result.Metrics["video_duration"]
	assert.True(t, ok)

	// Step failure message should be present.
	assert.Contains(
		t, result.Assertions[1].Message,
		"device disconnected",
	)
}
