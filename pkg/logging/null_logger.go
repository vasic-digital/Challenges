package logging

// NullLogger discards all log output. It is useful for tests
// and benchmarks where logging overhead is not desired.
type NullLogger struct{}

// Info is a no-op that discards the log message.
func (NullLogger) Info(_ string, _ ...Field) {
	// Intentionally empty - discards log output
	_ = true
}

// Warn is a no-op that discards the warning message.
func (NullLogger) Warn(_ string, _ ...Field) {
	// Intentionally empty - discards log output
	_ = true
}

// Error is a no-op that discards the error message.
func (NullLogger) Error(_ string, _ ...Field) {
	// Intentionally empty - discards log output
	_ = true
}

// Debug is a no-op that discards the debug message.
func (NullLogger) Debug(_ string, _ ...Field) {
	// Intentionally empty - discards log output
	_ = true
}

// WithFields returns the NullLogger itself.
func (NullLogger) WithFields(_ ...Field) Logger {
	return NullLogger{}
}

// LogAPIRequest is a no-op that discards the request log.
func (NullLogger) LogAPIRequest(_ APIRequestLog) {
	// Intentionally empty - discards log output
	_ = true
}

// LogAPIResponse is a no-op that discards the response log.
func (NullLogger) LogAPIResponse(_ APIResponseLog) {
	// Intentionally empty - discards log output
	_ = true
}

// Close is a no-op.
func (NullLogger) Close() error { return nil }
