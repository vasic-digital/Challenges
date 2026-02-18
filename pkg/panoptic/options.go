package panoptic

import "time"

// ChallengeOption configures a PanopticChallenge.
type ChallengeOption func(*PanopticChallenge)

// WithConfigPath sets the YAML config file path for the
// challenge.
func WithConfigPath(path string) ChallengeOption {
	return func(c *PanopticChallenge) {
		c.configPath = path
	}
}

// WithConfigBuilder sets a ConfigBuilder for programmatic
// config generation.
func WithConfigBuilder(b *ConfigBuilder) ChallengeOption {
	return func(c *PanopticChallenge) {
		c.configBuilder = b
	}
}

// WithRunOpts appends run options for the Panoptic adapter.
func WithRunOpts(opts ...RunOption) ChallengeOption {
	return func(c *PanopticChallenge) {
		c.runOpts = append(c.runOpts, opts...)
	}
}

// RunOption configures a single Panoptic run invocation.
type RunOption func(*runConfig)

// runConfig holds resolved run options.
type runConfig struct {
	outputDir string
	verbose   bool
	timeout   time.Duration
	env       map[string]string
}

// RunWithOutputDir overrides the output directory.
func RunWithOutputDir(dir string) RunOption {
	return func(c *runConfig) {
		c.outputDir = dir
	}
}

// RunWithVerbose enables verbose logging.
func RunWithVerbose() RunOption {
	return func(c *runConfig) {
		c.verbose = true
	}
}

// RunWithTimeout sets a timeout for the run.
func RunWithTimeout(d time.Duration) RunOption {
	return func(c *runConfig) {
		c.timeout = d
	}
}

// RunWithEnv adds environment variables to the run.
func RunWithEnv(env map[string]string) RunOption {
	return func(c *runConfig) {
		if c.env == nil {
			c.env = make(map[string]string)
		}
		for k, v := range env {
			c.env[k] = v
		}
	}
}

// AITestingOpts configures AI testing features.
type AITestingOpts struct {
	ErrorDetection      bool
	TestGeneration      bool
	VisionAnalysis      bool
	ConfidenceThreshold float64
}

// CloudOpts configures cloud integration.
type CloudOpts struct {
	Provider   string
	Bucket     string
	EnableSync bool
}

// EnterpriseOpts configures enterprise features.
type EnterpriseOpts struct {
	ConfigPath string
}

// resolveRunConfig applies RunOptions to produce a runConfig.
func resolveRunConfig(opts []RunOption) *runConfig {
	cfg := &runConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
