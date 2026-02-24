package yole

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// --- Infrastructure Challenges ---

// GradleBuildChallenge verifies all Gradle build tasks succeed.
type GradleBuildChallenge struct {
	challenge.BaseChallenge
	gradle  GradleAdapter
	targets []BuildTarget
}

// NewGradleBuildChallenge creates a challenge that compiles all
// application modules.
func NewGradleBuildChallenge(
	gradle GradleAdapter,
) *GradleBuildChallenge {
	return &GradleBuildChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"infra-gradle-build",
			"Gradle Build Verification",
			"Verifies all application modules compile "+
				"successfully",
			"infrastructure",
			nil,
		),
		gradle: gradle,
		targets: []BuildTarget{
			{"Android Debug", ":androidApp:assembleDebug"},
			{"Desktop JAR", ":desktopApp:jar"},
			{"Shared Library", ":shared:compileKotlinJvm"},
		},
	}
}

// Execute runs the Gradle build tasks.
func (c *GradleBuildChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	start := time.Now()
	outputs := make(map[string]string)
	metrics := make(map[string]challenge.MetricValue)
	var assertions []challenge.AssertionResult
	var errMsg string
	allPassed := true

	for _, build := range c.targets {
		c.ReportProgress(
			fmt.Sprintf("Building %s...", build.Name),
			nil,
		)
		res, err := c.gradle.RunTask(ctx, build.Task)
		if err != nil {
			allPassed = false
			assertions = append(assertions,
				challenge.AssertionResult{
					Type:     "build_succeeds",
					Target:   build.Name,
					Expected: "build success",
					Actual: fmt.Sprintf(
						"failed: %v", err,
					),
					Passed:  false,
					Message: fmt.Sprintf(
						"%s build failed", build.Name,
					),
				},
			)
			if res != nil {
				outputs[build.Name+"_output"] = res.Output
			}
			errMsg = fmt.Sprintf(
				"%s build failed: %v", build.Name, err,
			)
		} else {
			assertions = append(assertions,
				challenge.AssertionResult{
					Type:     "build_succeeds",
					Target:   build.Name,
					Expected: "build success",
					Actual:   "build success",
					Passed:   true,
					Message: fmt.Sprintf(
						"%s built in %v",
						build.Name, res.Duration,
					),
				},
			)
			metrics[build.Name] = challenge.MetricValue{
				Name:  build.Name + "_duration",
				Value: res.Duration.Seconds(),
				Unit:  "seconds",
			}
		}
	}

	status := challenge.StatusPassed
	if !allPassed {
		status = challenge.StatusFailed
	}

	return c.CreateResult(
		status, start, assertions, metrics, outputs, errMsg,
	), nil
}

// Validate checks that Gradle is available.
func (c *GradleBuildChallenge) Validate(
	ctx context.Context,
) error {
	if !c.gradle.Available(ctx) {
		return fmt.Errorf("gradle not available")
	}
	return nil
}

// --- Gradle Tests Challenge ---

// GradleTestsChallenge runs all Gradle tests and collects
// results.
type GradleTestsChallenge struct {
	challenge.BaseChallenge
	gradle  GradleAdapter
	targets []TestTarget
}

// NewGradleTestsChallenge creates a challenge that executes
// all unit and integration tests.
func NewGradleTestsChallenge(
	gradle GradleAdapter,
) *GradleTestsChallenge {
	return &GradleTestsChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"infra-gradle-tests",
			"Gradle Test Execution",
			"Runs all existing unit and integration "+
				"tests across all modules",
			"infrastructure",
			[]challenge.ID{"infra-gradle-build"},
		),
		gradle: gradle,
		targets: []TestTarget{
			{
				"Shared Unit Tests",
				":shared:testDebugUnitTest", "",
			},
			{
				"Shared Desktop Tests",
				":shared:desktopTest", "",
			},
			{
				"Android Robolectric Tests",
				":androidApp:testDebugUnitTest", "",
			},
		},
	}
}

