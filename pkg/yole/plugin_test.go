package yole

import (
	"testing"

	"digital.vasic.challenges/pkg/assertion"
	"digital.vasic.challenges/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYolePlugin_Interface(t *testing.T) {
	engine := assertion.NewEngine()
	p := NewYolePlugin(engine)

	// Verify it implements plugin.Plugin.
	var _ plugin.Plugin = p

	assert.Equal(t, PluginName, p.Name())
	assert.Equal(t, PluginVersion, p.Version())
}

func TestYolePlugin_Init(t *testing.T) {
	engine := assertion.NewEngine()
	p := NewYolePlugin(engine)

	err := p.Init(&plugin.PluginContext{})
	require.NoError(t, err)

	// Verify evaluators are registered.
	assert.True(t, engine.HasEvaluator("build_succeeds"))
	assert.True(t, engine.HasEvaluator("all_tests_pass"))
	assert.True(t, engine.HasEvaluator("lint_passes"))
	assert.True(t, engine.HasEvaluator("app_launches"))
	assert.True(t, engine.HasEvaluator("app_stable"))
	assert.True(t, engine.HasEvaluator("format_renders"))
	assert.True(t, engine.HasEvaluator("test_count_above"))
	assert.True(t, engine.HasEvaluator("no_test_failures"))
}

func TestYolePlugin_Init_NilEngine(t *testing.T) {
	p := NewYolePlugin(nil)

	err := p.Init(&plugin.PluginContext{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestYolePlugin_Init_Idempotent(t *testing.T) {
	engine := assertion.NewEngine()
	p := NewYolePlugin(engine)

	// First init succeeds.
	err := p.Init(&plugin.PluginContext{})
	require.NoError(t, err)

	// Second init fails because evaluators already registered.
	err = p.Init(&plugin.PluginContext{})
	assert.Error(t, err)
}

func TestYolePlugin_Registration(t *testing.T) {
	engine := assertion.NewEngine()
	p := NewYolePlugin(engine)

	// Register with plugin registry.
	reg := plugin.NewRegistry()
	err := reg.Register(p)
	require.NoError(t, err)

	assert.Equal(t, 1, reg.Count())

	found, ok := reg.Get(PluginName)
	assert.True(t, ok)
	assert.Equal(t, PluginName, found.Name())

	// Initialize via registry.
	err = reg.Init(PluginName, &plugin.PluginContext{})
	require.NoError(t, err)
	assert.True(t, reg.IsLoaded(PluginName))
}

func TestYolePlugin_Constants(t *testing.T) {
	assert.Equal(t, "yole", PluginName)
	assert.Equal(t, "1.0.0", PluginVersion)
}
