package userflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/assertion"
	"digital.vasic.challenges/pkg/plugin"
)

func TestUserFlowPlugin_Name(t *testing.T) {
	p := &UserFlowPlugin{}
	assert.Equal(t, "userflow", p.Name())
}

func TestUserFlowPlugin_Version(t *testing.T) {
	p := &UserFlowPlugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestUserFlowPlugin_Init_NilContext(t *testing.T) {
	p := &UserFlowPlugin{}
	err := p.Init(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context must not be nil")
}

func TestUserFlowPlugin_Init_MissingEngine(t *testing.T) {
	p := &UserFlowPlugin{}
	ctx := &plugin.PluginContext{
		Config: map[string]interface{}{},
	}
	err := p.Init(ctx)
	require.Error(t, err)
	assert.Contains(
		t, err.Error(), "assertion_engine not found",
	)
}

func TestUserFlowPlugin_Init_WrongType(t *testing.T) {
	p := &UserFlowPlugin{}
	ctx := &plugin.PluginContext{
		Config: map[string]interface{}{
			"assertion_engine": "not an engine",
		},
	}
	err := p.Init(ctx)
	require.Error(t, err)
	assert.Contains(
		t, err.Error(), "not *assertion.DefaultEngine",
	)
}

func TestUserFlowPlugin_Init_Success(t *testing.T) {
	engine := assertion.NewEngine()
	p := &UserFlowPlugin{}
	ctx := &plugin.PluginContext{
		Config: map[string]interface{}{
			"assertion_engine": engine,
		},
	}
	err := p.Init(ctx)
	require.NoError(t, err)

	// Verify evaluators were registered.
	assert.True(t, engine.HasEvaluator("build_succeeds"))
	assert.True(t, engine.HasEvaluator("all_tests_pass"))
	assert.True(t, engine.HasEvaluator("lint_passes"))
	assert.True(t, engine.HasEvaluator("app_launches"))
	assert.True(t, engine.HasEvaluator("app_stable"))
	assert.True(t, engine.HasEvaluator("status_code"))
	assert.True(t, engine.HasEvaluator("response_contains"))
	assert.True(t, engine.HasEvaluator("response_not_empty"))
	assert.True(t, engine.HasEvaluator("json_field_equals"))
	assert.True(t, engine.HasEvaluator("screenshot_exists"))
	assert.True(t, engine.HasEvaluator("flow_completes"))
	assert.True(t, engine.HasEvaluator("within_duration"))
}

func TestUserFlowPlugin_PluginInterface(t *testing.T) {
	var p plugin.Plugin = &UserFlowPlugin{}
	assert.NotNil(t, p)
	assert.Equal(t, "userflow", p.Name())
	assert.Equal(t, "1.0.0", p.Version())
}

func TestUserFlowPlugin_Registry(t *testing.T) {
	reg := plugin.NewRegistry()
	p := &UserFlowPlugin{}

	err := reg.Register(p)
	require.NoError(t, err)

	found, ok := reg.Get("userflow")
	assert.True(t, ok)
	assert.Equal(t, "userflow", found.Name())
}
