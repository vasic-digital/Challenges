// Package logging provides structured logging for the challenges
// framework with JSON, console, and multi-destination output.
package logging

// Logger defines the interface for structured challenge logging.
type Logger interface {
	// Info logs an informational message.
	Info(msg string, fields ...Field)

	// Warn logs a warning message.
	Warn(msg string, fields ...Field)

	// Error logs an error message.
	Error(msg string, fields ...Field)

	// Debug logs a debug-level message.
	Debug(msg string, fields ...Field)

	// WithFields returns a Logger with additional default
	// fields attached to every subsequent log entry.
	WithFields(fields ...Field) Logger

	// LogAPIRequest logs an outbound API request.
	LogAPIRequest(request APIRequestLog)

	// LogAPIResponse logs an inbound API response.
	LogAPIResponse(response APIResponseLog)

	// Close flushes any buffers and releases resources.
	Close() error
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value any
}

// APIRequestLog captures API request details.
type APIRequestLog struct {
	Timestamp  string            `json:"timestamp"`
	RequestID  string            `json:"request_id"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body,omitempty"`
	BodyLength int               `json:"body_length"`
}

// APIResponseLog captures API response details.
type APIResponseLog struct {
	Timestamp      string            `json:"timestamp"`
	RequestID      string            `json:"request_id"`
	StatusCode     int               `json:"status_code"`
	Headers        map[string]string `json:"headers"`
	BodyPreview    string            `json:"body_preview,omitempty"`
	BodyLength     int               `json:"body_length"`
	ResponseTimeMs int64             `json:"response_time_ms"`
}

// LogLevel represents logging severity levels.
type LogLevel int

const (
	// LevelDebug is the most verbose level.
	LevelDebug LogLevel = iota
	// LevelInfo is the default level.
	LevelInfo
	// LevelWarn indicates potential issues.
	LevelWarn
	// LevelError indicates failures.
	LevelError
)

// String returns the string representation of a log level.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
