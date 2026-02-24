package userflow

// ChallengeOption configures a userflow challenge.
type ChallengeOption func(*challengeConfig)

// challengeConfig holds resolved configuration for a userflow
// challenge.
type challengeConfig struct {
	containerized bool
	projectRoot   string
	runtimeName   string
}

// WithContainerized sets whether the challenge should run
// inside a container.
func WithContainerized(use bool) ChallengeOption {
	return func(c *challengeConfig) {
		c.containerized = use
	}
}

// WithProjectRoot sets the project root directory for the
// challenge.
func WithProjectRoot(root string) ChallengeOption {
	return func(c *challengeConfig) {
		c.projectRoot = root
	}
}

// WithRuntimeName sets the container runtime name (e.g.,
// "podman", "docker").
func WithRuntimeName(name string) ChallengeOption {
	return func(c *challengeConfig) {
		c.runtimeName = name
	}
}

// resolveChallengeConfig applies all options to a default
// configuration and returns the result.
func resolveChallengeConfig(
	opts []ChallengeOption,
) *challengeConfig {
	cfg := &challengeConfig{
		containerized: false,
		projectRoot:   ".",
		runtimeName:   "podman",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
