package userflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ADBCLIAdapter implements MobileAdapter by shelling out to
// ADB (Android Debug Bridge) commands. Package and activity
// names are configurable via MobileConfig.
type ADBCLIAdapter struct {
	config MobileConfig
}

// Compile-time interface check.
var _ MobileAdapter = (*ADBCLIAdapter)(nil)

// NewADBCLIAdapter creates an ADBCLIAdapter with the given
// mobile configuration. PackageName and ActivityName must be
// set in the config.
func NewADBCLIAdapter(config MobileConfig) *ADBCLIAdapter {
	return &ADBCLIAdapter{config: config}
}

// IsDeviceAvailable checks if a connected device is ready
// by parsing `adb devices` output for a "device" status.
func (a *ADBCLIAdapter) IsDeviceAvailable(
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

// InstallApp installs an APK onto the connected device.
func (a *ADBCLIAdapter) InstallApp(
	ctx context.Context, appPath string,
) error {
	args := a.deviceArgs("install", "-r", appPath)
	_, err := a.runADB(ctx, args...)
	if err != nil {
		return fmt.Errorf("install app: %w", err)
	}
	return nil
}

// LaunchApp starts the configured application using
// `am start`.
func (a *ADBCLIAdapter) LaunchApp(
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

// StopApp force-stops the configured application.
func (a *ADBCLIAdapter) StopApp(
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

// IsAppRunning checks if the configured package has a running
// process via `pidof`.
func (a *ADBCLIAdapter) IsAppRunning(
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

// TakeScreenshot captures the device screen, pulls the file
// to the host, reads it, and cleans up.
func (a *ADBCLIAdapter) TakeScreenshot(
	ctx context.Context,
) ([]byte, error) {
	remotePath := "/sdcard/screenshot.png"

	// Capture screenshot on device.
	captureArgs := a.deviceArgs(
		"shell", "screencap", "-p", remotePath,
	)
	if _, err := a.runADB(ctx, captureArgs...); err != nil {
		return nil, fmt.Errorf("screencap: %w", err)
	}

	// Pull to host.
	tmpFile := fmt.Sprintf(
		"/tmp/adb-screenshot-%d.png",
		time.Now().UnixNano(),
	)
	pullArgs := a.deviceArgs("pull", remotePath, tmpFile)
	if _, err := a.runADB(ctx, pullArgs...); err != nil {
		return nil, fmt.Errorf("pull screenshot: %w", err)
	}

	// Read the local file.
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf(
			"read screenshot: %w", err,
		)
	}

	// Cleanup.
	os.Remove(tmpFile)
	cleanArgs := a.deviceArgs(
		"shell", "rm", "-f", remotePath,
	)
	_, _ = a.runADB(ctx, cleanArgs...)

	return data, nil
}

// Tap performs a tap gesture at screen coordinates (x, y).
func (a *ADBCLIAdapter) Tap(
	ctx context.Context, x, y int,
) error {
	args := a.deviceArgs(
		"shell", "input", "tap",
		fmt.Sprintf("%d", x), fmt.Sprintf("%d", y),
	)
	_, err := a.runADB(ctx, args...)
	if err != nil {
		return fmt.Errorf("tap: %w", err)
	}
	return nil
}

// SendKeys types the given text on the device.
func (a *ADBCLIAdapter) SendKeys(
	ctx context.Context, text string,
) error {
	// Escape special characters for adb shell input.
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

// PressKey sends a key event to the device.
func (a *ADBCLIAdapter) PressKey(
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

// WaitForApp polls IsAppRunning every 500ms until the app is
// detected or the timeout expires.
func (a *ADBCLIAdapter) WaitForApp(
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

// RunInstrumentedTests runs Android instrumented tests via
// `am instrument` and parses the output.
func (a *ADBCLIAdapter) RunInstrumentedTests(
	ctx context.Context, testClass string,
) (*TestResult, error) {
	runner := a.config.PackageName +
		".test/androidx.test.runner.AndroidJUnitRunner"

	instrumentArgs := a.deviceArgs(
		"shell", "am", "instrument", "-w",
	)
	if testClass != "" {
		instrumentArgs = append(
			instrumentArgs, "-e", "class", testClass,
		)
	}
	instrumentArgs = append(instrumentArgs, runner)

	start := time.Now()
	out, err := a.runADB(ctx, instrumentArgs...)
	elapsed := time.Since(start)

	result := parseInstrumentOutput(out, elapsed)
	return result, err
}

// parseInstrumentOutput does basic parsing of the `am
// instrument` output to extract test counts.
func parseInstrumentOutput(
	output string, elapsed time.Duration,
) *TestResult {
	result := &TestResult{
		Duration: elapsed,
		Output:   output,
	}

	suite := TestSuite{Name: "instrumented"}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "OK (") {
			// e.g. "OK (5 tests)"
			var n int
			if _, err := fmt.Sscanf(
				line, "OK (%d tests)", &n,
			); err == nil {
				suite.Tests = n
			}
		}
		if strings.Contains(line, "FAILURES!!!") {
			suite.Failures++
		}
	}

	if suite.Tests > 0 || suite.Failures > 0 {
		result.Suites = append(result.Suites, suite)
		result.TotalTests = suite.Tests
		result.TotalFailed = suite.Failures
	}

	return result
}

// Close is a no-op for ADB. The device connection does not
// need explicit cleanup.
func (a *ADBCLIAdapter) Close(
	_ context.Context,
) error {
	return nil
}

// Available returns true if the `adb` binary is found in PATH.
func (a *ADBCLIAdapter) Available(
	_ context.Context,
) bool {
	_, err := exec.LookPath("adb")
	return err == nil
}

// deviceArgs prepends `-s <serial>` to the argument list if
// a device serial is configured.
func (a *ADBCLIAdapter) deviceArgs(
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

// runADB executes an adb command and returns combined output.
func (a *ADBCLIAdapter) runADB(
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
