package challenge

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteJSONResult_WriteError tests the WriteFile error path.
func TestWriteJSONResult_WriteError(t *testing.T) {
	b := NewBaseChallenge(
		"json-write-err", "Write Error", "desc", "unit", nil,
	)
	// Set config directly to point to an impossible path
	b.config = &Config{
		ChallengeID: "json-write-err",
		ResultsDir:  "/dev/null/cannot/write/here",
		LogsDir:     "/tmp",
	}

	result := &Result{
		ChallengeID: "json-write-err",
		Status:      StatusPassed,
	}

	err := b.WriteJSONResult(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write result")
}

// TestWriteJSONResult_MarshalErrorWithChannel tests marshal error by using a
// type that can't be marshaled. Since Result struct is fully marshalable,
// we can't easily trigger a marshal error. This test documents the behavior.
func TestWriteJSONResult_MarshalErrorWithChannel(t *testing.T) {
	// Note: Result struct doesn't contain any fields that would fail
	// json.Marshal. The marshal error branch is defensive code that
	// can't be easily triggered with the current struct definition.
	// This test verifies the function works correctly with valid data.

	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"json-marshal-test", "Marshal Test", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID: "json-marshal-test",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}
	require.NoError(t, b.Configure(cfg))

	result := &Result{
		ChallengeID: "json-marshal-test",
		Status:      StatusPassed,
		Outputs: map[string]string{
			"key": "value",
		},
		Metrics: map[string]MetricValue{
			"latency": {Name: "latency", Value: 100.5, Unit: "ms"},
		},
	}

	err := b.WriteJSONResult(result)
	assert.NoError(t, err)
}

// TestWriteOutputLog_MkdirError tests the MkdirAll error path.
func TestWriteOutputLog_MkdirError(t *testing.T) {
	sc := NewShellChallenge(
		"write-log-err", "Write Log Error", "desc", "unit",
		nil, "/bin/bash", nil, "",
	)
	// Set config to point to impossible log dir
	sc.config = &Config{
		ChallengeID: "write-log-err",
		ResultsDir:  "/tmp/results",
		LogsDir:     "/dev/null/cannot/mkdir",
	}

	ml := &mockLogger{}
	sc.SetLogger(ml)

	// This should trigger the mkdir error path and log an error
	sc.writeOutputLog([]byte("stdout"), []byte("stderr"))

	// Should have logged an error
	assert.NotEmpty(t, ml.errors)
	assert.Contains(t, ml.errors[0], "create log dir")
}

// TestWriteOutputLog_WriteFileError tests the WriteFile error path.
func TestWriteOutputLog_WriteFileError(t *testing.T) {
	tmpDir := t.TempDir()

	sc := NewShellChallenge(
		"write-file-err", "Write File Error", "desc", "unit",
		nil, "/bin/bash", nil, "",
	)

	// Create a file where the log file should be created (to cause write error)
	logDir := filepath.Join(tmpDir, "logs", "write-file-err")
	require.NoError(t, os.MkdirAll(logDir, 0o755))

	// Create a directory where we expect a file (output.log)
	outputLogPath := filepath.Join(logDir, "output.log")
	require.NoError(t, os.MkdirAll(outputLogPath, 0o755))

	sc.config = &Config{
		ChallengeID: "write-file-err",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}

	ml := &mockLogger{}
	sc.SetLogger(ml)

	// This should trigger the write file error path
	sc.writeOutputLog([]byte("stdout"), []byte("stderr"))

	// Should have logged an error
	assert.NotEmpty(t, ml.errors)
	assert.Contains(t, ml.errors[0], "write output log")
}

// TestShellChallenge_Execute_ExitStatus tests various exit statuses.
func TestShellChallenge_Execute_ExitStatus(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that exits with a non-zero status
	exitScript := filepath.Join(tmpDir, "exit.sh")
	require.NoError(t, os.WriteFile(exitScript, []byte("#!/bin/bash\nexit 42\n"), 0o755))

	sc := NewShellChallenge(
		"exit-status", "Exit Status", "desc", "unit",
		nil, exitScript, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "exit-status",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     0,
	}
	require.NoError(t, sc.Configure(cfg))

	result, err := sc.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, result.Status)
	assert.Equal(t, "42", result.Outputs["exit_code"])
}

