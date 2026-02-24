package userflow

import (
	"fmt"
	"strings"

	"digital.vasic.challenges/pkg/assertion"
)

// RegisterEvaluators registers all 12 userflow assertion
// evaluators with the given engine.
func RegisterEvaluators(
	engine *assertion.DefaultEngine,
) error {
	evaluators := map[string]assertion.Evaluator{
		"build_succeeds":    evaluateBuildSucceeds,
		"all_tests_pass":    evaluateAllTestsPass,
		"lint_passes":       evaluateLintPasses,
		"app_launches":      evaluateAppLaunches,
		"app_stable":        evaluateAppStable,
		"status_code":       evaluateStatusCode,
		"response_contains": evaluateResponseContains,
		"response_not_empty": evaluateResponseNotEmpty,
		"json_field_equals": evaluateJSONFieldEquals,
		"screenshot_exists": evaluateScreenshotExists,
		"flow_completes":    evaluateFlowCompletes,
		"within_duration":   evaluateWithinDuration,
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

// toIntVal converts a value to int. Supports int, int64,
// float64, and float32.
func toIntVal(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	default:
		return 0, false
	}
}

// evaluateBuildSucceeds checks that the value is a bool true.
func evaluateBuildSucceeds(
	def assertion.Definition, value any,
) (bool, string) {
	b, ok := value.(bool)
	if !ok {
		return false, fmt.Sprintf(
			"build_succeeds: expected bool, got %T", value,
		)
	}
	if b {
		return true, "build succeeded"
	}
	return false, "build failed"
}

// evaluateAllTestsPass checks that the failure count is 0.
func evaluateAllTestsPass(
	def assertion.Definition, value any,
) (bool, string) {
	failures, ok := toIntVal(value)
	if !ok {
		return false, fmt.Sprintf(
			"all_tests_pass: expected int, got %T", value,
		)
	}
	if failures == 0 {
		return true, "all tests passed (0 failures)"
	}
	return false, fmt.Sprintf(
		"tests failed: %d failures", failures,
	)
}

// evaluateLintPasses checks that the value is a bool true.
func evaluateLintPasses(
	def assertion.Definition, value any,
) (bool, string) {
	b, ok := value.(bool)
	if !ok {
		return false, fmt.Sprintf(
			"lint_passes: expected bool, got %T", value,
		)
	}
	if b {
		return true, "lint passed"
	}
	return false, "lint failed"
}

// evaluateAppLaunches checks that the value is a bool true.
func evaluateAppLaunches(
	def assertion.Definition, value any,
) (bool, string) {
	b, ok := value.(bool)
	if !ok {
		return false, fmt.Sprintf(
			"app_launches: expected bool, got %T", value,
		)
	}
	if b {
		return true, "app launched successfully"
	}
	return false, "app failed to launch"
}

// evaluateAppStable checks that the value is a bool true.
func evaluateAppStable(
	def assertion.Definition, value any,
) (bool, string) {
	b, ok := value.(bool)
	if !ok {
		return false, fmt.Sprintf(
			"app_stable: expected bool, got %T", value,
		)
	}
	if b {
		return true, "app is stable"
	}
	return false, "app is unstable"
}

// evaluateStatusCode checks that the int value equals
// def.Value.
func evaluateStatusCode(
	def assertion.Definition, value any,
) (bool, string) {
	actual, ok := toIntVal(value)
	if !ok {
		return false, fmt.Sprintf(
			"status_code: expected int, got %T", value,
		)
	}
	expected, ok := toIntVal(def.Value)
	if !ok {
		return false, fmt.Sprintf(
			"status_code: expected int for def.Value, got %T",
			def.Value,
		)
	}
	if actual == expected {
		return true, fmt.Sprintf(
			"status code is %d", actual,
		)
	}
	return false, fmt.Sprintf(
		"status code: expected %d, got %d", expected, actual,
	)
}

// evaluateResponseContains checks that the string value
// contains def.Value.
func evaluateResponseContains(
	def assertion.Definition, value any,
) (bool, string) {
	s, ok := value.(string)
	if !ok {
		return false, fmt.Sprintf(
			"response_contains: expected string, got %T",
			value,
		)
	}
	expected, ok := def.Value.(string)
	if !ok {
		return false, fmt.Sprintf(
			"response_contains: expected string for "+
				"def.Value, got %T", def.Value,
		)
	}
	if strings.Contains(s, expected) {
		return true, fmt.Sprintf(
			"response contains %q", expected,
		)
	}
	return false, fmt.Sprintf(
		"response does not contain %q", expected,
	)
}

// evaluateResponseNotEmpty checks that the value has
// non-zero length. Supports string and []byte.
func evaluateResponseNotEmpty(
	def assertion.Definition, value any,
) (bool, string) {
	switch v := value.(type) {
	case string:
		if len(v) > 0 {
			return true, "response is not empty"
		}
		return false, "response is empty"
	case []byte:
		if len(v) > 0 {
			return true, "response is not empty"
		}
		return false, "response is empty"
	default:
		return false, fmt.Sprintf(
			"response_not_empty: expected string or "+
				"[]byte, got %T", value,
		)
	}
}

// evaluateJSONFieldEquals checks that the value equals
// def.Value using fmt.Sprintf comparison.
func evaluateJSONFieldEquals(
	def assertion.Definition, value any,
) (bool, string) {
	actual := fmt.Sprintf("%v", value)
	expected := fmt.Sprintf("%v", def.Value)
	if actual == expected {
		return true, fmt.Sprintf(
			"field equals %q", expected,
		)
	}
	return false, fmt.Sprintf(
		"field: expected %q, got %q", expected, actual,
	)
}

// evaluateScreenshotExists checks that the []byte value
// has non-zero length.
func evaluateScreenshotExists(
	def assertion.Definition, value any,
) (bool, string) {
	b, ok := value.([]byte)
	if !ok {
		return false, fmt.Sprintf(
			"screenshot_exists: expected []byte, got %T",
			value,
		)
	}
	if len(b) > 0 {
		return true, "screenshot captured"
	}
	return false, "screenshot is empty"
}

// evaluateFlowCompletes checks that the value is a bool
// true.
func evaluateFlowCompletes(
	def assertion.Definition, value any,
) (bool, string) {
	b, ok := value.(bool)
	if !ok {
		return false, fmt.Sprintf(
			"flow_completes: expected bool, got %T", value,
		)
	}
	if b {
		return true, "flow completed successfully"
	}
	return false, "flow did not complete"
}

// evaluateWithinDuration checks that the int value (ms)
// is <= def.Value (ms).
func evaluateWithinDuration(
	def assertion.Definition, value any,
) (bool, string) {
	actual, ok := toIntVal(value)
	if !ok {
		return false, fmt.Sprintf(
			"within_duration: expected int, got %T", value,
		)
	}
	limit, ok := toIntVal(def.Value)
	if !ok {
		return false, fmt.Sprintf(
			"within_duration: expected int for def.Value, "+
				"got %T", def.Value,
		)
	}
	if actual <= limit {
		return true, fmt.Sprintf(
			"duration %dms within limit %dms",
			actual, limit,
		)
	}
	return false, fmt.Sprintf(
		"duration %dms exceeds limit %dms", actual, limit,
	)
}
