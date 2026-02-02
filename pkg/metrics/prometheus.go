package metrics

import (
	"time"
)

// PrometheusMetrics implements ChallengeMetrics using counters and histograms.
// It uses simple in-memory storage; real Prometheus integration is done
// by the host application using prometheus/client_golang.
type PrometheusMetrics struct {
	executions map[string]int
	assertions map[string]int
	durations  map[string][]time.Duration
	runTotal   int
	active     int
}

// NewPrometheusMetrics creates a new PrometheusMetrics instance.
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		executions: make(map[string]int),
		assertions: make(map[string]int),
		durations:  make(map[string][]time.Duration),
	}
}

func (m *PrometheusMetrics) RecordExecution(challengeID, status string, duration time.Duration) {
	key := challengeID + ":" + status
	m.executions[key]++
	m.durations[challengeID] = append(m.durations[challengeID], duration)
}

func (m *PrometheusMetrics) RecordAssertion(challengeID, evaluator string, passed bool) {
	status := "failed"
	if passed {
		status = "passed"
	}
	key := challengeID + ":" + evaluator + ":" + status
	m.assertions[key]++
}

func (m *PrometheusMetrics) IncrementRunTotal() {
	m.runTotal++
}

func (m *PrometheusMetrics) SetActiveChallenges(count int) {
	m.active = count
}

// ExecutionCount returns the count for a challenge+status combination.
func (m *PrometheusMetrics) ExecutionCount(challengeID, status string) int {
	return m.executions[challengeID+":"+status]
}

// RunTotal returns the total number of runs.
func (m *PrometheusMetrics) RunTotal() int {
	return m.runTotal
}

// ActiveChallenges returns the current active challenges gauge.
func (m *PrometheusMetrics) ActiveChallenges() int {
	return m.active
}
