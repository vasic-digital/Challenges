package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

func TestHTMLReporter_GenerateReport_Content(t *testing.T) {
	r := NewHTMLReporter(t.TempDir())
	result := makeTestResult()

	data, err := r.GenerateReport(result)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "<!DOCTYPE html>")
	assert.Contains(t, content, "<title>")
	assert.Contains(t, content, "Test Challenge")
	assert.Contains(t, content, "PASSED")
	assert.Contains(t, content, "status-passed")
	assert.Contains(t, content, "latency")
	assert.Contains(t, content, "120.50")
	assert.Contains(t, content, "</html>")
	assert.Contains(t, content, "Challenges Framework")
}

func TestHTMLReporter_GenerateReport_FailedStatus(
	t *testing.T,
) {
	r := NewHTMLReporter(t.TempDir())
	result := makeTestResult()
	result.Status = challenge.StatusFailed
	result.Error = "timeout exceeded"

	data, err := r.GenerateReport(result)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "status-failed")
	assert.Contains(t, content, "timeout exceeded")
}

func TestHTMLReporter_WriteReport(t *testing.T) {
	r := NewHTMLReporter(t.TempDir())
	result := makeTestResult()

	var buf bytes.Buffer
	err := r.WriteReport(&buf, result)
	require.NoError(t, err)
	assert.True(
		t, strings.HasPrefix(buf.String(), "<!DOCTYPE"),
	)
}

func TestHTMLReporter_GenerateMasterSummary(t *testing.T) {
	r := NewHTMLReporter(t.TempDir())
	results := makeTestResults()

	data, err := r.GenerateMasterSummary(results)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Master Summary")
	assert.Contains(t, content, "Test Challenge")
	assert.Contains(t, content, "Another Challenge")
	assert.Contains(t, content, "Statistics")
	assert.Contains(t, content, "50%")
}

func TestHTMLReporter_EscapesHTML(t *testing.T) {
	r := NewHTMLReporter(t.TempDir())
	result := makeTestResult()
	result.ChallengeName = "<script>alert('xss')</script>"

	data, err := r.GenerateReport(result)
	require.NoError(t, err)

	content := string(data)
	assert.NotContains(t, content, "<script>")
	assert.Contains(t, content, "&lt;script&gt;")
}

func TestHTMLReporter_NoMetrics(t *testing.T) {
	r := NewHTMLReporter(t.TempDir())
	result := makeTestResult()
	result.Metrics = nil

	data, err := r.GenerateReport(result)
	require.NoError(t, err)

	content := string(data)
	assert.NotContains(t, content, "<h2>Metrics</h2>")
}

func TestHTMLReporter_NoAssertions(t *testing.T) {
	r := NewHTMLReporter(t.TempDir())
	result := makeTestResult()
	result.Assertions = nil

	data, err := r.GenerateReport(result)
	require.NoError(t, err)

	content := string(data)
	assert.NotContains(t, content, "<h2>Assertions</h2>")
}

func TestHTMLReporter_NoOutputs(t *testing.T) {
	r := NewHTMLReporter(t.TempDir())
	result := makeTestResult()
	result.Outputs = nil

	data, err := r.GenerateReport(result)
	require.NoError(t, err)

	content := string(data)
	assert.NotContains(t, content, "<h2>Output Files</h2>")
}

func TestHTMLReporter_MetricsWithEmptyUnit(t *testing.T) {
	r := NewHTMLReporter(t.TempDir())
	result := makeTestResult()
	result.Metrics["nounit"] = challenge.MetricValue{
		Name:  "nounit",
		Value: 42.0,
		Unit:  "", // Empty unit should become "-"
	}

	data, err := r.GenerateReport(result)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "nounit")
}
