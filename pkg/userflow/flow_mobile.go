package userflow

// MobileFlow defines a sequence of mobile interactions to
// execute as a user flow test.
type MobileFlow struct {
	// Name identifies this mobile flow.
	Name string `json:"name"`

	// Description explains the purpose of this flow.
	Description string `json:"description"`

	// Config holds mobile device configuration.
	Config MobileConfig `json:"config"`

	// AppPath is the path to the application to install
	// and test.
	AppPath string `json:"app_path"`

	// Steps is the ordered sequence of mobile steps.
	Steps []MobileStep `json:"steps"`
}

// MobileStep defines a single step in a mobile flow.
type MobileStep struct {
	// Name identifies this step.
	Name string `json:"name"`

	// Action is the mobile action to perform (launch, tap,
	// send_keys, press_key, screenshot, wait, stop).
	Action string `json:"action"`

	// X is the X coordinate for tap actions.
	X int `json:"x,omitempty"`

	// Y is the Y coordinate for tap actions.
	Y int `json:"y,omitempty"`

	// Value is the text for send_keys or keycode for
	// press_key actions.
	Value string `json:"value,omitempty"`

	// Assertions define checks to run after this step.
	Assertions []StepAssertion `json:"assertions,omitempty"`
}
