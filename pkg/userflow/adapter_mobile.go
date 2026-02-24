package userflow

import (
	"context"
	"time"
)

// MobileAdapter defines the interface for mobile device testing.
// Implementations may wrap ADB (Android), XCUITest (iOS), or
// Appium for cross-platform mobile automation.
type MobileAdapter interface {
	// IsDeviceAvailable checks if the configured device is
	// connected and ready.
	IsDeviceAvailable(
		ctx context.Context,
	) (bool, error)

	// InstallApp installs the application from the given path
	// (APK, AAB, or IPA) onto the device.
	InstallApp(
		ctx context.Context, appPath string,
	) error

	// LaunchApp starts the configured application on the
	// device.
	LaunchApp(ctx context.Context) error

	// StopApp stops the running application on the device.
	StopApp(ctx context.Context) error

	// IsAppRunning checks if the application is currently
	// running on the device.
	IsAppRunning(ctx context.Context) (bool, error)

	// TakeScreenshot captures the current device screen as
	// a PNG image.
	TakeScreenshot(ctx context.Context) ([]byte, error)

	// Tap performs a tap gesture at the given screen
	// coordinates.
	Tap(ctx context.Context, x, y int) error

	// SendKeys types the given text into the currently
	// focused input.
	SendKeys(ctx context.Context, text string) error

	// PressKey sends a key event to the device (e.g.,
	// "KEYCODE_BACK", "KEYCODE_HOME").
	PressKey(ctx context.Context, keycode string) error

	// WaitForApp waits until the application is fully
	// launched and responsive, up to the given timeout.
	WaitForApp(
		ctx context.Context, timeout time.Duration,
	) error

	// RunInstrumentedTests runs instrumented tests on the
	// device, optionally filtered by test class.
	RunInstrumentedTests(
		ctx context.Context, testClass string,
	) (*TestResult, error)

	// Close disconnects from the device and releases
	// resources.
	Close(ctx context.Context) error

	// Available returns true if the mobile automation tool
	// is installed and a device is accessible.
	Available(ctx context.Context) bool
}
