package challenge

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResult_AllPassed_True(t *testing.T) {
	r := &Result{
		Assertions: []AssertionResult{
			{Type: "not_empty", Passed: true},
			{Type: "contains", Passed: true},
			{Type: "min_length", Passed: true},
		},
	}
	assert.True(t, r.AllPassed())
}

func TestResult_AllPassed_False(t *testing.T) {
	r := &Result{
		Assertions: []AssertionResult{
			{Type: "not_empty", Passed: true},
			{Type: "contains", Passed: false},
			{Type: "min_length", Passed: true},
		},
	}
	assert.False(t, r.AllPassed())
}

func TestResult_AllPassed_Empty(t *testing.T) {
	r := &Result{
		Assertions: []AssertionResult{},
	}
	assert.True(t, r.AllPassed())
}

func TestResult_AllPassed_Nil(t *testing.T) {
	r := &Result{}
	assert.True(t, r.AllPassed())
}

func TestResult_IsFinal_TerminalStatuses(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{StatusPending, false},
		{StatusRunning, false},
		{StatusPassed, true},
		{StatusFailed, true},
		{StatusSkipped, true},
		{StatusTimedOut, true},
		{StatusError, true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			r := &Result{Status: tt.status}
			assert.Equal(t, tt.expected, r.IsFinal())
		})
	}
}

func TestResult_StatusConstantValues(t *testing.T) {
	statuses := []string{
		StatusPending, StatusRunning, StatusPassed,
		StatusFailed, StatusSkipped, StatusTimedOut, StatusError,
	}
	for _, s := range statuses {
		assert.NotEmpty(t, s)
	}
	assert.Len(t, statuses, 7)
}

func TestResult_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	r := &Result{
		ChallengeID:   "test-1",
		ChallengeName: "Test Challenge",
		Status:        StatusPassed,
		StartTime:     now,
		EndTime:       now.Add(5 * time.Second),
		Duration:      5 * time.Second,
		Assertions: []AssertionResult{
			{
				Type:     "not_empty",
				Target:   "output",
				Passed:   true,
				Message:  "output is not empty",
				Expected: nil,
				Actual:   "some output",
			},
		},
		Metrics: map[string]MetricValue{
			"latency": {Name: "latency", Value: 42.5, Unit: "ms"},
		},
		Outputs: map[string]string{
			"output": "some output",
		},
		Logs: LogPaths{
			ChallengeLog: "/tmp/logs/challenge.log",
			OutputLog:    "/tmp/logs/output.log",
		},
	}

	data, err := json.Marshal(r)
	require.NoError(t, err)

	var decoded Result
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, r.ChallengeID, decoded.ChallengeID)
	assert.Equal(t, r.ChallengeName, decoded.ChallengeName)
	assert.Equal(t, r.Status, decoded.Status)
	assert.Len(t, decoded.Assertions, 1)
	assert.True(t, decoded.Assertions[0].Passed)
}

func TestMetricValue_Fields(t *testing.T) {
	mv := MetricValue{
		Name:  "response_time",
		Value: 123.45,
		Unit:  "ms",
	}
	assert.Equal(t, "response_time", mv.Name)
	assert.Equal(t, 123.45, mv.Value)
	assert.Equal(t, "ms", mv.Unit)
}

func TestLogPaths_Fields(t *testing.T) {
	lp := LogPaths{
		ChallengeLog: "/logs/challenge.log",
		OutputLog:    "/logs/output.log",
		APIRequests:  "/logs/api_requests.log",
		APIResponses: "/logs/api_responses.log",
	}
	assert.Equal(t, "/logs/challenge.log", lp.ChallengeLog)
	assert.Equal(t, "/logs/output.log", lp.OutputLog)
	assert.Equal(t, "/logs/api_requests.log", lp.APIRequests)
	assert.Equal(t, "/logs/api_responses.log", lp.APIResponses)
}

func TestAssertionResult_Fields(t *testing.T) {
	ar := AssertionResult{
		Type:     "contains",
		Target:   "response_body",
		Expected: "success",
		Actual:   "operation success",
		Passed:   true,
		Message:  "response body contains success",
	}
	assert.Equal(t, "contains", ar.Type)
	assert.Equal(t, "response_body", ar.Target)
	assert.True(t, ar.Passed)
}