// Execute runs each test task and collects JUnit results.
func (c *GradleTestsChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	start := time.Now()
	outputs := make(map[string]string)
	metrics := make(map[string]challenge.MetricValue)
	var assertions []challenge.AssertionResult
	var errMsg string
	allPassed := true
	totalTests := 0
	totalFailures := 0

	for _, tt := range c.targets {
		c.ReportProgress(
			fmt.Sprintf("Running %s...", tt.Name), nil,
		)
		res, err := c.gradle.RunTests(
			ctx, tt.Task, tt.Filter,
		)

		if res != nil {
			for _, suite := range res.Suites {
				totalTests += suite.Tests
				totalFailures += suite.Failures +
					suite.Errors
				outputs[fmt.Sprintf(
					"%s_%s", tt.Name, suite.Name,
				)] = fmt.Sprintf(
					"tests=%d failures=%d errors=%d "+
						"time=%.2fs",
					suite.Tests, suite.Failures,
					suite.Errors, suite.Time,
				)
			}
		}

		if err != nil {
			allPassed = false
			assertions = append(assertions,
				challenge.AssertionResult{
					Type:     "all_tests_pass",
					Target:   tt.Name,
					Expected: "all tests pass",
					Actual: fmt.Sprintf(
						"failed: %v", err,
					),
					Passed:  false,
					Message: fmt.Sprintf(
						"%s failed", tt.Name,
					),
				},
			)
			errMsg = fmt.Sprintf(
				"%s failed: %v", tt.Name, err,
			)
		} else {
			assertions = append(assertions,
				challenge.AssertionResult{
					Type:     "all_tests_pass",
					Target:   tt.Name,
					Expected: "all tests pass",
					Actual:   "all tests pass",
					Passed:   true,
					Message: fmt.Sprintf(
						"%s passed in %v",
						tt.Name, res.Duration,
					),
				},
			)
		}
	}

	outputs["total_tests"] = fmt.Sprintf("%d", totalTests)
	outputs["total_failures"] = fmt.Sprintf(
		"%d", totalFailures,
	)
	metrics["total_tests"] = challenge.MetricValue{
		Name: "total_tests", Value: float64(totalTests),
		Unit: "count",
	}
	metrics["total_failures"] = challenge.MetricValue{
		Name: "total_failures",
		Value: float64(totalFailures),
		Unit:  "count",
	}

	status := challenge.StatusPassed
	if !allPassed {
		status = challenge.StatusFailed
	}

	return c.CreateResult(
		status, start, assertions, metrics, outputs, errMsg,
	), nil
}

// --- Lint Challenge ---

// LintChallenge runs lint and static analysis checks.
type LintChallenge struct {
	challenge.BaseChallenge
	gradle GradleAdapter
}

// NewLintChallenge creates a challenge that runs Android lint
// and Detekt.
func NewLintChallenge(
	gradle GradleAdapter,
) *LintChallenge {
	return &LintChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"infra-lint",
			"Lint and Static Analysis",
			"Runs Android lint and Detekt static analysis",
			"infrastructure",
			[]challenge.ID{"infra-gradle-build"},
		),
		gradle: gradle,
	}
}

