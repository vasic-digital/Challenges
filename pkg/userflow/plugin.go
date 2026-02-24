package userflow

import (
	"fmt"

	"digital.vasic.challenges/pkg/assertion"
	"digital.vasic.challenges/pkg/plugin"
)

const (
	// PluginName is the unique name of the userflow plugin.
	PluginName = "userflow"
	// PluginVersion is the version of the userflow plugin.
	PluginVersion = "2.0.0"
)

// UserFlowPlugin implements plugin.Plugin for the userflow
// testing framework. It registers assertion evaluators for
// multi-platform user flow testing.
type UserFlowPlugin struct {
	engine *assertion.DefaultEngine
}

// Ensure UserFlowPlugin satisfies the plugin.Plugin interface.
var _ plugin.Plugin = (*UserFlowPlugin)(nil)

// Name returns the plugin's unique name.
func (p *UserFlowPlugin) Name() string {
	return PluginName
}

// Version returns the plugin's version string.
func (p *UserFlowPlugin) Version() string {
	return PluginVersion
}

// Init initializes the plugin by registering all userflow
// evaluators with the assertion engine. The engine must be
// provided in ctx.Config["assertion_engine"].
func (p *UserFlowPlugin) Init(
	ctx *plugin.PluginContext,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"userflow plugin: context must not be nil",
		)
	}

	engineVal, ok := ctx.Config["assertion_engine"]
	if !ok {
		return fmt.Errorf(
			"userflow plugin: assertion_engine not found " +
				"in plugin context",
		)
	}

	engine, ok := engineVal.(*assertion.DefaultEngine)
	if !ok {
		return fmt.Errorf(
			"userflow plugin: assertion_engine is not " +
				"*assertion.DefaultEngine",
		)
	}

	p.engine = engine
	return RegisterEvaluators(engine)
}
