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

// mockBrowserAdapter implements BrowserAdapter for testing.
type mockBrowserAdapter struct {
	initErr     error
	navigateErr error
	clickErr    error
	fillErr     error
	selectErr   error
	waitErr     error
	closeErr    error

	visibleResults  map[string]bool
	textResults     map[string]string
	evaluateResults map[string]string
	evaluateErr     error

	screenshotData  []byte
	screenshotErr   error
	screenshotCount int

	navigatedURLs []string
	clickedSels   []string
	filledSels    []string
	filledVals    []string
	closed        bool
}

func newMockBrowserAdapter() *mockBrowserAdapter {
	return &mockBrowserAdapter{
		visibleResults:  make(map[string]bool),
		textResults:     make(map[string]string),
		evaluateResults: make(map[string]string),
		screenshotData:  []byte{0x89, 0x50, 0x4E, 0x47},
	}
}

func (m *mockBrowserAdapter) Initialize(
	_ context.Context, _ BrowserConfig,
) error {
	return m.initErr
}

func (m *mockBrowserAdapter) Navigate(
	_ context.Context, url string,
) error {
	m.navigatedURLs = append(m.navigatedURLs, url)
	return m.navigateErr
}

func (m *mockBrowserAdapter) Click(
	_ context.Context, selector string,
) error {
	m.clickedSels = append(m.clickedSels, selector)
	return m.clickErr
}

func (m *mockBrowserAdapter) Fill(
	_ context.Context, selector, value string,
) error {
	m.filledSels = append(m.filledSels, selector)
	m.filledVals = append(m.filledVals, value)
	return m.fillErr
}

func (m *mockBrowserAdapter) SelectOption(
	_ context.Context, _, _ string,
) error {
	return m.selectErr
}

func (m *mockBrowserAdapter) IsVisible(
	_ context.Context, selector string,
) (bool, error) {
	if v, ok := m.visibleResults[selector]; ok {
		return v, nil
	}
	return true, nil
}

func (m *mockBrowserAdapter) WaitForSelector(
	_ context.Context, _ string, _ time.Duration,
) error {
	return m.waitErr
}

func (m *mockBrowserAdapter) GetText(
	_ context.Context, selector string,
) (string, error) {
	if t, ok := m.textResults[selector]; ok {
		return t, nil
	}
	return "", nil
}

func (m *mockBrowserAdapter) GetAttribute(
	_ context.Context, _, _ string,
) (string, error) {
	return "", nil
}

func (m *mockBrowserAdapter) Screenshot(
	_ context.Context,
) ([]byte, error) {
	m.screenshotCount++
	return m.screenshotData, m.screenshotErr
}

func (m *mockBrowserAdapter) EvaluateJS(
	_ context.Context, script string,
) (string, error) {
	if m.evaluateErr != nil {
		return "", m.evaluateErr
	}
	if r, ok := m.evaluateResults[script]; ok {
		return r, nil
	}
	return "", nil
}

func (m *mockBrowserAdapter) NetworkIntercept(
	_ context.Context,
	_ string,
	_ func(req *InterceptedRequest),
) error {
	return nil
}

func (m *mockBrowserAdapter) Close(
	_ context.Context,
) error {
	m.closed = true
	return m.closeErr
}

func (m *mockBrowserAdapter) Available(
	_ context.Context,
) bool {
	return true
}

// --- BrowserFlowChallenge tests ---

func TestNewBrowserFlowChallenge(t *testing.T) {
	adapter := newMockBrowserAdapter()
	flow := BrowserFlow{
		Name:     "test-flow",
		StartURL: "http://localhost:3000",
		Config: BrowserConfig{
			BrowserType: "chromium",
			Headless:    true,
		},
	}
	ch := NewBrowserFlowChallenge(
		"BROWSER-001", "Browser Test", "Test browser",
		nil, adapter, flow,
	)

	assert.Equal(
		t, challenge.ID("BROWSER-001"), ch.ID(),
	)
	assert.Equal(t, "Browser Test", ch.Name())
	assert.Equal(t, "browser", ch.Category())
}

func TestBrowserFlowChallenge_Execute_SimpleFlow(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
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

	ch := NewBrowserFlowChallenge(
		"BROWSER-002", "Login", "Test login",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 3)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed, "failed: %s", a.Message)
	}

	assert.True(t, adapter.closed)
	require.Len(t, adapter.navigatedURLs, 1)
	assert.Equal(
		t, "http://localhost:3000/login",
		adapter.navigatedURLs[0],
	)
	require.Len(t, adapter.filledSels, 2)
	assert.Equal(t, "#username", adapter.filledSels[0])
	assert.Equal(t, "admin", adapter.filledVals[0])
	require.Len(t, adapter.clickedSels, 1)
	assert.Equal(t, "#login-btn", adapter.clickedSels[0])
}

func TestBrowserFlowChallenge_Execute_InitFailure(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
	adapter.initErr = fmt.Errorf("chromium not found")

	flow := BrowserFlow{
		Name:     "fail-init",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-003", "Init Fail", "Init fails",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.Contains(
		t, result.Assertions[0].Message,
		"chromium not found",
	)
}

