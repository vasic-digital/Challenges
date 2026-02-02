package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrometheusMetrics_ImplementsInterface(t *testing.T) {
	var _ ChallengeMetrics = &PrometheusMetrics{}
}

func TestPrometheusMetrics_Durations(t *testing.T) {
	m := NewPrometheusMetrics()
	m.RecordExecution("ch-1", "passed", 2*time.Second)
	m.RecordExecution("ch-1", "passed", 3*time.Second)

	assert.Len(t, m.durations["ch-1"], 2)
}

func TestNoopMetrics_ImplementsInterface(t *testing.T) {
	var _ ChallengeMetrics = &NoopMetrics{}
}
