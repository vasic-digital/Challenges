package logging

import "strings"

// RedactingLogger is a decorator that redacts sensitive strings
// from log messages and field values before passing them to the
// inner logger.
type RedactingLogger struct {
	inner   Logger
	secrets []string
}

// NewRedactingLogger creates a logger that redacts the given
// secrets from all messages and string field values.
func NewRedactingLogger(
	inner Logger,
	secrets ...string,
) *RedactingLogger {
	return &RedactingLogger{
		inner:   inner,
		secrets: secrets,
	}
}

func (r *RedactingLogger) redact(msg string) string {
	result := msg
	for _, secret := range r.secrets {
		if secret != "" && len(secret) > 4 {
			result = strings.ReplaceAll(
				result, secret, redactValue(secret),
			)
		}
	}
	return result
}

// redactValue masks all but the first 4 characters.
func redactValue(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}

func (r *RedactingLogger) redactFields(
	fields []Field,
) []Field {
	result := make([]Field, len(fields))
	for i, f := range fields {
		if str, ok := f.Value.(string); ok {
			result[i] = Field{
				Key:   f.Key,
				Value: r.redact(str),
			}
		} else {
			result[i] = f
		}
	}
	return result
}

// Info logs a redacted informational message.
func (r *RedactingLogger) Info(
	msg string, fields ...Field,
) {
	r.inner.Info(r.redact(msg), r.redactFields(fields)...)
}

// Warn logs a redacted warning message.
func (r *RedactingLogger) Warn(
	msg string, fields ...Field,
) {
	r.inner.Warn(r.redact(msg), r.redactFields(fields)...)
}

// Error logs a redacted error message.
func (r *RedactingLogger) Error(
	msg string, fields ...Field,
) {
	r.inner.Error(r.redact(msg), r.redactFields(fields)...)
}

// Debug logs a redacted debug message.
func (r *RedactingLogger) Debug(
	msg string, fields ...Field,
) {
	r.inner.Debug(r.redact(msg), r.redactFields(fields)...)
}

// WithFields returns a RedactingLogger wrapping a new inner
// logger with the given fields applied.
func (r *RedactingLogger) WithFields(
	fields ...Field,
) Logger {
	return &RedactingLogger{
		inner: r.inner.WithFields(
			r.redactFields(fields)...,
		),
		secrets: r.secrets,
	}
}

// LogAPIRequest logs an API request with redacted headers.
func (r *RedactingLogger) LogAPIRequest(
	request APIRequestLog,
) {
	request.Headers = redactHeaders(request.Headers)
	r.inner.LogAPIRequest(request)
}

// LogAPIResponse logs an API response with redacted headers.
func (r *RedactingLogger) LogAPIResponse(
	response APIResponseLog,
) {
	response.Headers = redactHeaders(response.Headers)
	r.inner.LogAPIResponse(response)
}

// Close closes the inner logger.
func (r *RedactingLogger) Close() error {
	return r.inner.Close()
}

// redactHeaders replaces values of sensitive headers with
// "****".
func redactHeaders(
	headers map[string]string,
) map[string]string {
	if headers == nil {
		return nil
	}

	sensitiveKeys := map[string]bool{
		"authorization":  true,
		"x-api-key":      true,
		"api-key":         true,
		"x-auth-token":    true,
		"x-access-token":  true,
		"bearer":          true,
	}

	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if sensitiveKeys[strings.ToLower(k)] {
			result[k] = "****"
		} else {
			result[k] = v
		}
	}
	return result
}