// Execute runs lint and static analysis tasks.
func (c *LintChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	start := time.Now()
	outputs := make(map[string]string)
	metrics := make(map[string]challenge.MetricValue)
	var assertions []challenge.AssertionResult
	var errMsg string
	allPassed := true

	tasks := []struct {
		name string
		task string
	}{
		{"Android Lint", ":androidApp:lintDebug"},
		{"Detekt", "detekt"},
	}

	for _, t := range tasks {
		c.ReportProgress(
			fmt.Sprintf("Running %s...", t.name), nil,
		)
		res, err := c.gradle.RunTask(ctx, t.task)
		if err != nil {
			allPassed = false
			assertions = append(assertions,
				challenge.AssertionResult{
					Type:     "lint_passes",
					Target:   t.name,
					Expected: "passes",
					Actual: fmt.Sprintf(
						"failed: %v", err,
					),
					Passed:  false,
					Message: fmt.Sprintf(
						"%s failed", t.name,
					),
				},
			)
			if errMsg == "" {
				errMsg = fmt.Sprintf(
					"%s failed: %v", t.name, err,
				)
			}
		} else {
			assertions = append(assertions,
				challenge.AssertionResult{
					Type:     "lint_passes",
					Target:   t.name,
					Expected: "passes",
					Actual:   "passes",
					Passed:   true,
					Message: fmt.Sprintf(
						"%s passed in %v",
						t.name, res.Duration,
					),
				},
			)
			metrics[t.name+"_duration"] = challenge.MetricValue{
				Name:  t.name + "_duration",
				Value: res.Duration.Seconds(),
				Unit:  "seconds",
			}
		}
	}

	status := challenge.StatusPassed
	if !allPassed {
		status = challenge.StatusFailed
	}

	return c.CreateResult(
		status, start, assertions, metrics, outputs, errMsg,
	), nil
}

// --- Android Challenges ---

// RobolectricLaunchChallenge runs Robolectric app launch tests.
type RobolectricLaunchChallenge struct {
	challenge.BaseChallenge
	gradle GradleAdapter
}

// NewRobolectricLaunchChallenge creates a challenge that
// verifies Android app launch via Robolectric.
func NewRobolectricLaunchChallenge(
	gradle GradleAdapter,
) *RobolectricLaunchChallenge {
	return &RobolectricLaunchChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"android-robolectric-launch",
			"Android App Launch (Robolectric)",
			"Verifies the Android app launches without "+
				"crash using Robolectric",
			"android",
			[]challenge.ID{"infra-gradle-build"},
		),
		gradle: gradle,
	}
}

// Execute runs the Robolectric app launch test class.
func (c *RobolectricLaunchChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	start := time.Now()
	outputs := make(map[string]string)
	metrics := make(map[string]challenge.MetricValue)
	var assertions []challenge.AssertionResult
	var errMsg string

	c.ReportProgress(
		"Running Robolectric launch tests...", nil,
	)
	res, err := c.gradle.RunTests(
		ctx,
		":androidApp:testDebugUnitTest",
		"digital.vasic.yole.android.robolectric."+
			"AppLaunchRobolectricTest",
	)

	if err != nil {
		assertions = append(assertions,
			challenge.AssertionResult{
				Type:     "app_launches",
				Target:   "robolectric_launch",
				Expected: "app launches",
				Actual: fmt.Sprintf(
					"failed: %v", err,
				),
				Passed:  false,
				Message: "Robolectric launch tests failed",
			},
		)
		errMsg = fmt.Sprintf(
			"Robolectric launch failed: %v", err,
		)
		if res != nil {
			outputs["output"] = res.Output
		}
	} else {
		assertions = append(assertions,
			challenge.AssertionResult{
				Type:     "app_launches",
				Target:   "robolectric_launch",
				Expected: "app launches",
				Actual:   "app launches",
				Passed:   true,
				Message: fmt.Sprintf(
					"Launch tests passed in %v",
					res.Duration,
				),
			},
		)
		for _, suite := range res.Suites {
			outputs[suite.Name] = fmt.Sprintf(
				"tests=%d failures=%d time=%.2fs",
				suite.Tests, suite.Failures, suite.Time,
			)
		}
	}

	status := challenge.StatusPassed
	if err != nil {
		status = challenge.StatusFailed
	}

	return c.CreateResult(
		status, start, assertions, metrics, outputs, errMsg,
	), nil
}

// --- Robolectric Flows Challenge ---

// RobolectricFlowsChallenge runs all Robolectric user flow
// tests.
type RobolectricFlowsChallenge struct {
	challenge.BaseChallenge
	gradle      GradleAdapter
	testClasses []TestTarget
}

