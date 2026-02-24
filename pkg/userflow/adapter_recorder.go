package userflow

import (
	"context"
	"time"
)

// RecorderAdapter records UI testing sessions as video.
// Implementations may use CDP screencast, OS-level screen
// recording, or external tools.
type RecorderAdapter interface {
	// StartRecording begins recording the session.
	StartRecording(
		ctx context.Context, config RecordingConfig,
	) error

	// StopRecording stops the recording and returns the
	// result including file path, duration, and frame count.
	StopRecording(
		ctx context.Context,
	) (*RecordingResult, error)

	// IsRecording reports whether a recording is active.
	IsRecording() bool

	// Available reports whether the adapter can run.
	Available(ctx context.Context) bool
}

// RecordingConfig configures a recording session.
type RecordingConfig struct {
	URL       string `json:"url"`
	OutputDir string `json:"output_dir"`
	MaxFPS    int    `json:"max_fps"`
	MaxWidth  int    `json:"max_width"`
	MaxHeight int    `json:"max_height"`
	Format    string `json:"format"`
	Headless  bool   `json:"headless"`
}

// RecordingResult contains the outcome of a recording.
type RecordingResult struct {
	FilePath   string        `json:"file_path"`
	Duration   time.Duration `json:"duration"`
	FrameCount int           `json:"frame_count"`
	FileSize   int64         `json:"file_size"`
}
