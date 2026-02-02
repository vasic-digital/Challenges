package monitor

import (
	"sync"
	"testing"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"github.com/stretchr/testify/assert"
)

func TestEventCollector_Emit(t *testing.T) {
	c := NewEventCollector()

	var received []ChallengeEvent
	var mu sync.Mutex
	c.OnEvent(func(e ChallengeEvent) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	c.Emit(ChallengeEvent{
		Type:        EventStarted,
		ChallengeID: "ch-1",
		Name:        "Test",
	})

	mu.Lock()
	assert.Len(t, received, 1)
	assert.Equal(t, EventStarted, received[0].Type)
	assert.False(t, received[0].Timestamp.IsZero())
	mu.Unlock()
}

func TestEventCollector_EmitStarted(t *testing.T) {
	c := NewEventCollector()
	c.EmitStarted("ch-1", "Test Challenge")

	events := c.Events()
	assert.Len(t, events, 1)
	assert.Equal(t, EventStarted, events[0].Type)
	assert.Equal(t, challenge.ID("ch-1"), events[0].ChallengeID)
}

func TestEventCollector_EmitCompleted(t *testing.T) {
	c := NewEventCollector()
	c.EmitCompleted("ch-1", "Test", 5*time.Second)

	stats := c.Stats()
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 1, stats.Passed)
}

func TestEventCollector_EmitFailed(t *testing.T) {
	c := NewEventCollector()
	c.EmitFailed("ch-1", "Test", "assertion failed")

	stats := c.Stats()
	assert.Equal(t, 1, stats.Failed)

	events := c.Events()
	assert.Equal(t, "assertion failed", events[0].Message)
}

func TestEventCollector_Stats(t *testing.T) {
	c := NewEventCollector()
	c.EmitCompleted("ch-1", "Pass", time.Second)
	c.EmitFailed("ch-2", "Fail", "err")
	c.Emit(ChallengeEvent{Type: EventSkipped, ChallengeID: "ch-3"})
	c.Emit(ChallengeEvent{Type: EventTimedOut, ChallengeID: "ch-4"})

	stats := c.Stats()
	assert.Equal(t, 4, stats.Total)
	assert.Equal(t, 1, stats.Passed)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, 1, stats.Skipped)
	assert.Equal(t, 1, stats.TimedOut)
}

func TestEventCollector_Reset(t *testing.T) {
	c := NewEventCollector()
	c.EmitCompleted("ch-1", "Test", time.Second)
	c.Reset()

	assert.Empty(t, c.Events())
	assert.Equal(t, 0, c.Stats().Total)
}

func TestEventCollector_ConcurrentAccess(t *testing.T) {
	c := NewEventCollector()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.EmitStarted(challenge.ID("ch"), "Test")
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 100, c.Stats().Total)
}