// NewRobolectricFlowsChallenge creates a challenge that runs
// all Robolectric user flow test classes.
func NewRobolectricFlowsChallenge(
	gradle GradleAdapter,
) *RobolectricFlowsChallenge {
	pkg := "digital.vasic.yole.android.robolectric."
	return &RobolectricFlowsChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"android-robolectric-flows",
			"Android User Flows (Robolectric)",
			"Runs all Robolectric user flow tests",
			"android",
			[]challenge.ID{"android-robolectric-launch"},
		),
		gradle: gradle,
		testClasses: []TestTarget{
			{"Theme", ":androidApp:testDebugUnitTest",
				pkg + "ThemeRobolectricTest"},
			{"Navigation", ":androidApp:testDebugUnitTest",
				pkg + "NavigationRobolectricTest"},
			{"Settings", ":androidApp:testDebugUnitTest",
				pkg + "SettingsRobolectricTest"},
			{"File Editing", ":androidApp:testDebugUnitTest",
				pkg + "FileEditingRobolectricTest"},
			{"Format Detection",
				":androidApp:testDebugUnitTest",
				pkg + "FormatDetectionRobolectricTest"},
			{"Todo Workflow",
				":androidApp:testDebugUnitTest",
				pkg + "TodoWorkflowRobolectricTest"},
			{"QuickNote", ":androidApp:testDebugUnitTest",
				pkg + "QuickNoteRobolectricTest"},
			{"Backup/Restore",
				":androidApp:testDebugUnitTest",
				pkg + "BackupRestoreRobolectricTest"},
			{"Accessibility",
				":androidApp:testDebugUnitTest",
				pkg + "AccessibilityRobolectricTest"},
		},
	}
}

// Execute runs each Robolectric test class.
func (c *RobolectricFlowsChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	start := time.Now()
	outputs := make(map[string]string)
	metrics := make(map[string]challenge.MetricValue)
	var assertions []challenge.AssertionResult
	var errMsg string
	allPassed := true

	for _, tc := range c.testClasses {
		c.ReportProgress(
			fmt.Sprintf("Running %s tests...", tc.Name),
			nil,
		)
		res, err := c.gradle.RunTests(
			ctx, tc.Task, tc.Filter,
		)

		if err != nil {
			allPassed = false
			assertions = append(assertions,
				challenge.AssertionResult{
					Type:     "all_tests_pass",
					Target:   tc.Name,
					Expected: "tests pass",
					Actual: fmt.Sprintf(
						"failed: %v", err,
					),
					Passed: false,
					Message: fmt.Sprintf(
						"%s tests failed", tc.Name,
					),
				},
			)
			errMsg = fmt.Sprintf(
				"%s tests failed: %v", tc.Name, err,
			)
		} else {
			assertions = append(assertions,
				challenge.AssertionResult{
					Type:     "all_tests_pass",
					Target:   tc.Name,
					Expected: "tests pass",
					Actual:   "tests pass",
					Passed:   true,
					Message: fmt.Sprintf(
						"%s tests passed in %v",
						tc.Name, res.Duration,
					),
				},
			)
		}

		if res != nil {
			for _, suite := range res.Suites {
				outputs[fmt.Sprintf(
					"%s_%s", tc.Name, suite.Name,
				)] = fmt.Sprintf(
					"tests=%d failures=%d time=%.2fs",
					suite.Tests, suite.Failures,
					suite.Time,
				)
			}
		}
	}

	status := challenge.StatusPassed
	if !allPassed {
		status = challenge.StatusFailed
	}

	return c.CreateResult(
		status, start, assertions, metrics, outputs, errMsg,
	), nil
}

// --- UIAutomator Launch Challenge ---

// UIAutomatorLaunchChallenge tests app launch on a real device.
type UIAutomatorLaunchChallenge struct {
	challenge.BaseChallenge
	adb    ADBAdapter
	gradle GradleAdapter
}

