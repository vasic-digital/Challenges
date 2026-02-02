package metrics

import "time"

// ChallengeMetrics defines the interface for recording challenge metrics.
type ChallengeMetrics interface {
	// RecordExecution records a challenge execution.
	RecordExecution(challengeID, status string, duration time.Duration)
	// RecordAssertion records an assertion evaluation.
	RecordAssertion(challengeID, evaluator string, passed bool)
	// IncrementRunTotal increments the total run counter.
	IncrementRunTotal()
	// SetActiveChallenges sets the gauge of active challenges.
	SetActiveChallenges(count int)
}

// NoopMetrics is a no-op implementation of ChallengeMetrics
// useful for testing or when metrics collection is disabled.
type NoopMetrics struct{}

func (NoopMetrics) RecordExecution(_, _ string, _ time.Duration) {}
func (NoopMetrics) RecordAssertion(_, _ string, _ bool)          {}
func (NoopMetrics) IncrementRunTotal()                           {}
func (NoopMetrics) SetActiveChallenges(_ int)                    {}