func TestBrowserFlowChallenge_Execute_NavigateFailure(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
	adapter.navigateErr = fmt.Errorf("connection refused")

	flow := BrowserFlow{
		Name:     "fail-nav",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-004", "Nav Fail", "Navigate fails",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(t, result.Error, "connection refused")
}

func TestBrowserFlowChallenge_Execute_AssertVisible(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
	adapter.visibleResults["#dashboard"] = true

	flow := BrowserFlow{
		Name:     "assert-visible",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:     "check dashboard",
				Action:   "assert_visible",
				Selector: "#dashboard",
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-005", "Visible", "Check visible",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestBrowserFlowChallenge_Execute_AssertVisibleFails(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
	adapter.visibleResults["#missing"] = false

	flow := BrowserFlow{
		Name:     "assert-not-visible",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:     "check missing",
				Action:   "assert_visible",
				Selector: "#missing",
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-006", "Not Visible", "Not visible",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message, "not visible",
	)
}

func TestBrowserFlowChallenge_Execute_AssertText(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
	adapter.textResults["h1.title"] = "Welcome to Dashboard"

	flow := BrowserFlow{
		Name:     "assert-text",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:     "check title",
				Action:   "assert_text",
				Selector: "h1.title",
				Value:    "Welcome",
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-007", "Text", "Check text",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestBrowserFlowChallenge_Execute_AssertTextFails(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
	adapter.textResults["h1.title"] = "Login Page"

	flow := BrowserFlow{
		Name:     "assert-text-fail",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:     "check title",
				Action:   "assert_text",
				Selector: "h1.title",
				Value:    "Dashboard",
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-008", "Text Fail", "Text fails",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message, "not found",
	)
}

func TestBrowserFlowChallenge_Execute_AssertURL(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
	adapter.evaluateResults["window.location.href"] =
		"http://localhost:3000/dashboard"

	flow := BrowserFlow{
		Name:     "assert-url",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:   "check url",
				Action: "assert_url",
				Value:  "/dashboard",
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-009", "URL", "Check URL",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestBrowserFlowChallenge_Execute_Screenshot(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()

	flow := BrowserFlow{
		Name:     "screenshot-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:   "take screenshot",
				Action: "screenshot",
			},
			{
				Name:       "click and screenshot",
				Action:     "click",
				Selector:   "#btn",
				Screenshot: true,
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-010", "Screenshots", "Take screenshots",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)

	// 1 from explicit screenshot step + 1 from step.Screenshot.
	assert.Equal(t, 2, adapter.screenshotCount)
	assert.Equal(t, "2", result.Outputs["screenshot_count"])
}

func TestBrowserFlowChallenge_Execute_WaitAction(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()

	flow := BrowserFlow{
		Name:     "wait-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:     "wait for element",
				Action:   "wait",
				Selector: "#loaded",
				Timeout:  2 * time.Second,
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-011", "Wait", "Wait for element",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestBrowserFlowChallenge_Execute_EvaluateJS(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
	adapter.evaluateResults["return document.title"] =
		"My App"

	flow := BrowserFlow{
		Name:     "eval-js",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:   "get title",
				Action: "evaluate_js",
				Script: "return document.title",
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-012", "Eval JS", "Evaluate JS",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestBrowserFlowChallenge_Execute_UnknownAction(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()

	flow := BrowserFlow{
		Name:     "unknown-action",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:   "bad step",
				Action: "hover",
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-013", "Unknown", "Unknown action",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(
		t, result.Assertions[0].Message,
		"unknown browser action",
	)
}

func TestBrowserFlowChallenge_Execute_WithDeps(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()
	flow := BrowserFlow{
		Name:     "deps-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
	}
	deps := []challenge.ID{"ENV-SETUP", "API-HEALTH"}
	ch := NewBrowserFlowChallenge(
		"BROWSER-014", "With Deps", "Has deps",
		deps, adapter, flow,
	)

	assert.Equal(t, deps, ch.Dependencies())
}

func TestBrowserFlowChallenge_Execute_SelectAction(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()

	flow := BrowserFlow{
		Name:     "select-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:     "select option",
				Action:   "select",
				Selector: "#dropdown",
				Value:    "option2",
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-015", "Select", "Select option",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestBrowserFlowChallenge_Execute_StepWithAssertions(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()

	flow := BrowserFlow{
		Name:     "step-assertions",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{
				Name:     "click submit",
				Action:   "click",
				Selector: "#submit",
				Assertions: []StepAssertion{
					{
						Type:    "flow_completes",
						Target:  "submit",
						Message: "submit should work",
					},
				},
			},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-016", "Step Assert", "Step with assertions",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	// 1 for the click step + 1 for the step assertion.
	require.Len(t, result.Assertions, 2)
}

func TestBrowserFlowChallenge_Execute_Metrics(
	t *testing.T,
) {
	adapter := newMockBrowserAdapter()

	flow := BrowserFlow{
		Name:     "metrics-flow",
		StartURL: "http://localhost:3000",
		Config:   BrowserConfig{Headless: true},
		Steps: []BrowserStep{
			{Name: "step-a", Action: "click", Selector: "#a"},
			{Name: "step-b", Action: "click", Selector: "#b"},
		},
	}

	ch := NewBrowserFlowChallenge(
		"BROWSER-017", "Metrics", "Check metrics",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)

	_, ok := result.Metrics["total_duration"]
	assert.True(t, ok)
	steps := result.Metrics["steps_executed"]
	assert.Equal(t, 2.0, steps.Value)
	_, ok = result.Metrics["step_step-a_duration"]
	assert.True(t, ok)
	_, ok = result.Metrics["step_step-b_duration"]
	assert.True(t, ok)
}
