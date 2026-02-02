package plugin

import (
	"fmt"
)

// Loader handles plugin discovery and loading.
type Loader struct {
	registry *Registry
}

// NewLoader creates a new plugin loader.
func NewLoader(registry *Registry) *Loader {
	return &Loader{registry: registry}
}

// LoadAndInit registers and initializes a set of plugins.
func (l *Loader) LoadAndInit(plugins []Plugin, ctx *PluginContext) error {
	for _, p := range plugins {
		if err := l.registry.Register(p); err != nil {
			return fmt.Errorf("load plugin: %w", err)
		}
	}
	return l.registry.InitAll(ctx)
}

// LoadOne registers and initializes a single plugin.
func (l *Loader) LoadOne(p Plugin, ctx *PluginContext) error {
	if err := l.registry.Register(p); err != nil {
		return fmt.Errorf("load plugin: %w", err)
	}
	return l.registry.Init(p.Name(), ctx)
}
