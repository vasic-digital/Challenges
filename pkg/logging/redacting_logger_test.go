package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Info(msg string, fields ...Field) {
	m.Called(msg, fields)
}
func (m *mockLogger) Warn(msg string, fields ...Field) {
	m.Called(msg, fields)
}
func (m *mockLogger) Error(msg string, fields ...Field) {
	m.Called(msg, fields)
}
func (m *mockLogger) Debug(msg string, fields ...Field) {
	m.Called(msg, fields)
}

// fieldsMatcher matches any []Field value.
type fieldsMatcher struct{}

func (fieldsMatcher) Matches(v interface{}) bool {
	_, ok := v.([]Field)
	return ok
}
func (fieldsMatcher) String() string {
	return "[]Field"
}
func (m *mockLogger) WithFields(fields ...Field) Logger {
	args := m.Called(fields)
	return args.Get(0).(Logger)
}
func (m *mockLogger) LogAPIRequest(req APIRequestLog) {
	m.Called(req)
}
func (m *mockLogger) LogAPIResponse(resp APIResponseLog) {
	m.Called(resp)
}
func (m *mockLogger) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestRedactingLogger_RedactsMessage(t *testing.T) {
	inner := new(mockLogger)
	secret := "sk-ant-1234567890abcdef"
	logger := NewRedactingLogger(inner, secret)

	expected := "sk-a*******************"
	inner.On(
		"Info", "key: "+expected, mock.MatchedBy(func(fields []Field) bool {
			return true
		}),
	).Return()

	logger.Info("key: " + secret)
	inner.AssertExpectations(t)
}

func TestRedactingLogger_RedactsFields(t *testing.T) {
	inner := new(mockLogger)
	secret := "supersecretapikey123"
	logger := NewRedactingLogger(inner, secret)

	inner.On(
		"Warn", "msg", mock.MatchedBy(
			func(fields []Field) bool {
				if len(fields) != 1 {
					return false
				}
				val, ok := fields[0].Value.(string)
				return ok && val != secret
			},
		),
	).Return()

	logger.Warn("msg", LogField("token", secret))
	inner.AssertExpectations(t)
}

func TestRedactingLogger_ShortSecretIgnored(t *testing.T) {
	inner := new(mockLogger)
	logger := NewRedactingLogger(inner, "ab")

	inner.On("Error", "ab is short", mock.Anything).Return()

	logger.Error("ab is short")
	inner.AssertExpectations(t)
}

func TestRedactingLogger_WithFields(t *testing.T) {
	inner := new(mockLogger)
	childInner := new(mockLogger)
	secret := "longsecretvalue12345"
	logger := NewRedactingLogger(inner, secret)

	inner.On(
		"WithFields", mock.Anything,
	).Return(childInner)

	child := logger.WithFields(LogField("k", "v"))
	assert.NotNil(t, child)

	rl, ok := child.(*RedactingLogger)
	assert.True(t, ok)
	assert.Equal(t, []string{secret}, rl.secrets)
}

func TestRedactingLogger_LogAPIRequest_RedactsHeaders(
	t *testing.T,
) {
	inner := new(mockLogger)
	logger := NewRedactingLogger(inner, "secret")

	inner.On(
		"LogAPIRequest", mock.MatchedBy(
			func(req APIRequestLog) bool {
				return req.Headers["Authorization"] == "****"
			},
		),
	).Return()

	logger.LogAPIRequest(APIRequestLog{
		Headers: map[string]string{
			"Authorization": "Bearer token",
			"Content-Type":  "application/json",
		},
	})
	inner.AssertExpectations(t)
}

func TestRedactingLogger_LogAPIResponse_RedactsHeaders(
	t *testing.T,
) {
	inner := new(mockLogger)
	logger := NewRedactingLogger(inner, "secret")

	inner.On(
		"LogAPIResponse", mock.MatchedBy(
			func(resp APIResponseLog) bool {
				return resp.Headers["X-Api-Key"] == "****"
			},
		),
	).Return()

	logger.LogAPIResponse(APIResponseLog{
		Headers: map[string]string{
			"X-Api-Key":    "mykey",
			"Content-Type": "text/plain",
		},
	})
	inner.AssertExpectations(t)
}

func TestRedactingLogger_Close(t *testing.T) {
	inner := new(mockLogger)
	logger := NewRedactingLogger(inner, "secret")

	inner.On("Close").Return(nil)

	err := logger.Close()
	assert.NoError(t, err)
	inner.AssertExpectations(t)
}

func TestRedactHeaders_NilHeaders(t *testing.T) {
	result := redactHeaders(nil)
	assert.Nil(t, result)
}

func TestRedactHeaders_MixedHeaders(t *testing.T) {
	headers := map[string]string{
		"Authorization":  "Bearer abc",
		"X-Api-Key":      "key123",
		"Content-Type":   "application/json",
		"X-Auth-Token":   "tok",
		"Accept":         "text/html",
	}

	result := redactHeaders(headers)

	assert.Equal(t, "****", result["Authorization"])
	assert.Equal(t, "****", result["X-Api-Key"])
	assert.Equal(t, "application/json", result["Content-Type"])
	assert.Equal(t, "****", result["X-Auth-Token"])
	assert.Equal(t, "text/html", result["Accept"])
}
