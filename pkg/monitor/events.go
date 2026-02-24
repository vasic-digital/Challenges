package monitor

import (
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// EventType represents the type of challenge event.
type EventType string

const (
	EventStarted             EventType = "started"
	EventConfigured          EventType = "configured"
	EventValidated           EventType = "validated"
	EventExecuting           EventType = "executing"
	EventProgress            EventType = "progress"
	EventExecutingCompleted  EventType = "executing_completed"
	EventAssertionsEvaluated EventType = "assertions_evaluated"
	EventCleanupStarted      EventType = "cleanup_started"
	EventCleanupCompleted    EventType = "cleanup_completed"
	EventCompleted           EventType = "completed"
	EventFailed              EventType = "failed"
	EventSkipped             EventType = "skipped"
	EventTimedOut            EventType = "timed_out"
	EventStuck               EventType = "stuck"
	EventMetric              EventType = "metric"
	EventLog                 EventType = "log"
)

// ChallengeEvent represents a lifecycle event during challenge execution.
type ChallengeEvent struct {
	Type         EventType              `json:"type"`
	ChallengeID  challenge.ID           `json:"challenge_id"`
	Name         string                 `json:"name"`
	Category     string                 `json:"category,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	ProgressData map[string]interface{} `json:"progress_data,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Stage        string                 `json:"stage,omitempty"`
}
