package registry

import (
	"testing"

	"digital.vasic.challenges/pkg/challenge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDependencyOrder_NoDeps(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("a")))
	require.NoError(t, r.Register(newStub("b")))

	ordered, err := r.GetDependencyOrder()
	require.NoError(t, err)
	require.Len(t, ordered, 2)
	// Sorted alphabetically since no deps constrain order.
	assert.Equal(t, challenge.ID("a"), ordered[0].ID())
	assert.Equal(t, challenge.ID("b"), ordered[1].ID())
}

func TestGetDependencyOrder_LinearChain(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("c", "b")))
	require.NoError(t, r.Register(newStub("b", "a")))
	require.NoError(t, r.Register(newStub("a")))

	ordered, err := r.GetDependencyOrder()
	require.NoError(t, err)
	require.Len(t, ordered, 3)
	assert.Equal(t, challenge.ID("a"), ordered[0].ID())
	assert.Equal(t, challenge.ID("b"), ordered[1].ID())
	assert.Equal(t, challenge.ID("c"), ordered[2].ID())
}

func TestGetDependencyOrder_Diamond(t *testing.T) {
	// a -> b, a -> c, b -> d, c -> d
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("d")))
	require.NoError(t, r.Register(newStub("b", "d")))
	require.NoError(t, r.Register(newStub("c", "d")))
	require.NoError(t, r.Register(newStub("a", "b", "c")))

	ordered, err := r.GetDependencyOrder()
	require.NoError(t, err)
	require.Len(t, ordered, 4)

	// d must come first, a must come last.
	assert.Equal(t, challenge.ID("d"), ordered[0].ID())
	assert.Equal(t, challenge.ID("a"), ordered[3].ID())
}

func TestGetDependencyOrder_CycleDetected(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("a", "b")))
	require.NoError(t, r.Register(newStub("b", "a")))

	_, err := r.GetDependencyOrder()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestGetDependencyOrder_ThreeNodeCycle(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("a", "c")))
	require.NoError(t, r.Register(newStub("b", "a")))
	require.NoError(t, r.Register(newStub("c", "b")))

	_, err := r.GetDependencyOrder()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestGetDependencyOrder_Empty(t *testing.T) {
	r := NewRegistry()

	ordered, err := r.GetDependencyOrder()
	require.NoError(t, err)
	assert.Empty(t, ordered)
}

func TestGetDependencyOrder_SingleNode(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("solo")))

	ordered, err := r.GetDependencyOrder()
	require.NoError(t, err)
	require.Len(t, ordered, 1)
	assert.Equal(t, challenge.ID("solo"), ordered[0].ID())
}

func TestDetectCycle_Simple(t *testing.T) {
	challenges := map[challenge.ID]challenge.Challenge{
		"a": newStub("a", "b"),
		"b": newStub("b", "a"),
	}

	desc := detectCycle(challenges)
	assert.NotEmpty(t, desc)
	assert.NotEqual(t, "unknown cycle", desc)
}

func TestGetDeps_Missing(t *testing.T) {
	m := map[challenge.ID]challenge.Challenge{}
	deps := getDeps(m, "nonexistent")
	assert.Nil(t, deps)
}
