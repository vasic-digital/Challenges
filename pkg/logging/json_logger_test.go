package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONLogger_NewJSONLogger_Stdout(t *testing.T) {
	logger, err := NewJSONLogger(LoggerConfig{
		Level:   LevelInfo,
		Verbose: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, logger)
	assert.NoError(t, logger.Close())
}

func TestJSONLogger_NewJSONLogger_File(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	logger, err := NewJSONLogger(LoggerConfig{
		OutputPath: logPath,
		Level:      LevelDebug,
		Verbose:    true,
	})
	require.NoError(t, err)

	logger.Info("hello", LogField("key", "val"))
	logger.Debug("debug msg")
	require.NoError(t, logger.Close())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	lines := splitNonEmpty(string(data))
	require.Len(t, lines, 2)

	var entry LogEntry
	err = json.Unmarshal([]byte(lines[0]), &entry)
	require.NoError(t, err)
	assert.Equal(t, "INFO", entry.Level)
	assert.Equal(t, "hello", entry.Message)
	assert.Equal(t, "val", entry.Fields["key"])
}

func TestJSONLogger_LevelFiltering(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "level.log")

	logger, err := NewJSONLogger(LoggerConfig{
		OutputPath: logPath,
		Level:      LevelWarn,
		Verbose:    true,
	})
	require.NoError(t, err)

	logger.Debug("should not appear")
	logger.Info("should not appear")
	logger.Warn("should appear")
	logger.Error("should appear")
	require.NoError(t, logger.Close())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	lines := splitNonEmpty(string(data))
	assert.Len(t, lines, 2)
}

func TestJSONLogger_WithFields(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "fields.log")

	logger, err := NewJSONLogger(LoggerConfig{
		OutputPath: logPath,
		Level:      LevelInfo,
		Fields:     map[string]any{"base": "value"},
	})
	require.NoError(t, err)

	child := logger.WithFields(LogField("child", "yes"))
	child.Info("child message")
	require.NoError(t, logger.Close())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	var entry LogEntry
	err = json.Unmarshal(
		[]byte(splitNonEmpty(string(data))[0]), &entry,
	)
	require.NoError(t, err)
	assert.Equal(t, "value", entry.Fields["base"])
	assert.Equal(t, "yes", entry.Fields["child"])
}

func TestJSONLogger_LogAPIRequest(t *testing.T) {
	dir := t.TempDir()
	reqPath := filepath.Join(dir, "api_req.log")

	logger, err := NewJSONLogger(LoggerConfig{
		APIRequestLog: reqPath,
		Level:         LevelInfo,
	})
	require.NoError(t, err)

	logger.LogAPIRequest(APIRequestLog{
		RequestID: "req-1",
		Method:    "GET",
		URL:       "http://example.com",
	})
	require.NoError(t, logger.Close())

	data, err := os.ReadFile(reqPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "req-1")
}

func TestJSONLogger_LogAPIResponse(t *testing.T) {
	dir := t.TempDir()
	respPath := filepath.Join(dir, "api_resp.log")

	logger, err := NewJSONLogger(LoggerConfig{
		APIResponseLog: respPath,
		Level:          LevelInfo,
	})
	require.NoError(t, err)

	logger.LogAPIResponse(APIResponseLog{
		RequestID:      "req-1",
		StatusCode:     200,
		ResponseTimeMs: 42,
	})
	require.NoError(t, logger.Close())

	data, err := os.ReadFile(respPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "req-1")
}

func TestJSONLogger_ClosedLoggerNoop(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "closed.log")

	logger, err := NewJSONLogger(LoggerConfig{
		OutputPath: logPath,
		Level:      LevelInfo,
	})
	require.NoError(t, err)
	require.NoError(t, logger.Close())

	// Should not panic or write
	logger.Info("after close")

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Empty(t, splitNonEmpty(string(data)))
}

func TestSetupLogging(t *testing.T) {
	dir := t.TempDir()
	logger, err := SetupLogging(dir, true)
	require.NoError(t, err)
	require.NotNil(t, logger)

	logger.Info("setup test")
	require.NoError(t, logger.Close())

	_, err = os.Stat(
		filepath.Join(dir, "challenge.log"),
	)
	assert.NoError(t, err)
}

func splitNonEmpty(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				result = append(result, line)
			}
			start = i + 1
		}
	}
	if start < len(s) && s[start:] != "" {
		result = append(result, s[start:])
	}
	return result
}

