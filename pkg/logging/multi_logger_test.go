package logging

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testMockLogger is a mock logger for testing MultiLogger delegation.
type testMockLogger struct {
	mock.Mock
}

func (m *testMockLogger) Info(msg string, fields ...Field) {
	m.Called(msg, fields)
}

func (m *testMockLogger) Warn(msg string, fields ...Field) {
	m.Called(msg, fields)
}

func (m *testMockLogger) Error(msg string, fields ...Field) {
	m.Called(msg, fields)
}

func (m *testMockLogger) Debug(msg string, fields ...Field) {
	m.Called(msg, fields)
}

func (m *testMockLogger) WithFields(fields ...Field) Logger {
	args := m.Called(fields)
	return args.Get(0).(Logger)
}

func (m *testMockLogger) LogAPIRequest(req APIRequestLog) {
	m.Called(req)
}

func (m *testMockLogger) LogAPIResponse(resp APIResponseLog) {
	m.Called(resp)
}

func (m *testMockLogger) Close() error {
	args := m.Called()
	return args.Error(0)
}

// TestNewMultiLogger verifies MultiLogger creation.
func TestNewMultiLogger(t *testing.T) {
	tests := []struct {
		name     string
		loggers  []Logger
		wantLen  int
	}{
		{
			name:    "empty loggers",
			loggers: []Logger{},
			wantLen: 0,
		},
		{
			name:    "single logger",
			loggers: []Logger{NullLogger{}},
			wantLen: 1,
		},
		{
			name:    "multiple loggers",
			loggers: []Logger{NullLogger{}, NullLogger{}, NullLogger{}},
			wantLen: 3,
		},
		{
			name:    "nil slice",
			loggers: nil,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ml := NewMultiLogger(tt.loggers...)
			require.NotNil(t, ml)
			assert.Len(t, ml.loggers, tt.wantLen)
		})
	}
}

// TestMultiLogger_Info verifies Info delegates to all loggers.
func TestMultiLogger_Info(t *testing.T) {
	tests := []struct {
		name       string
		msg        string
		fields     []Field
		numLoggers int
	}{
		{
			name:       "no fields",
			msg:        "info message",
			fields:     nil,
			numLoggers: 2,
		},
		{
			name:       "with fields",
			msg:        "info with fields",
			fields:     []Field{LogField("key", "value")},
			numLoggers: 3,
		},
		{
			name:       "single logger",
			msg:        "single",
			fields:     []Field{StringField("a", "b"), IntField("c", 1)},
			numLoggers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := make([]*testMockLogger, tt.numLoggers)
			loggers := make([]Logger, tt.numLoggers)
			for i := 0; i < tt.numLoggers; i++ {
				m := new(testMockLogger)
				m.On("Info", tt.msg, tt.fields).Return()
				mocks[i] = m
				loggers[i] = m
			}

			ml := NewMultiLogger(loggers...)
			ml.Info(tt.msg, tt.fields...)

			for _, m := range mocks {
				m.AssertExpectations(t)
			}
		})
	}
}

// TestMultiLogger_Warn verifies Warn delegates to all loggers.
func TestMultiLogger_Warn(t *testing.T) {
	tests := []struct {
		name       string
		msg        string
		fields     []Field
		numLoggers int
	}{
		{
			name:       "no fields",
			msg:        "warn message",
			fields:     nil,
			numLoggers: 2,
		},
		{
			name:       "with fields",
			msg:        "warn with fields",
			fields:     []Field{LogField("severity", "high")},
			numLoggers: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := make([]*testMockLogger, tt.numLoggers)
			loggers := make([]Logger, tt.numLoggers)
			for i := 0; i < tt.numLoggers; i++ {
				m := new(testMockLogger)
				m.On("Warn", tt.msg, tt.fields).Return()
				mocks[i] = m
				loggers[i] = m
			}

			ml := NewMultiLogger(loggers...)
			ml.Warn(tt.msg, tt.fields...)

			for _, m := range mocks {
				m.AssertExpectations(t)
			}
		})
	}
}

// TestMultiLogger_Error verifies Error delegates to all loggers.
func TestMultiLogger_Error(t *testing.T) {
	tests := []struct {
		name       string
		msg        string
		fields     []Field
		numLoggers int
	}{
		{
			name:       "no fields",
			msg:        "error occurred",
			fields:     nil,
			numLoggers: 2,
		},
		{
			name:       "with error field",
			msg:        "operation failed",
			fields:     []Field{ErrorField(errors.New("test error"))},
			numLoggers: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := make([]*testMockLogger, tt.numLoggers)
			loggers := make([]Logger, tt.numLoggers)
			for i := 0; i < tt.numLoggers; i++ {
				m := new(testMockLogger)
				m.On("Error", tt.msg, tt.fields).Return()
				mocks[i] = m
				loggers[i] = m
			}

			ml := NewMultiLogger(loggers...)
			ml.Error(tt.msg, tt.fields...)

			for _, m := range mocks {
				m.AssertExpectations(t)
			}
		})
	}
}

