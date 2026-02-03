package report

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

func TestJSONReporter_GenerateReport_Pretty(t *testing.T) {
	r := NewJSONReporter(t.TempDir(), true)
	result := makeTestResult()

	data, err := r.GenerateReport(result)
	require.NoError(t, err)
	assert.Contains(t, string(data), "  ")
	assert.True(t, json.Valid(data))
}

func TestJSONReporter_GenerateReport_Compact(t *testing.T) {
	r := NewJSONReporter(t.TempDir(), false)
	result := makeTestResult()

	data, err := r.GenerateReport(result)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
	assert.NotContains(t, string(data), "\n  ")
}

func TestJSONReporter_GenerateMasterSummary(t *testing.T) {
	r := NewJSONReporter(t.TempDir(), true)
	results := makeTestResults()

	data, err := r.GenerateMasterSummary(results)
	require.NoError(t, err)

	var summary jsonMasterSummary
	err = json.Unmarshal(data, &summary)
	require.NoError(t, err)
	assert.Equal(t, 2, summary.TotalChallenges)
	assert.Equal(t, 1, summary.Passed)
	assert.Equal(t, 1, summary.Failed)
	assert.Len(t, summary.Results, 2)
}

func TestJSONReporter_WriteReport(t *testing.T) {
	r := NewJSONReporter(t.TempDir(), false)
	result := makeTestResult()

	var buf bytes.Buffer
	err := r.WriteReport(&buf, result)
	require.NoError(t, err)
	assert.True(t, json.Valid(buf.Bytes()))
}

func TestJSONReporter_GenerateMasterSummary_Empty(
	t *testing.T,
) {
	r := NewJSONReporter(t.TempDir(), true)
	var results []*challenge.Result

	data, err := r.GenerateMasterSummary(results)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
}

func TestJSONReporter_GenerateMasterSummary_Compact(t *testing.T) {
	r := NewJSONReporter(t.TempDir(), false)
	results := makeTestResults()

	data, err := r.GenerateMasterSummary(results)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
	// Compact format should not have newlines with indentation
	assert.NotContains(t, string(data), "\n  ")
}
