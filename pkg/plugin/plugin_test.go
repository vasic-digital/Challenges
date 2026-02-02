package plugin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPlugin struct {
	name    string
	version string
	initErr error
	inited  bool
}

func (m *mockPlugin) Name() string    { return m.name }
func (m *mockPlugin) Version() string { return m.version }
func (m *mockPlugin) Init(ctx *PluginContext) error {
	if m.initErr != nil {
		return m.initErr
	}
	m.inited = true
	return nil
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	err := r.Register(&mockPlugin{name: "test", version: "1.0"})
	assert.NoError(t, err)
	assert.Equal(t, 1, r.Count())

	// Duplicate
	err = r.Register(&mockPlugin{name: "test", version: "1.0"})
	assert.Error(t, err)

	// Nil plugin
	err = r.Register(nil)
	assert.Error(t, err)

	// Empty name
	err = r.Register(&mockPlugin{name: "", version: "1.0"})
	assert.Error(t, err)
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockPlugin{name: "test", version: "1.0"})

	p, ok := r.Get("test")
	assert.True(t, ok)
	assert.Equal(t, "test", p.Name())

	_, ok = r.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_InitAll(t *testing.T) {
	r := NewRegistry()
	p1 := &mockPlugin{name: "p1", version: "1.0"}
	p2 := &mockPlugin{name: "p2", version: "1.0"}
	r.Register(p1)
	r.Register(p2)

	err := r.InitAll(&PluginContext{})
	assert.NoError(t, err)
	assert.True(t, p1.inited)
	assert.True(t, p2.inited)
	assert.True(t, r.IsLoaded("p1"))
	assert.True(t, r.IsLoaded("p2"))
}

func TestRegistry_InitAll_Error(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockPlugin{name: "bad", version: "1.0", initErr: fmt.Errorf("init failed")})

	err := r.InitAll(&PluginContext{})
	assert.Error(t, err)
}

func TestRegistry_Init_AlreadyLoaded(t *testing.T) {
	r := NewRegistry()
	p := &mockPlugin{name: "test", version: "1.0"}
	r.Register(p)
	r.InitAll(&PluginContext{})

	// Second init should be no-op
	err := r.Init("test", &PluginContext{})
	assert.NoError(t, err)
}

func TestRegistry_Init_NotFound(t *testing.T) {
	r := NewRegistry()
	err := r.Init("nonexistent", &PluginContext{})
	assert.Error(t, err)
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockPlugin{name: "a", version: "1.0"})
	r.Register(&mockPlugin{name: "b", version: "1.0"})

	names := r.List()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "a")
	assert.Contains(t, names, "b")
}

// Suppress unused import warning for require
var _ = require.NoError
