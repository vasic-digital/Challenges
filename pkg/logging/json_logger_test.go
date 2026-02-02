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
