package report

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

func TestAppendToHistory_MarshalError(t *testing.T) {
	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history.jsonl")

	// Save original and restore after test
	originalMarshal := jsonMarshal
	t.Cleanup(func() { jsonMarshal = originalMarshal })

	// Inject a failing marshaler
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, assert.AnError
	}

	result := &challenge.Result{
		ChallengeID: "test-001",
		Status:      challenge.StatusPassed,
		EndTime:     time.Now(),
		Duration:    time.Second,
	}

	err := AppendToHistory(historyPath, result, "/tmp/results")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal history entry")
}

func TestSaveMasterSummary_MarshalError(t *testing.T) {
	dir := t.TempDir()

	// Save original and restore after test
	originalMarshal := jsonMarshalIndent
	t.Cleanup(func() { jsonMarshalIndent = originalMarshal })

	// Inject a failing marshaler
	jsonMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
		return nil, assert.AnError
	}

	summary := BuildMasterSummary(nil)

	err := SaveMasterSummary(summary, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal summary")
}

func TestSaveMasterSummary_WriteJSONError(t *testing.T) {
	dir := t.TempDir()

	// Create a file where the JSON summary file should be created
	// to cause WriteFile to fail
	ts := time.Now().Format("20060102_150405")
	jsonPath := filepath.Join(dir, "master_summary_"+ts+".json")
	require.NoError(t, os.MkdirAll(jsonPath, 0755))

	summary := &MasterSummary{
		ID:          "test",
		GeneratedAt: time.Now(),
	}
	// Force the same timestamp
	summary.GeneratedAt, _ = time.Parse("20060102_150405", ts)

	err := SaveMasterSummary(summary, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write JSON summary")
}

func TestSaveMasterSummary_WriteMarkdownError(t *testing.T) {
	dir := t.TempDir()

	summary := BuildMasterSummary(nil)

	// Create a directory where the markdown file should be written
	ts := summary.GeneratedAt.Format("20060102_150405")
	mdPath := filepath.Join(dir, "master_summary_"+ts+".md")
	require.NoError(t, os.MkdirAll(mdPath, 0755))

	err := SaveMasterSummary(summary, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write Markdown summary")
}

func TestJSONReporter_WriteReport_MarshalError(t *testing.T) {
	// Save original and restore after test
	originalMarshal := jsonReportMarshal
	t.Cleanup(func() { jsonReportMarshal = originalMarshal })

	// Inject a failing marshaler
	jsonReportMarshal = func(v any) ([]byte, error) {
		return nil, assert.AnError
	}

	r := NewJSONReporter(t.TempDir(), false)
	result := &challenge.Result{
		ChallengeID: "test-001",
		Status:      challenge.StatusPassed,
	}

	var buf bytes.Buffer
	err := r.WriteReport(&buf, result)
	assert.Error(t, err)
}

func TestJSONReporter_GenerateReport_MarshalIndentError(t *testing.T) {
	// Save original and restore after test
	originalMarshal := jsonReportMarshalIndent
	t.Cleanup(func() { jsonReportMarshalIndent = originalMarshal })

	// Inject a failing marshaler
	jsonReportMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
		return nil, assert.AnError
	}

	r := NewJSONReporter(t.TempDir(), true)
	result := &challenge.Result{
		ChallengeID: "test-001",
		Status:      challenge.StatusPassed,
	}

	_, err := r.GenerateReport(result)
	assert.Error(t, err)
}

func TestJSONReporter_GenerateMasterSummary_MarshalError(t *testing.T) {
	// Save original and restore after test
	originalMarshal := jsonReportMarshal
	t.Cleanup(func() { jsonReportMarshal = originalMarshal })

	// Inject a failing marshaler
	jsonReportMarshal = func(v any) ([]byte, error) {
		return nil, assert.AnError
	}

	r := NewJSONReporter(t.TempDir(), false)
	results := []*challenge.Result{
		{ChallengeID: "test-001", Status: challenge.StatusPassed},
	}

	_, err := r.GenerateMasterSummary(results)
	assert.Error(t, err)
}

func TestJSONReporter_GenerateMasterSummary_MarshalIndentError(t *testing.T) {
	// Save original and restore after test
	originalMarshal := jsonReportMarshalIndent
	t.Cleanup(func() { jsonReportMarshalIndent = originalMarshal })

	// Inject a failing marshaler
	jsonReportMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
		return nil, assert.AnError
	}

	r := NewJSONReporter(t.TempDir(), true)
	results := []*challenge.Result{
		{ChallengeID: "test-001", Status: challenge.StatusPassed},
	}

	_, err := r.GenerateMasterSummary(results)
	assert.Error(t, err)
}
