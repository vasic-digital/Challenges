package logging

// MultiLogger fans out log calls to multiple loggers.
type MultiLogger struct {
	loggers []Logger
}

// NewMultiLogger creates a logger that writes to multiple
// destinations.
func NewMultiLogger(loggers ...Logger) *MultiLogger {
	return &MultiLogger{loggers: loggers}
}

// Info logs to all loggers.
func (m *MultiLogger) Info(msg string, fields ...Field) {
	for _, l := range m.loggers {
		l.Info(msg, fields...)
	}
}

// Warn logs to all loggers.
func (m *MultiLogger) Warn(msg string, fields ...Field) {
	for _, l := range m.loggers {
		l.Warn(msg, fields...)
	}
}

// Error logs to all loggers.
func (m *MultiLogger) Error(msg string, fields ...Field) {
	for _, l := range m.loggers {
		l.Error(msg, fields...)
	}
}

// Debug logs to all loggers.
func (m *MultiLogger) Debug(msg string, fields ...Field) {
	for _, l := range m.loggers {
		l.Debug(msg, fields...)
	}
}

// WithFields returns a MultiLogger where each inner logger
// has the given fields applied.
func (m *MultiLogger) WithFields(
	fields ...Field,
) Logger {
	newLoggers := make([]Logger, len(m.loggers))
	for i, l := range m.loggers {
		newLoggers[i] = l.WithFields(fields...)
	}
	return &MultiLogger{loggers: newLoggers}
}

// LogAPIRequest logs to all loggers.
func (m *MultiLogger) LogAPIRequest(
	request APIRequestLog,
) {
	for _, l := range m.loggers {
		l.LogAPIRequest(request)
	}
}

// LogAPIResponse logs to all loggers.
func (m *MultiLogger) LogAPIResponse(
	response APIResponseLog,
) {
	for _, l := range m.loggers {
		l.LogAPIResponse(response)
	}
}

// Close closes all loggers, returning the last error.
func (m *MultiLogger) Close() error {
	var lastErr error
	for _, l := range m.loggers {
		if err := l.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
