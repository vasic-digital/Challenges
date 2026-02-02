package report

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"strings"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// HTMLReporter generates HTML reports from challenge results.
type HTMLReporter struct {
	outputDir string
}

// NewHTMLReporter creates a new HTML reporter.
func NewHTMLReporter(outputDir string) *HTMLReporter {
	return &HTMLReporter{outputDir: outputDir}
}

// GenerateReport creates an HTML report for a single challenge
// result.
func (r *HTMLReporter) GenerateReport(
	result *challenge.Result,
) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.WriteReport(&buf, result); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteReport writes an HTML report to the specified writer.
func (r *HTMLReporter) WriteReport(
	w io.Writer,
	result *challenge.Result,
) error {
	r.writeHeader(w, "Challenge Report: "+result.ChallengeName)

	fmt.Fprintf(
		w,
		"<h1>Challenge Report: %s</h1>\n",
		html.EscapeString(result.ChallengeName),
	)
	fmt.Fprintf(
		w,
		"<p><strong>Challenge ID:</strong> %s</p>\n",
		html.EscapeString(string(result.ChallengeID)),
	)
	fmt.Fprintf(
		w,
		"<p><strong>Generated:</strong> %s</p>\n",
		result.EndTime.Format(time.RFC3339),
	)

	r.writeSummaryTable(w, result)
	r.writeMetricsSection(w, result)
	r.writeAssertionsSection(w, result)
	r.writeOutputsSection(w, result)
	r.writeLogsSection(w, result)

	r.writeFooter(w)
	return nil
}

func (r *HTMLReporter) writeSummaryTable(
	w io.Writer,
	result *challenge.Result,
) {
	statusClass := "status-passed"
	if result.Status != challenge.StatusPassed {
		statusClass = "status-failed"
	}

	fmt.Fprintln(w, "<h2>Summary</h2>")
	fmt.Fprintln(w, "<table>")
	fmt.Fprintln(w, "<tr><th>Metric</th><th>Value</th></tr>")
	fmt.Fprintf(
		w,
		"<tr><td>Status</td><td class=\"%s\">"+
			"<strong>%s</strong></td></tr>\n",
		statusClass, strings.ToUpper(result.Status),
	)
	fmt.Fprintf(
		w,
		"<tr><td>Start Time</td><td>%s</td></tr>\n",
		result.StartTime.Format(time.RFC3339),
	)
	fmt.Fprintf(
		w,
		"<tr><td>End Time</td><td>%s</td></tr>\n",
		result.EndTime.Format(time.RFC3339),
	)
	fmt.Fprintf(
		w,
		"<tr><td>Duration</td><td>%v</td></tr>\n",
		result.Duration,
	)

	if result.Error != "" {
		fmt.Fprintf(
			w,
			"<tr><td>Error</td>"+
				"<td class=\"status-failed\">%s</td></tr>\n",
			html.EscapeString(result.Error),
		)
	}

	fmt.Fprintln(w, "</table>")
}

func (r *HTMLReporter) writeMetricsSection(
	w io.Writer,
	result *challenge.Result,
) {
	if len(result.Metrics) == 0 {
		return
	}

	fmt.Fprintln(w, "<h2>Metrics</h2>")
	fmt.Fprintln(w, "<table>")
	fmt.Fprintln(
		w,
		"<tr><th>Metric</th><th>Value</th>"+
			"<th>Unit</th></tr>",
	)

	for _, m := range result.Metrics {
		unit := m.Unit
		if unit == "" {
			unit = "-"
		}
		fmt.Fprintf(
			w,
			"<tr><td>%s</td><td>%.2f</td>"+
				"<td>%s</td></tr>\n",
			html.EscapeString(m.Name), m.Value,
			html.EscapeString(unit),
		)
	}

	fmt.Fprintln(w, "</table>")
}

