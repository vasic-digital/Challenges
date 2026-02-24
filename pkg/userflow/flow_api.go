package userflow

// APIFlow defines a sequence of API steps to execute as a
// user flow test.
type APIFlow struct {
	// Name identifies this API flow.
	Name string `json:"name"`

	// Description explains the purpose of this flow.
	Description string `json:"description"`

	// BaseURL is the base URL for all API requests in
	// this flow.
	BaseURL string `json:"base_url"`

	// Credentials holds authentication info for the flow.
	Credentials Credentials `json:"credentials"`

	// Steps is the ordered sequence of API steps.
	Steps []APIStep `json:"steps"`
}

// APIStep defines a single step in an API flow.
type APIStep struct {
	// Name identifies this step.
	Name string `json:"name"`

	// Method is the HTTP method (GET, POST, PUT, DELETE).
	Method string `json:"method"`

	// Path is the URL path relative to the flow's BaseURL.
	Path string `json:"path"`

	// Body is the JSON request body (for POST/PUT).
	Body string `json:"body,omitempty"`

	// Headers are additional request headers.
	Headers map[string]string `json:"headers,omitempty"`

	// Assertions define checks to run on the response.
	Assertions []StepAssertion `json:"assertions"`
}

// StepAssertion defines a single assertion on a step result.
type StepAssertion struct {
	// Type is the assertion evaluator type.
	Type string `json:"type"`

	// Target is the response field to check.
	Target string `json:"target"`

	// Value is the expected value.
	Value any `json:"value,omitempty"`

	// Message is the human-readable failure message.
	Message string `json:"message"`
}
