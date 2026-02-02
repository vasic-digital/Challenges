package monitor

import (
	"sync"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// DashboardData provides a real-time snapshot of challenge execution state.
type DashboardData struct {
	mu         sync.RWMutex
	RunID      string                         `json:"run_id"`
	StartTime  time.Time                      `json:"start_time"`
	Status     string                         `json:"status"` // running, completed, failed
	Challenges map[challenge.ID]ChallengeState `json:"challenges"`
	Summary    DashboardSummary               `json:"summary"`
}

// ChallengeState represents the current state of a challenge in the dashboard.
type ChallengeState struct {
	ID        challenge.ID  `json:"id"`
	Name      string        `json:"name"`
	Category  string        `json:"category"`
	Status    string        `json:"status"`
	StartTime *time.Time    `json:"start_time,omitempty"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
	Message   string        `json:"message,omitempty"`
}

// DashboardSummary holds aggregate stats for the dashboard.
type DashboardSummary struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	Failed   int     `json:"failed"`
	Skipped  int     `json:"skipped"`
	Running  int     `json:"running"`
	Pending  int     `json:"pending"`
	PassRate float64 `json:"pass_rate"`
	Elapsed  string  `json:"elapsed"`
}

// NewDashboardData creates a new dashboard data instance.
func NewDashboardData(runID string) *DashboardData {
	return &DashboardData{
		RunID:      runID,
		StartTime:  time.Now(),
		Status:     "running",
		Challenges: make(map[challenge.ID]ChallengeState),
	}
}

// UpdateFromEvent updates dashboard state from a challenge event.
func (d *DashboardData) UpdateFromEvent(event ChallengeEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	state, exists := d.Challenges[event.ChallengeID]
	if !exists {
		state = ChallengeState{
			ID:   event.ChallengeID,
			Name: event.Name,
		}
	}

	switch event.Type {
	case EventStarted:
		state.Status = "running"
		state.StartTime = &now
	case EventCompleted:
		state.Status = "passed"
		state.EndTime = &now
		state.Duration = event.Duration
	case EventFailed:
		state.Status = "failed"
		state.EndTime = &now
		state.Message = event.Message
	case EventSkipped:
		state.Status = "skipped"
	case EventTimedOut:
		state.Status = "timed_out"
		state.EndTime = &now
	}

	d.Challenges[event.ChallengeID] = state
	d.recalcSummary()
}

func (d *DashboardData) recalcSummary() {
	s := DashboardSummary{}
	for _, ch := range d.Challenges {
		s.Total++
		switch ch.Status {
		case "passed":
			s.Passed++
		case "failed":
			s.Failed++
		case "skipped":
			s.Skipped++
		case "running":
			s.Running++
		default:
			s.Pending++
		}
	}
	if completed := s.Passed + s.Failed; completed > 0 {
		s.PassRate = float64(s.Passed) / float64(completed) * 100
	}
	s.Elapsed = time.Since(d.StartTime).Round(time.Millisecond).String()
	d.Summary = s
}

// Snapshot returns a copy of the current dashboard state.
func (d *DashboardData) Snapshot() DashboardData {
	d.mu.RLock()
	defer d.mu.RUnlock()
	snap := *d
	snap.Challenges = make(map[challenge.ID]ChallengeState, len(d.Challenges))
	for k, v := range d.Challenges {
		snap.Challenges[k] = v
	}
	return snap
}

// SetStatus sets the overall run status.
func (d *DashboardData) SetStatus(status string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Status = status
}

// BuildDashboardData creates a DashboardData snapshot from an
// EventCollector by replaying all collected events.
func BuildDashboardData(
	collector *EventCollector,
) *DashboardData {
	data := NewDashboardData("snapshot")
	for _, event := range collector.Events() {
		data.UpdateFromEvent(event)
	}
	return data
}
