package monitor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDashboardData_UpdateFromEvent(t *testing.T) {
	d := NewDashboardData("run-1")

	d.UpdateFromEvent(ChallengeEvent{
		Type:        EventStarted,
		ChallengeID: "ch-1",
		Name:        "Test",
	})

	snap := d.Snapshot()
	assert.Equal(t, 1, snap.Summary.Total)
	assert.Equal(t, 1, snap.Summary.Running)
	assert.Equal(t, "running", snap.Challenges["ch-1"].Status)

	d.UpdateFromEvent(ChallengeEvent{
		Type:        EventCompleted,
		ChallengeID: "ch-1",
		Name:        "Test",
		Duration:    2 * time.Second,
	})

	snap = d.Snapshot()
	assert.Equal(t, "passed", snap.Challenges["ch-1"].Status)
	assert.Equal(t, 1, snap.Summary.Passed)
	assert.Equal(t, float64(100), snap.Summary.PassRate)
}

func TestDashboardData_FailedEvent(t *testing.T) {
	d := NewDashboardData("run-2")
	d.UpdateFromEvent(ChallengeEvent{
		Type:        EventFailed,
		ChallengeID: "ch-1",
		Name:        "Fail Test",
		Message:     "assertion error",
	})

	snap := d.Snapshot()
	assert.Equal(t, "failed", snap.Challenges["ch-1"].Status)
	assert.Equal(t, "assertion error", snap.Challenges["ch-1"].Message)
	assert.Equal(t, 1, snap.Summary.Failed)
}

func TestDashboardData_SetStatus(t *testing.T) {
	d := NewDashboardData("run-3")
	d.SetStatus("completed")
	snap := d.Snapshot()
	assert.Equal(t, "completed", snap.Status)
}

func TestDashboardData_Snapshot_IsCopy(t *testing.T) {
	d := NewDashboardData("run-4")
	d.UpdateFromEvent(ChallengeEvent{
		Type:        EventStarted,
		ChallengeID: "ch-1",
		Name:        "Test",
	})

	snap := d.Snapshot()
	snap.Challenges["ch-2"] = ChallengeState{ID: "ch-2"}

	// Original should be unmodified
	d.mu.RLock()
	_, exists := d.Challenges["ch-2"]
	d.mu.RUnlock()
	assert.False(t, exists)
}
