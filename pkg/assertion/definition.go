// Package assertion provides an extensible assertion evaluation
// engine for the Challenges module. It ships with 16 built-in
// evaluator types and supports custom evaluator registration.
package assertion

// Definition describes a single assertion to evaluate against
// a challenge output or metric value.
type Definition struct {
	// Type is the evaluator type (e.g., "contains",
	// "not_empty", "min_length").
	Type string `json:"type"`

	// Target is the name of the output or metric to check.
	Target string `json:"target"`

	// Value is the expected value for single-value assertions.
	Value any `json:"value,omitempty"`

	// Values holds expected values for multi-value assertions
	// (e.g., "contains_any").
	Values []any `json:"values,omitempty"`

	// Message is a human-readable description shown on
	// failure.
	Message string `json:"message"`
}

// Result captures the outcome of evaluating a single assertion.
type Result struct {
	// Type is the assertion type that was evaluated.
	Type string `json:"type"`

	// Target is the name of the output or metric checked.
	Target string `json:"target"`

	// Expected is the value the assertion expected.
	Expected any `json:"expected"`

	// Actual is the value that was observed.
	Actual any `json:"actual"`

	// Passed indicates whether the assertion succeeded.
	Passed bool `json:"passed"`

	// Message is a human-readable description of the outcome.
	Message string `json:"message"`
}
