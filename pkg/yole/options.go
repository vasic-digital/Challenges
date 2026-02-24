package yole

// ChallengeOption configures a Yole challenge.
type ChallengeOption func(*challengeConfig)

// challengeConfig holds resolved challenge options.
type challengeConfig struct {
	useDocker   bool
	projectRoot string
}

// WithDocker enables Docker-based Gradle execution.
func WithDocker(use bool) ChallengeOption {
	return func(c *challengeConfig) {
		c.useDocker = use
	}
}

// WithProjectRoot sets the Yole project root directory.
func WithProjectRoot(root string) ChallengeOption {
	return func(c *challengeConfig) {
		c.projectRoot = root
	}
}

// resolveChallengeConfig applies options.
func resolveChallengeConfig(
	opts []ChallengeOption,
) *challengeConfig {
	cfg := &challengeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
