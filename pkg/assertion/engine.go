package assertion

import (
	"fmt"
	"sync"
)

// Engine defines the interface for assertion evaluation engines.
type Engine interface {
	// Evaluate checks a single assertion against the given
	// value.
	Evaluate(assertion Definition, value any) Result

	// EvaluateAll checks multiple assertions against a map of
	// named values. Each assertion's Target field is used as
	// the key into the values map.
	EvaluateAll(
		assertions []Definition,
		values map[string]any,
	) []Result

	// Register adds a custom evaluator for the given assertion
	// type. Returns an error if the type is already registered.
	Register(assertionType string, evaluator Evaluator) error
}

// DefaultEngine is the standard Engine implementation. It is
// safe for concurrent use.
type DefaultEngine struct {
	mu         sync.RWMutex
	evaluators map[string]Evaluator
}

// NewEngine creates a DefaultEngine with all 16 built-in
// evaluators pre-registered.
func NewEngine() *DefaultEngine {
	e := &DefaultEngine{
		evaluators: make(map[string]Evaluator),
	}
	e.registerDefaults()
	return e
}

// registerDefaults registers all 16 built-in evaluators.
func (e *DefaultEngine) registerDefaults() {
	e.evaluators["not_empty"] = evaluateNotEmpty
	e.evaluators["not_mock"] = evaluateNotMock
	e.evaluators["contains"] = evaluateContains
	e.evaluators["contains_any"] = evaluateContainsAny
	e.evaluators["min_length"] = evaluateMinLength
	e.evaluators["quality_score"] = evaluateQualityScore
	e.evaluators["reasoning_present"] = evaluateReasoningPresent
	e.evaluators["code_valid"] = evaluateCodeValid
	e.evaluators["min_count"] = evaluateMinCount
	e.evaluators["exact_count"] = evaluateExactCount
	e.evaluators["max_latency"] = evaluateMaxLatency
	e.evaluators["all_valid"] = evaluateAllValid
	e.evaluators["no_duplicates"] = evaluateNoDuplicates
	e.evaluators["all_pass"] = evaluateAllPass
	e.evaluators["no_mock_responses"] = evaluateNoMockResponses
	e.evaluators["min_score"] = evaluateMinScore
}

// Register adds a custom evaluator for the given assertion type.
// Returns an error if the type is already registered.
func (e *DefaultEngine) Register(
	assertionType string,
	evaluator Evaluator,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.evaluators[assertionType]; exists {
		return fmt.Errorf(
			"assertion type already registered: %s",
			assertionType,
		)
	}

	e.evaluators[assertionType] = evaluator
	return nil
}

// Evaluate runs a single assertion against the provided value.
func (e *DefaultEngine) Evaluate(
	assertion Definition,
	value any,
) Result {
	e.mu.RLock()
	evaluator, exists := e.evaluators[assertion.Type]
	e.mu.RUnlock()

	if !exists {
		return Result{
			Type:   assertion.Type,
			Target: assertion.Target,
			Passed: false,
			Message: fmt.Sprintf(
				"unknown assertion type: %s",
				assertion.Type,
			),
		}
	}

	passed, message := evaluator(assertion, value)

	return Result{
		Type:     assertion.Type,
		Target:   assertion.Target,
		Expected: assertion.Value,
		Actual:   value,
		Passed:   passed,
		Message:  message,
	}
}

// EvaluateAll runs multiple assertions against a map of named
// values. Each assertion's Target field is used as the key into
// the values map. If a target is missing, the assertion fails.
func (e *DefaultEngine) EvaluateAll(
	assertions []Definition,
	values map[string]any,
) []Result {
	results := make([]Result, 0, len(assertions))

	for _, a := range assertions {
		value, exists := values[a.Target]
		if !exists {
			results = append(results, Result{
				Type:   a.Type,
				Target: a.Target,
				Passed: false,
				Message: fmt.Sprintf(
					"target not found: %s", a.Target,
				),
			})
			continue
		}

		results = append(results, e.Evaluate(a, value))
	}

	return results
}

// HasEvaluator returns true if the given assertion type has a
// registered evaluator.
func (e *DefaultEngine) HasEvaluator(
	assertionType string,
) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.evaluators[assertionType]
	return exists
}