// TestMultiLogger_Debug verifies Debug delegates to all loggers.
func TestMultiLogger_Debug(t *testing.T) {
	tests := []struct {
		name       string
		msg        string
		fields     []Field
		numLoggers int
	}{
		{
			name:       "no fields",
			msg:        "debug message",
			fields:     nil,
			numLoggers: 2,
		},
		{
			name:       "with multiple fields",
			msg:        "debug trace",
			fields:     []Field{StringField("trace", "abc"), Int64Field("ts", 12345)},
			numLoggers: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := make([]*testMockLogger, tt.numLoggers)
			loggers := make([]Logger, tt.numLoggers)
			for i := 0; i < tt.numLoggers; i++ {
				m := new(testMockLogger)
				m.On("Debug", tt.msg, tt.fields).Return()
				mocks[i] = m
				loggers[i] = m
			}

			ml := NewMultiLogger(loggers...)
			ml.Debug(tt.msg, tt.fields...)

			for _, m := range mocks {
				m.AssertExpectations(t)
			}
		})
	}
}

// TestMultiLogger_WithFields verifies WithFields creates new MultiLogger.
func TestMultiLogger_WithFields(t *testing.T) {
	tests := []struct {
		name       string
		fields     []Field
		numLoggers int
	}{
		{
			name:       "single field",
			fields:     []Field{LogField("request_id", "abc123")},
			numLoggers: 2,
		},
		{
			name:       "multiple fields",
			fields:     []Field{StringField("a", "1"), StringField("b", "2")},
			numLoggers: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := make([]*testMockLogger, tt.numLoggers)
			childMocks := make([]*testMockLogger, tt.numLoggers)
			loggers := make([]Logger, tt.numLoggers)

			for i := 0; i < tt.numLoggers; i++ {
				m := new(testMockLogger)
				child := new(testMockLogger)
				m.On("WithFields", tt.fields).Return(child)
				mocks[i] = m
				childMocks[i] = child
				loggers[i] = m
			}

			ml := NewMultiLogger(loggers...)
			result := ml.WithFields(tt.fields...)

			require.NotNil(t, result)
			multiResult, ok := result.(*MultiLogger)
			require.True(t, ok)
			assert.Len(t, multiResult.loggers, tt.numLoggers)

			for _, m := range mocks {
				m.AssertExpectations(t)
			}
		})
	}
}

// TestMultiLogger_LogAPIRequest verifies LogAPIRequest delegates to all.
func TestMultiLogger_LogAPIRequest(t *testing.T) {
	request := APIRequestLog{
		RequestID: "req-123",
		Method:    "POST",
		URL:       "https://api.example.com/v1/test",
		Headers:   map[string]string{"Content-Type": "application/json"},
	}

	mocks := make([]*testMockLogger, 2)
	loggers := make([]Logger, 2)
	for i := 0; i < 2; i++ {
		m := new(testMockLogger)
		m.On("LogAPIRequest", request).Return()
		mocks[i] = m
		loggers[i] = m
	}

	ml := NewMultiLogger(loggers...)
	ml.LogAPIRequest(request)

	for _, m := range mocks {
		m.AssertExpectations(t)
	}
}

// TestMultiLogger_LogAPIResponse verifies LogAPIResponse delegates to all.
func TestMultiLogger_LogAPIResponse(t *testing.T) {
	response := APIResponseLog{
		RequestID:      "req-123",
		StatusCode:     200,
		Headers:        map[string]string{"Content-Type": "application/json"},
		ResponseTimeMs: 150,
	}

	mocks := make([]*testMockLogger, 2)
	loggers := make([]Logger, 2)
	for i := 0; i < 2; i++ {
		m := new(testMockLogger)
		m.On("LogAPIResponse", response).Return()
		mocks[i] = m
		loggers[i] = m
	}

	ml := NewMultiLogger(loggers...)
	ml.LogAPIResponse(response)

	for _, m := range mocks {
		m.AssertExpectations(t)
	}
}

// TestMultiLogger_Close verifies Close calls all loggers and returns last error.
func TestMultiLogger_Close(t *testing.T) {
	tests := []struct {
		name       string
		errors     []error
		wantErr    error
	}{
		{
			name:    "all succeed",
			errors:  []error{nil, nil},
			wantErr: nil,
		},
		{
			name:    "first fails only",
			errors:  []error{errors.New("first error"), nil},
			wantErr: errors.New("first error"),
		},
		{
			name:    "last fails",
			errors:  []error{nil, errors.New("last error")},
			wantErr: errors.New("last error"),
		},
		{
			name:    "both fail returns last",
			errors:  []error{errors.New("first"), errors.New("second")},
			wantErr: errors.New("second"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := make([]*testMockLogger, len(tt.errors))
			loggers := make([]Logger, len(tt.errors))
			for i, err := range tt.errors {
				m := new(testMockLogger)
				m.On("Close").Return(err)
				mocks[i] = m
				loggers[i] = m
			}

			ml := NewMultiLogger(loggers...)
			err := ml.Close()

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			for _, m := range mocks {
				m.AssertExpectations(t)
			}
		})
	}
}

// TestMultiLogger_EmptyLoggers verifies operations on empty MultiLogger.
func TestMultiLogger_EmptyLoggers(t *testing.T) {
	ml := NewMultiLogger()

	// All operations should be no-ops and not panic
	ml.Info("test")
	ml.Warn("test")
	ml.Error("test")
	ml.Debug("test")
	ml.LogAPIRequest(APIRequestLog{})
	ml.LogAPIResponse(APIResponseLog{})

	child := ml.WithFields(LogField("k", "v"))
	require.NotNil(t, child)

	err := ml.Close()
	assert.NoError(t, err)
}
