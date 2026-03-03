package userflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// EspressoAdapter implements MobileAdapter for running
// Android instrumented tests via the Espresso framework.
// It combines ADB for device interaction and Gradle for
// building and executing connectedAndroidTest tasks.
type EspressoAdapter struct {
	projectDir        string
	config            MobileConfig
	gradleWrapper     string
	module            string
	testRunner        string
	instrumentArgs    map[string]string
}

// Compile-time interface check.
var _ MobileAdapter = (*EspressoAdapter)(nil)

// EspressoOption configures an EspressoAdapter.
type EspressoOption func(*EspressoAdapter)

// WithEspressoGradleWrapper sets a custom path to the
// Gradle wrapper script. Defaults to ./gradlew in the
// project directory.
func WithEspressoGradleWrapper(
	path string,
) EspressoOption {
	return func(a *EspressoAdapter) {
		a.gradleWrapper = path
	}
}

// WithEspressoModule sets the Gradle module prefix
// (e.g., ":app") for multi-module projects.
func WithEspressoModule(module string) EspressoOption {
	return func(a *EspressoAdapter) {
		a.module = module
	}
}

// WithEspressoTestRunner sets a custom instrumentation
// test runner class. Defaults to
// "androidx.test.runner.AndroidJUnitRunner".
func WithEspressoTestRunner(
	runner string,
) EspressoOption {
	return func(a *EspressoAdapter) {
		a.testRunner = runner
	}
}

// WithEspressoInstrumentationArgs sets additional
// instrumentation arguments passed to the test runner
// as key-value pairs.
func WithEspressoInstrumentationArgs(
	args map[string]string,
) EspressoOption {
	return func(a *EspressoAdapter) {
		a.instrumentArgs = args
	}
}

