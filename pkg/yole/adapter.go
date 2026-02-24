package yole

import (
	"context"
	"time"
)

// GradleAdapter abstracts Gradle task execution, allowing
// different implementations (CLI subprocess, Docker, mock).
type GradleAdapter interface {
	// RunTask executes a Gradle task and returns the result.
	RunTask(
		ctx context.Context, task string, args ...string,
	) (*GradleRunResult, error)

	// RunTests executes a Gradle test task with optional
	// filter and parses JUnit XML results.
	RunTests(
		ctx context.Context, task string, testFilter string,
	) (*GradleRunResult, error)

	// Available returns true if Gradle is reachable.
	Available(ctx context.Context) bool
}

// ADBAdapter abstracts Android device/emulator management.
type ADBAdapter interface {
	// IsDeviceAvailable checks if a device is connected.
	IsDeviceAvailable(ctx context.Context) (bool, error)

	// InstallAPK installs an APK on the connected device.
	InstallAPK(ctx context.Context, apkPath string) error

	// LaunchApp starts the app on the device.
	LaunchApp(ctx context.Context) error

	// StopApp force-stops the app.
	StopApp(ctx context.Context) error

	// IsAppRunning checks if the app process is active.
	IsAppRunning(ctx context.Context) (bool, error)

	// TakeScreenshot captures the device screen.
	TakeScreenshot(ctx context.Context) ([]byte, error)

	// WaitForApp waits until the app is running or timeout.
	WaitForApp(
		ctx context.Context, timeout time.Duration,
	) error

	// Available returns true if ADB is reachable.
	Available(ctx context.Context) bool
}

// PlaywrightAdapter abstracts browser automation.
type PlaywrightAdapter interface {
	// Initialize sets up the browser instance.
	Initialize(
		ctx context.Context, browserType string,
	) error

	// Navigate goes to the specified URL.
	Navigate(ctx context.Context, url string) error

	// Click clicks an element matching the selector.
	Click(ctx context.Context, selector string) error

	// ClickByText clicks an element containing text.
	ClickByText(ctx context.Context, text string) error

	// IsVisible checks if an element is visible.
	IsVisible(
		ctx context.Context, selector string,
	) (bool, error)

	// Screenshot takes a screenshot.
	Screenshot(ctx context.Context) ([]byte, error)

	// Close shuts down the browser.
	Close(ctx context.Context) error

	// Available returns true if Playwright is installed.
	Available(ctx context.Context) bool
}

// ProcessAdapter abstracts JVM/native process lifecycle.
type ProcessAdapter interface {
	// LaunchJVM starts a JVM application from a JAR file.
	LaunchJVM(
		ctx context.Context, jarPath string, args ...string,
	) error

	// IsRunning checks if the process is alive.
	IsRunning() bool

	// WaitForReady waits until the process is running.
	WaitForReady(
		ctx context.Context, timeout time.Duration,
	) error

	// Stop gracefully terminates the process.
	Stop() error
}
