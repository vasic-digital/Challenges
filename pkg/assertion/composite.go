package assertion

import "fmt"

// AllPassComposite creates an Evaluator that checks whether all
// results in a slice have passed. It is a wrapper around the
// all_pass built-in for programmatic use.
func AllPassComposite(
	engine Engine,
	assertions []Definition,
	values map[string]any,
) Result {
	results := engine.EvaluateAll(assertions, values)

	for _, r := range results {
		if !r.Passed {
			return Result{
				Type:    "all_pass",
				Passed:  false,
				Message: fmt.Sprintf(
					"assertion '%s' on target '%s' failed: %s",
					r.Type, r.Target, r.Message,
				),
			}
		}
	}

	return Result{
		Type:    "all_pass",
		Passed:  true,
		Message: fmt.Sprintf(
			"all %d assertions passed", len(results),
		),
	}
}

// AnyPassComposite creates an Evaluator that checks whether at
// least one result in a slice has passed.
func AnyPassComposite(
	engine Engine,
	assertions []Definition,
	values map[string]any,
) Result {
	results := engine.EvaluateAll(assertions, values)

	for _, r := range results {
		if r.Passed {
			return Result{
				Type:   "any_pass",
				Passed: true,
				Message: fmt.Sprintf(
					"assertion '%s' on target '%s' passed",
					r.Type, r.Target,
				),
			}
		}
	}

	return Result{
		Type:   "any_pass",
		Passed: false,
		Message: fmt.Sprintf(
			"none of %d assertions passed",
			len(results),
		),
	}
}

// CompositeAllPass returns an Evaluator function that runs a
// fixed set of sub-assertions and requires all to pass.
func CompositeAllPass(
	engine Engine,
	subAssertions []Definition,
) Evaluator {
	return func(_ Definition, value any) (bool, string) {
		values := map[string]any{}
		for _, a := range subAssertions {
			values[a.Target] = value
		}
		r := AllPassComposite(engine, subAssertions, values)
		return r.Passed, r.Message
	}
}

// CompositeAnyPass returns an Evaluator function that runs a
// fixed set of sub-assertions and requires at least one to
// pass.
func CompositeAnyPass(
	engine Engine,
	subAssertions []Definition,
) Evaluator {
	return func(_ Definition, value any) (bool, string) {
		values := map[string]any{}
		for _, a := range subAssertions {
			values[a.Target] = value
		}
		r := AnyPassComposite(engine, subAssertions, values)
		return r.Passed, r.Message
	}
}
