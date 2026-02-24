package userflow

// BrowserFlow defines a sequence of browser interactions to
// execute as a user flow test.
type BrowserFlow struct {
	// Name identifies this browser flow.
	Name string `json:"name"`

	// Description explains the purpose of this flow.
	Description string `json:"description"`

	// StartURL is the initial URL to navigate to.
	StartURL string `json:"start_url"`

	// Config holds browser configuration.
	Config BrowserConfig `json:"config"`

	// Steps is the ordered sequence of browser steps.
	Steps []BrowserStep `json:"steps"`
}

// BrowserStep defines a single step in a browser flow.
type BrowserStep struct {
	// Name identifies this step.
	Name string `json:"name"`

	// Action is the browser action to perform (navigate,
	// click, fill, select, wait, screenshot, evaluate_js).
	Action string `json:"action"`

	// Selector is the CSS selector for the target element.
	Selector string `json:"selector,omitempty"`

	// Value is the value for fill/select/navigate actions.
	Value string `json:"value,omitempty"`

	// Script is the JavaScript to evaluate (for
	// evaluate_js action).
	Script string `json:"script,omitempty"`

	// Assertions define checks to run after this step.
	Assertions []StepAssertion `json:"assertions,omitempty"`
}
