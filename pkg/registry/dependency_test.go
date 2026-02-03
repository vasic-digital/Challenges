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

func TestDetectCycle_NoCycle(t *testing.T) {
	// Create a graph with no cycle
	challenges := map[challenge.ID]challenge.Challenge{
		"a": newStub("a"),
		"b": newStub("b", "a"),
		"c": newStub("c", "b"),
	}

	desc := detectCycle(challenges)
	// When no cycle is found, should return "unknown cycle"
	assert.Equal(t, "unknown cycle", desc)
}

func TestDetectCycle_SelfCycle(t *testing.T) {
	// A challenge that depends on itself
	challenges := map[challenge.ID]challenge.Challenge{
		"self": newStub("self", "self"),
	}

	desc := detectCycle(challenges)
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "self")
}

func TestDetectCycle_LongCycle(t *testing.T) {
	// a -> b -> c -> d -> a
	challenges := map[challenge.ID]challenge.Challenge{
		"a": newStub("a", "d"),
		"b": newStub("b", "a"),
		"c": newStub("c", "b"),
		"d": newStub("d", "c"),
	}

	desc := detectCycle(challenges)
	assert.NotEmpty(t, desc)
	assert.NotEqual(t, "unknown cycle", desc)
}

func TestGetDeps_WithDeps(t *testing.T) {
	a := newStub("a")
	b := newStub("b", "a", "c")
	c := newStub("c")

	challenges := map[challenge.ID]challenge.Challenge{
		"a": a,
		"b": b,
		"c": c,
	}

	deps := getDeps(challenges, "b")
	assert.Len(t, deps, 2)
	// Should be sorted
	assert.Equal(t, challenge.ID("a"), deps[0])
	assert.Equal(t, challenge.ID("c"), deps[1])
}

func TestTopologicalSort_WithExternalDeps(t *testing.T) {
	// Dependency on non-existent challenge
	challenges := map[challenge.ID]challenge.Challenge{
		"a": newStub("a", "external"),
	}

	// This should detect the cycle / missing dependency
	ordered, err := topologicalSort(challenges)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
	assert.Nil(t, ordered)
}

func TestDetectCycle_DisconnectedComponents(t *testing.T) {
	// Create a graph with multiple disconnected components
	// where DFS from one component already colors nodes
	// that are then skipped when starting from another component
	challenges := map[challenge.ID]challenge.Challenge{
		"a": newStub("a"),
		"b": newStub("b"),
		"c": newStub("c"),
		"d": newStub("d", "c"), // d depends on c
	}

	// No cycle - should return "unknown cycle" since this function
	// is called when topological sort already detected an issue
	desc := detectCycle(challenges)
	assert.Equal(t, "unknown cycle", desc)
}

func TestDetectCycle_SkipAlreadyVisited(t *testing.T) {
	// Create a graph where node "b" is visited via "a"
	// and then encountered again when iterating sorted IDs
	// Key: IDs are sorted, so "a" is processed first,
	// which visits "b" through dependencies, then "b" is
	// skipped when we reach it in the main loop.
	challenges := map[challenge.ID]challenge.Challenge{
		"a": newStub("a", "b"), // a depends on b
		"b": newStub("b"),      // b is independent
	}

	// No cycle - when we iterate sorted IDs:
	// 1. Process "a": colors "a" gray, visits "b" (colors gray then black)
	// 2. Process "b": already black (visited), so continue
	desc := detectCycle(challenges)
	assert.Equal(t, "unknown cycle", desc)
}
