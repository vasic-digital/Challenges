package userflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultADBPath       = "adb"
	defaultDevicePath    = "/sdcard/challenge_recording.mp4"
	defaultRecordingFPS  = 30
	adbRecordingFileName = "adb_recording.mp4"
)

// ADBRecorderAdapter implements RecorderAdapter by shelling
// out to `adb shell screenrecord` to capture the Android
// device screen. The recording is stored on-device and pulled
// to the local output directory when stopped.
type ADBRecorderAdapter struct {
	adbPath      string
	deviceSerial string
	recording    bool
	recordStart  time.Time
	devicePath   string
	outputDir    string
	cmd          *exec.Cmd
	mu           sync.Mutex
}

// Compile-time interface check.
var _ RecorderAdapter = (*ADBRecorderAdapter)(nil)

// NewADBRecorderAdapter creates an ADBRecorderAdapter. If
// adbPath is empty, it defaults to "adb". The deviceSerial
// is optional; when set, all commands use `adb -s <serial>`.
func NewADBRecorderAdapter(
	adbPath, deviceSerial string,
) *ADBRecorderAdapter {
	if adbPath == "" {
		adbPath = defaultADBPath
	}
	return &ADBRecorderAdapter{
		adbPath:      adbPath,
		deviceSerial: deviceSerial,
		devicePath:   defaultDevicePath,
	}
}

// StartRecording begins recording the device screen by
// launching `adb shell screenrecord` as a background process.
// The process runs until interrupted by StopRecording.
func (a *ADBRecorderAdapter) StartRecording(
	ctx context.Context, config RecordingConfig,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.recording {
		return fmt.Errorf("recording already in progress")
	}

	a.outputDir = config.OutputDir
	if a.outputDir == "" {
		a.outputDir = os.TempDir()
	}

	args := a.deviceArgs(
		"shell", "screenrecord", a.devicePath,
	)

	cmd := exec.CommandContext(
		ctx, a.adbPath, args...,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start screenrecord: %w", err)
	}

	a.cmd = cmd
	a.recording = true
	a.recordStart = time.Now()
	return nil
}

// StopRecording stops the active screenrecord process by
// sending an interrupt signal, waits for it to finish writing,
// then pulls the recording from the device to the local
// output directory. It returns metadata including file size,
// estimated duration, and estimated frame count.
func (a *ADBRecorderAdapter) StopRecording(
	ctx context.Context,
) (*RecordingResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.recording {
		return nil, fmt.Errorf("no recording in progress")
	}

	elapsed := time.Since(a.recordStart)

	// Signal the screenrecord process to stop gracefully.
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Signal(os.Interrupt)
		_ = a.cmd.Wait()
	}

	a.recording = false
	a.cmd = nil

	// Pull the recording from the device.
	localPath := filepath.Join(
		a.outputDir, adbRecordingFileName,
	)
	pullArgs := a.deviceArgs(
		"pull", a.devicePath, localPath,
	)
	if _, err := a.runADB(ctx, pullArgs...); err != nil {
		return nil, fmt.Errorf(
			"pull recording: %w", err,
		)
	}

	// Clean up the on-device file.
	rmArgs := a.deviceArgs(
		"shell", "rm", "-f", a.devicePath,
	)
	_, _ = a.runADB(ctx, rmArgs...)

	// Get local file size.
	var fileSize int64
	if info, err := os.Stat(localPath); err == nil {
		fileSize = info.Size()
	}

	// Estimate frame count from elapsed time and default
	// FPS (screenrecord defaults to ~30fps).
	durationSecs := elapsed.Seconds()
	frameCount := int(
		durationSecs * float64(defaultRecordingFPS),
	)

	return &RecordingResult{
		FilePath:   localPath,
		Duration:   elapsed,
		FrameCount: frameCount,
		FileSize:   fileSize,
	}, nil
}

// IsRecording reports whether a recording is currently active.
func (a *ADBRecorderAdapter) IsRecording() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.recording
}

// Available checks if the adb binary exists and at least one
// device is connected. If a deviceSerial is configured, it
// checks for that specific device.
func (a *ADBRecorderAdapter) Available(
	ctx context.Context,
) bool {
	if _, err := os.Stat(a.adbPath); err != nil {
		if _, err := exec.LookPath(a.adbPath); err != nil {
			return false
		}
	}

	args := a.deviceArgs("devices")
	out, err := a.runADB(ctx, args...)
	if err != nil {
		return false
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" ||
			strings.HasPrefix(line, "List of") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "device" {
			if a.deviceSerial == "" {
				return true
			}
			if fields[0] == a.deviceSerial {
				return true
			}
		}
	}
	return false
}

// deviceArgs prepends `-s <serial>` to the argument list if
// a device serial is configured.
func (a *ADBRecorderAdapter) deviceArgs(
	args ...string,
) []string {
	if a.deviceSerial != "" {
		return append(
			[]string{"-s", a.deviceSerial},
			args...,
		)
	}
	return args
}

// runADB executes an adb command and returns combined output.
func (a *ADBRecorderAdapter) runADB(
	ctx context.Context, args ...string,
) (string, error) {
	cmd := exec.CommandContext(ctx, a.adbPath, args...)
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
