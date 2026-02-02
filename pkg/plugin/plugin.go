package plugin

import (
	"fmt"
	"sync"
)

// Plugin defines the interface for extending the challenge framework.
type Plugin interface {
	// Name returns the plugin's unique name.
	Name() string
	// Version returns the plugin's version string.
	Version() string
	// Init initializes the plugin with the given context.
	Init(ctx *PluginContext) error
}

// PluginContext provides access to framework components during initialization.
type PluginContext struct {
	Config map[string]interface{}
}

// Registry manages plugin registration and initialization.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	loaded  map[string]bool
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
		loaded:  make(map[string]bool),
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(p Plugin) error {
	if p == nil {
		return fmt.Errorf("plugin cannot be nil")
	}
	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}

	r.plugins[name] = p
	return nil
}

// Get retrieves a registered plugin by name.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	return p, ok
}

// InitAll initializes all registered plugins that haven't been loaded yet.
func (r *Registry) InitAll(ctx *PluginContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, p := range r.plugins {
		if r.loaded[name] {
			continue
		}
		if err := p.Init(ctx); err != nil {
			return fmt.Errorf("init plugin %q: %w", name, err)
		}
		r.loaded[name] = true
	}
	return nil
}

// Init initializes a specific plugin by name.
func (r *Registry) Init(name string, ctx *PluginContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}
	if r.loaded[name] {
		return nil
	}
	if err := p.Init(ctx); err != nil {
		return fmt.Errorf("init plugin %q: %w", name, err)
	}
	r.loaded[name] = true
	return nil
}

// List returns all registered plugin names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// IsLoaded checks if a plugin has been initialized.
func (r *Registry) IsLoaded(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.loaded[name]
}

// Count returns the number of registered plugins.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.plugins)
}
