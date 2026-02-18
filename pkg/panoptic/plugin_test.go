package panoptic

import (
	"testing"

	"digital.vasic.challenges/pkg/assertion"
	"digital.vasic.challenges/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanopticPlugin_Interface(t *testing.T) {
	engine := assertion.NewEngine()
	p := NewPanopticPlugin(engine)

	// Verify it implements plugin.Plugin.
	var _ plugin.Plugin = p

	assert.Equal(t, PluginName, p.Name())
	assert.Equal(t, PluginVersion, p.Version())
}

func TestPanopticPlugin_Init(t *testing.T) {
	engine := assertion.NewEngine()
	p := NewPanopticPlugin(engine)

	err := p.Init(&plugin.PluginContext{})
	require.NoError(t, err)

	// Verify evaluators are registered.
	assert.True(t, engine.HasEvaluator("screenshot_exists"))
	assert.True(t, engine.HasEvaluator("video_exists"))
	assert.True(t, engine.HasEvaluator("no_ui_errors"))
	assert.True(t, engine.HasEvaluator("ai_confidence_above"))
	assert.True(t, engine.HasEvaluator("all_apps_passed"))
	assert.True(t, engine.HasEvaluator("max_duration"))
	assert.True(t, engine.HasEvaluator("report_exists"))
	assert.True(t, engine.HasEvaluator("app_count"))
}

func TestPanopticPlugin_Init_NilEngine(t *testing.T) {
	p := NewPanopticPlugin(nil)

	err := p.Init(&plugin.PluginContext{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestPanopticPlugin_Init_Idempotent(t *testing.T) {
	engine := assertion.NewEngine()
	p := NewPanopticPlugin(engine)

	// First init succeeds.
	err := p.Init(&plugin.PluginContext{})
	require.NoError(t, err)

	// Second init fails because evaluators already registered.
	err = p.Init(&plugin.PluginContext{})
	assert.Error(t, err)
}

func TestPanopticPlugin_Registration(t *testing.T) {
	engine := assertion.NewEngine()
	p := NewPanopticPlugin(engine)

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