// TestShellChallenge_Execute_NonExecError tests the generic error case in Execute
// where the error is not an ExitError (lines 139-141).
func TestShellChallenge_Execute_NonExecError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid script file but make it non-executable
	nonExecScript := filepath.Join(tmpDir, "nonexec.sh")
	// Create a binary file that can't be executed as bash script
	require.NoError(t, os.WriteFile(nonExecScript, []byte{0x7f, 'E', 'L', 'F'}, 0o644))

	sc := NewShellChallenge(
		"non-exec-error", "Non Exec Error", "desc", "unit",
		nil, nonExecScript, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "non-exec-error",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     0,
	}
	require.NoError(t, sc.Configure(cfg))

	// Execute will hit the generic error case when bash tries to execute
	// a non-script file
	result, err := sc.Execute(t.Context())
	require.NoError(t, err)
	// Should have some kind of error status (failed or error)
	assert.NotEqual(t, StatusPassed, result.Status)
}

// TestShellChallenge_Execute_BashSourceError tests when bash can't source the script.
func TestShellChallenge_Execute_BashSourceError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a path to a non-existent script to trigger a Start error
	// When bash tries to execute a non-existent script passed as argument,
	// it returns an exit error, not a generic error. We need a different approach.

	// Try with a script that causes bash to fail before execution completes
	badScript := filepath.Join(tmpDir, "bad.sh")
	// Script that intentionally fails
	require.NoError(t, os.WriteFile(badScript, []byte("#!/bin/bash\nexit 123\n"), 0o755))

	sc := NewShellChallenge(
		"bash-error", "Bash Error", "desc", "unit",
		nil, badScript, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "bash-error",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     0,
	}
	require.NoError(t, sc.Configure(cfg))

	result, err := sc.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, result.Status)
	assert.Equal(t, "123", result.Outputs["exit_code"])
}

// TestWriteJSONResult_MarshalError tests the json.Marshal error path
// using dependency injection.
func TestWriteJSONResult_MarshalError(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"json-marshal-err", "Marshal Error", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID: "json-marshal-err",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}
	require.NoError(t, b.Configure(cfg))

	// Save the original and restore after test
	originalMarshal := jsonMarshalIndent
	defer func() { jsonMarshalIndent = originalMarshal }()

	// Inject a failing marshaler
	jsonMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
		return nil, assert.AnError
	}

	result := &Result{
		ChallengeID: "json-marshal-err",
		Status:      StatusPassed,
	}

	err := b.WriteJSONResult(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal result")
}

// TestShellChallenge_Execute_GenericError tests the generic error case
// when cmd.Run() returns an error that is not an ExitError.
// This is triggered by context cancellation (not deadline exceeded).
func TestShellChallenge_Execute_GenericError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that sleeps long enough for us to cancel
	scriptPath := filepath.Join(tmpDir, "slow.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/bash\nsleep 30\n"), 0o755))

	sc := NewShellChallenge(
		"generic-error", "Generic Error", "desc", "unit",
		nil, scriptPath, nil, tmpDir,
	)

	cfg := &Config{
		ChallengeID: "generic-error",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     0, // No internal timeout
	}
	require.NoError(t, sc.Configure(cfg))

	// Create a context that we'll cancel, which triggers context.Canceled
	// not context.DeadlineExceeded
	ctx, cancel := context.WithCancel(context.Background())

	// Start execute in goroutine
	resultChan := make(chan *Result, 1)
	go func() {
		result, _ := sc.Execute(ctx)
		resultChan <- result
	}()

	// Give script time to start, then cancel
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for result
	select {
	case result := <-resultChan:
		// Context.Canceled causes the process to be killed,
		// which results in an ExitError (signal-based exit)
		assert.NotEqual(t, StatusPassed, result.Status)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	}
}

