package report

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoricalEntry_Fields(t *testing.T) {
	now := time.Now()
	entry := HistoricalEntry{
		Timestamp:        now,
		ChallengeID:      "test-1",
		Status:           "passed",
		Duration:         "5s",
		AssertionsPassed: 3,
		AssertionsTotal:  3,
		ResultsPath:      "/tmp/results/test-1",
	}
	assert.Equal(t, "test-1", entry.ChallengeID)
	assert.Equal(t, "passed", entry.Status)
	assert.Equal(t, "5s", entry.Duration)
	assert.Equal(t, 3, entry.AssertionsPassed)
	assert.Equal(t, 3, entry.AssertionsTotal)
	assert.Equal(t, "/tmp/results/test-1", entry.ResultsPath)
}

func TestHistoricalEntry_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	entry := HistoricalEntry{
		Timestamp:        now,
		ChallengeID:      "challenge-abc",
		Status:           "failed",
		Duration:         "10.5s",
		AssertionsPassed: 2,
		AssertionsTotal:  5,
		ResultsPath:      "/results/abc",
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded HistoricalEntry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, entry.ChallengeID, decoded.ChallengeID)
	assert.Equal(t, entry.Status, decoded.Status)
	assert.Equal(t, entry.Duration, decoded.Duration)
	assert.Equal(t, entry.AssertionsPassed, decoded.AssertionsPassed)
	assert.Equal(t, entry.AssertionsTotal, decoded.AssertionsTotal)
}

func TestHistoricalEntry_JSONTags(t *testing.T) {
	entry := HistoricalEntry{
		Timestamp:        time.Now(),
		ChallengeID:      "test-json",
		Status:           "passed",
		Duration:         "1s",
		AssertionsPassed: 1,
		AssertionsTotal:  1,
		ResultsPath:      "/results",
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "challenge_id")
	assert.Contains(t, raw, "status")
	assert.Contains(t, raw, "duration")
	assert.Contains(t, raw, "assertions_passed")
	assert.Contains(t, raw, "assertions_total")
	assert.Contains(t, raw, "results_path")
	assert.Contains(t, raw, "timestamp")
}

func TestHistoricalEntry_ZeroValues(t *testing.T) {
	entry := HistoricalEntry{}
	assert.Empty(t, entry.ChallengeID)
	assert.Empty(t, entry.Status)
	assert.Empty(t, entry.Duration)
	assert.Zero(t, entry.AssertionsPassed)
	assert.Zero(t, entry.AssertionsTotal)
	assert.Empty(t, entry.ResultsPath)
	assert.True(t, entry.Timestamp.IsZero())
}
