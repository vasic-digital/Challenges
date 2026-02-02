// Package registry provides challenge registration, discovery,
// and dependency-ordered retrieval.
package registry

import (
	"fmt"
	"sort"
	"sync"

	"digital.vasic.challenges/pkg/challenge"
)

// Registry defines the interface for managing challenges and
// their definitions.
type Registry interface {
	// Register adds a challenge implementation.
	Register(c challenge.Challenge) error

	// RegisterDefinition adds a declarative definition.
	RegisterDefinition(def *challenge.Definition) error

	// Get retrieves a challenge by ID.
	Get(id challenge.ID) (challenge.Challenge, error)

	// GetDefinition retrieves a definition by ID.
	GetDefinition(
		id challenge.ID,
	) (*challenge.Definition, error)

	// List returns all registered challenges sorted by ID.
	List() []challenge.Challenge

	// ListDefinitions returns all registered definitions
	// sorted by ID.
	ListDefinitions() []*challenge.Definition

	// ListByCategory returns challenges whose definition
	// matches the given category.
	ListByCategory(category string) []challenge.Challenge

	// GetDependencyOrder returns challenges in topological
	// (dependency) order.
	GetDependencyOrder() ([]challenge.Challenge, error)

	// ValidateDependencies checks that every dependency
	// referenced by a challenge is also registered.
	ValidateDependencies() error

	// Clear removes all challenges and definitions.
	Clear()

	// Count returns the number of registered challenges.
	Count() int
}

// DefaultRegistry is the standard Registry implementation.
// It is safe for concurrent use.
type DefaultRegistry struct {
	mu          sync.RWMutex
	challenges  map[challenge.ID]challenge.Challenge
	definitions map[challenge.ID]*challenge.Definition
}

// NewRegistry creates a new, empty DefaultRegistry.
func NewRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		challenges:  make(map[challenge.ID]challenge.Challenge),
		definitions: make(map[challenge.ID]*challenge.Definition),
	}
}

// Default is the package-level default registry instance.
var Default = NewRegistry()

// Register adds a challenge to the registry. Returns an error
// if a challenge with the same ID is already registered.
func (r *DefaultRegistry) Register(
	c challenge.Challenge,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := c.ID()
	if _, exists := r.challenges[id]; exists {
		return fmt.Errorf(
			"challenge already registered: %s", id,
		)
	}

	r.challenges[id] = c
	return nil
}

// RegisterDefinition adds a declarative challenge definition.
// Returns an error if a definition with the same ID already
// exists.
func (r *DefaultRegistry) RegisterDefinition(
	def *challenge.Definition,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.definitions[def.ID]; exists {
		return fmt.Errorf(
			"challenge definition already registered: %s",
			def.ID,
		)
	}

	r.definitions[def.ID] = def
	return nil
}

// Get retrieves a challenge by ID.
func (r *DefaultRegistry) Get(
	id challenge.ID,
) (challenge.Challenge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, exists := r.challenges[id]
	if !exists {
		return nil, fmt.Errorf(
			"challenge not found: %s", id,
		)
	}
	return c, nil
}

// GetDefinition retrieves a definition by ID.
func (r *DefaultRegistry) GetDefinition(
	id challenge.ID,
) (*challenge.Definition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, exists := r.definitions[id]
	if !exists {
		return nil, fmt.Errorf(
			"challenge definition not found: %s", id,
		)
	}
	return def, nil
}

// List returns all registered challenges sorted by ID.
func (r *DefaultRegistry) List() []challenge.Challenge {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(
		[]challenge.Challenge, 0, len(r.challenges),
	)
	for _, c := range r.challenges {
		out = append(out, c)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID() < out[j].ID()
	})
	return out
}

// ListDefinitions returns all registered definitions sorted
// by ID.
func (r *DefaultRegistry) ListDefinitions() []*challenge.Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(
		[]*challenge.Definition, 0, len(r.definitions),
	)
	for _, d := range r.definitions {
		out = append(out, d)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

// ListByCategory returns challenges whose definition matches
// the given category. Challenges without a corresponding
// definition are excluded.
func (r *DefaultRegistry) ListByCategory(
	category string,
) []challenge.Challenge {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []challenge.Challenge
	for id, c := range r.challenges {
		if def, ok := r.definitions[id]; ok {
			if def.Category == category {
				out = append(out, c)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID() < out[j].ID()
	})
	return out
}

// GetDependencyOrder returns challenges in topological order
// using Kahn's algorithm. Returns an error if a dependency
// cycle is detected.
func (r *DefaultRegistry) GetDependencyOrder() (
	[]challenge.Challenge, error,
) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return topologicalSort(r.challenges)
}

// ValidateDependencies checks that every dependency referenced
// by a registered challenge is also registered. Returns the
// first missing dependency found.
func (r *DefaultRegistry) ValidateDependencies() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for id, c := range r.challenges {
		for _, dep := range c.Dependencies() {
			if _, exists := r.challenges[dep]; !exists {
				return fmt.Errorf(
					"challenge %s has unregistered "+
						"dependency: %s",
					id, dep,
				)
			}
		}
	}
	return nil
}

// Clear removes all challenges and definitions.
func (r *DefaultRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.challenges = make(
		map[challenge.ID]challenge.Challenge,
	)
	r.definitions = make(
		map[challenge.ID]*challenge.Definition,
	)
}

// Count returns the number of registered challenges.
func (r *DefaultRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.challenges)
}
