package challenge

import "encoding/json"

// Definition describes a challenge declaratively. It captures
// all metadata needed to instantiate, configure, and evaluate
// a challenge without requiring Go code.
type Definition struct {
	ID                ID              `json:"id"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	Category          string          `json:"category"`
	Dependencies      []ID            `json:"dependencies"`
	EstimatedDuration string          `json:"estimated_duration"`
	Inputs            []Input         `json:"inputs"`
	Outputs           []Output        `json:"outputs"`
	Assertions        []AssertionDef  `json:"assertions"`
	Metrics           []string        `json:"metrics"`
	Configuration     json.RawMessage `json:"configuration,omitempty"`
}

// Input describes a named input parameter for a challenge.
type Input struct {
	// Name is the parameter name.
	Name string `json:"name"`

	// Source describes where the input comes from (e.g., "env",
	// "dependency:<id>", "config").
	Source string `json:"source"`

	// Required indicates whether the input must be present
	// for the challenge to execute.
	Required bool `json:"required"`
}

// Output describes a named output produced by a challenge.
type Output struct {
	// Name is the output identifier.
	Name string `json:"name"`

	// Type describes the output format (e.g., "string",
	// "json", "file").
	Type string `json:"type"`

	// Description explains what this output represents.
	Description string `json:"description"`
}

// AssertionDef defines a single assertion to evaluate against
// challenge outputs or metrics.
type AssertionDef struct {
	// Type is the assertion type (e.g., "equals", "contains",
	// "greater_than", "not_empty", "regex").
	Type string `json:"type"`

	// Target is the name of the output or metric to check.
	Target string `json:"target"`

	// Value is the expected value for single-value assertions.
	Value any `json:"value,omitempty"`

	// Values holds expected values for multi-value assertions
	// (e.g., "one_of").
	Values []any `json:"values,omitempty"`

	// Message is a human-readable description shown on failure.
	Message string `json:"message"`
}