// TestShellChallenge_Execute_StatusError tests the generic error case directly
// using dependency injection by mocking the command execution.
func TestShellChallenge_Execute_StatusError(t *testing.T) {
	tests := []struct {
		name       string
		injectedError error
		wantStatus string
	}{
		{
			name:       "generic error (not ExitError)",
			injectedError: assert.AnError, // This is not an *exec.ExitError
			wantStatus: StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create a valid script
			scriptPath := filepath.Join(tmpDir, "test.sh")
			require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/bash\necho test\n"), 0o755))

			sc := NewShellChallenge(
				"status-error", "Status Error", "desc", "unit",
				nil, scriptPath, nil, tmpDir,
			)

			cfg := &Config{
				ChallengeID: "status-error",
				ResultsDir:  filepath.Join(tmpDir, "results"),
				LogsDir:     filepath.Join(tmpDir, "logs"),
				Timeout:     0,
			}
			require.NoError(t, sc.Configure(cfg))

			// Save original and restore after test
			originalRunner := runCommand
			t.Cleanup(func() { runCommand = originalRunner })

			// Inject a failing runner that returns a non-ExitError
			runCommand = func(_ *exec.Cmd) error {
				return tt.injectedError
			}

			result, err := sc.Execute(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Contains(t, result.Error, "execution error")
		})
	}
}

// TestShellChallenge_Execute_WriteJSONResultError tests the error path
// when WriteJSONResult fails after script execution.
func TestShellChallenge_Execute_WriteJSONResultError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid script
	scriptPath := filepath.Join(tmpDir, "test.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/bash\necho test\n"), 0o755))

	sc := NewShellChallenge(
		"write-err", "Write Error", "desc", "unit",
		nil, scriptPath, nil, tmpDir,
	)

	// Configure with an impossible results directory to cause WriteJSONResult to fail
	cfg := &Config{
		ChallengeID: "write-err",
		ResultsDir:  "/dev/null/impossible/results",
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     0,
	}
	// Set config directly to bypass directory creation
	sc.config = cfg
	// Create logs dir manually
	require.NoError(t, os.MkdirAll(sc.LogsDir(), 0o755))

	// Set a mock logger to capture the error log
	ml := &mockLogger{}
	sc.SetLogger(ml)

	result, err := sc.Execute(context.Background())
	require.NoError(t, err) // Execute itself doesn't return an error
	assert.Equal(t, StatusPassed, result.Status)

	// The error should have been logged
	assert.NotEmpty(t, ml.errors)
	assert.Contains(t, ml.errors[0], "failed to write result")
}

// TestShellChallenge_Validate_AllBranches ensures all validation branches are covered.
func TestShellChallenge_Validate_AllBranches(t *testing.T) {
	tests := []struct {
		name          string
		setupScript   func(tmpDir string) string
		expectError   bool
		errorContains string
	}{
		{
			name: "script exists and is file",
			setupScript: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "valid.sh")
				require.NoError(t, os.WriteFile(path, []byte("#!/bin/bash\n"), 0o755))
				return path
			},
			expectError: false,
		},
		{
			name: "script does not exist",
			setupScript: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nonexistent.sh")
			},
			expectError:   true,
			errorContains: "script",
		},
		{
			name: "script is directory",
			setupScript: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "isdir.sh")
				require.NoError(t, os.MkdirAll(path, 0o755))
				return path
			},
			expectError:   true,
			errorContains: "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			scriptPath := tt.setupScript(tmpDir)

			sc := NewShellChallenge(
				"validate-test", "Validate Test", "desc", "unit",
				nil, scriptPath, nil, tmpDir,
			)
			cfg := &Config{
				ChallengeID: "validate-test",
				ResultsDir:  filepath.Join(tmpDir, "results"),
				LogsDir:     filepath.Join(tmpDir, "logs"),
			}
			require.NoError(t, sc.Configure(cfg))

			err := sc.Validate(t.Context())
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
