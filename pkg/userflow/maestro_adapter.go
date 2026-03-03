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

// MaestroFlow represents a Maestro YAML flow definition
// consisting of a sequence of commands executed on a mobile
// device.
type MaestroFlow struct {
	AppID    string   `yaml:"appId,omitempty"`
	Commands []string `yaml:"-"`
}

// toYAML serializes the flow into a Maestro-compatible YAML
// string. Each command is written as a top-level list entry.
func (f *MaestroFlow) toYAML() string {
	var b strings.Builder
	if f.AppID != "" {
		b.WriteString(
			fmt.Sprintf("appId: %s\n", f.AppID),
		)
	}
	b.WriteString("---\n")
	for _, cmd := range f.Commands {
		b.WriteString(fmt.Sprintf("- %s\n", cmd))
	}
	return b.String()
}

// MaestroCLIAdapter implements MobileAdapter using the Maestro
// CLI, a YAML-driven mobile testing framework. Each mobile
// action generates a YAML flow file in a temporary directory
// and invokes `maestro test` to execute it.
//
// Usage:
//
//	adapter := NewMaestroCLIAdapter(MobileConfig{
//	    PackageName:  "com.example.app",
//	    DeviceSerial: "emulator-5554",
//	})
//	available := adapter.Available(ctx)
type MaestroCLIAdapter struct {
	config  MobileConfig
	tempDir string
}

// Compile-time interface check.
var _ MobileAdapter = (*MaestroCLIAdapter)(nil)

// NewMaestroCLIAdapter creates a MaestroCLIAdapter with the
// given mobile configuration.
func NewMaestroCLIAdapter(
	config MobileConfig,
) *MaestroCLIAdapter {
	return &MaestroCLIAdapter{config: config}
}

// IsDeviceAvailable checks if a connected device is ready by
// parsing the output of `maestro devices`.
func (a *MaestroCLIAdapter) IsDeviceAvailable(
	ctx context.Context,
) (bool, error) {
	out, err := a.runMaestro(ctx, "devices")
	if err != nil {
		return false, fmt.Errorf(
			"check device: %w", err,
		)
	}

	// Maestro lists connected devices. If a specific serial
	// is configured, check for it; otherwise check for any
	// device line.
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if a.config.DeviceSerial != "" {
			if strings.Contains(
				line, a.config.DeviceSerial,
			) {
				return true, nil
			}
		} else if strings.Contains(line, "Connected") ||
			strings.Contains(line, "device") ||
			strings.Contains(line, "emulator") {
			return true, nil
		}
	}
	return false, nil
}

// InstallApp installs the application from the given path onto
// the device by generating a flow with an installApp command.
func (a *MaestroCLIAdapter) InstallApp(
	ctx context.Context, appPath string,
) error {
	flow := &MaestroFlow{
		AppID: a.config.PackageName,
		Commands: []string{
			fmt.Sprintf(
				"installApp: %s",
				escapeYAMLString(appPath),
			),
		},
	}

	_, err := a.runFlow(ctx, flow)
	if err != nil {
		return fmt.Errorf("install app: %w", err)
	}
	return nil
}

// LaunchApp starts the configured application on the device
// by generating a flow with a launchApp command.
func (a *MaestroCLIAdapter) LaunchApp(
	ctx context.Context,
) error {
	flow := &MaestroFlow{
		AppID: a.config.PackageName,
		Commands: []string{
			fmt.Sprintf(
				"launchApp: %s",
				escapeYAMLString(a.config.PackageName),
			),
		},
	}

	_, err := a.runFlow(ctx, flow)
	if err != nil {
		return fmt.Errorf("launch app: %w", err)
	}
	return nil
}

// StopApp stops the running application on the device by
// generating a flow with a stopApp command.
func (a *MaestroCLIAdapter) StopApp(
	ctx context.Context,
) error {
	flow := &MaestroFlow{
		AppID: a.config.PackageName,
		Commands: []string{
			fmt.Sprintf(
				"stopApp: %s",
				escapeYAMLString(a.config.PackageName),
			),
		},
	}

	_, err := a.runFlow(ctx, flow)
	if err != nil {
		return fmt.Errorf("stop app: %w", err)
	}
	return nil
}

// IsAppRunning checks if the application is currently running
// by inspecting the Maestro view hierarchy for the configured
// package name.
func (a *MaestroCLIAdapter) IsAppRunning(
	ctx context.Context,
) (bool, error) {
	out, err := a.runMaestro(ctx, "hierarchy")
	if err != nil {
		// hierarchy command failure indicates no app visible.
		return false, nil
	}

	return strings.Contains(
		out, a.config.PackageName,
	), nil
}

// TakeScreenshot captures the current device screen as a PNG
// image using `maestro screenshot`.
func (a *MaestroCLIAdapter) TakeScreenshot(
	ctx context.Context,
) ([]byte, error) {
	if err := a.ensureTempDir(); err != nil {
		return nil, err
	}

	screenshotFile := filepath.Join(
		a.tempDir,
		fmt.Sprintf(
			"screenshot-%d.png",
			time.Now().UnixNano(),
		),
	)

	_, err := a.runMaestro(
		ctx, "screenshot", screenshotFile,
	)
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}

	data, err := os.ReadFile(screenshotFile)
	if err != nil {
		return nil, fmt.Errorf(
			"read screenshot: %w", err,
		)
	}

	os.Remove(screenshotFile)
	return data, nil
}

