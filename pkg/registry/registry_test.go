package registry

import (
	"context"
	"testing"

	"digital.vasic.challenges/pkg/challenge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubChallenge is a minimal Challenge implementation for
// testing.
type stubChallenge struct {
	id       challenge.ID
	name     string
	desc     string
	category string
	deps     []challenge.ID
}

func (s *stubChallenge) ID() challenge.ID            { return s.id }
func (s *stubChallenge) Name() string                { return s.name }
func (s *stubChallenge) Description() string         { return s.desc }
func (s *stubChallenge) Category() string            { return s.category }
func (s *stubChallenge) Dependencies() []challenge.ID { return s.deps }

func (s *stubChallenge) Configure(
	_ *challenge.Config,
) error {
	return nil
}

func (s *stubChallenge) Validate(
	_ context.Context,
) error {
	return nil
}

func (s *stubChallenge) Execute(
	_ context.Context,
) (*challenge.Result, error) {
	return &challenge.Result{Status: challenge.StatusPassed}, nil
}

func (s *stubChallenge) Cleanup(_ context.Context) error {
	return nil
}

func newStub(
	id string, deps ...string,
) *stubChallenge {
	depIDs := make([]challenge.ID, len(deps))
	for i, d := range deps {
		depIDs[i] = challenge.ID(d)
	}
	return &stubChallenge{
		id:   challenge.ID(id),
		name: id,
		desc: "stub " + id,
		deps: depIDs,
	}
}

func TestDefaultRegistry_Register_Success(t *testing.T) {
	r := NewRegistry()
	err := r.Register(newStub("a"))
	require.NoError(t, err)
	assert.Equal(t, 1, r.Count())
}

func TestDefaultRegistry_Register_Duplicate(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("a")))

	err := r.Register(newStub("a"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestDefaultRegistry_Get_Found(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("x")))

	c, err := r.Get("x")
	require.NoError(t, err)
	assert.Equal(t, challenge.ID("x"), c.ID())
}

func TestDefaultRegistry_Get_NotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDefaultRegistry_RegisterDefinition(t *testing.T) {
	r := NewRegistry()
	def := &challenge.Definition{
		ID:       "def1",
		Name:     "Def 1",
		Category: "core",
	}

	require.NoError(t, r.RegisterDefinition(def))

	got, err := r.GetDefinition("def1")
	require.NoError(t, err)
	assert.Equal(t, "Def 1", got.Name)
}

func TestDefaultRegistry_RegisterDefinition_Dup(t *testing.T) {
	r := NewRegistry()
	def := &challenge.Definition{ID: "d1"}
	require.NoError(t, r.RegisterDefinition(def))

	err := r.RegisterDefinition(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestDefaultRegistry_List_Sorted(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("c")))
	require.NoError(t, r.Register(newStub("a")))
	require.NoError(t, r.Register(newStub("b")))

	list := r.List()
	require.Len(t, list, 3)
	assert.Equal(t, challenge.ID("a"), list[0].ID())
	assert.Equal(t, challenge.ID("b"), list[1].ID())
	assert.Equal(t, challenge.ID("c"), list[2].ID())
}

func TestDefaultRegistry_ListDefinitions_Sorted(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.RegisterDefinition(
		&challenge.Definition{ID: "z"},
	))
	require.NoError(t, r.RegisterDefinition(
		&challenge.Definition{ID: "a"},
	))

	defs := r.ListDefinitions()
	require.Len(t, defs, 2)
	assert.Equal(t, challenge.ID("a"), defs[0].ID)
	assert.Equal(t, challenge.ID("z"), defs[1].ID)
}

func TestDefaultRegistry_ListByCategory(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("a")))
	require.NoError(t, r.Register(newStub("b")))
	require.NoError(t, r.RegisterDefinition(
		&challenge.Definition{ID: "a", Category: "core"},
	))
	require.NoError(t, r.RegisterDefinition(
		&challenge.Definition{ID: "b", Category: "e2e"},
	))

	core := r.ListByCategory("core")
	require.Len(t, core, 1)
	assert.Equal(t, challenge.ID("a"), core[0].ID())

	e2e := r.ListByCategory("e2e")
	require.Len(t, e2e, 1)
	assert.Equal(t, challenge.ID("b"), e2e[0].ID())

	none := r.ListByCategory("missing")
	assert.Empty(t, none)
}

func TestDefaultRegistry_ValidateDependencies_OK(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("a")))
	require.NoError(t, r.Register(newStub("b", "a")))

	assert.NoError(t, r.ValidateDependencies())
}

func TestDefaultRegistry_ValidateDependencies_Missing(
	t *testing.T,
) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("b", "missing")))

	err := r.ValidateDependencies()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unregistered dependency")
}

func TestDefaultRegistry_Clear(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(newStub("a")))
	require.NoError(t, r.RegisterDefinition(
		&challenge.Definition{ID: "a"},
	))

	r.Clear()
	assert.Equal(t, 0, r.Count())
	assert.Empty(t, r.ListDefinitions())
}

func TestDefaultRegistry_Count(t *testing.T) {
	r := NewRegistry()
	assert.Equal(t, 0, r.Count())

	require.NoError(t, r.Register(newStub("a")))
	assert.Equal(t, 1, r.Count())

	require.NoError(t, r.Register(newStub("b")))
	assert.Equal(t, 2, r.Count())
}

func TestDefaultPackageLevelInstance(t *testing.T) {
	// Default should be a valid registry instance.
	assert.NotNil(t, Default)
	assert.Equal(t, 0, Default.Count())
}
