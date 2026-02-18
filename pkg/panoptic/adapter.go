package panoptic

import "context"

// PanopticAdapter abstracts the execution of Panoptic, allowing
// different implementations (CLI subprocess, mock, etc.).
type PanopticAdapter interface {
	// Run executes a Panoptic config file and returns the
	// aggregated result. The configPath should point to a
	// valid Panoptic YAML config.
	Run(
		ctx context.Context,
		configPath string,
		opts ...RunOption,
	) (*PanopticRunResult, error)

	// Version returns the Panoptic binary version string.
	Version(ctx context.Context) (string, error)

	// Available returns true if the Panoptic binary is
	// reachable and executable.
	Available(ctx context.Context) bool
}
