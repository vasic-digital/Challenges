package challenge

import "time"

// Config holds runtime configuration for a challenge execution.
type Config struct {
	// ChallengeID identifies which challenge this config is for.
	ChallengeID ID `json:"challenge_id"`

	// ResultsDir is the directory where result JSON files are
	// written.
	ResultsDir string `json:"results_dir"`

	// LogsDir is the directory where log files are written.
	LogsDir string `json:"logs_dir"`

	// Timeout is the maximum duration for challenge execution.
	// A zero value means no timeout.
	Timeout time.Duration `json:"timeout"`

	// Verbose enables detailed logging output.
	Verbose bool `json:"verbose"`

	// Environment holds key-value pairs injected into the
	// challenge execution environment.
	Environment map[string]string `json:"environment"`

	// Dependencies maps challenge IDs to the file paths of
	// their result JSON files, allowing challenges to read
	// outputs from upstream dependencies.
	Dependencies map[ID]string `json:"dependencies"`
}

// NewConfig creates a Config with sensible defaults.
func NewConfig(id ID) *Config {
	return &Config{
		ChallengeID:  id,
		ResultsDir:   "results",
		LogsDir:      "logs",
		Timeout:      5 * time.Minute,
		Environment:  make(map[string]string),
		Dependencies: make(map[ID]string),
	}
}

// GetEnv returns the value of an environment variable from
// the config, or the fallback if not set.
func (c *Config) GetEnv(key, fallback string) string {
	if c.Environment == nil {
		return fallback
	}
	if v, ok := c.Environment[key]; ok {
		return v
	}
	return fallback
}
