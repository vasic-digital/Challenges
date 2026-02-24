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

// mockMobileAdapter implements MobileAdapter for testing.
type mockMobileAdapter struct {
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

	tappedCoords [][2]int
	sentKeys     []string
	pressedKeys  []string
}

func newMockMobileAdapter() *mockMobileAdapter {
	return &mockMobileAdapter{
		running:        true,
		screenshotData: []byte{0x89, 0x50, 0x4E, 0x47},
		testResults:    make(map[string]*TestResult),
		testErrors:     make(map[string]error),
	}
}

func (m *mockMobileAdapter) IsDeviceAvailable(
	_ context.Context,
) (bool, error) {
	return true, nil
}

func (m *mockMobileAdapter) InstallApp(
	_ context.Context, _ string,
) error {
	m.installed = true
	return m.installErr
}

func (m *mockMobileAdapter) LaunchApp(
	_ context.Context,
) error {
	m.launched = true
	return m.launchErr
}

func (m *mockMobileAdapter) StopApp(
	_ context.Context,
) error {
	m.stopped = true
	return m.stopErr
}

func (m *mockMobileAdapter) IsAppRunning(
	_ context.Context,
) (bool, error) {
	return m.running, m.runningErr
}

func (m *mockMobileAdapter) TakeScreenshot(
	_ context.Context,
) ([]byte, error) {
	return m.screenshotData, m.screenshotErr
}

func (m *mockMobileAdapter) Tap(
	_ context.Context, x, y int,
) error {
	m.tappedCoords = append(m.tappedCoords, [2]int{x, y})
	return m.tapErr
}

func (m *mockMobileAdapter) SendKeys(
	_ context.Context, text string,
) error {
	m.sentKeys = append(m.sentKeys, text)
	return m.sendKeysErr
}

func (m *mockMobileAdapter) PressKey(
	_ context.Context, keycode string,
) error {
	m.pressedKeys = append(m.pressedKeys, keycode)
	return m.pressKeyErr
}

func (m *mockMobileAdapter) WaitForApp(
	_ context.Context, _ time.Duration,
) error {
	return m.waitErr
}

func (m *mockMobileAdapter) RunInstrumentedTests(
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

func (m *mockMobileAdapter) Close(
	_ context.Context,
) error {
	return m.closeErr
}

func (m *mockMobileAdapter) Available(
	_ context.Context,
) bool {
	return true
}

// --- MobileLaunchChallenge tests ---

func TestNewMobileLaunchChallenge(t *testing.T) {
	adapter := newMockMobileAdapter()
	ch := NewMobileLaunchChallenge(
		"MOB-001", "Launch App", "Launch and verify",
		nil, adapter, "/tmp/app.apk",
		100*time.Millisecond,
	)

	assert.Equal(
		t, challenge.ID("MOB-001"), ch.ID(),
	)
	assert.Equal(t, "Launch App", ch.Name())
	assert.Equal(t, "mobile", ch.Category())
}

func TestMobileLaunchChallenge_Execute_Success(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	ch := NewMobileLaunchChallenge(
		"MOB-002", "Launch", "Launch app",
		nil, adapter, "/tmp/app.apk",
		50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)

	// install + launch + stability.
	require.Len(t, result.Assertions, 3)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed, "failed: %s", a.Message)
	}

	assert.True(t, adapter.installed)
	assert.True(t, adapter.launched)
	assert.True(t, adapter.stopped)

	dur, ok := result.Metrics["launch_duration"]
	require.True(t, ok)
	assert.Equal(t, "s", dur.Unit)
}

func TestMobileLaunchChallenge_Execute_InstallFails(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	adapter.installErr = fmt.Errorf("APK not found")

	ch := NewMobileLaunchChallenge(
		"MOB-003", "Launch", "Install fails",
		nil, adapter, "/tmp/missing.apk",
		50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Message, "APK not found",
	)
}

func TestMobileLaunchChallenge_Execute_LaunchFails(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	adapter.launchErr = fmt.Errorf("activity not found")

	ch := NewMobileLaunchChallenge(
		"MOB-004", "Launch", "Launch fails",
		nil, adapter, "/tmp/app.apk",
		50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 2)
	assert.True(t, result.Assertions[0].Passed)
	assert.False(t, result.Assertions[1].Passed)
}

func TestMobileLaunchChallenge_Execute_AppCrashes(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	adapter.running = false

	ch := NewMobileLaunchChallenge(
		"MOB-005", "Launch", "App crashes",
		nil, adapter, "/tmp/app.apk",
		50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 3)
	assert.False(t, result.Assertions[2].Passed)
	assert.Contains(
		t, result.Assertions[2].Message, "crashed",
	)
}

// --- MobileFlowChallenge tests ---

func TestNewMobileFlowChallenge(t *testing.T) {
	adapter := newMockMobileAdapter()
	flow := MobileFlow{
		Name: "test-flow",
		Steps: []MobileStep{
			{Name: "tap", Action: "tap", X: 100, Y: 200},
		},
	}
	ch := NewMobileFlowChallenge(
		"MOB-FLOW-001", "Flow", "Mobile flow",
		nil, adapter, flow,
	)

	assert.Equal(
		t, challenge.ID("MOB-FLOW-001"), ch.ID(),
	)
	assert.Equal(t, "mobile", ch.Category())
}

