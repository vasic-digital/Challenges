package report

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

func makeTestResult() *challenge.Result {
	return &challenge.Result{
		ChallengeID:   "test-001",
		ChallengeName: "Test Challenge",
		Status:        challenge.StatusPassed,
		StartTime:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:       time.Date(2026, 1, 1, 0, 0, 5, 0, time.UTC),
		Duration:      5 * time.Second,
		Assertions: []challenge.AssertionResult{
			{
				Type:    "not_empty",
				Target:  "response",
				Passed:  true,
				Message: "response is not empty",
			},
			{
				Type:    "contains",
				Target:  "body",
				Passed:  false,
				Message: "body missing keyword",
			},
		},
		Metrics: map[string]challenge.MetricValue{
			"latency": {
				Name: "latency", Value: 120.5, Unit: "ms",
			},
		},
		Outputs: map[string]string{
			"result": "/tmp/result.json",
		},
		Logs: challenge.LogPaths{
			ChallengeLog: "/tmp/challenge.log",
			OutputLog:    "/tmp/output.log",
			APIRequests:  "/tmp/api_req.log",
			APIResponses: "/tmp/api_resp.log",
		},
	}
}

func makeTestResults() []*challenge.Result {
	return []*challenge.Result{
		makeTestResult(),
		{
			ChallengeID:   "test-002",
			ChallengeName: "Another Challenge",
			Status:        challenge.StatusFailed,
			StartTime:     time.Date(2026, 1, 1, 0, 0, 6, 0, time.UTC),
			EndTime:       time.Date(2026, 1, 1, 0, 0, 8, 0, time.UTC),
			Duration:      2 * time.Second,
			Error:         "connection refused",
			Logs: challenge.LogPaths{
				ChallengeLog: "/tmp/ch2.log",
				OutputLog:    "/tmp/out2.log",
			},
		},
	}
}

func TestReporter_MarkdownImplementsInterface(t *testing.T) {
	var _ Reporter = &MarkdownReporter{}
}

func TestReporter_JSONImplementsInterface(t *testing.T) {
	var _ Reporter = &JSONReporter{}
}

func TestReporter_HTMLImplementsInterface(t *testing.T) {
	var _ Reporter = &HTMLReporter{}
}

func TestReporter_AllReporters_GenerateReport(t *testing.T) {
	result := makeTestResult()

	reporters := map[string]Reporter{
		"markdown": NewMarkdownReporter(t.TempDir()),
		"json":     NewJSONReporter(t.TempDir(), true),
		"html":     NewHTMLReporter(t.TempDir()),
	}

	for name, rpt := range reporters {
		t.Run(name, func(t *testing.T) {
			data, err := rpt.GenerateReport(result)
			require.NoError(t, err)
			assert.NotEmpty(t, data)
		})
	}
}

func TestReporter_AllReporters_WriteReport(t *testing.T) {
	result := makeTestResult()

	reporters := map[string]Reporter{
		"markdown": NewMarkdownReporter(t.TempDir()),
		"json":     NewJSONReporter(t.TempDir(), true),
		"html":     NewHTMLReporter(t.TempDir()),
	}

	for name, rpt := range reporters {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			err := rpt.WriteReport(&buf, result)
			require.NoError(t, err)
			assert.NotEmpty(t, buf.String())
		})
	}
}

func TestReporter_AllReporters_GenerateMasterSummary(
	t *testing.T,
) {
	results := makeTestResults()

	reporters := map[string]Reporter{
		"markdown": NewMarkdownReporter(t.TempDir()),
		"json":     NewJSONReporter(t.TempDir(), true),
		"html":     NewHTMLReporter(t.TempDir()),
	}

	for name, rpt := range reporters {
		t.Run(name, func(t *testing.T) {
			data, err := rpt.GenerateMasterSummary(results)
			require.NoError(t, err)
			assert.NotEmpty(t, data)
		})
	}
}
