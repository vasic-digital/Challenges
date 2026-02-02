package assertion

import (
	"fmt"
	"regexp"
	"strings"
)

// evaluateNotEmpty checks that a value is non-nil and non-empty.
func evaluateNotEmpty(
	_ Definition,
	value any,
) (bool, string) {
	if value == nil {
		return false, "value is nil"
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return false, "string is empty"
		}
	case []any:
		if len(v) == 0 {
			return false, "array is empty"
		}
	case map[string]any:
		if len(v) == 0 {
			return false, "map is empty"
		}
	}

	return true, "value is not empty"
}

// evaluateNotMock checks that a string value does not contain
// common mock/placeholder patterns.
func evaluateNotMock(
	_ Definition,
	value any,
) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return true, "value is not a string"
	}

	mockPatterns := []string{
		"lorem ipsum",
		"placeholder",
		"mock response",
		"TODO",
		"not implemented",
		"[MOCK]",
		"test response",
		"dummy",
		"sample output",
	}

	lower := strings.ToLower(str)
	for _, pattern := range mockPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return false, fmt.Sprintf(
				"response appears to be mocked (contains '%s')",
				pattern,
			)
		}
	}

	return true, "response is not mocked"
}

// evaluateContains checks that a string value contains the
// expected substring (case-insensitive).
func evaluateContains(
	assertion Definition,
	value any,
) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "value is not a string"
	}

	expected, ok := assertion.Value.(string)
	if !ok {
		return false, "expected value is not a string"
	}

	if strings.Contains(
		strings.ToLower(str),
		strings.ToLower(expected),
	) {
		return true, fmt.Sprintf("contains '%s'", expected)
	}

	return false, fmt.Sprintf(
		"does not contain '%s'", expected,
	)
}

// evaluateContainsAny checks that a string value contains at
// least one of the expected substrings.
func evaluateContainsAny(
	assertion Definition,
	value any,
) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "value is not a string"
	}

	lower := strings.ToLower(str)

	var values []string
	switch v := assertion.Value.(type) {
	case string:
		values = strings.Split(v, ",")
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				values = append(values, s)
			}
		}
	case []string:
		values = v
	default:
		if assertion.Values != nil {
			for _, item := range assertion.Values {
				if s, ok := item.(string); ok {
					values = append(values, s)
				}
			}
		}
	}

	for _, expected := range values {
		trimmed := strings.TrimSpace(expected)
		if strings.Contains(
			lower, strings.ToLower(trimmed),
		) {
			return true, fmt.Sprintf(
				"contains '%s'", expected,
			)
		}
	}

	return false, fmt.Sprintf(
		"does not contain any of: %v", values,
	)
}

// evaluateMinLength checks that a string value meets a minimum
// character length.
func evaluateMinLength(
	assertion Definition,
	value any,
) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "value is not a string"
	}

	minLength, ok := toInt(assertion.Value)
	if !ok {
		return false, "expected value is not a number"
	}

	actual := len(str)
	if actual >= minLength {
		return true, fmt.Sprintf(
			"length %d >= %d", actual, minLength,
		)
	}

	return false, fmt.Sprintf(
		"length %d < %d", actual, minLength,
	)
}

// evaluateQualityScore checks that a numeric value meets a
// minimum quality score threshold.
func evaluateQualityScore(
	assertion Definition,
	value any,
) (bool, string) {
	score, ok := toFloat64(value)
	if !ok {
		return false, "value is not a number"
	}

	minScore, ok := toFloat64(assertion.Value)
	if !ok {
		return false, "expected value is not a number"
	}

	if score >= minScore {
		return true, fmt.Sprintf(
			"quality score %.2f >= %.2f", score, minScore,
		)
	}

	return false, fmt.Sprintf(
		"quality score %.2f < %.2f", score, minScore,
	)
}

// evaluateReasoningPresent checks that a string value contains
// reasoning indicator words.
func evaluateReasoningPresent(
	_ Definition,
	value any,
) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "value is not a string"
	}

	indicators := []string{
		"because", "therefore", "since", "thus",
		"step", "first", "then", "next",
		"reason", "explanation", "conclude",
		"let me", "let's",
	}

	lower := strings.ToLower(str)
	for _, indicator := range indicators {
		if strings.Contains(lower, indicator) {
			return true, fmt.Sprintf(
				"reasoning present (found '%s')", indicator,
			)
		}
	}

	return false, "no reasoning indicators found"
}

// evaluateCodeValid checks that a string value contains
// recognizable code patterns or code block markers.
func evaluateCodeValid(
	_ Definition,
	value any,
) (bool, string) {
	str, ok := value.(string)
	if !ok {
		return false, "value is not a string"
	}

	hasCodeBlock := strings.Contains(str, "```") ||
		strings.Contains(str, "    ")

	codePatterns := []string{
		`func\s+\w+`,
		`def\s+\w+`,
		`class\s+\w+`,
		`function\s+\w+`,
		`=>\s*\{`,
		`public\s+\w+`,
		`import\s+`,
		`return\s+`,
	}

	for _, pattern := range codePatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(str) {
			return true, "valid code detected"
		}
	}

	if hasCodeBlock {
		return true, "code block present"
	}

	return false, "no valid code detected"
}

