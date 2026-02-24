package userflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface check.
var _ MobileAdapter = (*ADBCLIAdapter)(nil)

func TestADBCLIAdapter_Constructor(t *testing.T) {
	config := MobileConfig{
		PackageName:  "com.example.app",
		ActivityName: ".MainActivity",
		DeviceSerial: "emulator-5554",
	}
	adapter := NewADBCLIAdapter(config)
	assert.NotNil(t, adapter)
	assert.Equal(
		t, "com.example.app", adapter.config.PackageName,
	)
	assert.Equal(
		t, ".MainActivity", adapter.config.ActivityName,
	)
	assert.Equal(
		t, "emulator-5554", adapter.config.DeviceSerial,
	)
}

func TestADBCLIAdapter_DeviceArgs_WithSerial(t *testing.T) {
	adapter := NewADBCLIAdapter(MobileConfig{
		DeviceSerial: "device-123",
	})
	args := adapter.deviceArgs("shell", "ls")
	assert.Equal(
		t,
		[]string{"-s", "device-123", "shell", "ls"},
		args,
	)
}

func TestADBCLIAdapter_DeviceArgs_NoSerial(t *testing.T) {
	adapter := NewADBCLIAdapter(MobileConfig{})
	args := adapter.deviceArgs("shell", "ls")
	assert.Equal(t, []string{"shell", "ls"}, args)
}

func TestADBCLIAdapter_Available_Graceful(t *testing.T) {
	// This test verifies Available() returns a boolean
	// without panicking, regardless of whether adb is
	// installed.
	adapter := NewADBCLIAdapter(MobileConfig{
		PackageName:  "com.test.app",
		ActivityName: ".MainActivity",
	})
	result := adapter.Available(context.Background())
	// result is either true or false; no panic.
	assert.IsType(t, true, result)
}

func TestADBCLIAdapter_Close_NoOp(t *testing.T) {
	adapter := NewADBCLIAdapter(MobileConfig{})
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
}

func TestADBCLIAdapter_ParseInstrumentOutput(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantTests  int
		wantFailed int
	}{
		{
			name:       "all_pass",
			output:     "OK (5 tests)",
			wantTests:  5,
			wantFailed: 0,
		},
		{
			name:       "with_failure",
			output:     "FAILURES!!!\nTests run: 3",
			wantTests:  0,
			wantFailed: 1,
		},
		{
			name:       "empty_output",
			output:     "",
			wantTests:  0,
			wantFailed: 0,
		},
		{
			name: "multiline_pass",
			output: "INSTRUMENTATION_STATUS: test=testOne\n" +
				"OK (12 tests)\n" +
				"INSTRUMENTATION_CODE: -1",
			wantTests:  12,
			wantFailed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInstrumentOutput(
				tt.output, time.Second,
			)
			assert.Equal(
				t, tt.wantTests, result.TotalTests,
			)
			assert.Equal(
				t, tt.wantFailed, result.TotalFailed,
			)
		})
	}
}

func TestADBCLIAdapter_ConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config MobileConfig
	}{
		{
			name: "full_config",
			config: MobileConfig{
				PackageName:  "com.vasic.catalogizer",
				ActivityName: ".ui.MainActivity",
				DeviceSerial: "emulator-5554",
			},
		},
		{
			name: "minimal_config",
			config: MobileConfig{
				PackageName:  "com.test",
				ActivityName: ".Main",
			},
		},
		{
			name:   "empty_config",
			config: MobileConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewADBCLIAdapter(tt.config)
			assert.NotNil(t, adapter)
			assert.Equal(
				t, tt.config.PackageName,
				adapter.config.PackageName,
			)
		})
	}
}