// Tap performs a tap gesture at the given screen coordinates
// by generating a flow with a tapOn point command.
func (a *MaestroCLIAdapter) Tap(
	ctx context.Context, x, y int,
) error {
	flow := &MaestroFlow{
		AppID: a.config.PackageName,
		Commands: []string{
			fmt.Sprintf(
				`tapOn:
    point: "%d,%d"`, x, y,
			),
		},
	}

	_, err := a.runFlow(ctx, flow)
	if err != nil {
		return fmt.Errorf("tap: %w", err)
	}
	return nil
}

// SendKeys types the given text into the currently focused
// input by generating a flow with an inputText command.
func (a *MaestroCLIAdapter) SendKeys(
	ctx context.Context, text string,
) error {
	flow := &MaestroFlow{
		AppID: a.config.PackageName,
		Commands: []string{
			fmt.Sprintf(
				"inputText: %s",
				escapeYAMLString(text),
			),
		},
	}

	_, err := a.runFlow(ctx, flow)
	if err != nil {
		return fmt.Errorf("send keys: %w", err)
	}
	return nil
}

// PressKey sends a key event to the device by generating a
// flow with a pressKey command. Supports Maestro key names
// such as "back", "home", "enter".
func (a *MaestroCLIAdapter) PressKey(
	ctx context.Context, keycode string,
) error {
	flow := &MaestroFlow{
		AppID: a.config.PackageName,
		Commands: []string{
			fmt.Sprintf(
				"pressKey: %s",
				escapeYAMLString(keycode),
			),
		},
	}

	_, err := a.runFlow(ctx, flow)
	if err != nil {
		return fmt.Errorf("press key: %w", err)
	}
	return nil
}

// WaitForApp waits until the application is fully launched and
// responsive by repeatedly running a flow that asserts the app
// has launched, up to the given timeout.
func (a *MaestroCLIAdapter) WaitForApp(
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

// RunInstrumentedTests is not applicable for Maestro. Returns
// a nil TestResult since Maestro uses its own YAML-based test
// format rather than instrumented test runners.
func (a *MaestroCLIAdapter) RunInstrumentedTests(
	_ context.Context, _ string,
) (*TestResult, error) {
	return nil, nil
}

// Close removes the temporary flow directory and releases
// resources.
func (a *MaestroCLIAdapter) Close(
	_ context.Context,
) error {
	if a.tempDir != "" {
		os.RemoveAll(a.tempDir)
		a.tempDir = ""
	}
	return nil
}

// Available returns true if the `maestro` CLI binary is found
// in PATH.
func (a *MaestroCLIAdapter) Available(
	_ context.Context,
) bool {
	cmd := exec.Command("maestro", "--version")
	err := cmd.Run()
	return err == nil
}

// --- internal helpers ---

// ensureTempDir creates the temporary directory for flow
// files if it does not already exist.
func (a *MaestroCLIAdapter) ensureTempDir() error {
	if a.tempDir != "" {
		return nil
	}
	dir, err := os.MkdirTemp("", "maestro-flows-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	a.tempDir = dir
	return nil
}

// runFlow writes a MaestroFlow to a temporary YAML file and
// executes it via `maestro test`.
func (a *MaestroCLIAdapter) runFlow(
	ctx context.Context, flow *MaestroFlow,
) (string, error) {
	if err := a.ensureTempDir(); err != nil {
		return "", err
	}

	flowFile := filepath.Join(
		a.tempDir,
		fmt.Sprintf(
			"flow_%d.yaml", time.Now().UnixNano(),
		),
	)
	yaml := flow.toYAML()
	if err := os.WriteFile(
		flowFile, []byte(yaml), 0644,
	); err != nil {
		return "", fmt.Errorf(
			"write flow file: %w", err,
		)
	}
	defer os.Remove(flowFile)

	args := []string{"test", flowFile}
	if a.config.DeviceSerial != "" {
		args = append(
			args,
			"--device", a.config.DeviceSerial,
		)
	}

	return a.runMaestro(ctx, args...)
}

// runMaestro executes a maestro CLI command and returns the
// combined output.
func (a *MaestroCLIAdapter) runMaestro(
	ctx context.Context, args ...string,
) (string, error) {
	cmd := exec.CommandContext(
		ctx, "maestro", args...,
	)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf(
			"maestro %v: %w\noutput: %s",
			args, err, buf.String(),
		)
	}
	return buf.String(), nil
}

// escapeYAMLString wraps a value in double quotes if it
// contains special YAML characters, otherwise returns it
// as-is.
func escapeYAMLString(s string) string {
	if strings.ContainsAny(s, ": #[]{}|>&*!%@`'\"\\") {
		escaped := strings.ReplaceAll(s, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}
	return s
}
