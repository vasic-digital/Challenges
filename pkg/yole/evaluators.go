package yole

import (
	"fmt"

	"digital.vasic.challenges/pkg/assertion"
)

// RegisterEvaluators registers all Yole-specific assertion
// evaluators with the given assertion engine.
func RegisterEvaluators(
	engine *assertion.DefaultEngine,
) error {
	evaluators := map[string]assertion.Evaluator{
		"build_succeeds":      evaluateBuildSucceeds,
		"all_tests_pass":      evaluateAllTestsPass,
		"lint_passes":         evaluateLintPasses,
		"app_launches":        evaluateAppLaunches,
		"app_stable":          evaluateAppStable,
		"format_renders":      evaluateFormatRenders,
		"test_count_above":    evaluateTestCountAbove,
		"no_test_failures":    evaluateNoTestFailures,
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

func evaluateBuildSucceeds(
	def assertion.Definition, value any,
) (bool, string) {
	success, ok := value.(bool)
	if !ok {
		return false, fmt.Sprintf(
			"expected bool, got %T", value,
		)
	}
	if success {
		return true, "build succeeded"
	}
	return false, "build failed"
}

func evaluateAllTestsPass(
	def assertion.Definition, value any,
) (bool, string) {
	failures := toIntVal(value)
	if failures == 0 {
		return true, "all tests passed"
	}
	return false, fmt.Sprintf(
		"%d test failures", failures,
	)
}

func evaluateLintPasses(
	def assertion.Definition, value any,
) (bool, string) {
	success, ok := value.(bool)
	if !ok {
		return false, fmt.Sprintf(
			"expected bool, got %T", value,
		)
	}
	if success {
		return true, "lint passed"
	}
	return false, "lint failed"
}

func evaluateAppLaunches(
	def assertion.Definition, value any,
) (bool, string) {
	running, ok := value.(bool)
	if !ok {
		return false, fmt.Sprintf(
			"expected bool, got %T", value,
		)
	}
	if running {
		return true, "app launched successfully"
	}
	return false, "app failed to launch"
}

func evaluateAppStable(
	def assertion.Definition, value any,
) (bool, string) {
	running, ok := value.(bool)
	if !ok {
		return false, fmt.Sprintf(
			"expected bool, got %T", value,
		)
	}
	if running {
		return true, "app is stable (still running)"
	}
	return false, "app crashed after launch"
}

func evaluateFormatRenders(
	def assertion.Definition, value any,
) (bool, string) {
	length := toIntVal(value)
	if length > 0 {
		return true, fmt.Sprintf(
			"format rendered %d chars", length,
		)
	}
	return false, "format rendered empty content"
}

func evaluateTestCountAbove(
	def assertion.Definition, value any,
) (bool, string) {
	count := toIntVal(value)
	minCount := 0
	if def.Value != nil {
		minCount = toIntVal(def.Value)
	}
	if count >= minCount {
		return true, fmt.Sprintf(
			"%d tests (>= %d)", count, minCount,
		)
	}
	return false, fmt.Sprintf(
		"%d tests (< %d)", count, minCount,
	)
}

func evaluateNoTestFailures(
	def assertion.Definition, value any,
) (bool, string) {
	failures := toIntVal(value)
	if failures == 0 {
		return true, "no test failures"
	}
	return false, fmt.Sprintf(
		"%d test failures", failures,
	)
}

func toIntVal(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}
