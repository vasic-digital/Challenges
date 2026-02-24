package userflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockRecorder implements RecorderAdapter for testing.
type mockRecorder struct {
	recording bool
	available bool
	startErr  error
	stopErr   error
	result    *RecordingResult
}

func (m *mockRecorder) StartRecording(
	_ context.Context, _ RecordingConfig,
) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.recording = true
	return nil
}

func (m *mockRecorder) StopRecording(
	_ context.Context,
) (*RecordingResult, error) {
	if m.stopErr != nil {
		return nil, m.stopErr
	}
	m.recording = false
	return m.result, nil
}

func (m *mockRecorder) IsRecording() bool {
	return m.recording
}

func (m *mockRecorder) Available(_ context.Context) bool {
	return m.available
}

// Compile-time interface check for mockRecorder.
var _ RecorderAdapter = (*mockRecorder)(nil)

func TestRecordingConfig_Defaults(t *testing.T) {
	cfg := RecordingConfig{
		URL:       "http://localhost:3000",
		OutputDir: "/tmp/recordings",
		MaxFPS:    30,
		MaxWidth:  1920,
		MaxHeight: 1080,
		Format:    "mp4",
		Headless:  true,
	}

	assert.Equal(t, "http://localhost:3000", cfg.URL)
	assert.Equal(t, "/tmp/recordings", cfg.OutputDir)
	assert.Equal(t, 30, cfg.MaxFPS)
	assert.Equal(t, 1920, cfg.MaxWidth)
	assert.Equal(t, 1080, cfg.MaxHeight)
	assert.Equal(t, "mp4", cfg.Format)
	assert.True(t, cfg.Headless)
}

func TestRecordingResult_Duration(t *testing.T) {
	// Simulate the duration conversion that
	// PanopticRecorderAdapter.StopRecording performs.
	durationMs := int64(5432)
	result := RecordingResult{
		FilePath:   "/tmp/recordings/session.mp4",
		Duration:   time.Duration(durationMs) * time.Millisecond,
		FrameCount: 163,
		FileSize:   2048576,
	}

	assert.Equal(t, 5432*time.Millisecond, result.Duration)
	assert.Equal(
		t, "/tmp/recordings/session.mp4", result.FilePath,
	)
	assert.Equal(t, 163, result.FrameCount)
	assert.Equal(t, int64(2048576), result.FileSize)
}

func TestPanopticRecorderAdapter_Available_NotFound(
	t *testing.T,
) {
	adapter := NewPanopticRecorderAdapter(
		"/nonexistent/path/to/panoptic",
	)
	ctx := context.Background()

	assert.False(t, adapter.Available(ctx))
}

func TestPanopticRecorderAdapter_IsRecording_Initial(
	t *testing.T,
) {
	adapter := NewPanopticRecorderAdapter("/usr/bin/panoptic")

	assert.False(t, adapter.IsRecording())
}

func TestMockRecorder_Interface(t *testing.T) {
	ctx := context.Background()
	m := &mockRecorder{
		available: true,
		result: &RecordingResult{
			FilePath:   "/tmp/test.mp4",
			Duration:   3 * time.Second,
			FrameCount: 90,
			FileSize:   1024,
		},
	}

	assert.True(t, m.Available(ctx))
	assert.False(t, m.IsRecording())

	err := m.StartRecording(ctx, RecordingConfig{
		URL:       "http://localhost:8080",
		OutputDir: "/tmp",
		MaxFPS:    30,
	})
	assert.NoError(t, err)
	assert.True(t, m.IsRecording())

	result, err := m.StopRecording(ctx)
	assert.NoError(t, err)
	assert.False(t, m.IsRecording())
	assert.Equal(t, "/tmp/test.mp4", result.FilePath)
	assert.Equal(t, 3*time.Second, result.Duration)
	assert.Equal(t, 90, result.FrameCount)
	assert.Equal(t, int64(1024), result.FileSize)
}