// evaluateMinCount checks that a countable value (int, float64,
// slice, or map) meets a minimum count.
func evaluateMinCount(
	assertion Definition,
	value any,
) (bool, string) {
	count, ok := toCount(value)
	if !ok {
		return false, "value is not countable"
	}

	minCount, ok := toInt(assertion.Value)
	if !ok {
		return false, "expected value is not a number"
	}

	if count >= minCount {
		return true, fmt.Sprintf(
			"count %d >= %d", count, minCount,
		)
	}

	return false, fmt.Sprintf(
		"count %d < %d", count, minCount,
	)
}

// evaluateExactCount checks that a countable value exactly
// matches the expected count.
func evaluateExactCount(
	assertion Definition,
	value any,
) (bool, string) {
	count, ok := toCount(value)
	if !ok {
		return false, "value is not countable"
	}

	expected, ok := toInt(assertion.Value)
	if !ok {
		return false, "expected value is not a number"
	}

	if count == expected {
		return true, fmt.Sprintf(
			"count %d == %d", count, expected,
		)
	}

	return false, fmt.Sprintf(
		"count %d != %d", count, expected,
	)
}

// evaluateMaxLatency checks that a numeric latency value does
// not exceed the specified maximum (in milliseconds).
func evaluateMaxLatency(
	assertion Definition,
	value any,
) (bool, string) {
	latency, ok := toInt64(value)
	if !ok {
		return false, "value is not a number"
	}

	maxLatency, ok := toInt64(assertion.Value)
	if !ok {
		return false, "expected value is not a number"
	}

	if latency <= maxLatency {
		return true, fmt.Sprintf(
			"latency %dms <= %dms", latency, maxLatency,
		)
	}

	return false, fmt.Sprintf(
		"latency %dms > %dms", latency, maxLatency,
	)
}

// evaluateAllValid checks that every item in a slice is
// non-nil and non-empty.
func evaluateAllValid(
	_ Definition,
	value any,
) (bool, string) {
	items, ok := value.([]any)
	if !ok {
		return false, "value is not an array"
	}

	for i, item := range items {
		if item == nil {
			return false, fmt.Sprintf(
				"item %d is nil", i,
			)
		}
		if str, ok := item.(string); ok && str == "" {
			return false, fmt.Sprintf(
				"item %d is empty", i,
			)
		}
	}

	return true, "all items are valid"
}

// evaluateNoDuplicates checks that a slice contains no
// duplicate values (compared via fmt.Sprintf("%v")).
func evaluateNoDuplicates(
	_ Definition,
	value any,
) (bool, string) {
	items, ok := value.([]any)
	if !ok {
		return false, "value is not an array"
	}

	seen := make(map[string]bool, len(items))
	for _, item := range items {
		key := fmt.Sprintf("%v", item)
		if seen[key] {
			return false, fmt.Sprintf(
				"duplicate found: %s", key,
			)
		}
		seen[key] = true
	}

	return true, "no duplicates found"
}

// evaluateAllPass checks that all items in a slice of results
// have passed. Accepts []Result or []any with map entries
// containing a "passed" key.
func evaluateAllPass(
	_ Definition,
	value any,
) (bool, string) {
	results, ok := value.([]Result)
	if !ok {
		items, ok := value.([]any)
		if !ok {
			return false, "value is not an array of results"
		}
		for i, item := range items {
			if m, ok := item.(map[string]any); ok {
				if passed, exists := m["passed"]; exists {
					if p, ok := passed.(bool); ok && !p {
						return false, fmt.Sprintf(
							"item %d failed", i,
						)
					}
				}
			}
		}
		return true, "all items passed"
	}

	for _, result := range results {
		if !result.Passed {
			return false, fmt.Sprintf(
				"assertion '%s' failed: %s",
				result.Type, result.Message,
			)
		}
	}

	return true, "all assertions passed"
}

// evaluateNoMockResponses checks that none of the items in a
// slice (or a single value) contain mock patterns.
func evaluateNoMockResponses(
	assertion Definition,
	value any,
) (bool, string) {
	responses, ok := value.([]any)
	if !ok {
		return evaluateNotMock(assertion, value)
	}

	for i, resp := range responses {
		if passed, msg := evaluateNotMock(assertion, resp); !passed {
			return false, fmt.Sprintf(
				"response %d: %s", i, msg,
			)
		}
	}

	return true, "no mock responses detected"
}

// evaluateMinScore is an alias for evaluateQualityScore.
func evaluateMinScore(
	assertion Definition,
	value any,
) (bool, string) {
	return evaluateQualityScore(assertion, value)
}

// --- helpers ---

// toInt converts an any value to int.
func toInt(v any) (int, bool) {
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

// toInt64 converts an any value to int64.
func toInt64(v any) (int64, bool) {
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

// toFloat64 converts an any value to float64.
func toFloat64(v any) (float64, bool) {
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

// toCount extracts an integer count from a value. It handles
// int, float64, []any, and map[string]any.
func toCount(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case float64:
		return int(val), true
	case int64:
		return int(val), true
	case []any:
		return len(val), true
	case map[string]any:
		return len(val), true
	}
	return 0, false
}
