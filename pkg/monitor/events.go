package monitor

import (
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// EventType represents the type of challenge event.
type EventType string

const (
	EventStarted   EventType = "started"
	EventCompleted EventType = "completed"
	EventFailed    EventType = "failed"
	EventSkipped   EventType = "skipped"
	EventTimedOut  EventType = "timed_out"
	EventMetric    EventType = "metric"
	EventLog       EventType = "log"
)

// ChallengeEvent represents a lifecycle event during challenge execution.
type ChallengeEvent struct {
	Type        EventType    `json:"type"`
	ChallengeID challenge.ID `json:"challenge_id"`
	Name        string       `json:"name"`
	Category    string       `json:"category,omitempty"`
	Status      string       `json:"status,omitempty"`
	Message     string       `json:"message,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Timestamp   time.Time    `json:"timestamp"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}
