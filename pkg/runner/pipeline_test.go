package runner

import (
	"context"
	"errors"
	"testing"

	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.challenges/pkg/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeline_Execute_Success(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	p := NewPipeline(r)

	c, err := reg.Get("a")
	require.NoError(t, err)

	result, err := p.Execute(
		context.Background(), c,
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestPipeline_Execute_PreHookFails(t *testing.T) {
	reg := registry.NewRegistry()
	s := newStub("a")
	require.NoError(t, reg.Register(s))

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	p.AddPreHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		return errors.New("pre-hook fail")
	})

	result, err := p.Execute(
		context.Background(), s,
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusError, result.Status)
	assert.Contains(t, result.Error, "pre-hook")
}

func TestPipeline_Execute_PreHookErrorPropagation(
	t *testing.T,
) {
	tests := []struct {
		name     string
		hookErr  error
		wantErr  string
		wantStat string
	}{
		{
			name:     "generic pre-hook error",
			hookErr:  errors.New("setup failed"),
			wantErr:  "setup failed",
			wantStat: challenge.StatusError,
		},
		{
			name:     "wrapped error",
			hookErr:  errors.New("env check: missing VAR"),
			wantErr:  "missing VAR",
			wantStat: challenge.StatusError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newStub("a")
			reg := setupRegistry(t, s)
			r := NewRunner(
				WithRegistry(reg),
				WithResultsDir(t.TempDir()),
			)
			p := NewPipeline(r)
			p.AddPreHook(func(
				_ context.Context,
				_ challenge.Challenge,
				_ *challenge.Config,
			) error {
				return tc.hookErr
			})

			result, err := p.Execute(
				context.Background(), s,
				challenge.NewConfig("a"),
			)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStat, result.Status)
			assert.Contains(t, result.Error, tc.wantErr)
		})
	}
}

func TestPipeline_Execute_MultiplePreHooks_StopsOnFirst(
	t *testing.T,
) {
	s := newStub("a")
	reg := setupRegistry(t, s)
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	var order []string
	p.AddPreHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		order = append(order, "pre1")
		return errors.New("stop here")
	})
	p.AddPreHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		order = append(order, "pre2")
		return nil
	})

	result, err := p.Execute(
		context.Background(), s,
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusError, result.Status)
	// Second pre-hook should not have been called.
	assert.Equal(t, []string{"pre1"}, order)
}

func TestPipeline_Execute_PostHookWarning(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	p := NewPipeline(r)
	p.AddPostHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		return errors.New("post-hook warning")
	})

	c, err := reg.Get("a")
	require.NoError(t, err)

	// Post-hook errors are warnings, not failures.
	result, err := p.Execute(
		context.Background(), c,
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestPipeline_Execute_PostHookMultiple(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))
	logger := &stubLogger{}
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
		WithLogger(logger),
	)
	p := NewPipeline(r)

	var postOrder []string
	p.AddPostHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		postOrder = append(postOrder, "post1")
		return errors.New("warn1")
	})
	p.AddPostHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		postOrder = append(postOrder, "post2")
		return nil
	})

	c, err := reg.Get("a")
	require.NoError(t, err)

	result, err := p.Execute(
		context.Background(), c,
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	// Both post-hooks should have run despite first error.
	assert.Equal(t, []string{"post1", "post2"}, postOrder)
}

func TestPipeline_ExecuteSequence(t *testing.T) {
	a := newStub("a")
	b := newStub("b")
	reg := setupRegistry(t, a, b)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	challenges := []challenge.Challenge{a, b}
	results, err := p.ExecuteSequence(
		context.Background(), challenges,
		challenge.NewConfig(""),
	)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, challenge.ID("a"), results[0].ChallengeID)
	assert.Equal(t, challenge.ID("b"), results[1].ChallengeID)
}

