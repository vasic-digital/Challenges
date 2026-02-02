package report

import (
	"encoding/json"
	"io"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// JSONReporter generates JSON reports from challenge results.
type JSONReporter struct {
	outputDir string
	pretty    bool
}

// NewJSONReporter creates a new JSON reporter. When pretty is
// true, output is indented for readability.
func NewJSONReporter(
	outputDir string,
	pretty bool,
) *JSONReporter {
	return &JSONReporter{
		outputDir: outputDir,
		pretty:    pretty,
	}
}

// GenerateReport creates a JSON report for a single challenge
// result.
func (r *JSONReporter) GenerateReport(
	result *challenge.Result,
) ([]byte, error) {
	if r.pretty {
		return json.MarshalIndent(result, "", "  ")
	}
	return json.Marshal(result)
}

// jsonMasterSummary is the JSON structure for a master summary.
type jsonMasterSummary struct {
	GeneratedAt     time.Time          `json:"generated_at"`
	TotalChallenges int                `json:"total_challenges"`
	Passed          int                `json:"passed"`
	Failed          int                `json:"failed"`
	TotalDuration   time.Duration      `json:"total_duration"`
	Results         []*challenge.Result `json:"results"`
}

// GenerateMasterSummary creates a JSON summary of all challenge
// results.
func (r *JSONReporter) GenerateMasterSummary(
	results []*challenge.Result,
) ([]byte, error) {
	summary := jsonMasterSummary{
		GeneratedAt:     time.Now(),
		TotalChallenges: len(results),
		Results:         results,
	}

	for _, res := range results {
		if res.Status == challenge.StatusPassed {
			summary.Passed++
		} else {
			summary.Failed++
		}
		summary.TotalDuration += res.Duration
	}

	if r.pretty {
		return json.MarshalIndent(summary, "", "  ")
	}
	return json.Marshal(summary)
}

// WriteReport writes a JSON report to the specified writer.
func (r *JSONReporter) WriteReport(
	w io.Writer,
	result *challenge.Result,
) error {
	data, err := r.GenerateReport(result)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