// NewUIAutomatorLaunchChallenge creates a challenge that
// installs and launches the app on a connected device.
func NewUIAutomatorLaunchChallenge(
	adb ADBAdapter, gradle GradleAdapter,
) *UIAutomatorLaunchChallenge {
	return &UIAutomatorLaunchChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"android-uiautomator-launch",
			"Android App Launch (Device)",
			"Installs and launches the Android app on "+
				"a real device or emulator via ADB",
			"android",
			[]challenge.ID{"infra-gradle-build"},
		),
		adb:    adb,
		gradle: gradle,
	}
}

// Validate checks that a device is available.
func (c *UIAutomatorLaunchChallenge) Validate(
	ctx context.Context,
) error {
	if !c.adb.Available(ctx) {
		return fmt.Errorf("ADB not available")
	}
	available, err := c.adb.IsDeviceAvailable(ctx)
	if err != nil || !available {
		return fmt.Errorf(
			"no Android device or emulator available",
		)
	}
	return nil
}

// Execute builds, installs, and launches the app.
func (c *UIAutomatorLaunchChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	start := time.Now()
	outputs := make(map[string]string)
	metrics := make(map[string]challenge.MetricValue)
	var assertions []challenge.AssertionResult

	// Build debug APK
	c.ReportProgress("Building debug APK...", nil)
	buildRes, err := c.gradle.RunTask(
		ctx, ":androidApp:assembleDebug",
	)
	if err != nil {
		return c.CreateResult(
			challenge.StatusFailed, start,
			[]challenge.AssertionResult{{
				Type: "build_succeeds", Target: "apk_build",
				Expected: "APK builds",
				Actual:   fmt.Sprintf("failed: %v", err),
				Passed:  false,
				Message: "APK build failed",
			}},
			metrics, outputs,
			fmt.Sprintf("APK build failed: %v", err),
		), nil
	}
	assertions = append(assertions,
		challenge.AssertionResult{
			Type: "build_succeeds", Target: "apk_build",
			Expected: "APK builds", Actual: "APK builds",
			Passed: true,
			Message: fmt.Sprintf(
				"APK built in %v", buildRes.Duration,
			),
		},
	)

	// Install APK
	c.ReportProgress("Installing APK...", nil)
	apkPath := filepath.Join(
		"androidApp", "build", "outputs", "apk",
		"debug", "androidApp-debug.apk",
	)
	if err := c.adb.InstallAPK(ctx, apkPath); err != nil {
		return c.CreateResult(
			challenge.StatusFailed, start, assertions,
			metrics, outputs,
			fmt.Sprintf("APK install failed: %v", err),
		), nil
	}

	// Launch app
	c.ReportProgress("Launching app...", nil)
	if err := c.adb.LaunchApp(ctx); err != nil {
		return c.CreateResult(
			challenge.StatusFailed, start, assertions,
			metrics, outputs,
			fmt.Sprintf("App launch failed: %v", err),
		), nil
	}

	// Wait for app to be running
	if err := c.adb.WaitForApp(
		ctx, 15*time.Second,
	); err != nil {
		return c.CreateResult(
			challenge.StatusFailed, start, assertions,
			metrics, outputs,
			fmt.Sprintf("App did not start: %v", err),
		), nil
	}

	// Verify stability
	time.Sleep(3 * time.Second)
	running, _ := c.adb.IsAppRunning(ctx)
	assertions = append(assertions,
		challenge.AssertionResult{
			Type: "app_stable", Target: "app_stability",
			Expected: "app running after 3s",
			Actual: fmt.Sprintf("running=%v", running),
			Passed:  running,
			Message: map[bool]string{
				true:  "App stable",
				false: "App crashed",
			}[running],
		},
	)

	// Screenshot
	screenshot, err := c.adb.TakeScreenshot(ctx)
	if err == nil && len(screenshot) > 0 {
		outputs["screenshot_size"] = fmt.Sprintf(
			"%d bytes", len(screenshot),
		)
	}

	_ = c.adb.StopApp(ctx)

	status := challenge.StatusPassed
	if !running {
		status = challenge.StatusFailed
	}

	return c.CreateResult(
		status, start, assertions, metrics, outputs, "",
	), nil
}

