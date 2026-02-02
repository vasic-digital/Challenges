package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

// ConsoleLogger provides colored console output.
type ConsoleLogger struct {
	mu      sync.Mutex
	output  io.Writer
	verbose bool
	fields  map[string]any
}

// NewConsoleLogger creates a console logger. When verbose is
// true, debug messages are emitted.
func NewConsoleLogger(verbose bool) *ConsoleLogger {
	return &ConsoleLogger{
		output:  os.Stdout,
		verbose: verbose,
		fields:  make(map[string]any),
	}
}

func (c *ConsoleLogger) log(
	level LogLevel, color, msg string, fields ...Field,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ts := time.Now().Format("15:04:05")
	levelStr := level.String()

	var fieldStr string
	if len(fields) > 0 {
		parts := make([]string, 0, len(fields))
		for _, f := range fields {
			parts = append(
				parts,
				fmt.Sprintf("%s=%v", f.Key, f.Value),
			)
		}
		fieldStr = " " + colorGray +
			fmt.Sprintf("{%s}", strings.Join(parts, ", ")) +
			colorReset
	}

	fmt.Fprintf(
		c.output, "%s%s%s [%s%-5s%s] %s%s\n",
		colorGray, ts, colorReset,
		color, levelStr, colorReset,
		msg, fieldStr,
	)
}

// Info logs an informational message.
func (c *ConsoleLogger) Info(msg string, fields ...Field) {
	c.log(LevelInfo, colorBlue, msg, fields...)
}

// Warn logs a warning message.
func (c *ConsoleLogger) Warn(msg string, fields ...Field) {
	c.log(LevelWarn, colorYellow, msg, fields...)
}

// Error logs an error message.
func (c *ConsoleLogger) Error(msg string, fields ...Field) {
	c.log(LevelError, colorRed, msg, fields...)
}

// Debug logs a debug message only if verbose is enabled.
func (c *ConsoleLogger) Debug(msg string, fields ...Field) {
	if c.verbose {
		c.log(LevelDebug, colorGray, msg, fields...)
	}
}

// WithFields returns a new Logger with additional default
// fields.
func (c *ConsoleLogger) WithFields(
	fields ...Field,
) Logger {
	newFields := make(map[string]any)
	for k, v := range c.fields {
		newFields[k] = v
	}
	for _, f := range fields {
		newFields[f.Key] = f.Value
	}
	return &ConsoleLogger{
		output:  c.output,
		verbose: c.verbose,
		fields:  newFields,
	}
}

// LogAPIRequest logs an API request summary to the console.
func (c *ConsoleLogger) LogAPIRequest(
	request APIRequestLog,
) {
	c.Info("API Request",
		Field{Key: "request_id", Value: request.RequestID},
		Field{Key: "method", Value: request.Method},
		Field{Key: "url", Value: request.URL},
	)
}

// LogAPIResponse logs an API response summary to the console.
func (c *ConsoleLogger) LogAPIResponse(
	response APIResponseLog,
) {
	c.Info("API Response",
		Field{Key: "request_id", Value: response.RequestID},
		Field{Key: "status", Value: response.StatusCode},
		Field{Key: "time_ms", Value: response.ResponseTimeMs},
	)
}

// Close is a no-op for ConsoleLogger.
func (c *ConsoleLogger) Close() error {
	return nil
}
