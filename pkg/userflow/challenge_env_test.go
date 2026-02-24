package userflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

func TestNewEnvironmentSetupChallenge(t *testing.T) {
	called := false
	setup := func(_ context.Context) error {
		called = true
		return nil
	}
	ch := NewEnvironmentSetupChallenge(
		"ENV-SETUP", setup, 30*time.Second,
	)

	assert.Equal(
		t, challenge.ID("ENV-SETUP"), ch.ID(),
	)
	assert.Equal(t, "Environment Setup", ch.Name())
	assert.Equal(t, "environment", ch.Category())
	assert.Empty(t, ch.Dependencies())
	assert.False(t, called)
}

func TestEnvironmentSetupChallenge_Execute_Success(
	t *testing.T,
) {
	setup := func(_ context.Context) error {
		return nil
	}
	ch := NewEnvironmentSetupChallenge(
		"ENV-001", setup, 5*time.Second,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.True(t, result.Assertions[0].Passed)
	assert.Equal(
		t, "setup_succeeds",
		result.Assertions[0].Target,
	)
	assert.Contains(
		t, result.Assertions[0].Message,
		"completed successfully",
	)

	dur, ok := result.Metrics["setup_duration"]
	require.True(t, ok)
	assert.Equal(t, "s", dur.Unit)
	assert.GreaterOrEqual(t, dur.Value, 0.0)
}

func TestEnvironmentSetupChallenge_Execute_Failure(
	t *testing.T,
) {
	setup := func(_ context.Context) error {
		return fmt.Errorf("container start failed")
	}
	ch := NewEnvironmentSetupChallenge(
		"ENV-002", setup, 5*time.Second,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Message,
		"container start failed",
	)
	assert.Equal(t, "container start failed", result.Error)
}

func TestEnvironmentSetupChallenge_Execute_Timeout(
	t *testing.T,
) {
	setup := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	}
	ch := NewEnvironmentSetupChallenge(
		"ENV-003", setup, 50*time.Millisecond,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.Contains(t, result.Error, "context deadline exceeded")
}

func TestEnvironmentSetupChallenge_Execute_NoTimeout(
	t *testing.T,
) {
	setup := func(_ context.Context) error {
		return nil
	}
	ch := NewEnvironmentSetupChallenge(
		"ENV-004", setup, 0,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
}

func TestNewEnvironmentTeardownChallenge(t *testing.T) {
	called := false
	teardown := func(_ context.Context) error {
		called = true
		return nil
	}
	ch := NewEnvironmentTeardownChallenge(
		"ENV-TEARDOWN", teardown,
	)

	assert.Equal(
		t, challenge.ID("ENV-TEARDOWN"), ch.ID(),
	)
	assert.Equal(t, "Environment Teardown", ch.Name())
	assert.Equal(t, "environment", ch.Category())
	assert.Empty(t, ch.Dependencies())
	assert.False(t, called)
}

func TestEnvironmentTeardownChallenge_Execute_Success(
	t *testing.T,
) {
	teardown := func(_ context.Context) error {
		return nil
	}
	ch := NewEnvironmentTeardownChallenge(
		"ENV-TD-001", teardown,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.True(t, result.Assertions[0].Passed)
	assert.Equal(
		t, "teardown_succeeds",
		result.Assertions[0].Target,
	)
	assert.Contains(
		t, result.Assertions[0].Message,
		"completed successfully",
	)

	dur, ok := result.Metrics["teardown_duration"]
	require.True(t, ok)
	assert.Equal(t, "s", dur.Unit)
}

func TestEnvironmentTeardownChallenge_Execute_Failure(
	t *testing.T,
) {
	teardown := func(_ context.Context) error {
		return fmt.Errorf("container stop timed out")
	}
	ch := NewEnvironmentTeardownChallenge(
		"ENV-TD-002", teardown,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Message,
		"container stop timed out",
	)
	assert.Equal(
		t, "container stop timed out", result.Error,
	)
}
