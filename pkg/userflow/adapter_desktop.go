package userflow

import (
	"context"
	"time"
)

// DesktopAdapter defines the interface for desktop application
// testing. Implementations may wrap Tauri's WebDriver support,
// Wails testing, or native OS accessibility APIs.
type DesktopAdapter interface {
	// LaunchApp starts the desktop application with the given
	// configuration.
	LaunchApp(
		ctx context.Context, config DesktopAppConfig,
	) error

	// IsAppRunning checks if the desktop application process
	// is currently running.
	IsAppRunning(ctx context.Context) (bool, error)

	// Navigate loads the given URL in the application's
	// webview (for Tauri/Wails/Electron apps).
	Navigate(ctx context.Context, url string) error

	// Click performs a click on the element matching the
	// selector within the application's webview.
	Click(ctx context.Context, selector string) error

	// Fill types a value into the input matching the selector
	// within the application's webview.
	Fill(
		ctx context.Context, selector, value string,
	) error

	// IsVisible returns whether the element matching the
	// selector is currently visible.
	IsVisible(
		ctx context.Context, selector string,
	) (bool, error)

	// WaitForSelector waits until an element matching the
	// selector appears within the application's webview,
	// up to the given timeout.
	WaitForSelector(
		ctx context.Context,
		selector string,
		timeout time.Duration,
	) error

	// Screenshot captures the current application window as
	// a PNG image.
	Screenshot(ctx context.Context) ([]byte, error)

	// InvokeCommand sends an IPC command to the desktop app's
	// backend (e.g., Tauri invoke) and returns the response.
	InvokeCommand(
		ctx context.Context,
		command string,
		args ...string,
	) (string, error)

	// WaitForWindow waits until the application's main window
	// is visible, up to the given timeout.
	WaitForWindow(
		ctx context.Context, timeout time.Duration,
	) error

	// Close shuts down the application and releases resources.
	Close(ctx context.Context) error

	// Available returns true if the desktop testing
	// environment is ready.
	Available(ctx context.Context) bool
}
