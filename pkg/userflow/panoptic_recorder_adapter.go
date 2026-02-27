package userflow

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// PanopticRecorderAdapter implements RecorderAdapter by
// invoking the Panoptic CLI's record subcommand.
type PanopticRecorderAdapter struct {
	binaryPath string
	sessionID  string
	recording  bool
	cmd        *exec.Cmd
}

// Compile-time interface check.
var _ RecorderAdapter = (*PanopticRecorderAdapter)(nil)

// NewPanopticRecorderAdapter creates a PanopticRecorderAdapter
// that wraps the given Panoptic binary.
func NewPanopticRecorderAdapter(
	binaryPath string,
) *PanopticRecorderAdapter {
	return &PanopticRecorderAdapter{binaryPath: binaryPath}
}

// StartRecording begins a recording session by launching
// `panoptic record start` as a background process. The
// record start process blocks (stays alive while recording),
// so it is started with cmd.Start() and the session ID is
// read asynchronously from stdout.
func (a *PanopticRecorderAdapter) StartRecording(
	ctx context.Context, config RecordingConfig,
) error {
	if a.recording {
		return fmt.Errorf("recording already in progress")
	}

	args := []string{
		"record", "start",
		"--url", config.URL,
		"--output", config.OutputDir,
		"--fps", fmt.Sprintf("%d", config.MaxFPS),
		"--max-width", fmt.Sprintf("%d", config.MaxWidth),
		"--max-height", fmt.Sprintf("%d", config.MaxHeight),
		fmt.Sprintf("--headless=%t", config.Headless),
	}

	cmd := exec.CommandContext(ctx, a.binaryPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start recording: %w", err)
	}

	// Read the session ID from the first line of stdout.
	// The process continues running in the background.
	sessionCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			sessionCh <- strings.TrimSpace(scanner.Text())
		} else {
			if scanErr := scanner.Err(); scanErr != nil {
				errCh <- fmt.Errorf(
					"read session id: %w", scanErr,
				)
			} else {
				errCh <- fmt.Errorf(
					"read session id: unexpected EOF",
				)
			}
		}
	}()

	select {
	case sid := <-sessionCh:
		a.sessionID = sid
		a.recording = true
		a.cmd = cmd
		return nil
	case readErr := <-errCh:
		_ = cmd.Process.Kill()
		return readErr
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return fmt.Errorf(
			"start recording: %w", ctx.Err(),
		)
	}
}

// stopResult is the JSON structure returned by
// `panoptic record stop`.
type stopResult struct {
	FilePath   string `json:"file_path"`
	DurationMs int64  `json:"duration_ms"`
	FrameCount int    `json:"frame_count"`
	FileSize   int64  `json:"file_size"`
}

// StopRecording stops the active recording session by running
// `panoptic record stop --session <id>` and returns the
// recording result parsed from JSON output.
func (a *PanopticRecorderAdapter) StopRecording(
	ctx context.Context,
) (*RecordingResult, error) {
	if !a.recording {
		return nil, fmt.Errorf("no recording in progress")
	}

	args := []string{
		"record", "stop",
		"--session", a.sessionID,
	}

	cmd := exec.CommandContext(ctx, a.binaryPath, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("stop recording: %w", err)
	}

	var sr stopResult
	if err := json.Unmarshal(out, &sr); err != nil {
		return nil, fmt.Errorf(
			"parse stop result: %w", err,
		)
	}

	a.recording = false
	a.sessionID = ""
	a.cmd = nil

	return &RecordingResult{
		FilePath:   sr.FilePath,
		Duration:   time.Duration(sr.DurationMs) * time.Millisecond,
		FrameCount: sr.FrameCount,
		FileSize:   sr.FileSize,
	}, nil
}

// IsRecording reports whether a recording is currently active.
func (a *PanopticRecorderAdapter) IsRecording() bool {
	return a.recording
}

// Available checks if the Panoptic binary exists on disk or
// can be found in PATH.
func (a *PanopticRecorderAdapter) Available(
	_ context.Context,
) bool {
	if _, err := os.Stat(a.binaryPath); err == nil {
		return true
	}
	if _, err := exec.LookPath(a.binaryPath); err == nil {
		return true
	}
	return false
}

// Reset stops any in-progress recording and resets the adapter state.
// This should be called before starting a new challenge to ensure
// no lingering recording state from previous failed challenges.
func (a *PanopticRecorderAdapter) Reset(ctx context.Context) error {
	// Try to stop any running recording
	if a.recording || a.cmd != nil {
		// Try to stop via CLI
		stopCmd := exec.CommandContext(ctx, a.binaryPath, "record", "stop")
		_ = stopCmd.Run()

		// Kill the process if still running
		if a.cmd != nil && a.cmd.Process != nil {
			_ = a.cmd.Process.Kill()
		}
	}
	// Reset state
	a.recording = false
	a.cmd = nil
	a.sessionID = ""
	return nil
}
