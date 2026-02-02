package infra

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainersAdapter_EnsureRunning(t *testing.T) {
	called := false
	a := NewContainersAdapter(
		WithEnsureFunc(func(ctx context.Context, name string) error {
			called = true
			assert.Equal(t, "redis", name)
			return nil
		}),
	)

	err := a.EnsureRunning(context.Background(), "redis")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestContainersAdapter_EnsureRunning_NotConfigured(t *testing.T) {
	a := NewContainersAdapter()
	err := a.EnsureRunning(context.Background(), "redis")
	assert.Error(t, err)
}

func TestContainersAdapter_Release_Optional(t *testing.T) {
	a := NewContainersAdapter()
	err := a.Release(context.Background(), "redis")
	assert.NoError(t, err) // Release is optional, no error
}

func TestContainersAdapter_HealthCheck(t *testing.T) {
	a := NewContainersAdapter(
		WithHealthFunc(func(ctx context.Context, name string) error {
			if name == "healthy" {
				return nil
			}
			return fmt.Errorf("service %s unhealthy", name)
		}),
	)

	assert.NoError(t, a.HealthCheck(context.Background(), "healthy"))
	assert.Error(t, a.HealthCheck(context.Background(), "sick"))
}

func TestContainersAdapter_Shutdown(t *testing.T) {
	shutdownCalled := false
	a := NewContainersAdapter(
		WithShutdownFunc(func(ctx context.Context) error {
			shutdownCalled = true
			return nil
		}),
	)

	err := a.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, shutdownCalled)
}

func TestContainersAdapter_Shutdown_Optional(t *testing.T) {
	a := NewContainersAdapter()
	err := a.Shutdown(context.Background())
	assert.NoError(t, err)
}
