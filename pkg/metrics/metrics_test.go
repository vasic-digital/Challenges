package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrometheusMetrics_RecordExecution(t *testing.T) {
	m := NewPrometheusMetrics()
	m.RecordExecution("ch-1", "passed", 2*time.Second)
	m.RecordExecution("ch-1", "passed", 3*time.Second)
	m.RecordExecution("ch-2", "failed", time.Second)

	assert.Equal(t, 2, m.ExecutionCount("ch-1", "passed"))
	assert.Equal(t, 1, m.ExecutionCount("ch-2", "failed"))
	assert.Equal(t, 0, m.ExecutionCount("ch-3", "passed"))
}

func TestPrometheusMetrics_RecordAssertion(t *testing.T) {
	m := NewPrometheusMetrics()
	m.RecordAssertion("ch-1", "not_empty", true)
	m.RecordAssertion("ch-1", "not_empty", false)

	assert.Equal(t, 1, m.assertions["ch-1:not_empty:passed"])
	assert.Equal(t, 1, m.assertions["ch-1:not_empty:failed"])
}

func TestPrometheusMetrics_RunTotal(t *testing.T) {
	m := NewPrometheusMetrics()
	m.IncrementRunTotal()
	m.IncrementRunTotal()
	assert.Equal(t, 2, m.RunTotal())
}

func TestPrometheusMetrics_ActiveChallenges(t *testing.T) {
	m := NewPrometheusMetrics()
	m.SetActiveChallenges(5)
	assert.Equal(t, 5, m.ActiveChallenges())
}

func TestNoopMetrics(t *testing.T) {
	m := &NoopMetrics{}
	// Should not panic
	m.RecordExecution("ch", "passed", time.Second)
	m.RecordAssertion("ch", "test", true)
	m.IncrementRunTotal()
	m.SetActiveChallenges(0)
}