func TestJSONLogger_NewJSONLogger_CreateDirError(t *testing.T) {
	// Try to create logger with output path in a non-existent directory
	// that cannot be created (path through a file, not a directory)
	_, err := NewJSONLogger(LoggerConfig{
		OutputPath: "/dev/null/impossible/path/log.txt",
		Level:      LevelInfo,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create log directory")
}

func TestJSONLogger_NewJSONLogger_OpenFileError(t *testing.T) {
	// Create a directory where we'll try to open a file
	dir := t.TempDir()
	// Create a subdirectory with the same name as the log file
	logPath := filepath.Join(dir, "test.log")
	require.NoError(t, os.MkdirAll(logPath, 0755))

	_, err := NewJSONLogger(LoggerConfig{
		OutputPath: logPath,
		Level:      LevelInfo,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open log file")
}

func TestJSONLogger_NewJSONLogger_APIRequestLogError(t *testing.T) {
	dir := t.TempDir()
	// Create a directory where the API request log file should be
	reqPath := filepath.Join(dir, "api_req.log")
	require.NoError(t, os.MkdirAll(reqPath, 0755))

	_, err := NewJSONLogger(LoggerConfig{
		APIRequestLog: reqPath,
		Level:         LevelInfo,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open API request log")
}

func TestJSONLogger_NewJSONLogger_APIResponseLogError(t *testing.T) {
	dir := t.TempDir()
	// Create a directory where the API response log file should be
	respPath := filepath.Join(dir, "api_resp.log")
	require.NoError(t, os.MkdirAll(respPath, 0755))

	_, err := NewJSONLogger(LoggerConfig{
		APIResponseLog: respPath,
		Level:          LevelInfo,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open API response log")
}

func TestJSONLogger_LogAPIRequest_NilWriter(t *testing.T) {
	// Logger with no API request log configured
	logger, err := NewJSONLogger(LoggerConfig{
		Level: LevelInfo,
	})
	require.NoError(t, err)

	// Should not panic when apiRequestLog is nil
	logger.LogAPIRequest(APIRequestLog{
		RequestID: "req-1",
		Method:    "GET",
		URL:       "http://example.com",
	})
	require.NoError(t, logger.Close())
}

func TestJSONLogger_LogAPIResponse_NilWriter(t *testing.T) {
	// Logger with no API response log configured
	logger, err := NewJSONLogger(LoggerConfig{
		Level: LevelInfo,
	})
	require.NoError(t, err)

	// Should not panic when apiResponseLog is nil
	logger.LogAPIResponse(APIResponseLog{
		RequestID:      "req-1",
		StatusCode:     200,
		ResponseTimeMs: 42,
	})
	require.NoError(t, logger.Close())
}

func TestJSONLogger_Close_WithFileErrors(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "main.log")
	reqPath := filepath.Join(dir, "api_req.log")
	respPath := filepath.Join(dir, "api_resp.log")

	logger, err := NewJSONLogger(LoggerConfig{
		OutputPath:     logPath,
		APIRequestLog:  reqPath,
		APIResponseLog: respPath,
		Level:          LevelInfo,
	})
	require.NoError(t, err)

	// Close should properly close all files
	err = logger.Close()
	assert.NoError(t, err)

	// Second close - the logger should handle the closed state gracefully
	// (it's fine if it returns an error for already closed files)
	_ = logger.Close()
}

func TestJSONLogger_NilFields(t *testing.T) {
	// Test that nil fields in config don't cause issues
	logger, err := NewJSONLogger(LoggerConfig{
		Level:  LevelInfo,
		Fields: nil,
	})
	require.NoError(t, err)
	assert.NotNil(t, logger.fields)
	require.NoError(t, logger.Close())
}

func TestJSONLogger_LogMarshalError(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "marshal_err.log")

	logger, err := NewJSONLogger(LoggerConfig{
		OutputPath: logPath,
		Level:      LevelInfo,
	})
	require.NoError(t, err)

	// Save original and restore after test
	originalMarshal := jsonMarshal
	t.Cleanup(func() { jsonMarshal = originalMarshal })

	// Inject a failing marshaler
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, assert.AnError
	}

	// This should return early without writing anything
	logger.Info("test message")

	require.NoError(t, logger.Close())

	// File should be empty because marshal failed
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Empty(t, splitNonEmpty(string(data)))
}

func TestJSONLogger_LogAPIRequestMarshalError(t *testing.T) {
	dir := t.TempDir()
	reqPath := filepath.Join(dir, "api_req.log")

	logger, err := NewJSONLogger(LoggerConfig{
		APIRequestLog: reqPath,
		Level:         LevelInfo,
	})
	require.NoError(t, err)

	// Save original and restore after test
	originalMarshal := jsonMarshal
	t.Cleanup(func() { jsonMarshal = originalMarshal })

	// Inject a failing marshaler
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, assert.AnError
	}

	// This should return early without writing anything
	logger.LogAPIRequest(APIRequestLog{
		RequestID: "req-1",
		Method:    "GET",
		URL:       "http://example.com",
	})

	require.NoError(t, logger.Close())

	// File should be empty because marshal failed
	data, err := os.ReadFile(reqPath)
	require.NoError(t, err)
	assert.Empty(t, splitNonEmpty(string(data)))
}

func TestJSONLogger_LogAPIResponseMarshalError(t *testing.T) {
	dir := t.TempDir()
	respPath := filepath.Join(dir, "api_resp.log")

	logger, err := NewJSONLogger(LoggerConfig{
		APIResponseLog: respPath,
		Level:          LevelInfo,
	})
	require.NoError(t, err)

	// Save original and restore after test
	originalMarshal := jsonMarshal
	t.Cleanup(func() { jsonMarshal = originalMarshal })

	// Inject a failing marshaler
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, assert.AnError
	}

	// This should return early without writing anything
	logger.LogAPIResponse(APIResponseLog{
		RequestID:      "req-1",
		StatusCode:     200,
		ResponseTimeMs: 42,
	})

	require.NoError(t, logger.Close())

	// File should be empty because marshal failed
	data, err := os.ReadFile(respPath)
	require.NoError(t, err)
	assert.Empty(t, splitNonEmpty(string(data)))
}
