package logging

// NullLogger discards all log output. It is useful for tests
// and benchmarks where logging overhead is not desired.
type NullLogger struct{}

// Info is a no-op.
func (NullLogger) Info(_ string, _ ...Field) {}

// Warn is a no-op.
func (NullLogger) Warn(_ string, _ ...Field) {}

// Error is a no-op.
func (NullLogger) Error(_ string, _ ...Field) {}

// Debug is a no-op.
func (NullLogger) Debug(_ string, _ ...Field) {}

// WithFields returns the NullLogger itself.
func (NullLogger) WithFields(_ ...Field) Logger {
	return NullLogger{}
}

// LogAPIRequest is a no-op.
func (NullLogger) LogAPIRequest(_ APIRequestLog) {}

// LogAPIResponse is a no-op.
func (NullLogger) LogAPIResponse(_ APIResponseLog) {}

// Close is a no-op.
func (NullLogger) Close() error { return nil }
