package runner

import (
	"context"

	"digital.vasic.challenges/pkg/challenge"
)

// Pipeline represents a sequence of hooks and a runner that
// executes challenges with pre- and post-processing steps.
type Pipeline struct {
	runner    *DefaultRunner
	preHooks  []Hook
	postHooks []Hook
}

// NewPipeline creates a Pipeline wrapping the given runner.
func NewPipeline(runner *DefaultRunner) *Pipeline {
	return &Pipeline{
		runner: runner,
	}
}

// AddPreHook appends a pre-execution hook to the pipeline.
func (p *Pipeline) AddPreHook(h Hook) {
	p.preHooks = append(p.preHooks, h)
}

// AddPostHook appends a post-execution hook to the pipeline.
func (p *Pipeline) AddPostHook(h Hook) {
	p.postHooks = append(p.postHooks, h)
}

// Execute runs a challenge through the pipeline:
// pre-hooks -> runner.executeChallenge -> post-hooks.
func (p *Pipeline) Execute(
	ctx context.Context,
	c challenge.Challenge,
	config *challenge.Config,
) (*challenge.Result, error) {
	// Run pipeline-level pre-hooks.
	for _, hook := range p.preHooks {
		if err := hook(ctx, c, config); err != nil {
			return &challenge.Result{
				ChallengeID:   c.ID(),
				ChallengeName: c.Name(),
				Status:        challenge.StatusError,
				Error: "pipeline pre-hook failed: " +
					err.Error(),
			}, nil
		}
	}

	// Execute via runner.
	result, err := p.runner.executeChallenge(ctx, c, config)
	if err != nil {
		return result, err
	}

	// Run pipeline-level post-hooks.
	for _, hook := range p.postHooks {
		if hookErr := hook(ctx, c, config); hookErr != nil {
			p.runner.logEvent(
				"pipeline_post_hook_warning",
				map[string]any{
					"challenge_id": c.ID(),
					"warning":      hookErr.Error(),
				},
			)
		}
	}

	return result, nil
}

// ExecuteSequence runs multiple challenges through the pipeline
// in order.
func (p *Pipeline) ExecuteSequence(
	ctx context.Context,
	challenges []challenge.Challenge,
	config *challenge.Config,
) ([]*challenge.Result, error) {
	results := make(
		[]*challenge.Result, 0, len(challenges),
	)

	for _, c := range challenges {
		cfg := *config
		cfg.ChallengeID = c.ID()

		result, err := p.Execute(ctx, c, &cfg)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}
