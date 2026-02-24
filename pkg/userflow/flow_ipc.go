package userflow

// IPCCommand defines a command to send to a desktop
// application's backend via IPC (e.g., Tauri invoke).
type IPCCommand struct {
	// Name identifies this IPC command.
	Name string `json:"name"`

	// Command is the IPC command name recognized by the
	// desktop app backend.
	Command string `json:"command"`

	// Args are the arguments to pass with the command.
	Args []string `json:"args,omitempty"`

	// ExpectedResult is the expected response string for
	// validation.
	ExpectedResult string `json:"expected_result,omitempty"`

	// Assertions define checks to run on the IPC response.
	Assertions []StepAssertion `json:"assertions,omitempty"`
}
