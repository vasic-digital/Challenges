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

// mockDesktopAdapter implements DesktopAdapter for testing.
type mockDesktopAdapter struct {
	launchErr   error
	navigateErr error
	clickErr    error
	fillErr     error
	waitSelErr  error
	waitWinErr  error
	closeErr    error

	running    bool
	runningErr error

	visibleResults map[string]bool

	screenshotData  []byte
	screenshotErr   error
	screenshotCount int

	commandResults map[string]string
	commandErrors  map[string]error

	navigatedURLs []string
	clickedSels   []string
	closed        bool
}

func newMockDesktopAdapter() *mockDesktopAdapter {
	return &mockDesktopAdapter{
		running:        true,
		visibleResults: make(map[string]bool),
		screenshotData: []byte{0x89, 0x50, 0x4E, 0x47},
		commandResults: make(map[string]string),
		commandErrors:  make(map[string]error),
	}
}

func (m *mockDesktopAdapter) LaunchApp(
	_ context.Context, _ DesktopAppConfig,
) error {
	return m.launchErr
}

func (m *mockDesktopAdapter) IsAppRunning(
	_ context.Context,
) (bool, error) {
	return m.running, m.runningErr
}

func (m *mockDesktopAdapter) Navigate(
	_ context.Context, url string,
) error {
	m.navigatedURLs = append(m.navigatedURLs, url)
	return m.navigateErr
}

func (m *mockDesktopAdapter) Click(
	_ context.Context, selector string,
) error {
	m.clickedSels = append(m.clickedSels, selector)
	return m.clickErr
}

func (m *mockDesktopAdapter) Fill(
	_ context.Context, _, _ string,
) error {
	return m.fillErr
}

func (m *mockDesktopAdapter) IsVisible(
	_ context.Context, selector string,
) (bool, error) {
	if v, ok := m.visibleResults[selector]; ok {
		return v, nil
	}
	return true, nil
}

func (m *mockDesktopAdapter) WaitForSelector(
	_ context.Context, _ string, _ time.Duration,
) error {
	return m.waitSelErr
}

func (m *mockDesktopAdapter) Screenshot(
	_ context.Context,
) ([]byte, error) {
	m.screenshotCount++
	return m.screenshotData, m.screenshotErr
}

func (m *mockDesktopAdapter) InvokeCommand(
	_ context.Context, command string, _ ...string,
) (string, error) {
	if err, ok := m.commandErrors[command]; ok {
		return "", err
	}
	if r, ok := m.commandResults[command]; ok {
		return r, nil
	}
	return "", nil
}

func (m *mockDesktopAdapter) WaitForWindow(
	_ context.Context, _ time.Duration,
) error {
	return m.waitWinErr
}

func (m *mockDesktopAdapter) Close(
	_ context.Context,
) error {
	m.closed = true
	return m.closeErr
}

func (m *mockDesktopAdapter) Available(
	_ context.Context,
) bool {
	return true
}

// --- DesktopLaunchChallenge tests ---

func TestNewDesktopLaunchChallenge(t *testing.T) {
	adapter := newMockDesktopAdapter()
	config := DesktopAppConfig{
		BinaryPath: "/usr/bin/app",
	}
	ch := NewDesktopLaunchChallenge(
		"DESK-001", "Launch", "Launch desktop app",
		nil, adapter, config, 100*time.Millisecond,
	)

	assert.Equal(
		t, challenge.ID("DESK-001"), ch.ID(),
	)
	assert.Equal(t, "Launch", ch.Name())
	assert.Equal(t, "desktop", ch.Category())
}

func TestDesktopLaunchChallenge_Execute_Success(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	config := DesktopAppConfig{
		BinaryPath: "/usr/bin/app",
	}
	ch := NewDesktopLaunchChallenge(
		"DESK-002", "Launch", "Launch app",
		nil, adapter, config, 50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)

	// launch + window + stability.
	require.Len(t, result.Assertions, 3)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed, "failed: %s", a.Message)
	}
	assert.True(t, adapter.closed)

	dur, ok := result.Metrics["launch_duration"]
	require.True(t, ok)
	assert.Equal(t, "s", dur.Unit)
}

