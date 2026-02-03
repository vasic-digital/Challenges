package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNullLogger_Info verifies Info is a no-op.
func TestNullLogger_Info(t *testing.T) {
	tests := []struct {
		name   string
		msg    string
		fields []Field
	}{
		{
			name:   "empty message",
			msg:    "",
			fields: nil,
		},
		{
			name:   "simple message",
			msg:    "test message",
			fields: nil,
		},
		{
			name:   "with fields",
			msg:    "message with fields",
			fields: []Field{LogField("key", "value"), IntField("count", 42)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nl := NullLogger{}
			// Should not panic
			nl.Info(tt.msg, tt.fields...)
		})
	}
}

// TestNullLogger_Warn verifies Warn is a no-op.
func TestNullLogger_Warn(t *testing.T) {
	tests := []struct {
		name   string
		msg    string
		fields []Field
	}{
		{
			name:   "empty message",
			msg:    "",
			fields: nil,
		},
		{
			name:   "warning message",
			msg:    "potential issue detected",
			fields: nil,
		},
		{
			name:   "with fields",
			msg:    "warning with context",
			fields: []Field{StringField("component", "database")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nl := NullLogger{}
			// Should not panic
			nl.Warn(tt.msg, tt.fields...)
		})
	}
}

// TestNullLogger_Error verifies Error is a no-op.
func TestNullLogger_Error(t *testing.T) {
	tests := []struct {
		name   string
		msg    string
		fields []Field
	}{
		{
			name:   "empty message",
			msg:    "",
			fields: nil,
		},
		{
			name:   "error message",
			msg:    "operation failed",
			fields: nil,
		},
		{
			name:   "with error field",
			msg:    "critical failure",
			fields: []Field{ErrorField(assert.AnError)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nl := NullLogger{}
			// Should not panic
			nl.Error(tt.msg, tt.fields...)
		})
	}
}

// TestNullLogger_Debug verifies Debug is a no-op.
func TestNullLogger_Debug(t *testing.T) {
	tests := []struct {
		name   string
		msg    string
		fields []Field
	}{
		{
			name:   "empty message",
			msg:    "",
			fields: nil,
		},
		{
			name:   "debug trace",
			msg:    "entering function",
			fields: nil,
		},
		{
			name:   "with multiple fields",
			msg:    "detailed debug info",
			fields: []Field{
				StringField("func", "process"),
				Int64Field("duration_ns", 12345),
				BoolField("success", true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nl := NullLogger{}
			// Should not panic
			nl.Debug(tt.msg, tt.fields...)
		})
	}
}

// TestNullLogger_WithFields verifies WithFields returns NullLogger.
func TestNullLogger_WithFields(t *testing.T) {
	tests := []struct {
		name   string
		fields []Field
	}{
		{
			name:   "no fields",
			fields: nil,
		},
		{
			name:   "single field",
			fields: []Field{LogField("request_id", "abc")},
		},
		{
			name:   "multiple fields",
			fields: []Field{StringField("a", "1"), StringField("b", "2")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nl := NullLogger{}
			result := nl.WithFields(tt.fields...)

			require.NotNil(t, result)
			_, ok := result.(NullLogger)
			assert.True(t, ok, "WithFields should return NullLogger")
		})
	}
}

// TestNullLogger_LogAPIRequest verifies LogAPIRequest is a no-op.
func TestNullLogger_LogAPIRequest(t *testing.T) {
	tests := []struct {
		name    string
		request APIRequestLog
	}{
		{
			name:    "empty request",
			request: APIRequestLog{},
		},
		{
			name: "full request",
			request: APIRequestLog{
				Timestamp:  "2024-01-15T10:30:00Z",
				RequestID:  "req-123",
				Method:     "POST",
				URL:        "https://api.example.com/v1/data",
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"key": "value"}`,
				BodyLength: 16,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nl := NullLogger{}
			// Should not panic
			nl.LogAPIRequest(tt.request)
		})
	}
}

// TestNullLogger_LogAPIResponse verifies LogAPIResponse is a no-op.
func TestNullLogger_LogAPIResponse(t *testing.T) {
	tests := []struct {
		name     string
		response APIResponseLog
	}{
		{
			name:     "empty response",
			response: APIResponseLog{},
		},
		{
			name: "full response",
			response: APIResponseLog{
				Timestamp:      "2024-01-15T10:30:01Z",
				RequestID:      "req-123",
				StatusCode:     200,
				Headers:        map[string]string{"Content-Type": "application/json"},
				BodyPreview:    `{"status": "ok"}`,
				BodyLength:     16,
				ResponseTimeMs: 150,
			},
		},
		{
			name: "error response",
			response: APIResponseLog{
				RequestID:      "req-456",
				StatusCode:     500,
				ResponseTimeMs: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nl := NullLogger{}
			// Should not panic
			nl.LogAPIResponse(tt.response)
		})
	}
}

// TestNullLogger_Close verifies Close returns nil.
func TestNullLogger_Close(t *testing.T) {
	nl := NullLogger{}
	err := nl.Close()
	assert.NoError(t, err)
}

// TestNullLogger_ChainedOperations verifies chained WithFields and logging.
func TestNullLogger_ChainedOperations(t *testing.T) {
	nl := NullLogger{}

	// Chain multiple WithFields calls
	child1 := nl.WithFields(LogField("level1", "a"))
	child2 := child1.WithFields(LogField("level2", "b"))
	child3 := child2.WithFields(LogField("level3", "c"))

	// All should work without panic
	child3.Info("chained message")
	child3.Warn("chained warning")
	child3.Error("chained error")
	child3.Debug("chained debug")

	err := child3.Close()
	assert.NoError(t, err)
}

// TestNullLogger_ConcurrentAccess verifies thread safety.
func TestNullLogger_ConcurrentAccess(t *testing.T) {
	nl := NullLogger{}
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			nl.Info("concurrent message", IntField("goroutine", n))
			nl.Warn("concurrent warning", IntField("goroutine", n))
			nl.Error("concurrent error", IntField("goroutine", n))
			nl.Debug("concurrent debug", IntField("goroutine", n))
			nl.WithFields(IntField("n", n))
			nl.LogAPIRequest(APIRequestLog{RequestID: "req"})
			nl.LogAPIResponse(APIResponseLog{RequestID: "req"})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
