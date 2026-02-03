package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// jsonMarshal is a variable for dependency injection in tests.
var jsonMarshal = json.Marshal

// LogEntry represents a single JSON log entry.
type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// LoggerConfig configures the JSONLogger.
type LoggerConfig struct {
	OutputPath     string
	APIRequestLog  string
	APIResponseLog string
	Level          LogLevel
	Verbose        bool
	Fields         map[string]any
}

// JSONLogger implements Logger with JSON Lines output.
type JSONLogger struct {
	mu             sync.Mutex
	output         io.Writer
	apiRequestLog  io.Writer
	apiResponseLog io.Writer
	level          LogLevel
	fields         map[string]any
	verbose        bool
	closed         bool
}

// NewJSONLogger creates a new JSON logger. If OutputPath is
// empty, logs are written to stdout.
func NewJSONLogger(config LoggerConfig) (*JSONLogger, error) {
	logger := &JSONLogger{
		level:   config.Level,
		verbose: config.Verbose,
		fields:  config.Fields,
	}

	if logger.fields == nil {
		logger.fields = make(map[string]any)
	}

	if config.OutputPath != "" {
		dir := filepath.Dir(config.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf(
				"failed to create log directory: %w", err,
			)
		}
		file, err := os.OpenFile(
			config.OutputPath,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to open log file: %w", err,
			)
		}
		logger.output = file
	} else {
		logger.output = os.Stdout
	}

	if config.APIRequestLog != "" {
		file, err := os.OpenFile(
			config.APIRequestLog,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to open API request log: %w", err,
			)
		}
		logger.apiRequestLog = file
	}

	if config.APIResponseLog != "" {
		file, err := os.OpenFile(
			config.APIResponseLog,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to open API response log: %w", err,
			)
		}
		logger.apiResponseLog = file
	}

	return logger, nil
}

func (l *JSONLogger) log(
	level LogLevel, msg string, fields ...Field,
) {
	if l.closed {
		return
	}
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Level:     level.String(),
		Message:   msg,
		Fields:    make(map[string]any),
	}

	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	for _, f := range fields {
		entry.Fields[f.Key] = f.Value
	}

	data, err := jsonMarshal(entry)
	if err != nil {
		return
	}

	fmt.Fprintln(l.output, string(data))
}

// Info logs an informational message.
func (l *JSONLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

// Warn logs a warning message.
func (l *JSONLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

// Error logs an error message.
func (l *JSONLogger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

// Debug logs a debug message only if verbose is enabled.
func (l *JSONLogger) Debug(msg string, fields ...Field) {
	if l.verbose {
		l.log(LevelDebug, msg, fields...)
	}
}

// WithFields returns a new Logger with additional default
// fields.
func (l *JSONLogger) WithFields(fields ...Field) Logger {
	newFields := make(map[string]any)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for _, f := range fields {
		newFields[f.Key] = f.Value
	}

	return &JSONLogger{
		output:         l.output,
		apiRequestLog:  l.apiRequestLog,
		apiResponseLog: l.apiResponseLog,
		level:          l.level,
		verbose:        l.verbose,
		fields:         newFields,
	}
}

// LogAPIRequest logs an API request to the dedicated request
// log.
func (l *JSONLogger) LogAPIRequest(request APIRequestLog) {
	if l.apiRequestLog == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := jsonMarshal(request)
	if err != nil {
		return
	}

	fmt.Fprintln(l.apiRequestLog, string(data))
}

// LogAPIResponse logs an API response to the dedicated response
// log.
func (l *JSONLogger) LogAPIResponse(response APIResponseLog) {
	if l.apiResponseLog == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := jsonMarshal(response)
	if err != nil {
		return
	}

	fmt.Fprintln(l.apiResponseLog, string(data))
}

// Close flushes and closes all underlying writers.
func (l *JSONLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.closed = true

	var errs []error

	if closer, ok := l.output.(io.Closer); ok &&
		l.output != os.Stdout {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if closer, ok := l.apiRequestLog.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if closer, ok := l.apiResponseLog.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// SetupLogging creates a logging configuration for a challenge
// in the given logs directory.
func SetupLogging(
	logsDir string,
	verbose bool,
) (*JSONLogger, error) {
	config := LoggerConfig{
		OutputPath: filepath.Join(
			logsDir, "challenge.log",
		),
		APIRequestLog: filepath.Join(
			logsDir, "api_requests.log",
		),
		APIResponseLog: filepath.Join(
			logsDir, "api_responses.log",
		),
		Level:   LevelInfo,
		Verbose: verbose,
	}

	if verbose {
		config.Level = LevelDebug
	}

	return NewJSONLogger(config)
}