func (r *HTMLReporter) writeAssertionsSection(
	w io.Writer,
	result *challenge.Result,
) {
	if len(result.Assertions) == 0 {
		return
	}

	fmt.Fprintln(w, "<h2>Assertions</h2>")
	fmt.Fprintln(w, "<table>")
	fmt.Fprintln(
		w,
		"<tr><th>Type</th><th>Target</th>"+
			"<th>Passed</th><th>Message</th></tr>",
	)

	passedCount := 0
	for _, a := range result.Assertions {
		passedStr := "No"
		cls := "status-failed"
		if a.Passed {
			passedStr = "Yes"
			cls = "status-passed"
			passedCount++
		}
		fmt.Fprintf(
			w,
			"<tr><td>%s</td><td>%s</td>"+
				"<td class=\"%s\">%s</td>"+
				"<td>%s</td></tr>\n",
			html.EscapeString(a.Type),
			html.EscapeString(a.Target),
			cls, passedStr,
			html.EscapeString(a.Message),
		)
	}

	fmt.Fprintln(w, "</table>")

	total := len(result.Assertions)
	pct := float64(passedCount) / float64(total) * 100
	fmt.Fprintf(
		w,
		"<p><strong>Pass Rate:</strong> %d/%d (%.0f%%)</p>\n",
		passedCount, total, pct,
	)
}

func (r *HTMLReporter) writeOutputsSection(
	w io.Writer,
	result *challenge.Result,
) {
	if len(result.Outputs) == 0 {
		return
	}

	fmt.Fprintln(w, "<h2>Output Files</h2>")
	fmt.Fprintln(w, "<table>")
	fmt.Fprintln(
		w, "<tr><th>Name</th><th>Path</th></tr>",
	)

	for name, path := range result.Outputs {
		fmt.Fprintf(
			w,
			"<tr><td>%s</td>"+
				"<td><code>%s</code></td></tr>\n",
			html.EscapeString(name),
			html.EscapeString(path),
		)
	}

	fmt.Fprintln(w, "</table>")
}

func (r *HTMLReporter) writeLogsSection(
	w io.Writer,
	result *challenge.Result,
) {
	fmt.Fprintln(w, "<h2>Log Files</h2>")
	fmt.Fprintln(w, "<table>")
	fmt.Fprintln(
		w, "<tr><th>Log Type</th><th>Path</th></tr>",
	)

	fmt.Fprintf(
		w,
		"<tr><td>Challenge Log</td>"+
			"<td><code>%s</code></td></tr>\n",
		html.EscapeString(result.Logs.ChallengeLog),
	)
	fmt.Fprintf(
		w,
		"<tr><td>Output Log</td>"+
			"<td><code>%s</code></td></tr>\n",
		html.EscapeString(result.Logs.OutputLog),
	)
	if result.Logs.APIRequests != "" {
		fmt.Fprintf(
			w,
			"<tr><td>API Requests</td>"+
				"<td><code>%s</code></td></tr>\n",
			html.EscapeString(result.Logs.APIRequests),
		)
	}
	if result.Logs.APIResponses != "" {
		fmt.Fprintf(
			w,
			"<tr><td>API Responses</td>"+
				"<td><code>%s</code></td></tr>\n",
			html.EscapeString(result.Logs.APIResponses),
		)
	}

	fmt.Fprintln(w, "</table>")
}

// GenerateMasterSummary creates an HTML summary of all
// challenge results.
func (r *HTMLReporter) GenerateMasterSummary(
	results []*challenge.Result,
) ([]byte, error) {
	var buf bytes.Buffer

	r.writeHeader(
		&buf, "Challenges Framework - Master Summary",
	)

	fmt.Fprintln(
		&buf,
		"<h1>Challenges Framework - Master Summary</h1>",
	)
	fmt.Fprintf(
		&buf,
		"<p><strong>Generated:</strong> %s</p>\n",
		time.Now().Format(time.RFC3339),
	)

	r.writeMasterOverview(&buf, results)
	r.writeMasterStats(&buf, results)
	r.writeMasterDetails(&buf, results)
	r.writeFooter(&buf)

	return buf.Bytes(), nil
}

func (r *HTMLReporter) writeMasterOverview(
	w io.Writer,
	results []*challenge.Result,
) {
	fmt.Fprintln(w, "<h2>Overview</h2>")
	fmt.Fprintln(w, "<table>")
	fmt.Fprintln(
		w,
		"<tr><th>Challenge</th><th>Status</th>"+
			"<th>Duration</th><th>Last Run</th></tr>",
	)

	for _, result := range results {
		cls := "status-passed"
		if result.Status != challenge.StatusPassed {
			cls = "status-failed"
		}
		fmt.Fprintf(
			w,
			"<tr><td>%s</td>"+
				"<td class=\"%s\">%s</td>"+
				"<td>%v</td><td>%s</td></tr>\n",
			html.EscapeString(result.ChallengeName),
			cls, strings.ToUpper(result.Status),
			result.Duration,
			result.EndTime.Format("2006-01-02 15:04:05"),
		)
	}

	fmt.Fprintln(w, "</table>")
}