func TestPipeline_ExecuteSequence_Empty(t *testing.T) {
	r := NewRunner(WithResultsDir(t.TempDir()))
	p := NewPipeline(r)

	results, err := p.ExecuteSequence(
		context.Background(),
		nil,
		challenge.NewConfig(""),
	)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestPipeline_ExecuteSequence_SingleChallenge(
	t *testing.T,
) {
	s := newStub("only")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	results, err := p.ExecuteSequence(
		context.Background(),
		[]challenge.Challenge{s},
		challenge.NewConfig(""),
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, challenge.StatusPassed, results[0].Status)
}

func TestPipeline_ExecuteSequence_PreHookApplied(
	t *testing.T,
) {
	a := newStub("a")
	b := newStub("b")
	reg := setupRegistry(t, a, b)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	callCount := 0
	p.AddPreHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		callCount++
		return nil
	})

	challenges := []challenge.Challenge{a, b}
	results, err := p.ExecuteSequence(
		context.Background(), challenges,
		challenge.NewConfig(""),
	)
	require.NoError(t, err)
	require.Len(t, results, 2)
	// Pre-hook should be called once per challenge.
	assert.Equal(t, 2, callCount)
}

func TestPipeline_HookOrder(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	var order []string
	p := NewPipeline(r)

	p.AddPreHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		order = append(order, "pre1")
		return nil
	})
	p.AddPreHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		order = append(order, "pre2")
		return nil
	})
	p.AddPostHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		order = append(order, "post1")
		return nil
	})

	c, err := reg.Get("a")
	require.NoError(t, err)

	_, err = p.Execute(
		context.Background(), c,
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"pre1", "pre2", "post1"}, order)
}

func TestPipeline_Execute_ReceivesCorrectChallenge(
	t *testing.T,
) {
	s := newStub("target")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	var receivedID challenge.ID
	p.AddPreHook(func(
		_ context.Context,
		c challenge.Challenge,
		_ *challenge.Config,
	) error {
		receivedID = c.ID()
		return nil
	})

	result, err := p.Execute(
		context.Background(), s,
		challenge.NewConfig("target"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Equal(t, challenge.ID("target"), receivedID)
}

func TestPipeline_Execute_ReceivesCorrectConfig(
	t *testing.T,
) {
	s := newStub("a")
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	cfg := challenge.NewConfig("a")
	cfg.Verbose = true
	cfg.Environment = map[string]string{
		"KEY": "VALUE",
	}

	var receivedVerbose bool
	var receivedEnv map[string]string
	p.AddPreHook(func(
		_ context.Context,
		_ challenge.Challenge,
		c *challenge.Config,
	) error {
		receivedVerbose = c.Verbose
		receivedEnv = c.Environment
		return nil
	})

	_, err := p.Execute(
		context.Background(), s, cfg,
	)
	require.NoError(t, err)
	assert.True(t, receivedVerbose)
	assert.Equal(t, "VALUE", receivedEnv["KEY"])
}

func TestPipeline_ExecuteSequence_ConfigureError(t *testing.T) {
	a := newStub("a")
	a.configureErr = errors.New("configure failed")
	reg := setupRegistry(t, a)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	results, err := p.ExecuteSequence(
		context.Background(),
		[]challenge.Challenge{a},
		challenge.NewConfig(""),
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, challenge.StatusError, results[0].Status)
}

func TestPipeline_ExecuteSequence_ContextCancellation(t *testing.T) {
	a := newStub("a")
	b := newStub("b")
	reg := setupRegistry(t, a, b)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results, err := p.ExecuteSequence(
		ctx,
		[]challenge.Challenge{a, b},
		challenge.NewConfig(""),
	)
	// The sequence should handle cancellation gracefully
	_ = results
	_ = err
}

func TestPipeline_Execute_MultiplePreHooksAllPass(t *testing.T) {
	s := newStub("a")
	reg := setupRegistry(t, s)
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)
	p := NewPipeline(r)

	var callOrder []string
	p.AddPreHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		callOrder = append(callOrder, "pre1")
		return nil
	})
	p.AddPreHook(func(
		_ context.Context,
		_ challenge.Challenge,
		_ *challenge.Config,
	) error {
		callOrder = append(callOrder, "pre2")
		return nil
	})

	result, err := p.Execute(
		context.Background(), s,
		challenge.NewConfig("a"),
	)
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Equal(t, []string{"pre1", "pre2"}, callOrder)
}