// --- Desktop Launch Challenge ---

// DesktopLaunchChallenge launches the desktop JVM app.
type DesktopLaunchChallenge struct {
	challenge.BaseChallenge
	gradle  GradleAdapter
	process ProcessAdapter
}

// NewDesktopLaunchChallenge creates a challenge that builds
// and launches the desktop app.
func NewDesktopLaunchChallenge(
	gradle GradleAdapter, process ProcessAdapter,
) *DesktopLaunchChallenge {
	return &DesktopLaunchChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"desktop-launch",
			"Desktop App Launch",
			"Builds and launches the desktop JVM app",
			"desktop",
			[]challenge.ID{"infra-gradle-build"},
		),
		gradle:  gradle,
		process: process,
	}
}

// Execute builds and launches the desktop JAR.
func (c *DesktopLaunchChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	start := time.Now()
	outputs := make(map[string]string)
	metrics := make(map[string]challenge.MetricValue)
	var assertions []challenge.AssertionResult

	// Build desktop JAR
	c.ReportProgress("Building desktop JAR...", nil)
	buildRes, err := c.gradle.RunTask(
		ctx, ":desktopApp:jar",
	)
	if err != nil {
		return c.CreateResult(
			challenge.StatusFailed, start,
			[]challenge.AssertionResult{{
				Type: "build_succeeds",
				Target: "desktop_build",
				Expected: "JAR builds",
				Actual: fmt.Sprintf("failed: %v", err),
				Passed: false, Message: "JAR build failed",
			}},
			metrics, outputs,
			fmt.Sprintf("JAR build failed: %v", err),
		), nil
	}
	metrics["build_duration"] = challenge.MetricValue{
		Name:  "build_duration",
		Value: buildRes.Duration.Seconds(),
		Unit:  "seconds",
	}

	// Launch
	jarPath := filepath.Join(
		"desktopApp", "build", "libs", "desktopApp.jar",
	)
	c.ReportProgress("Launching desktop app...", nil)
	if err := c.process.LaunchJVM(ctx, jarPath); err != nil {
		return c.CreateResult(
			challenge.StatusFailed, start, assertions,
			metrics, outputs,
			fmt.Sprintf("Launch failed: %v", err),
		), nil
	}
	defer c.process.Stop()

	if err := c.process.WaitForReady(
		ctx, 15*time.Second,
	); err != nil {
		return c.CreateResult(
			challenge.StatusFailed, start, assertions,
			metrics, outputs,
			fmt.Sprintf("App did not start: %v", err),
		), nil
	}

	time.Sleep(5 * time.Second)
	running := c.process.IsRunning()
	assertions = append(assertions,
		challenge.AssertionResult{
			Type: "app_stable", Target: "desktop_stable",
			Expected: "app running after 5s",
			Actual: fmt.Sprintf("running=%v", running),
			Passed:  running,
			Message: map[bool]string{
				true:  "Desktop app stable",
				false: "Desktop app crashed",
			}[running],
		},
	)

	status := challenge.StatusPassed
	if !running {
		status = challenge.StatusFailed
	}

	return c.CreateResult(
		status, start, assertions, metrics, outputs, "",
	), nil
}

// --- Desktop User Flows Challenge ---

// DesktopUserFlowsChallenge runs desktop tests via Gradle.
type DesktopUserFlowsChallenge struct {
	challenge.BaseChallenge
	gradle GradleAdapter
}

// NewDesktopUserFlowsChallenge creates a challenge that runs
// all desktop-specific tests.
func NewDesktopUserFlowsChallenge(
	gradle GradleAdapter,
) *DesktopUserFlowsChallenge {
	return &DesktopUserFlowsChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"desktop-user-flows",
			"Desktop User Flows",
			"Runs all desktop-specific tests",
			"desktop",
			[]challenge.ID{"desktop-launch"},
		),
		gradle: gradle,
	}
}

