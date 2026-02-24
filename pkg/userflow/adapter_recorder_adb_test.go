package userflow

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ RecorderAdapter = (*ADBRecorderAdapter)(nil)

// adbAvailable returns true if the adb binary is found in
// PATH. Tests that need a real device should call this and
// skip when false.
func adbAvailable() bool {
	_, err := exec.LookPath("adb")
	return err == nil
}

func TestNewADBRecorderAdapter_Defaults(t *testing.T) {
	adapter := NewADBRecorderAdapter("", "")

	assert.Equal(t, defaultADBPath, adapter.adbPath)
	assert.Empty(t, adapter.deviceSerial)
	assert.Equal(
		t, defaultDevicePath, adapter.devicePath,
	)
	assert.False(t, adapter.recording)
	assert.Nil(t, adapter.cmd)
}

func TestNewADBRecorderAdapter_CustomPath(t *testing.T) {
	adapter := NewADBRecorderAdapter(
		"/opt/android/platform-tools/adb",
		"emulator-5554",
	)

	assert.Equal(
		t,
		"/opt/android/platform-tools/adb",
		adapter.adbPath,
	)
	assert.Equal(
		t, "emulator-5554", adapter.deviceSerial,
	)
}

func TestADBRecorderAdapter_IsRecording_Initial(
	t *testing.T,
) {
	adapter := NewADBRecorderAdapter("adb", "")

	assert.False(t, adapter.IsRecording())
}

func TestADBRecorderAdapter_Available_NoADB(t *testing.T) {
	adapter := NewADBRecorderAdapter(
		"/nonexistent/path/to/adb", "",
	)
	ctx := context.Background()

	assert.False(t, adapter.Available(ctx))
}

func TestADBRecorderAdapter_StartRecording_NoADB(
	t *testing.T,
) {
	adapter := NewADBRecorderAdapter(
		"/nonexistent/path/to/adb", "",
	)
	ctx := context.Background()

	err := adapter.StartRecording(ctx, RecordingConfig{
		OutputDir: t.TempDir(),
		MaxFPS:    30,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "start screenrecord")
	assert.False(t, adapter.IsRecording())
}

func TestADBRecorderAdapter_StopRecording_NotStarted(
	t *testing.T,
) {
	adapter := NewADBRecorderAdapter("adb", "")
	ctx := context.Background()

	result, err := adapter.StopRecording(ctx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(
		t, err.Error(), "no recording in progress",
	)
}

func TestADBRecorderAdapter_DeviceArgs_WithSerial(
	t *testing.T,
) {
	adapter := NewADBRecorderAdapter(
		"adb", "device-123",
	)

	args := adapter.deviceArgs("shell", "screenrecord")
	assert.Equal(
		t,
		[]string{
			"-s", "device-123",
			"shell", "screenrecord",
		},
		args,
	)
}

func TestADBRecorderAdapter_DeviceArgs_NoSerial(
	t *testing.T,
) {
	adapter := NewADBRecorderAdapter("adb", "")

	args := adapter.deviceArgs("shell", "screenrecord")
	assert.Equal(
		t,
		[]string{"shell", "screenrecord"},
		args,
	)
}

func TestADBRecorderAdapter_StartRecording_AlreadyRecording(
	t *testing.T,
) {
	adapter := NewADBRecorderAdapter("adb", "")
	// Manually set recording state to simulate an active
	// recording without needing a real adb process.
	adapter.mu.Lock()
	adapter.recording = true
	adapter.mu.Unlock()

	ctx := context.Background()
	err := adapter.StartRecording(ctx, RecordingConfig{
		OutputDir: t.TempDir(),
	})

	require.Error(t, err)
	assert.Contains(
		t, err.Error(), "recording already in progress",
	)
}

func TestADBRecorderAdapter_Available_WithSerial_NoADB(
	t *testing.T,
) {
	adapter := NewADBRecorderAdapter(
		"/nonexistent/adb", "specific-device",
	)
	ctx := context.Background()

	assert.False(t, adapter.Available(ctx))
}
