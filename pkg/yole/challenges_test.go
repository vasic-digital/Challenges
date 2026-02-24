package yole

import (
	"context"
	"fmt"
	"testing"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Adapters ---

// mockGradle implements GradleAdapter for testing.
type mockGradle struct {
	available    bool
	taskResults  map[string]*GradleRunResult
	taskErrors   map[string]error
	testResults  map[string]*GradleRunResult
	testErrors   map[string]error
}

func newMockGradle(available bool) *mockGradle {
	return &mockGradle{
		available:   available,
		taskResults: make(map[string]*GradleRunResult),
		taskErrors:  make(map[string]error),
		testResults: make(map[string]*GradleRunResult),
		testErrors:  make(map[string]error),
	}
}

func (m *mockGradle) RunTask(
	_ context.Context, task string, _ ...string,
) (*GradleRunResult, error) {
	res := m.taskResults[task]
	err := m.taskErrors[task]
	if res == nil && err == nil {
		res = &GradleRunResult{
			Task: task, Success: true,
			Duration: 1 * time.Second,
		}
	}
	return res, err
}

func (m *mockGradle) RunTests(
	_ context.Context, task string, filter string,
) (*GradleRunResult, error) {
	key := task
	if filter != "" {
		key = task + ":" + filter
	}
	// Check exact key (task:filter) first.
	if res, ok := m.testResults[key]; ok {
		return res, m.testErrors[key]
	}
	if err, ok := m.testErrors[key]; ok {
		return m.testResults[key], err
	}
	// Fallback to task-only key.
	if res, ok := m.testResults[task]; ok {
		return res, m.testErrors[task]
	}
	if err, ok := m.testErrors[task]; ok {
		return m.testResults[task], err
	}
	return &GradleRunResult{
		Task: task, Success: true,
		Duration: 1 * time.Second,
	}, nil
}

func (m *mockGradle) Available(_ context.Context) bool {
	return m.available
}

// mockADB implements ADBAdapter for testing.
type mockADB struct {
	available       bool
	deviceAvailable bool
	appRunning      bool
	installErr      error
	launchErr       error
	waitErr         error
	screenshotData  []byte
}

func (m *mockADB) IsDeviceAvailable(
	_ context.Context,
) (bool, error) {
	return m.deviceAvailable, nil
}

func (m *mockADB) InstallAPK(
	_ context.Context, _ string,
) error {
	return m.installErr
}

func (m *mockADB) LaunchApp(_ context.Context) error {
	return m.launchErr
}

func (m *mockADB) StopApp(_ context.Context) error {
	return nil
}

func (m *mockADB) IsAppRunning(
	_ context.Context,
) (bool, error) {
	return m.appRunning, nil
}

func (m *mockADB) TakeScreenshot(
	_ context.Context,
) ([]byte, error) {
	return m.screenshotData, nil
}

func (m *mockADB) WaitForApp(
	_ context.Context, _ time.Duration,
) error {
	return m.waitErr
}

func (m *mockADB) Available(_ context.Context) bool {
	return m.available
}

// mockProcess implements ProcessAdapter for testing.
type mockProcess struct {
	running    bool
	launchErr  error
	waitErr    error
}

func (m *mockProcess) LaunchJVM(
	_ context.Context, _ string, _ ...string,
) error {
	return m.launchErr
}

func (m *mockProcess) IsRunning() bool {
	return m.running
}

func (m *mockProcess) WaitForReady(
	_ context.Context, _ time.Duration,
) error {
	return m.waitErr
}

func (m *mockProcess) Stop() error {
	return nil
}

// --- Helper ---

func configureChallengeForTest(
	t *testing.T, c challenge.Challenge,
) {
	t.Helper()
	cfg := challenge.NewConfig("test")
	cfg.ResultsDir = t.TempDir()
	cfg.LogsDir = t.TempDir()
	require.NoError(t, c.Configure(cfg))
}

// --- GradleBuildChallenge Tests ---

func TestNewGradleBuildChallenge(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewGradleBuildChallenge(gradle)

	assert.Equal(t,
		challenge.ID("infra-gradle-build"), c.ID(),
	)
	assert.Equal(t,
		"Gradle Build Verification", c.Name(),
	)
	assert.Equal(t, "infrastructure", c.Category())
}

func TestGradleBuildChallenge_Validate(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewGradleBuildChallenge(gradle)
	configureChallengeForTest(t, c)

	assert.NoError(t, c.Validate(context.Background()))
}

func TestGradleBuildChallenge_Validate_NotAvailable(
	t *testing.T,
) {
	gradle := newMockGradle(false)
	c := NewGradleBuildChallenge(gradle)
	configureChallengeForTest(t, c)

	err := c.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestGradleBuildChallenge_Execute_AllPass(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewGradleBuildChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Len(t, result.Assertions, 3) // 3 build targets
	for _, a := range result.Assertions {
		assert.True(t, a.Passed,
			"assertion %s failed", a.Target,
		)
	}
}

func TestGradleBuildChallenge_Execute_OneFails(t *testing.T) {
	gradle := newMockGradle(true)
	gradle.taskErrors[":androidApp:assembleDebug"] =
		fmt.Errorf("compilation error")
	c := NewGradleBuildChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
	failedCount := 0
	for _, a := range result.Assertions {
		if !a.Passed {
			failedCount++
		}
	}
	assert.Equal(t, 1, failedCount)
}

// --- GradleTestsChallenge Tests ---

func TestNewGradleTestsChallenge(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewGradleTestsChallenge(gradle)

	assert.Equal(t,
		challenge.ID("infra-gradle-tests"), c.ID(),
	)
	assert.Contains(t, c.Dependencies(),
		challenge.ID("infra-gradle-build"),
	)
}

func TestGradleTestsChallenge_Execute_AllPass(t *testing.T) {
	gradle := newMockGradle(true)
	gradle.testResults[":shared:testDebugUnitTest"] =
		&GradleRunResult{
			Success:  true,
			Duration: 2 * time.Second,
			Suites: []JUnitTestSuite{
				{
					Name: "FormatTests", Tests: 100,
					Failures: 0, Errors: 0,
				},
			},
		}
	c := NewGradleTestsChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Contains(t, result.Outputs, "total_tests")
}

func TestGradleTestsChallenge_Execute_SomeFail(t *testing.T) {
	gradle := newMockGradle(true)
	gradle.testErrors[":shared:desktopTest"] =
		fmt.Errorf("test failures")
	c := NewGradleTestsChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

// --- LintChallenge Tests ---

func TestNewLintChallenge(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewLintChallenge(gradle)

	assert.Equal(t, challenge.ID("infra-lint"), c.ID())
	assert.Contains(t, c.Dependencies(),
		challenge.ID("infra-gradle-build"),
	)
}

func TestLintChallenge_Execute_AllPass(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewLintChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Len(t, result.Assertions, 2)
}

func TestLintChallenge_Execute_LintFails(t *testing.T) {
	gradle := newMockGradle(true)
	gradle.taskErrors[":androidApp:lintDebug"] =
		fmt.Errorf("lint violations found")
	c := NewLintChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

// --- RobolectricLaunchChallenge Tests ---

func TestNewRobolectricLaunchChallenge(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewRobolectricLaunchChallenge(gradle)

	assert.Equal(t,
		challenge.ID("android-robolectric-launch"), c.ID(),
	)
	assert.Equal(t, "android", c.Category())
}

func TestRobolectricLaunchChallenge_Execute_Pass(
	t *testing.T,
) {
	gradle := newMockGradle(true)
	c := NewRobolectricLaunchChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Len(t, result.Assertions, 1)
	assert.True(t, result.Assertions[0].Passed)
}

func TestRobolectricLaunchChallenge_Execute_Fail(
	t *testing.T,
) {
	gradle := newMockGradle(true)
	key := ":androidApp:testDebugUnitTest:" +
		"digital.vasic.yole.android.robolectric." +
		"AppLaunchRobolectricTest"
	gradle.testErrors[key] = fmt.Errorf("app crashed")
	gradle.testResults[key] = &GradleRunResult{
		Output: "CRASH",
	}
	c := NewRobolectricLaunchChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
}

// --- RobolectricFlowsChallenge Tests ---

func TestNewRobolectricFlowsChallenge(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewRobolectricFlowsChallenge(gradle)

	assert.Equal(t,
		challenge.ID("android-robolectric-flows"), c.ID(),
	)
	assert.Contains(t, c.Dependencies(),
		challenge.ID("android-robolectric-launch"),
	)
}

func TestRobolectricFlowsChallenge_Execute_AllPass(
	t *testing.T,
) {
	gradle := newMockGradle(true)
	c := NewRobolectricFlowsChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
	// 9 test classes
	assert.Len(t, result.Assertions, 9)
}

func TestRobolectricFlowsChallenge_Execute_SomeFail(
	t *testing.T,
) {
	gradle := newMockGradle(true)
	key := ":androidApp:testDebugUnitTest:" +
		"digital.vasic.yole.android.robolectric." +
		"ThemeRobolectricTest"
	gradle.testErrors[key] = fmt.Errorf("theme test failed")
	c := NewRobolectricFlowsChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

// --- UIAutomatorLaunchChallenge Tests ---

func TestNewUIAutomatorLaunchChallenge(t *testing.T) {
	adb := &mockADB{available: true, deviceAvailable: true}
	gradle := newMockGradle(true)
	c := NewUIAutomatorLaunchChallenge(adb, gradle)

	assert.Equal(t,
		challenge.ID("android-uiautomator-launch"), c.ID(),
	)
}

func TestUIAutomatorLaunch_Validate_NoADB(t *testing.T) {
	adb := &mockADB{available: false}
	gradle := newMockGradle(true)
	c := NewUIAutomatorLaunchChallenge(adb, gradle)
	configureChallengeForTest(t, c)

	err := c.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ADB not available")
}

func TestUIAutomatorLaunch_Validate_NoDevice(t *testing.T) {
	adb := &mockADB{
		available: true, deviceAvailable: false,
	}
	gradle := newMockGradle(true)
	c := NewUIAutomatorLaunchChallenge(adb, gradle)
	configureChallengeForTest(t, c)

	err := c.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Android device")
}

func TestUIAutomatorLaunch_Execute_Success(t *testing.T) {
	adb := &mockADB{
		available: true, deviceAvailable: true,
		appRunning:     true,
		screenshotData: []byte("png-data"),
	}
	gradle := newMockGradle(true)
	c := NewUIAutomatorLaunchChallenge(adb, gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Contains(t, result.Outputs, "screenshot_size")
}

func TestUIAutomatorLaunch_Execute_BuildFails(t *testing.T) {
	adb := &mockADB{available: true, deviceAvailable: true}
	gradle := newMockGradle(true)
	gradle.taskErrors[":androidApp:assembleDebug"] =
		fmt.Errorf("build error")
	c := NewUIAutomatorLaunchChallenge(adb, gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

func TestUIAutomatorLaunch_Execute_InstallFails(
	t *testing.T,
) {
	adb := &mockADB{
		available: true, deviceAvailable: true,
		installErr: fmt.Errorf("install failed"),
	}
	gradle := newMockGradle(true)
	c := NewUIAutomatorLaunchChallenge(adb, gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

func TestUIAutomatorLaunch_Execute_LaunchFails(
	t *testing.T,
) {
	adb := &mockADB{
		available: true, deviceAvailable: true,
		launchErr: fmt.Errorf("launch failed"),
	}
	gradle := newMockGradle(true)
	c := NewUIAutomatorLaunchChallenge(adb, gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

func TestUIAutomatorLaunch_Execute_AppCrashes(t *testing.T) {
	adb := &mockADB{
		available: true, deviceAvailable: true,
		appRunning: false,
	}
	gradle := newMockGradle(true)
	c := NewUIAutomatorLaunchChallenge(adb, gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

// --- DesktopLaunchChallenge Tests ---

func TestNewDesktopLaunchChallenge(t *testing.T) {
	gradle := newMockGradle(true)
	process := &mockProcess{running: true}
	c := NewDesktopLaunchChallenge(gradle, process)

	assert.Equal(t,
		challenge.ID("desktop-launch"), c.ID(),
	)
	assert.Equal(t, "desktop", c.Category())
}

func TestDesktopLaunch_Execute_Success(t *testing.T) {
	gradle := newMockGradle(true)
	process := &mockProcess{running: true}
	c := NewDesktopLaunchChallenge(gradle, process)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestDesktopLaunch_Execute_BuildFails(t *testing.T) {
	gradle := newMockGradle(true)
	gradle.taskErrors[":desktopApp:jar"] =
		fmt.Errorf("jar build failed")
	process := &mockProcess{running: true}
	c := NewDesktopLaunchChallenge(gradle, process)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

func TestDesktopLaunch_Execute_LaunchFails(t *testing.T) {
	gradle := newMockGradle(true)
	process := &mockProcess{
		launchErr: fmt.Errorf("java not found"),
	}
	c := NewDesktopLaunchChallenge(gradle, process)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

func TestDesktopLaunch_Execute_WaitFails(t *testing.T) {
	gradle := newMockGradle(true)
	process := &mockProcess{
		waitErr: fmt.Errorf("timed out"),
	}
	c := NewDesktopLaunchChallenge(gradle, process)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

func TestDesktopLaunch_Execute_AppCrashes(t *testing.T) {
	gradle := newMockGradle(true)
	process := &mockProcess{running: false}
	c := NewDesktopLaunchChallenge(gradle, process)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

// --- DesktopUserFlowsChallenge Tests ---

func TestNewDesktopUserFlowsChallenge(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewDesktopUserFlowsChallenge(gradle)

	assert.Equal(t,
		challenge.ID("desktop-user-flows"), c.ID(),
	)
	assert.Contains(t, c.Dependencies(),
		challenge.ID("desktop-launch"),
	)
}

func TestDesktopUserFlows_Execute_Pass(t *testing.T) {
	gradle := newMockGradle(true)
	gradle.testResults[":desktopApp:test"] =
		&GradleRunResult{
			Success:  true,
			Duration: 5 * time.Second,
			Suites: []JUnitTestSuite{
				{
					Name: "DesktopTests", Tests: 10,
					Failures: 0,
				},
			},
		}
	c := NewDesktopUserFlowsChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Contains(t, result.Outputs, "DesktopTests")
}

func TestDesktopUserFlows_Execute_Fail(t *testing.T) {
	gradle := newMockGradle(true)
	gradle.testErrors[":desktopApp:test"] =
		fmt.Errorf("desktop tests failed")
	c := NewDesktopUserFlowsChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}

// --- WebLaunchChallenge Tests ---

func TestNewWebLaunchChallenge(t *testing.T) {
	gradle := newMockGradle(true)
	c := NewWebLaunchChallenge(gradle)

	assert.Equal(t, challenge.ID("web-launch"), c.ID())
	assert.Equal(t, "web", c.Category())
}

func TestWebLaunch_Execute_Pass(t *testing.T) {
	gradle := newMockGradle(true)
	gradle.testResults[":webApp:wasmJsBrowserTest"] =
		&GradleRunResult{
			Success:  true,
			Duration: 3 * time.Second,
			Suites: []JUnitTestSuite{
				{
					Name: "WasmTests", Tests: 5,
					Failures: 0,
				},
			},
		}
	c := NewWebLaunchChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestWebLaunch_Execute_Fail(t *testing.T) {
	gradle := newMockGradle(true)
	gradle.testErrors[":webApp:wasmJsBrowserTest"] =
		fmt.Errorf("wasm tests failed")
	c := NewWebLaunchChallenge(gradle)
	configureChallengeForTest(t, c)

	result, err := c.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, challenge.StatusFailed, result.Status)
}