func TestDesktopLaunchChallenge_Execute_LaunchFails(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.launchErr = fmt.Errorf("binary not found")
	config := DesktopAppConfig{
		BinaryPath: "/usr/bin/missing",
	}

	ch := NewDesktopLaunchChallenge(
		"DESK-003", "Launch", "Launch fails",
		nil, adapter, config, 50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.Contains(
		t, result.Assertions[0].Message,
		"binary not found",
	)
}

func TestDesktopLaunchChallenge_Execute_WindowFails(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.waitWinErr = fmt.Errorf("window timeout")
	config := DesktopAppConfig{
		BinaryPath: "/usr/bin/app",
	}

	ch := NewDesktopLaunchChallenge(
		"DESK-004", "Launch", "Window fails",
		nil, adapter, config, 50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 2)
	assert.True(t, result.Assertions[0].Passed)
	assert.False(t, result.Assertions[1].Passed)
}

func TestDesktopLaunchChallenge_Execute_AppCrashes(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.running = false
	config := DesktopAppConfig{
		BinaryPath: "/usr/bin/app",
	}

	ch := NewDesktopLaunchChallenge(
		"DESK-005", "Launch", "App crashes",
		nil, adapter, config, 50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[2].Message, "crashed",
	)
}

// --- DesktopFlowChallenge tests ---

func TestNewDesktopFlowChallenge(t *testing.T) {
	adapter := newMockDesktopAdapter()
	flow := BrowserFlow{
		Name:     "desktop-flow",
		StartURL: "http://localhost:3000",
	}
	ch := NewDesktopFlowChallenge(
		"DESK-FLOW-001", "Flow", "Desktop flow",
		nil, adapter, flow,
	)

	assert.Equal(
		t, challenge.ID("DESK-FLOW-001"), ch.ID(),
	)
	assert.Equal(t, "desktop", ch.Category())
}

func TestDesktopFlowChallenge_Execute_SimpleFlow(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	flow := BrowserFlow{
		Name:     "nav-flow",
		StartURL: "http://localhost:3000",
		Steps: []BrowserStep{
			{
				Name:     "click menu",
				Action:   "click",
				Selector: "#menu",
			},
			{
				Name:   "go to settings",
				Action: "navigate",
				Value:  "http://localhost:3000/settings",
			},
		},
	}

	ch := NewDesktopFlowChallenge(
		"DESK-FLOW-002", "Flow", "Simple flow",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 2)

	// Start URL + navigate step.
	require.Len(t, adapter.navigatedURLs, 2)
	assert.Equal(
		t, "http://localhost:3000",
		adapter.navigatedURLs[0],
	)
	assert.Equal(
		t, "http://localhost:3000/settings",
		adapter.navigatedURLs[1],
	)
	require.Len(t, adapter.clickedSels, 1)
	assert.Equal(t, "#menu", adapter.clickedSels[0])
}

func TestDesktopFlowChallenge_Execute_NavigateFails(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.navigateErr = fmt.Errorf("webview error")
	flow := BrowserFlow{
		Name:     "fail-nav",
		StartURL: "http://localhost:3000",
	}

	ch := NewDesktopFlowChallenge(
		"DESK-FLOW-003", "Flow", "Navigate fails",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(t, result.Error, "webview error")
}

func TestDesktopFlowChallenge_Execute_AssertVisible(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.visibleResults["#sidebar"] = true

	flow := BrowserFlow{
		Name:     "visible-flow",
		StartURL: "http://localhost:3000",
		Steps: []BrowserStep{
			{
				Name:     "check sidebar",
				Action:   "assert_visible",
				Selector: "#sidebar",
			},
		},
	}

	ch := NewDesktopFlowChallenge(
		"DESK-FLOW-004", "Flow", "Assert visible",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestDesktopFlowChallenge_Execute_ScreenshotStep(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()

	flow := BrowserFlow{
		Name:     "screenshot-flow",
		StartURL: "http://localhost:3000",
		Steps: []BrowserStep{
			{
				Name:   "capture",
				Action: "screenshot",
			},
			{
				Name:       "click and capture",
				Action:     "click",
				Selector:   "#btn",
				Screenshot: true,
			},
		},
	}

	ch := NewDesktopFlowChallenge(
		"DESK-FLOW-005", "Flow", "Screenshots",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Equal(t, 2, adapter.screenshotCount)
	assert.Equal(t, "2", result.Outputs["screenshot_count"])
}

func TestDesktopFlowChallenge_Execute_UnsupportedAction(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()

	flow := BrowserFlow{
		Name:     "unsupported",
		StartURL: "http://localhost:3000",
		Steps: []BrowserStep{
			{
				Name:   "evaluate js",
				Action: "evaluate_js",
				Value:  "document.title",
			},
		},
	}

	ch := NewDesktopFlowChallenge(
		"DESK-FLOW-006", "Flow", "Unsupported action",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message,
		"unsupported desktop action",
	)
}

func TestDesktopFlowChallenge_Execute_WaitAction(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()

	flow := BrowserFlow{
		Name:     "wait-flow",
		StartURL: "http://localhost:3000",
		Steps: []BrowserStep{
			{
				Name:     "wait for loaded",
				Action:   "wait",
				Selector: "#loaded",
				Timeout:  2 * time.Second,
			},
		},
	}

	ch := NewDesktopFlowChallenge(
		"DESK-FLOW-007", "Flow", "Wait action",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestDesktopFlowChallenge_Execute_FillAction(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()

	flow := BrowserFlow{
		Name:     "fill-flow",
		StartURL: "http://localhost:3000",
		Steps: []BrowserStep{
			{
				Name:     "fill input",
				Action:   "fill",
				Selector: "#search",
				Value:    "query",
			},
		},
	}

	ch := NewDesktopFlowChallenge(
		"DESK-FLOW-008", "Flow", "Fill action",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestDesktopFlowChallenge_Execute_Metrics(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()

	flow := BrowserFlow{
		Name:     "metrics-flow",
		StartURL: "http://localhost:3000",
		Steps: []BrowserStep{
			{Name: "a", Action: "click", Selector: "#a"},
			{Name: "b", Action: "click", Selector: "#b"},
		},
	}

	ch := NewDesktopFlowChallenge(
		"DESK-FLOW-009", "Flow", "Metrics",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)

	_, ok := result.Metrics["total_duration"]
	assert.True(t, ok)
	steps := result.Metrics["steps_executed"]
	assert.Equal(t, 2.0, steps.Value)
}

// --- DesktopIPCChallenge tests ---

func TestNewDesktopIPCChallenge(t *testing.T) {
	adapter := newMockDesktopAdapter()
	commands := []IPCCommand{
		{Name: "get-config", Command: "get_config"},
	}
	ch := NewDesktopIPCChallenge(
		"IPC-001", "IPC", "Test IPC",
		nil, adapter, commands,
	)

	assert.Equal(
		t, challenge.ID("IPC-001"), ch.ID(),
	)
	assert.Equal(t, "desktop", ch.Category())
}

func TestDesktopIPCChallenge_Execute_AllPass(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.commandResults["get_config"] =
		`{"theme":"dark","lang":"en"}`
	adapter.commandResults["get_version"] = "1.2.3"

	commands := []IPCCommand{
		{
			Name:           "get-config",
			Command:        "get_config",
			ExpectedResult: `"theme":"dark"`,
		},
		{
			Name:           "get-version",
			Command:        "get_version",
			ExpectedResult: "1.2.3",
		},
	}

	ch := NewDesktopIPCChallenge(
		"IPC-002", "IPC", "All pass",
		nil, adapter, commands,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 2)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed, "failed: %s", a.Message)
	}

	cmdMetric := result.Metrics["commands_executed"]
	assert.Equal(t, 2.0, cmdMetric.Value)

	assert.Equal(
		t, `{"theme":"dark","lang":"en"}`,
		result.Outputs["get-config"],
	)
}

func TestDesktopIPCChallenge_Execute_ResponseMismatch(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.commandResults["get_config"] =
		`{"theme":"light"}`

	commands := []IPCCommand{
		{
			Name:           "get-config",
			Command:        "get_config",
			ExpectedResult: `"theme":"dark"`,
		},
	}

	ch := NewDesktopIPCChallenge(
		"IPC-003", "IPC", "Mismatch",
		nil, adapter, commands,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Message, "mismatch",
	)
}

func TestDesktopIPCChallenge_Execute_CommandError(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.commandErrors["bad_cmd"] = fmt.Errorf(
		"unknown command",
	)

	commands := []IPCCommand{
		{
			Name:           "bad-cmd",
			Command:        "bad_cmd",
			ExpectedResult: "anything",
		},
	}

	ch := NewDesktopIPCChallenge(
		"IPC-004", "IPC", "Error",
		nil, adapter, commands,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message,
		"unknown command",
	)
}

func TestDesktopIPCChallenge_Execute_NoExpectedResult(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.commandResults["ping"] = "pong"

	commands := []IPCCommand{
		{
			Name:    "ping",
			Command: "ping",
		},
	}

	ch := NewDesktopIPCChallenge(
		"IPC-005", "IPC", "No expected",
		nil, adapter, commands,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Equal(
		t, "ipc_no_error", result.Assertions[0].Type,
	)
}

func TestDesktopIPCChallenge_Execute_WithAssertions(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.commandResults["get_info"] =
		`{"version":"2.0","name":"app"}`

	commands := []IPCCommand{
		{
			Name:    "get-info",
			Command: "get_info",
			Assertions: []StepAssertion{
				{
					Type:    "response_contains",
					Target:  "version",
					Value:   "2.0",
					Message: "should contain version",
				},
				{
					Type:    "not_empty",
					Target:  "response",
					Message: "should not be empty",
				},
			},
		},
	}

	ch := NewDesktopIPCChallenge(
		"IPC-006", "IPC", "With assertions",
		nil, adapter, commands,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	// 1 ipc_no_error + 2 step assertions.
	require.Len(t, result.Assertions, 3)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed, "failed: %s", a.Message)
	}
}

func TestDesktopIPCChallenge_Execute_AssertionFails(
	t *testing.T,
) {
	adapter := newMockDesktopAdapter()
	adapter.commandResults["get_info"] = `{"version":"1.0"}`

	commands := []IPCCommand{
		{
			Name:    "get-info",
			Command: "get_info",
			Assertions: []StepAssertion{
				{
					Type:    "response_contains",
					Target:  "version",
					Value:   "2.0",
					Message: "should contain 2.0",
				},
			},
		},
	}

	ch := NewDesktopIPCChallenge(
		"IPC-007", "IPC", "Assertion fails",
		nil, adapter, commands,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
}
