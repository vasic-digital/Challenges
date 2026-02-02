package logging

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsoleLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		output:  &buf,
		verbose: false,
		fields:  make(map[string]any),
	}

	logger.Info("hello world")

	output := buf.String()
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "hello world")
}

func TestConsoleLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		output:  &buf,
		verbose: false,
		fields:  make(map[string]any),
	}

	logger.Warn("warning message")

	output := buf.String()
	assert.Contains(t, output, "WARN")
	assert.Contains(t, output, "warning message")
}

func TestConsoleLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		output:  &buf,
		verbose: false,
		fields:  make(map[string]any),
	}

	logger.Error("error occurred")

	output := buf.String()
	assert.Contains(t, output, "ERROR")
	assert.Contains(t, output, "error occurred")
}

func TestConsoleLogger_Debug_Verbose(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		output:  &buf,
		verbose: true,
		fields:  make(map[string]any),
	}

	logger.Debug("debug info")
	assert.Contains(t, buf.String(), "debug info")
}

func TestConsoleLogger_Debug_NotVerbose(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		output:  &buf,
		verbose: false,
		fields:  make(map[string]any),
	}

	logger.Debug("debug info")
	assert.Empty(t, buf.String())
}

func TestConsoleLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		output:  &buf,
		verbose: false,
		fields:  make(map[string]any),
	}

	child := logger.WithFields(LogField("env", "test"))
	assert.NotNil(t, child)

	cl, ok := child.(*ConsoleLogger)
	assert.True(t, ok)
	assert.Equal(t, "test", cl.fields["env"])
}

func TestConsoleLogger_InfoWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		output:  &buf,
		verbose: false,
		fields:  make(map[string]any),
	}

	logger.Info("msg", LogField("key", "val"))
	assert.Contains(t, buf.String(), "key=val")
}

func TestConsoleLogger_LogAPIRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		output:  &buf,
		verbose: false,
		fields:  make(map[string]any),
	}

	logger.LogAPIRequest(APIRequestLog{
		RequestID: "r1",
		Method:    "POST",
		URL:       "http://localhost",
	})

	output := buf.String()
	assert.Contains(t, output, "r1")
	assert.Contains(t, output, "POST")
}

func TestConsoleLogger_LogAPIResponse(t *testing.T) {
	var buf bytes.Buffer
	logger := &ConsoleLogger{
		output:  &buf,
		verbose: false,
		fields:  make(map[string]any),
	}

	logger.LogAPIResponse(APIResponseLog{
		RequestID:      "r1",
		StatusCode:     200,
		ResponseTimeMs: 50,
	})

	output := buf.String()
	assert.Contains(t, output, "r1")
	assert.Contains(t, output, "200")
}

func TestConsoleLogger_Close(t *testing.T) {
	logger := NewConsoleLogger(false)
	assert.NoError(t, logger.Close())
}
