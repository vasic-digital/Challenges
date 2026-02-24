package yole

import (
	"fmt"

	"digital.vasic.challenges/pkg/assertion"
	"digital.vasic.challenges/pkg/plugin"
)

const (
	// PluginName is the canonical name for the Yole plugin.
	PluginName = "yole"
	// PluginVersion is the current version of the plugin.
	PluginVersion = "1.0.0"
)

// YolePlugin implements plugin.Plugin and registers all
// Yole-specific assertion evaluators with the framework.
type YolePlugin struct {
	engine *assertion.DefaultEngine
}

// NewYolePlugin creates a YolePlugin that will register
// evaluators with the given assertion engine.
func NewYolePlugin(
	engine *assertion.DefaultEngine,
) *YolePlugin {
	return &YolePlugin{engine: engine}
}

// Name returns the plugin name.
func (p *YolePlugin) Name() string {
	return PluginName
}

// Version returns the plugin version.
func (p *YolePlugin) Version() string {
	return PluginVersion
}

// Init registers all Yole assertion evaluators with the
// assertion engine. The PluginContext config is not used.
func (p *YolePlugin) Init(
	_ *plugin.PluginContext,
) error {
	if p.engine == nil {
		return fmt.Errorf(
			"yole plugin: assertion engine is nil",
		)
	}
	return RegisterEvaluators(p.engine)
}
