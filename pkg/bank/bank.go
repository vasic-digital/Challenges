package bank

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"digital.vasic.challenges/pkg/challenge"
)

// Bank manages collections of challenge definitions loaded from files.
type Bank struct {
	mu          sync.RWMutex
	definitions map[challenge.ID]*challenge.Definition
	sources     []string
}

// New creates a new empty Bank.
func New() *Bank {
	return &Bank{
		definitions: make(map[challenge.ID]*challenge.Definition),
	}
}

// LoadFile loads challenge definitions from a JSON file.
func (b *Bank) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read bank file %s: %w", path, err)
	}

	var file BankFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse bank file %s: %w", path, err)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range file.Challenges {
		def := &file.Challenges[i]
		if def.ID == "" {
			return fmt.Errorf("challenge at index %d in %s has no ID", i, path)
		}
		b.definitions[def.ID] = def
	}
	b.sources = append(b.sources, path)
	return nil
}

// LoadDir loads all .json files from a directory.
func (b *Bank) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read bank directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".json" {
			continue
		}
		if err := b.LoadFile(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves a challenge definition by ID.
func (b *Bank) Get(id challenge.ID) (*challenge.Definition, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	def, ok := b.definitions[id]
	return def, ok
}

// All returns all loaded definitions.
func (b *Bank) All() []*challenge.Definition {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]*challenge.Definition, 0, len(b.definitions))
	for _, def := range b.definitions {
		result = append(result, def)
	}
	return result
}

// ByCategory returns definitions filtered by category.
func (b *Bank) ByCategory(category string) []*challenge.Definition {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var result []*challenge.Definition
	for _, def := range b.definitions {
		if def.Category == category {
			result = append(result, def)
		}
	}
	return result
}

// Count returns the number of loaded definitions.
func (b *Bank) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.definitions)
}

// Sources returns the list of loaded file paths.
func (b *Bank) Sources() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]string, len(b.sources))
	copy(result, b.sources)
	return result
}
