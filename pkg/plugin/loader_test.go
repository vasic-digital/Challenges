package plugin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_LoadAndInit(t *testing.T) {
	r := NewRegistry()
	l := NewLoader(r)

	plugins := []Plugin{
		&mockPlugin{name: "p1", version: "1.0"},
		&mockPlugin{name: "p2", version: "2.0"},
	}

	err := l.LoadAndInit(plugins, &PluginContext{})
	assert.NoError(t, err)
	assert.Equal(t, 2, r.Count())
	assert.True(t, r.IsLoaded("p1"))
	assert.True(t, r.IsLoaded("p2"))
}

func TestLoader_LoadOne(t *testing.T) {
	r := NewRegistry()
	l := NewLoader(r)

	err := l.LoadOne(&mockPlugin{name: "single", version: "1.0"}, &PluginContext{})
	assert.NoError(t, err)
	assert.True(t, r.IsLoaded("single"))
}

func TestLoader_LoadAndInit_DuplicateError(t *testing.T) {
	r := NewRegistry()
	l := NewLoader(r)

	plugins := []Plugin{
		&mockPlugin{name: "same", version: "1.0"},
		&mockPlugin{name: "same", version: "2.0"},
	}

	err := l.LoadAndInit(plugins, &PluginContext{})
	assert.Error(t, err)
}

func TestLoader_LoadOne_RegisterError(t *testing.T) {
	r := NewRegistry()
	l := NewLoader(r)

	// Register a plugin first
	require.NoError(t, l.LoadOne(&mockPlugin{name: "first", version: "1.0"}, &PluginContext{}))

	// Try to register the same plugin again
	err := l.LoadOne(&mockPlugin{name: "first", version: "2.0"}, &PluginContext{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load plugin")
}

func TestLoader_LoadOne_InitError(t *testing.T) {
	r := NewRegistry()
	l := NewLoader(r)

	err := l.LoadOne(&mockPlugin{
		name:    "fail-init",
		version: "1.0",
		initErr: fmt.Errorf("init failed"),
	}, &PluginContext{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "init plugin")
}