func TestMobileFlowChallenge_Execute_FullFlow(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
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
			{Name: "screenshot", Action: "screenshot"},
			{Name: "stop", Action: "stop"},
		},
	}

	ch := NewMobileFlowChallenge(
		"MOB-FLOW-002", "Login", "Login flow",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 8)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed, "failed: %s", a.Message)
	}

	assert.True(t, adapter.launched)
	require.Len(t, adapter.tappedCoords, 1)
	assert.Equal(t, [2]int{540, 960}, adapter.tappedCoords[0])
	require.Len(t, adapter.sentKeys, 1)
	assert.Equal(t, "admin", adapter.sentKeys[0])
	require.Len(t, adapter.pressedKeys, 1)
	assert.Equal(
		t, "KEYCODE_BACK", adapter.pressedKeys[0],
	)

	steps := result.Metrics["steps_executed"]
	assert.Equal(t, 8.0, steps.Value)
}

func TestMobileFlowChallenge_Execute_StepFails(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	adapter.tapErr = fmt.Errorf("device disconnected")

	flow := MobileFlow{
		Name: "fail-flow",
		Steps: []MobileStep{
			{Name: "tap btn", Action: "tap",
				X: 100, Y: 200},
		},
	}

	ch := NewMobileFlowChallenge(
		"MOB-FLOW-003", "Fail", "Step fails",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message,
		"device disconnected",
	)
}

func TestMobileFlowChallenge_Execute_UnknownAction(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	flow := MobileFlow{
		Name: "unknown-flow",
		Steps: []MobileStep{
			{Name: "swipe", Action: "swipe"},
		},
	}

	ch := NewMobileFlowChallenge(
		"MOB-FLOW-004", "Unknown", "Unknown action",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message,
		"unknown mobile action",
	)
}

func TestMobileFlowChallenge_Execute_AssertRunningFails(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	adapter.running = false

	flow := MobileFlow{
		Name: "assert-fail",
		Steps: []MobileStep{
			{Name: "check", Action: "assert_running"},
		},
	}

	ch := NewMobileFlowChallenge(
		"MOB-FLOW-005", "Assert", "Assert fails",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message, "not running",
	)
}

func TestMobileFlowChallenge_Execute_StepAssertions(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	flow := MobileFlow{
		Name: "step-assert",
		Steps: []MobileStep{
			{
				Name: "screenshot", Action: "screenshot",
				Assertions: []StepAssertion{
					{
						Type:    "screenshot_exists",
						Target:  "screen",
						Message: "should exist",
					},
				},
			},
		},
	}

	ch := NewMobileFlowChallenge(
		"MOB-FLOW-006", "Assert", "Step assertions",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	// 1 step + 1 step assertion.
	require.Len(t, result.Assertions, 2)
}

// --- InstrumentedTestChallenge tests ---

func TestNewInstrumentedTestChallenge(t *testing.T) {
	adapter := newMockMobileAdapter()
	ch := NewInstrumentedTestChallenge(
		"INST-001", "Instrumented", "Run device tests",
		nil, adapter,
		[]string{"com.example.LoginTest"},
	)

	assert.Equal(
		t, challenge.ID("INST-001"), ch.ID(),
	)
	assert.Equal(t, "mobile", ch.Category())
}

func TestInstrumentedTestChallenge_Execute_AllPass(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	adapter.testResults["com.example.LoginTest"] = &TestResult{
		TotalTests: 8, TotalFailed: 0,
		TotalErrors: 0,
	}
	adapter.testResults["com.example.HomeTest"] = &TestResult{
		TotalTests: 12, TotalFailed: 0,
		TotalErrors: 0,
	}

	ch := NewInstrumentedTestChallenge(
		"INST-002", "Instrumented", "All pass",
		nil, adapter,
		[]string{
			"com.example.LoginTest",
			"com.example.HomeTest",
		},
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 2)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed)
	}

	total := result.Metrics["total_tests"]
	assert.Equal(t, 20.0, total.Value)
	failures := result.Metrics["total_failures"]
	assert.Equal(t, 0.0, failures.Value)
}

func TestInstrumentedTestChallenge_Execute_SomeFail(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	adapter.testResults["com.example.LoginTest"] = &TestResult{
		TotalTests: 8, TotalFailed: 2,
		TotalErrors: 0,
	}

	ch := NewInstrumentedTestChallenge(
		"INST-003", "Instrumented", "Some fail",
		nil, adapter,
		[]string{"com.example.LoginTest"},
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Actual, "2 failures",
	)
}

func TestInstrumentedTestChallenge_Execute_Error(
	t *testing.T,
) {
	adapter := newMockMobileAdapter()
	adapter.testErrors["com.example.Test"] = fmt.Errorf(
		"device disconnected",
	)

	ch := NewInstrumentedTestChallenge(
		"INST-004", "Instrumented", "Error",
		nil, adapter,
		[]string{"com.example.Test"},
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message,
		"device disconnected",
	)
}
