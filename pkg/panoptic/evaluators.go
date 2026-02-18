package panoptic

import (
	"fmt"
	"os"
	"strings"

	"digital.vasic.challenges/pkg/assertion"
)

// RegisterEvaluators registers all 8 Panoptic-specific assertion
// evaluators with the given engine.
func RegisterEvaluators(engine *assertion.DefaultEngine) error {
	evaluators := map[string]assertion.Evaluator{
		"screenshot_exists":  evaluateScreenshotExists,
		"video_exists":       evaluateVideoExists,
		"no_ui_errors":       evaluateNoUIErrors,
		"ai_confidence_above": evaluateAIConfidenceAbove,
		"all_apps_passed":    evaluateAllAppsPassed,
		"max_duration":       evaluateMaxDuration,
		"report_exists":      evaluateReportExists,
		"app_count":          evaluateAppCount,
	}

	for name, eval := range evaluators {
		if err := engine.Register(name, eval); err != nil {
			return fmt.Errorf(
				"register evaluator %s: %w", name, err,
			)
		}
	}
	return nil
}

// evaluateScreenshotExists checks that at least N screenshots
// were captured. Target key: "screenshots" ([]any of strings)
// or "total_screenshots" (int). Value: minimum count.
func evaluateScreenshotExists(
	def assertion.Definition,
	value any,
) (bool, string) {
	minCount := 1
	if n, ok := toIntVal(def.Value); ok {
		minCount = n
	}

	count := countItems(value)
	if count >= minCount {
		return true, fmt.Sprintf(
			"%d screenshots captured (>= %d)",
			count, minCount,
		)
	}
	return false, fmt.Sprintf(
		"%d screenshots captured (< %d required)",
		count, minCount,
	)
}

// evaluateVideoExists checks that at least N videos were
// recorded. Target key: "videos" ([]any) or "total_videos"
// (int). Value: minimum count.
func evaluateVideoExists(
	def assertion.Definition,
	value any,
) (bool, string) {
	minCount := 1
	if n, ok := toIntVal(def.Value); ok {
		minCount = n
	}

	count := countItems(value)
	if count >= minCount {
		return true, fmt.Sprintf(
			"%d videos recorded (>= %d)",
			count, minCount,
		)
	}
	return false, fmt.Sprintf(
		"%d videos recorded (< %d required)",
		count, minCount,
	)
}

// evaluateNoUIErrors checks that the AI error detection report
// found no errors. Target key: "ai_error_report" (string path).
// The evaluator passes if the file is empty, missing, or
// contains no error indicators.
func evaluateNoUIErrors(
	_ assertion.Definition,
	value any,
) (bool, string) {
	path, ok := value.(string)
	if !ok || path == "" {
		return true, "no AI error report generated (assumed clean)"
	}

	if !fileExists(path) {
		return true, "AI error report file not found (assumed clean)"
	}

	// Read and check for error indicators.
	data, err := readFileFunc(path)
	if err != nil {
		return false, fmt.Sprintf(
			"failed to read AI error report: %v", err,
		)
	}

	content := strings.ToLower(string(data))
	errorIndicators := []string{
		`"errors":`, `"error_count":`,
		`"critical":`, `"failures":`,
	}

	for _, indicator := range errorIndicators {
		if strings.Contains(content, indicator) {
			// Check if error count is non-zero.
			if strings.Contains(content, `"error_count": 0`) ||
				strings.Contains(content, `"error_count":0`) {
				continue
			}
			return false, fmt.Sprintf(
				"AI error report contains error indicator: %s",
				indicator,
			)
		}
	}

	return true, "no UI errors detected by AI"
}

// evaluateAIConfidenceAbove checks that the AI confidence score
// meets or exceeds a threshold. Target key: "ai_confidence"
// (float64). Value: minimum threshold.
func evaluateAIConfidenceAbove(
	def assertion.Definition,
	value any,
) (bool, string) {
	threshold := 0.75
	if f, ok := toFloatVal(def.Value); ok {
		threshold = f
	}

	confidence, ok := toFloatVal(value)
	if !ok {
		return false, "ai_confidence is not a number"
	}

	if confidence >= threshold {
		return true, fmt.Sprintf(
			"AI confidence %.2f >= %.2f",
			confidence, threshold,
		)
	}
	return false, fmt.Sprintf(
		"AI confidence %.2f < %.2f",
		confidence, threshold,
	)
}

// evaluateAllAppsPassed checks that every tested app succeeded.
// Target key: "all_apps_passed" (bool).
func evaluateAllAppsPassed(
	_ assertion.Definition,
	value any,
) (bool, string) {
	passed, ok := value.(bool)
	if !ok {
		return false, "all_apps_passed is not a boolean"
	}

	if passed {
		return true, "all apps passed"
	}
	return false, "one or more apps failed"
}

// evaluateMaxDuration checks that no single app exceeded the
// given duration limit in milliseconds. Target key:
// "max_duration_ms" (int64). Value: maximum allowed ms.
func evaluateMaxDuration(
	def assertion.Definition,
	value any,
) (bool, string) {
	maxMs, ok := toInt64Val(def.Value)
	if !ok {
		return false, "expected value is not a number"
	}

	actual, ok := toInt64Val(value)
	if !ok {
		return false, "max_duration_ms is not a number"
	}

	if actual <= maxMs {
		return true, fmt.Sprintf(
			"max duration %dms <= %dms limit",
			actual, maxMs,
		)
	}
	return false, fmt.Sprintf(
		"max duration %dms > %dms limit",
		actual, maxMs,
	)
}

// evaluateReportExists checks that HTML or JSON reports were
// generated. Target key: "report_html_exists" or
// "report_json_exists" (bool).
func evaluateReportExists(
	_ assertion.Definition,
	value any,
) (bool, string) {
	exists, ok := value.(bool)
	if !ok {
		return false, "report_exists value is not a boolean"
	}

	if exists {
		return true, "report was generated"
	}
	return false, "report was not generated"
}

// evaluateAppCount checks that the expected number of apps were
// tested. Target key: "app_count" (int). Value: expected count.
func evaluateAppCount(
	def assertion.Definition,
	value any,
) (bool, string) {
	expected, ok := toIntVal(def.Value)
	if !ok {
		return false, "expected value is not a number"
	}

	actual, ok := toIntVal(value)
	if !ok {
		return false, "app_count is not a number"
	}

	if actual == expected {
		return true, fmt.Sprintf(
			"app count %d == %d", actual, expected,
		)
	}
	return false, fmt.Sprintf(
		"app count %d != %d", actual, expected,
	)
}

// --- helpers ---

// readFileFunc is a variable for dependency injection in tests.
var readFileFunc = readFileDefault

func readFileDefault(path string) ([]byte, error) {
	return osReadFile(path)
}

// osReadFile wraps os.ReadFile for testability.
var osReadFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func toIntVal(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	}
	return 0, false
}

func toInt64Val(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	}
	return 0, false
}

func toFloatVal(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func countItems(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case int64:
		return int(val)
	case []any:
		return len(val)
	case []string:
		return len(val)
	}
	return 0
}
