package challenge

import "time"

// Status constants for challenge execution outcomes.
const (
	StatusPending  = "pending"
	StatusRunning  = "running"
	StatusPassed   = "passed"
	StatusFailed   = "failed"
	StatusSkipped  = "skipped"
	StatusTimedOut = "timed_out"
	StatusError    = "error"
)

// Result captures the complete outcome of a challenge execution,
// including timing, assertion results, metrics, and log paths.
type Result struct {
	// ChallengeID is the unique identifier of the challenge.
	ChallengeID ID `json:"challenge_id"`

	// ChallengeName is the human-readable name.
	ChallengeName string `json:"challenge_name"`

	// Status is one of the Status* constants.
	Status string `json:"status"`

	// StartTime is when execution began.
	StartTime time.Time `json:"start_time"`

	// EndTime is when execution finished.
	EndTime time.Time `json:"end_time"`

	// Duration is the wall-clock execution time.
	Duration time.Duration `json:"duration"`

	// Assertions holds the evaluated assertion results.
	Assertions []AssertionResult `json:"assertions"`

	// Metrics holds named metric values collected during
	// execution.
	Metrics map[string]MetricValue `json:"metrics"`

	// Outputs holds named string outputs produced by the
	// challenge.
	Outputs map[string]string `json:"outputs"`

	// Logs contains paths to log files written during execution.
	Logs LogPaths `json:"logs"`

	// Error contains the error message if the challenge failed
	// with an unexpected error.
	Error string `json:"error,omitempty"`
}

// AssertionResult captures the outcome of a single assertion
// evaluation.
type AssertionResult struct {
	// Type is the assertion type that was evaluated.
	Type string `json:"type"`

	// Target is the name of the output or metric checked.
	Target string `json:"target"`

	// Expected is the value the assertion expected.
	Expected any `json:"expected"`

	// Actual is the value that was observed.
	Actual any `json:"actual"`

	// Passed indicates whether the assertion succeeded.
	Passed bool `json:"passed"`

	// Message is a human-readable description of the result.
	Message string `json:"message"`
}

// MetricValue represents a single named metric with its unit.
type MetricValue struct {
	// Name is the metric identifier.
	Name string `json:"name"`

	// Value is the numeric metric value.
	Value float64 `json:"value"`

	// Unit describes the measurement unit (e.g., "ms", "bytes",
	// "requests/sec").
	Unit string `json:"unit"`
}

// LogPaths holds file paths for logs generated during challenge
// execution.
type LogPaths struct {
	// ChallengeLog is the main challenge execution log.
	ChallengeLog string `json:"challenge_log"`

	// OutputLog captures stdout/stderr from the challenge.
	OutputLog string `json:"output_log"`

	// APIRequests logs outbound API request details.
	APIRequests string `json:"api_requests"`

	// APIResponses logs inbound API response details.
	APIResponses string `json:"api_responses"`
}

// AllPassed returns true if every assertion in the result passed.
func (r *Result) AllPassed() bool {
	for _, a := range r.Assertions {
		if !a.Passed {
			return false
		}
	}
	return true
}

// IsFinal returns true if the status is a terminal state.
func (r *Result) IsFinal() bool {
	switch r.Status {
	case StatusPassed, StatusFailed, StatusSkipped,
		StatusTimedOut, StatusError:
		return true
	}
	return false
}
