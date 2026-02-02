// Package report provides report generation for challenge results.
package report

import (
	"io"

	"digital.vasic.challenges/pkg/challenge"
)

// Reporter defines the interface for generating challenge reports.
type Reporter interface {
	// GenerateReport creates a report for a single challenge
	// result.
	GenerateReport(result *challenge.Result) ([]byte, error)

	// GenerateMasterSummary creates a summary of all challenge
	// results.
	GenerateMasterSummary(
		results []*challenge.Result,
	) ([]byte, error)

	// WriteReport writes a report to the specified writer.
	WriteReport(w io.Writer, result *challenge.Result) error
}