// Execute runs the desktop test suite.
func (c *DesktopUserFlowsChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	start := time.Now()
	outputs := make(map[string]string)
	metrics := make(map[string]challenge.MetricValue)
	var assertions []challenge.AssertionResult
	var errMsg string

	c.ReportProgress("Running desktop tests...", nil)
	res, err := c.gradle.RunTests(
		ctx, ":desktopApp:test", "",
	)

	if err != nil {
		assertions = append(assertions,
			challenge.AssertionResult{
				Type:     "all_tests_pass",
				Target:   "desktop_tests",
				Expected: "all tests pass",
				Actual:   fmt.Sprintf("failed: %v", err),
				Passed:   false,
				Message:  "Desktop tests failed",
			},
		)
		errMsg = fmt.Sprintf(
			"Desktop tests failed: %v", err,
		)
	} else {
		assertions = append(assertions,
			challenge.AssertionResult{
				Type:     "all_tests_pass",
				Target:   "desktop_tests",
				Expected: "all tests pass",
				Actual:   "all tests pass",
				Passed:   true,
				Message: fmt.Sprintf(
					"Desktop tests passed in %v",
					res.Duration,
				),
			},
		)
	}

	if res != nil {
		for _, suite := range res.Suites {
			outputs[suite.Name] = fmt.Sprintf(
				"tests=%d failures=%d time=%.2fs",
				suite.Tests, suite.Failures, suite.Time,
			)
		}
	}

	status := challenge.StatusPassed
	if err != nil {
		status = challenge.StatusFailed
	}

	return c.CreateResult(
		status, start, assertions, metrics, outputs, errMsg,
	), nil
}

// --- Web Launch Challenge ---

// WebLaunchChallenge builds and tests the Wasm web app.
type WebLaunchChallenge struct {
	challenge.BaseChallenge
	gradle GradleAdapter
}

// NewWebLaunchChallenge creates a challenge that builds the
// web app and runs Wasm browser tests.
func NewWebLaunchChallenge(
	gradle GradleAdapter,
) *WebLaunchChallenge {
	return &WebLaunchChallenge{
		BaseChallenge: challenge.NewBaseChallenge(
			"web-launch",
			"Web App Launch",
			"Builds the Wasm web app and runs browser "+
				"tests",
			"web",
			[]challenge.ID{"infra-gradle-build"},
		),
		gradle: gradle,
	}
}

// Execute runs web tests via Gradle.
func (c *WebLaunchChallenge) Execute(
	ctx context.Context,
) (*challenge.Result, error) {
	start := time.Now()
	outputs := make(map[string]string)
	metrics := make(map[string]challenge.MetricValue)
	var assertions []challenge.AssertionResult
	var errMsg string

	c.ReportProgress("Running web tests...", nil)
	res, err := c.gradle.RunTests(
		ctx, ":webApp:wasmJsBrowserTest", "",
	)

	if err != nil {
		assertions = append(assertions,
			challenge.AssertionResult{
				Type:     "all_tests_pass",
				Target:   "web_tests",
				Expected: "web tests pass",
				Actual:   fmt.Sprintf("failed: %v", err),
				Passed:   false,
				Message:  "Web tests failed",
			},
		)
		errMsg = fmt.Sprintf(
			"Web tests failed: %v", err,
		)
	} else {
		assertions = append(assertions,
			challenge.AssertionResult{
				Type:     "all_tests_pass",
				Target:   "web_tests",
				Expected: "web tests pass",
				Actual:   "web tests pass",
				Passed:   true,
				Message: fmt.Sprintf(
					"Web tests passed in %v",
					res.Duration,
				),
			},
		)
	}

	if res != nil {
		for _, suite := range res.Suites {
			outputs[suite.Name] = fmt.Sprintf(
				"tests=%d failures=%d time=%.2fs",
				suite.Tests, suite.Failures, suite.Time,
			)
		}
	}

	status := challenge.StatusPassed
	if err != nil {
		status = challenge.StatusFailed
	}

	return c.CreateResult(
		status, start, assertions, metrics, outputs, errMsg,
	), nil
}
