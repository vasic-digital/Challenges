package monitor

import (
	"sync"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// EventCollector captures challenge events and timing data.
type EventCollector struct {
	mu       sync.RWMutex
	events   []ChallengeEvent
	handlers []func(ChallengeEvent)
	stats    CollectorStats
}

// CollectorStats holds aggregate statistics.
type CollectorStats struct {
	Total     int           `json:"total"`
	Passed    int           `json:"passed"`
	Failed    int           `json:"failed"`
	Skipped   int           `json:"skipped"`
	TimedOut  int           `json:"timed_out"`
	StartTime time.Time    `json:"start_time"`
	Duration  time.Duration `json:"duration"`
}

// NewEventCollector creates a new event collector.
func NewEventCollector() *EventCollector {
	return &EventCollector{
		events: make([]ChallengeEvent, 0, 64),
		stats:  CollectorStats{StartTime: time.Now()},
	}
}

// OnEvent registers a handler to be called for each event.
func (c *EventCollector) OnEvent(handler func(ChallengeEvent)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, handler)
}

// Emit records an event and notifies all handlers.
func (c *EventCollector) Emit(event ChallengeEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	c.mu.Lock()
	c.events = append(c.events, event)
	c.stats.Total++
	switch event.Type {
	case EventCompleted:
		c.stats.Passed++
	case EventFailed:
		c.stats.Failed++
	case EventSkipped:
		c.stats.Skipped++
	case EventTimedOut:
		c.stats.TimedOut++
	}
	c.stats.Duration = time.Since(c.stats.StartTime)
	handlers := make([]func(ChallengeEvent), len(c.handlers))
	copy(handlers, c.handlers)
	c.mu.Unlock()

	for _, h := range handlers {
		h(event)
	}
}

// EmitStarted emits a challenge started event.
func (c *EventCollector) EmitStarted(id challenge.ID, name string) {
	c.Emit(ChallengeEvent{
		Type:        EventStarted,
		ChallengeID: id,
		Name:        name,
		Timestamp:   time.Now(),
	})
}

// EmitCompleted emits a challenge completed event.
func (c *EventCollector) EmitCompleted(id challenge.ID, name string, duration time.Duration) {
	c.Emit(ChallengeEvent{
		Type:        EventCompleted,
		ChallengeID: id,
		Name:        name,
		Status:      "passed",
		Duration:    duration,
		Timestamp:   time.Now(),
	})
}

// EmitFailed emits a challenge failed event.
func (c *EventCollector) EmitFailed(id challenge.ID, name string, msg string) {
	c.Emit(ChallengeEvent{
		Type:        EventFailed,
		ChallengeID: id,
		Name:        name,
		Status:      "failed",
		Message:     msg,
		Timestamp:   time.Now(),
	})
}

// Events returns a copy of all collected events.
func (c *EventCollector) Events() []ChallengeEvent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]ChallengeEvent, len(c.events))
	copy(result, c.events)
	return result
}

// Stats returns the current aggregate statistics.
func (c *EventCollector) Stats() CollectorStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s := c.stats
	s.Duration = time.Since(s.StartTime)
	return s
}

// Reset clears all collected events and statistics.
func (c *EventCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = c.events[:0]
	c.stats = CollectorStats{StartTime: time.Now()}
}