// NewEspressoAdapter creates an EspressoAdapter with the
// given project directory and mobile configuration.
// Options may override the Gradle wrapper path, module,
// test runner, and instrumentation arguments.
func NewEspressoAdapter(
	projectDir string,
	config MobileConfig,
	opts ...EspressoOption,
) *EspressoAdapter {
	a := &EspressoAdapter{
		projectDir: projectDir,
		config:     config,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// gradlePath returns the resolved path to the Gradle
// wrapper.
func (a *EspressoAdapter) gradlePath() string {
	if a.gradleWrapper != "" {
		return a.gradleWrapper
	}
	return filepath.Join(a.projectDir, "gradlew")
}

// taskName prepends the module prefix to a task name when
// a module is configured (e.g., ":app:installDebug").
func (a *EspressoAdapter) taskName(
	task string,
) string {
	if a.module != "" {
		return a.module + ":" + task
	}
	return task
}

// resolvedRunner returns the configured test runner or
// the default AndroidJUnitRunner.
func (a *EspressoAdapter) resolvedRunner() string {
	if a.testRunner != "" {
		return a.testRunner
	}
	return "androidx.test.runner.AndroidJUnitRunner"
}

// IsDeviceAvailable checks if a connected Android device
// is ready by parsing `adb devices` output.
func (a *EspressoAdapter) IsDeviceAvailable(
	ctx context.Context,
) (bool, error) {
	args := a.deviceArgs("devices")
	out, err := a.runADB(ctx, args...)
	if err != nil {
		return false, fmt.Errorf(
			"check device: %w", err,
		)
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(
			line, "List of",
		) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "device" {
			return true, nil
		}
	}
	return false, nil
}

// InstallApp builds and installs the debug APK onto the
// connected device via `./gradlew installDebug`.
func (a *EspressoAdapter) InstallApp(
	ctx context.Context, _ string,
) error {
	task := a.taskName("installDebug")
	_, err := a.runGradle(ctx, task)
	if err != nil {
		return fmt.Errorf("install app: %w", err)
	}
	return nil
}

// LaunchApp starts the configured application on the device
// using `adb shell am start -n package/activity`.
func (a *EspressoAdapter) LaunchApp(
	ctx context.Context,
) error {
	component := fmt.Sprintf(
		"%s/%s",
		a.config.PackageName, a.config.ActivityName,
	)
	args := a.deviceArgs(
		"shell", "am", "start", "-n", component,
	)
	_, err := a.runADB(ctx, args...)
	if err != nil {
		return fmt.Errorf("launch app: %w", err)
	}
	return nil
}

// StopApp force-stops the configured application on the
// device using `adb shell am force-stop`.
func (a *EspressoAdapter) StopApp(
	ctx context.Context,
) error {
	args := a.deviceArgs(
		"shell", "am", "force-stop",
		a.config.PackageName,
	)
	_, err := a.runADB(ctx, args...)
	if err != nil {
		return fmt.Errorf("stop app: %w", err)
	}
	return nil
}

// IsAppRunning checks if the configured package has a
// running process via `adb shell pidof`.
func (a *EspressoAdapter) IsAppRunning(
	ctx context.Context,
) (bool, error) {
	args := a.deviceArgs(
		"shell", "pidof", a.config.PackageName,
	)
	out, err := a.runADB(ctx, args...)
	if err != nil {
		// pidof returns exit code 1 when not found.
		return false, nil
	}
	return strings.TrimSpace(out) != "", nil
}

// TakeScreenshot captures the device screen using
// `adb exec-out screencap -p` and returns the PNG data.
func (a *EspressoAdapter) TakeScreenshot(
	ctx context.Context,
) ([]byte, error) {
	args := a.deviceArgs(
		"exec-out", "screencap", "-p",
	)
	cmd := exec.CommandContext(ctx, "adb", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf(
			"screencap: %s: %w",
			stderr.String(), err,
		)
	}
	return stdout.Bytes(), nil
}

// Tap performs a tap gesture at screen coordinates (x, y)
// using `adb shell input tap`.
func (a *EspressoAdapter) Tap(
	ctx context.Context, x, y int,
) error {
	args := a.deviceArgs(
		"shell", "input", "tap",
		fmt.Sprintf("%d", x),
		fmt.Sprintf("%d", y),
	)
	_, err := a.runADB(ctx, args...)
	if err != nil {
		return fmt.Errorf("tap: %w", err)
	}
	return nil
}

// SendKeys types the given text on the device using
// `adb shell input text`.
func (a *EspressoAdapter) SendKeys(
	ctx context.Context, text string,
) error {
	// Escape spaces for adb shell input.
	escaped := strings.ReplaceAll(text, " ", "%s")
	args := a.deviceArgs(
		"shell", "input", "text", escaped,
	)
	_, err := a.runADB(ctx, args...)
	if err != nil {
		return fmt.Errorf("send keys: %w", err)
	}
	return nil
}

// PressKey sends a key event to the device using
// `adb shell input keyevent`.
func (a *EspressoAdapter) PressKey(
	ctx context.Context, keycode string,
) error {
	args := a.deviceArgs(
		"shell", "input", "keyevent", keycode,
	)
	_, err := a.runADB(ctx, args...)
	if err != nil {
		return fmt.Errorf("press key: %w", err)
	}
	return nil
}

// WaitForApp polls IsAppRunning every 500ms until the
// application is detected or the timeout expires.
func (a *EspressoAdapter) WaitForApp(
	ctx context.Context, timeout time.Duration,
) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		running, err := a.IsAppRunning(ctx)
		if err == nil && running {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"wait for app: %w", ctx.Err(),
			)
		case <-deadline:
			return fmt.Errorf(
				"wait for app: timed out after %s",
				timeout,
			)
		case <-ticker.C:
			// continue polling
		}
	}
}

