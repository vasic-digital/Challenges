package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
