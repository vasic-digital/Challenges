package panoptic

import (
	"fmt"

	"digital.vasic.challenges/pkg/assertion"
	"digital.vasic.challenges/pkg/plugin"
)

const (
	// PluginName is the canonical name for the Panoptic plugin.
	PluginName = "panoptic"
	// PluginVersion is the current version of the plugin.
	PluginVersion = "1.0.0"
)

// PanopticPlugin implements plugin.Plugin and registers all
// Panoptic-specific assertion evaluators with the framework.
type PanopticPlugin struct {
	engine *assertion.DefaultEngine
}

// NewPanopticPlugin creates a PanopticPlugin that will register
// evaluators with the given assertion engine.
func NewPanopticPlugin(
	engine *assertion.DefaultEngine,
) *PanopticPlugin {
	return &PanopticPlugin{engine: engine}
}

// Name returns the plugin name.
func (p *PanopticPlugin) Name() string {
	return PluginName
}

// Version returns the plugin version.
func (p *PanopticPlugin) Version() string {
	return PluginVersion
}

// Init registers all 8 Panoptic assertion evaluators with the
// assertion engine. The PluginContext config is not used.
func (p *PanopticPlugin) Init(
	_ *plugin.PluginContext,
) error {
	if p.engine == nil {
		return fmt.Errorf(
			"panoptic plugin: assertion engine is nil",
		)
	}
	return RegisterEvaluators(p.engine)
}