// RunInstrumentedTests runs Espresso instrumented tests
// via `./gradlew connectedDebugAndroidTest`. When a
// testClass is specified it is passed via --tests. Results
// are parsed from JUnit XML in the standard connected test
// output directory.
func (a *EspressoAdapter) RunInstrumentedTests(
	ctx context.Context, testClass string,
) (*TestResult, error) {
	task := a.taskName("connectedDebugAndroidTest")
	args := []string{task}

	// Apply instrumentation arguments.
	for k, v := range a.instrumentArgs {
		args = append(
			args,
			fmt.Sprintf(
				"-Pandroid.testInstrumentationRunnerArguments.%s=%s",
				k, v,
			),
		)
	}

	if testClass != "" {
		args = append(args, "--tests", testClass)
	}

	start := time.Now()
	output, runErr := a.runGradle(ctx, args...)
	elapsed := time.Since(start)

	// Parse JUnit XML from standard output directories.
	suites := a.collectConnectedTestResults()
	if len(suites) > 0 {
		result := JUnitToTestResult(
			suites, elapsed, output,
		)
		return result, runErr
	}

	// No JUnit XML found; return basic result.
	result := &TestResult{
		Duration: elapsed,
		Output:   output,
	}
	if runErr != nil {
		result.TotalFailed = 1
	}
	return result, runErr
}

// collectConnectedTestResults searches standard connected
// test JUnit XML output directories and parses all found
// XML files.
func (a *EspressoAdapter) collectConnectedTestResults() []JUnitTestSuite {
	searchDirs := []string{
		"build/outputs/androidTest-results/connected",
		"app/build/outputs/androidTest-results/connected",
	}
	if a.module != "" {
		mod := strings.TrimPrefix(a.module, ":")
		searchDirs = append(
			searchDirs,
			filepath.Join(
				mod,
				"build/outputs/androidTest-results/connected",
			),
		)
	}

	var allSuites []JUnitTestSuite
	for _, base := range searchDirs {
		dir := filepath.Join(a.projectDir, base)
		// Try direct XML files first.
		matches, err := filepath.Glob(
			filepath.Join(dir, "*.xml"),
		)
		if err != nil || len(matches) == 0 {
			// Try one level of nesting.
			matches, _ = filepath.Glob(
				filepath.Join(dir, "*", "*.xml"),
			)
		}
		for _, m := range matches {
			data, err := os.ReadFile(m)
			if err != nil {
				continue
			}
			suites, err := ParseJUnitXML(data)
			if err != nil {
				continue
			}
			allSuites = append(allSuites, suites...)
		}
	}
	return allSuites
}

// Close is a no-op for the Espresso adapter. Device
// connections do not require explicit cleanup.
func (a *EspressoAdapter) Close(
	_ context.Context,
) error {
	return nil
}

// Available returns true if both the `adb` binary and
// the Gradle wrapper are accessible and functional.
func (a *EspressoAdapter) Available(
	ctx context.Context,
) bool {
	// Check adb is in PATH.
	if _, err := exec.LookPath("adb"); err != nil {
		return false
	}
	// Check gradle wrapper exists and runs.
	wrapper := a.gradlePath()
	if _, err := os.Stat(wrapper); err != nil {
		return false
	}
	cmd := exec.CommandContext(
		ctx, wrapper, "--version",
	)
	cmd.Dir = a.projectDir
	return cmd.Run() == nil
}

// deviceArgs prepends `-s <serial>` to the argument list
// if a device serial is configured.
func (a *EspressoAdapter) deviceArgs(
	args ...string,
) []string {
	if a.config.DeviceSerial != "" {
		return append(
			[]string{"-s", a.config.DeviceSerial},
			args...,
		)
	}
	return args
}

// runADB executes an adb command and returns combined
// output.
func (a *EspressoAdapter) runADB(
	ctx context.Context, args ...string,
) (string, error) {
	cmd := exec.CommandContext(ctx, "adb", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf(
			"adb %v: %w", args, err,
		)
	}
	return buf.String(), nil
}

// runGradle executes a Gradle wrapper command with
// resource limits (nice -n 19, ionice -c 3) and returns
// the combined output.
func (a *EspressoAdapter) runGradle(
	ctx context.Context, args ...string,
) (string, error) {
	wrapper := a.gradlePath()

	// Build resource-limited command:
	// nice -n 19 ionice -c 3 ./gradlew <args>
	cmdArgs := []string{
		"-n", "19",
		"ionice", "-c", "3",
		wrapper,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(
		ctx, "nice", cmdArgs...,
	)
	cmd.Dir = a.projectDir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf(
			"espresso gradle %v: %w", args, err,
		)
	}
	return string(out), nil
}
