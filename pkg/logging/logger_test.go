package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestLogField(t *testing.T) {
	f := LogField("key", "value")
	assert.Equal(t, "key", f.Key)
	assert.Equal(t, "value", f.Value)
}

func TestStringField(t *testing.T) {
	f := StringField("name", "test")
	assert.Equal(t, "name", f.Key)
	assert.Equal(t, "test", f.Value)
}

func TestIntField(t *testing.T) {
	f := IntField("count", 42)
	assert.Equal(t, "count", f.Key)
	assert.Equal(t, 42, f.Value)
}

func TestInt64Field(t *testing.T) {
	f := Int64Field("ts", int64(1234567890))
	assert.Equal(t, "ts", f.Key)
	assert.Equal(t, int64(1234567890), f.Value)
}

func TestFloat64Field(t *testing.T) {
	f := Float64Field("score", 3.14)
	assert.Equal(t, "score", f.Key)
	assert.Equal(t, 3.14, f.Value)
}

func TestBoolField(t *testing.T) {
	f := BoolField("enabled", true)
	assert.Equal(t, "enabled", f.Key)
	assert.Equal(t, true, f.Value)
}

func TestErrorField_WithError(t *testing.T) {
	err := assert.AnError
	f := ErrorField(err)
	assert.Equal(t, "error", f.Key)
	assert.Equal(t, err.Error(), f.Value)
}

func TestErrorField_Nil(t *testing.T) {
	f := ErrorField(nil)
	assert.Equal(t, "error", f.Key)
	assert.Equal(t, "<nil>", f.Value)
}

func TestNullLogger_ImplementsInterface(t *testing.T) {
	var _ Logger = NullLogger{}
}

func TestNullLogger_AllMethodsSucceed(t *testing.T) {
	l := NullLogger{}
	l.Info("test")
	l.Warn("test")
	l.Error("test")
	l.Debug("test")
	l.LogAPIRequest(APIRequestLog{})
	l.LogAPIResponse(APIResponseLog{})

	child := l.WithFields(LogField("k", "v"))
	assert.NotNil(t, child)

	err := l.Close()
	assert.NoError(t, err)
}

func TestMultiLogger_ImplementsInterface(t *testing.T) {
	var _ Logger = &MultiLogger{}
}

func TestConsoleLogger_ImplementsInterface(t *testing.T) {
	var _ Logger = &ConsoleLogger{}
}

func TestJSONLogger_ImplementsInterface(t *testing.T) {
	var _ Logger = &JSONLogger{}
}

func TestRedactingLogger_ImplementsInterface(t *testing.T) {
	var _ Logger = &RedactingLogger{}
}
