package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

func TestBuildMasterSummary_Basic(t *testing.T) {
	results := makeTestResults()

	summary := BuildMasterSummary(results)

	assert.NotEmpty(t, summary.ID)
	assert.NotZero(t, summary.GeneratedAt)
	assert.Equal(t, 2, summary.TotalChallenges)
	assert.Equal(t, 1, summary.PassedChallenges)
	assert.Equal(t, 1, summary.FailedChallenges)
	assert.Equal(t, 0.5, summary.AveragePassRate)
	assert.Len(t, summary.Challenges, 2)
}

func TestBuildMasterSummary_Empty(t *testing.T) {
	summary := BuildMasterSummary(nil)

	assert.Equal(t, 0, summary.TotalChallenges)
	assert.Equal(t, float64(0), summary.AveragePassRate)
	assert.Empty(t, summary.Challenges)
}

func TestBuildMasterSummary_AssertionCounts(t *testing.T) {
	results := makeTestResults()

	summary := BuildMasterSummary(results)

	assert.Equal(t, 1, summary.Challenges[0].AssertionsPassed)
	assert.Equal(t, 2, summary.Challenges[0].AssertionsTotal)
	assert.Equal(t, 0, summary.Challenges[1].AssertionsPassed)
	assert.Equal(t, 0, summary.Challenges[1].AssertionsTotal)
}

func TestSaveMasterSummary(t *testing.T) {
	dir := t.TempDir()
	results := makeTestResults()
	summary := BuildMasterSummary(results)

	err := SaveMasterSummary(summary, dir)
	require.NoError(t, err)

	// Check JSON file exists
	matches, err := filepath.Glob(
		filepath.Join(dir, "master_summary_*.json"),
	)
	require.NoError(t, err)
	assert.Len(t, matches, 1)

	data, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.True(t, json.Valid(data))

	// Check Markdown file exists
	mdMatches, err := filepath.Glob(
		filepath.Join(dir, "master_summary_*.md"),
	)
	require.NoError(t, err)
	assert.Len(t, mdMatches, 1)

	// Check symlinks
	_, err = os.Lstat(
		filepath.Join(dir, "latest_summary.json"),
	)
	assert.NoError(t, err)
	_, err = os.Lstat(
		filepath.Join(dir, "latest_summary.md"),
	)
	assert.NoError(t, err)
}

func TestSaveMasterSummary_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	summary := BuildMasterSummary(nil)

	err := SaveMasterSummary(summary, dir)
	require.NoError(t, err)

	_, err = os.Stat(dir)
	assert.NoError(t, err)
}

func TestAppendToHistory(t *testing.T) {
	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history.jsonl")

	result := makeTestResult()
	err := AppendToHistory(
		historyPath, result, "/tmp/results",
	)
	require.NoError(t, err)

	// Append another entry
	result.ChallengeID = challenge.ID("test-002")
	err = AppendToHistory(
		historyPath, result, "/tmp/results2",
	)
	require.NoError(t, err)

	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)

	lines := splitNonEmpty(string(data))
	assert.Len(t, lines, 2)

	var entry HistoricalEntry
	err = json.Unmarshal([]byte(lines[0]), &entry)
	require.NoError(t, err)
	assert.Equal(t, "test-001", entry.ChallengeID)
	assert.Equal(t, "passed", entry.Status)
	assert.Equal(t, 1, entry.AssertionsPassed)
	assert.Equal(t, 2, entry.AssertionsTotal)
}

func splitNonEmpty(s string) []string {
	var result []string
	for _, line := range splitLines(s) {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
