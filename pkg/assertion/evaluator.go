package assertion

// Evaluator is a function that evaluates a single assertion type
// against a concrete value. It returns whether the assertion
// passed and a human-readable explanation.
type Evaluator func(assertion Definition, value any) (bool, string)
