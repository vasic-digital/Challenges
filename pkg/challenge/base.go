package challenge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BaseChallenge provides a reusable foundation for building
// challenges using the template method pattern. Embed this struct
// and override Execute to implement custom challenge logic.
type BaseChallenge struct {
	id           ID
	name         string
	description  string
	category     string
	dependencies []ID
	config       *Config
	logger       Logger
	assertions   AssertionEngine
}

// NewBaseChallenge creates a BaseChallenge with the given identity
// fields. Logger and AssertionEngine can be set later via setters.
func NewBaseChallenge(
	id ID,
	name, description, category string,
	deps []ID,
) BaseChallenge {
	if deps == nil {
		deps = []ID{}
	}
	return BaseChallenge{
		id:           id,
		name:         name,
		description:  description,
		category:     category,
		dependencies: deps,
	}
}

// ID returns the challenge identifier.
func (b *BaseChallenge) ID() ID { return b.id }

// Name returns the challenge name.
func (b *BaseChallenge) Name() string { return b.name }

// Description returns the challenge description.
func (b *BaseChallenge) Description() string {
	return b.description
}

// Category returns the challenge category.
func (b *BaseChallenge) Category() string { return b.category }

// Dependencies returns the challenge dependency IDs.
func (b *BaseChallenge) Dependencies() []ID {
	return b.dependencies
}

// Config returns the current runtime configuration, or nil if
// Configure has not been called.
func (b *BaseChallenge) Config() *Config { return b.config }

// SetLogger sets the logger used by this challenge.
func (b *BaseChallenge) SetLogger(l Logger) {
	b.logger = l
}

// SetAssertionEngine sets the assertion engine used by this
// challenge.
func (b *BaseChallenge) SetAssertionEngine(e AssertionEngine) {
	b.assertions = e
}

// Configure stores the runtime config and ensures output
// directories exist.
func (b *BaseChallenge) Configure(config *Config) error {
	if config == nil {
		return fmt.Errorf("config must not be nil")
	}
	b.config = config

	if err := os.MkdirAll(b.ResultsDir(), 0o755); err != nil {
		return fmt.Errorf(
			"create results dir %s: %w",
			b.ResultsDir(), err,
		)
	}
	if err := os.MkdirAll(b.LogsDir(), 0o755); err != nil {
		return fmt.Errorf(
			"create logs dir %s: %w",
			b.LogsDir(), err,
		)
	}
	return nil
}

// Validate performs basic precondition checks. Override to add
// custom validation; call BaseChallenge.Validate first.
func (b *BaseChallenge) Validate(_ context.Context) error {
	if b.config == nil {
		return fmt.Errorf(
			"challenge %s: not configured", b.id,
		)
	}
	return nil
}

// Cleanup is a no-op by default. Override to release resources.
func (b *BaseChallenge) Cleanup(_ context.Context) error {
	if b.logger != nil {
		return b.logger.Close()
	}
	return nil
}

// ResultsDir returns the results directory path for this
// challenge.
func (b *BaseChallenge) ResultsDir() string {
	if b.config == nil {
		return "results"
	}
	return filepath.Join(
		b.config.ResultsDir,
		string(b.id),
	)
}

// LogsDir returns the logs directory path for this challenge.
func (b *BaseChallenge) LogsDir() string {
	if b.config == nil {
		return "logs"
	}
	return filepath.Join(
		b.config.LogsDir,
		string(b.id),
	)
}

// GetEnv returns an environment variable from the config, or
// the fallback value if not set.
func (b *BaseChallenge) GetEnv(
	key, fallback string,
) string {
	if b.config == nil {
		return fallback
	}
	return b.config.GetEnv(key, fallback)
}

// WriteJSONResult serializes a Result to a JSON file in the
// results directory.
func (b *BaseChallenge) WriteJSONResult(
	r *Result,
) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	path := filepath.Join(
		b.ResultsDir(), "result.json",
	)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write result %s: %w", path, err)
	}
	return nil
}

// WriteMarkdownReport writes a human-readable Markdown summary
// of the result to the results directory.
func (b *BaseChallenge) WriteMarkdownReport(
	r *Result,
) error {
	md := fmt.Sprintf(
		"# %s\n\n"+
			"**ID**: %s\n"+
			"**Status**: %s\n"+
			"**Duration**: %s\n\n"+
			"## Assertions\n\n",
		r.ChallengeName,
		r.ChallengeID,
		r.Status,
		r.Duration,
	)
	for _, a := range r.Assertions {
		status := "PASS"
		if !a.Passed {
			status = "FAIL"
		}
		md += fmt.Sprintf(
			"- [%s] %s: %s\n",
			status, a.Target, a.Message,
		)
	}
	if r.Error != "" {
		md += fmt.Sprintf("\n## Error\n\n```\n%s\n```\n", r.Error)
	}
	path := filepath.Join(
		b.ResultsDir(), "report.md",
	)
	if err := os.WriteFile(
		path, []byte(md), 0o644,
	); err != nil {
		return fmt.Errorf("write report %s: %w", path, err)
	}
	return nil
}

// ReadDependencyResult reads and unmarshals the result JSON
// produced by an upstream dependency.
func (b *BaseChallenge) ReadDependencyResult(
	depID ID,
) (*Result, error) {
	if b.config == nil {
		return nil, fmt.Errorf("not configured")
	}
	path, ok := b.config.Dependencies[depID]
	if !ok {
		return nil, fmt.Errorf(
			"dependency %s: path not found in config",
			depID,
		)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"read dependency result %s: %w", path, err,
		)
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf(
			"unmarshal dependency result %s: %w", path, err,
		)
	}
	return &result, nil
}

// EvaluateAssertions uses the AssertionEngine to evaluate all
// assertions from a Definition against the provided values.
func (b *BaseChallenge) EvaluateAssertions(
	defs []AssertionDef,
	values map[string]any,
) []AssertionResult {
	if b.assertions == nil {
		results := make([]AssertionResult, len(defs))
		for i, d := range defs {
			results[i] = AssertionResult{
				Type:    d.Type,
				Target:  d.Target,
				Passed:  false,
				Message: "no assertion engine configured",
			}
		}
		return results
	}
	return b.assertions.EvaluateAll(defs, values)
}

// CreateResult builds a Result pre-populated with this
// challenge's identity and the given status and timing.
func (b *BaseChallenge) CreateResult(
	status string,
	start time.Time,
	assertions []AssertionResult,
	metrics map[string]MetricValue,
	outputs map[string]string,
	errMsg string,
) *Result {
	end := time.Now()
	return &Result{
		ChallengeID:   b.id,
		ChallengeName: b.name,
		Status:        status,
		StartTime:     start,
		EndTime:       end,
		Duration:      end.Sub(start),
		Assertions:    assertions,
		Metrics:       metrics,
		Outputs:       outputs,
		Logs: LogPaths{
			ChallengeLog: filepath.Join(
				b.LogsDir(), "challenge.log",
			),
			OutputLog: filepath.Join(
				b.LogsDir(), "output.log",
			),
			APIRequests: filepath.Join(
				b.LogsDir(), "api_requests.log",
			),
			APIResponses: filepath.Join(
				b.LogsDir(), "api_responses.log",
			),
		},
		Error: errMsg,
	}
}

// logInfo logs at info level if a logger is available.
func (b *BaseChallenge) logInfo(msg string, args ...any) {
	if b.logger != nil {
		b.logger.Info(msg, args...)
	}
}

// logError logs at error level if a logger is available.
func (b *BaseChallenge) logError(msg string, args ...any) {
	if b.logger != nil {
		b.logger.Error(msg, args...)
	}
}