func (r *HTMLReporter) writeMasterStats(
	w io.Writer,
	results []*challenge.Result,
) {
	passedCount := 0
	totalDuration := time.Duration(0)
	for _, res := range results {
		if res.Status == challenge.StatusPassed {
			passedCount++
		}
		totalDuration += res.Duration
	}

	fmt.Fprintln(w, "<h2>Statistics</h2>")
	fmt.Fprintln(w, "<table>")
	fmt.Fprintln(w, "<tr><th>Metric</th><th>Value</th></tr>")
	fmt.Fprintf(
		w,
		"<tr><td>Total Challenges</td>"+
			"<td>%d</td></tr>\n",
		len(results),
	)
	fmt.Fprintf(
		w,
		"<tr><td>Passed</td><td>%d</td></tr>\n",
		passedCount,
	)
	fmt.Fprintf(
		w,
		"<tr><td>Failed</td><td>%d</td></tr>\n",
		len(results)-passedCount,
	)

	if len(results) > 0 {
		pct := float64(passedCount) /
			float64(len(results)) * 100
		fmt.Fprintf(
			w,
			"<tr><td>Pass Rate</td>"+
				"<td>%.0f%%</td></tr>\n",
			pct,
		)
	}

	fmt.Fprintf(
		w,
		"<tr><td>Total Duration</td>"+
			"<td>%v</td></tr>\n",
		totalDuration,
	)
	fmt.Fprintln(w, "</table>")
}

func (r *HTMLReporter) writeMasterDetails(
	w io.Writer,
	results []*challenge.Result,
) {
	fmt.Fprintln(w, "<h2>Challenge Details</h2>")

	for _, result := range results {
		fmt.Fprintf(
			w,
			"<h3>%s</h3>\n",
			html.EscapeString(result.ChallengeName),
		)
		fmt.Fprintf(
			w,
			"<p><strong>Status:</strong> %s</p>\n",
			strings.ToUpper(result.Status),
		)
		fmt.Fprintf(
			w,
			"<p><strong>Duration:</strong> %v</p>\n",
			result.Duration,
		)

		if result.Error != "" {
			fmt.Fprintf(
				w,
				"<p><strong>Error:</strong> %s</p>\n",
				html.EscapeString(result.Error),
			)
		}
	}
}

func (r *HTMLReporter) writeHeader(w io.Writer, title string) {
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
body {
  font-family: -apple-system, BlinkMacSystemFont,
    "Segoe UI", Roboto, sans-serif;
  max-width: 960px;
  margin: 0 auto;
  padding: 20px;
  color: #333;
  background: #f9f9f9;
}
h1 { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
h2 { color: #2c3e50; margin-top: 30px; }
h3 { color: #34495e; }
table {
  border-collapse: collapse;
  width: 100%%;
  margin: 10px 0;
  background: #fff;
}
th, td {
  border: 1px solid #ddd;
  padding: 8px 12px;
  text-align: left;
}
th { background: #3498db; color: #fff; }
tr:nth-child(even) { background: #f2f2f2; }
.status-passed { color: #27ae60; font-weight: bold; }
.status-failed { color: #e74c3c; font-weight: bold; }
code {
  background: #ecf0f1;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 0.9em;
}
footer {
  margin-top: 40px;
  padding-top: 10px;
  border-top: 1px solid #ddd;
  color: #7f8c8d;
  font-size: 0.9em;
}
</style>
</head>
<body>
`, html.EscapeString(title))
}

func (r *HTMLReporter) writeFooter(w io.Writer) {
	fmt.Fprintln(w, "<footer>")
	fmt.Fprintln(
		w, "<p>Generated by Challenges Framework</p>",
	)
	fmt.Fprintln(w, "</footer>")
	fmt.Fprintln(w, "</body>")
	fmt.Fprintln(w, "</html>")
}
