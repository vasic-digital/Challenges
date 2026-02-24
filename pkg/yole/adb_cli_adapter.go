package yole

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ADBCLIAdapter manages Android devices/emulators via ADB
// subprocess.
type ADBCLIAdapter struct {
	DeviceSerial string
	PackageName  string
	ActivityName string
}

// NewADBCLIAdapter creates an ADBCLIAdapter for the Yole
// Android app.
func NewADBCLIAdapter() *ADBCLIAdapter {
	return &ADBCLIAdapter{
		PackageName:  "digital.vasic.yole.android",
		ActivityName: "digital.vasic.yole.android.MainActivity",
	}
}

// IsDeviceAvailable checks if a device or emulator is
// connected.
func (a *ADBCLIAdapter) IsDeviceAvailable(
	ctx context.Context,
) (bool, error) {
	cmd := exec.CommandContext(ctx, "adb", "devices")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf(
			"adb not available: %w", err,
		)
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "device") &&
			!strings.Contains(trimmed, "offline") &&
			trimmed != "" {
			return true, nil
		}
	}
	return false, nil
}

// InstallAPK installs the APK on the connected device.
func (a *ADBCLIAdapter) InstallAPK(
	ctx context.Context, apkPath string,
) error {
	args := []string{"install", "-r", apkPath}
	if a.DeviceSerial != "" {
		args = append(
			[]string{"-s", a.DeviceSerial}, args...,
		)
	}
	cmd := exec.CommandContext(ctx, "adb", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"install failed: %s: %w",
			string(output), err,
		)
	}
	return nil
}

// LaunchApp starts the Yole app on the device.
func (a *ADBCLIAdapter) LaunchApp(
	ctx context.Context,
) error {
	component := fmt.Sprintf(
		"%s/%s", a.PackageName, a.ActivityName,
	)
	args := []string{"shell", "am", "start", "-n", component}
	if a.DeviceSerial != "" {
		args = append(
			[]string{"-s", a.DeviceSerial}, args...,
		)
	}
	cmd := exec.CommandContext(ctx, "adb", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"launch failed: %s: %w",
			string(output), err,
		)
	}
	return nil
}

// StopApp force-stops the Yole app.
func (a *ADBCLIAdapter) StopApp(
	ctx context.Context,
) error {
	args := []string{
		"shell", "am", "force-stop", a.PackageName,
	}
	if a.DeviceSerial != "" {
		args = append(
			[]string{"-s", a.DeviceSerial}, args...,
		)
	}
	cmd := exec.CommandContext(ctx, "adb", args...)
	_, err := cmd.CombinedOutput()
	return err
}

// IsAppRunning checks if the Yole app process is active.
func (a *ADBCLIAdapter) IsAppRunning(
	ctx context.Context,
) (bool, error) {
	args := []string{"shell", "pidof", a.PackageName}
	if a.DeviceSerial != "" {
		args = append(
			[]string{"-s", a.DeviceSerial}, args...,
		)
	}
	cmd := exec.CommandContext(ctx, "adb", args...)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// TakeScreenshot captures the device screen.
func (a *ADBCLIAdapter) TakeScreenshot(
	ctx context.Context,
) ([]byte, error) {
	tmpPath := fmt.Sprintf(
		"/sdcard/screenshot_%d.png",
		time.Now().UnixMilli(),
	)
	args := []string{"shell", "screencap", "-p", tmpPath}
	if a.DeviceSerial != "" {
		args = append(
			[]string{"-s", a.DeviceSerial}, args...,
		)
	}

	cmd := exec.CommandContext(ctx, "adb", args...)
	if _, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf(
			"screencap failed: %w", err,
		)
	}

	pullArgs := []string{
		"pull", tmpPath, "/tmp/screenshot.png",
	}
	if a.DeviceSerial != "" {
		pullArgs = append(
			[]string{"-s", a.DeviceSerial}, pullArgs...,
		)
	}
	cmd = exec.CommandContext(ctx, "adb", pullArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf(
			"pull screenshot failed: %w", err,
		)
	}

	rmArgs := []string{"shell", "rm", tmpPath}
	_ = exec.CommandContext(ctx, "adb", rmArgs...).Run()

	data, err := os.ReadFile("/tmp/screenshot.png")
	if err != nil {
		return nil, fmt.Errorf(
			"read screenshot failed: %w", err,
		)
	}
	return data, nil
}

// WaitForApp waits until the app is running or timeout.
func (a *ADBCLIAdapter) WaitForApp(
	ctx context.Context, timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, _ := a.IsAppRunning(ctx)
		if running {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf(
		"app did not start within %v", timeout,
	)
}

// Available checks if adb is in PATH.
func (a *ADBCLIAdapter) Available(
	ctx context.Context,
) bool {
	_, err := exec.LookPath("adb")
	return err == nil
}
