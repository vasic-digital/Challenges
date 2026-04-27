package bank

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

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

// LoadFile loads challenge definitions from a JSON or YAML file. The
// parser is selected by file extension: .json uses encoding/json,
// .yaml and .yml use gopkg.in/yaml.v3. Any other extension is treated
// as JSON to preserve backwards compatibility with the original
// pre-2026-04-11 behaviour.
//
// YAML support was added to let HelixQA use its existing YAML test
// banks without maintaining a second parser path — see HelixQA's
// banks/*.yaml files and the CLAUDE.md note at pkg/bank which has
// documented "load definitions from JSON/YAML" as a requirement.
func (b *Bank) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read bank file %s: %w", path, err)
	}

	file, err := parseBankFile(path, data)
	if err != nil {
		return err
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

// parseBankFile routes a raw file byte slice through the correct
// parser based on the filename extension. Centralised so LoadFile and
// any future loader variants share the same format detection.
//
// YAML path: we parse into a generic interface, normalise any
// map[interface{}]interface{} values into map[string]interface{},
// then re-encode as JSON and hand to json.Unmarshal. This lets the
// YAML loader reuse the existing `json:"..."` struct tags on
// BankFile and challenge.Definition without duplicating them as yaml
// tags — the single source of truth for field naming stays in one
// place and snake_case tags like `estimated_duration` keep working.
func parseBankFile(path string, data []byte) (*BankFile, error) {
	var file BankFile
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		var raw interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse bank file %s: %w", path, err)
		}
		normalised := normaliseYAMLValue(raw)
		jsonBytes, err := json.Marshal(normalised)
		if err != nil {
			return nil, fmt.Errorf("marshal bank file %s: %w", path, err)
		}
		if err := json.Unmarshal(jsonBytes, &file); err != nil {
			return nil, fmt.Errorf("parse bank file %s: %w", path, err)
		}
	default:
		if err := json.Unmarshal(data, &file); err != nil {
			return nil, fmt.Errorf("parse bank file %s: %w", path, err)
		}
	}
	// HelixQA banks use "test_cases" as the root key — fold those
	// into Challenges so every caller only ever reads one slice.
	if len(file.Challenges) == 0 && len(file.TestCases) > 0 {
		file.Challenges = file.TestCases
	}
	return &file, nil
}

// normaliseYAMLValue recursively walks a value produced by
// yaml.Unmarshal and converts every map[interface{}]interface{} into
// map[string]interface{}. yaml.v3 can return interface-keyed maps
// which json.Marshal refuses to encode; normalising here is the
// minimal-surface fix. Non-map types (slices, strings, numbers,
// booleans, nil) pass through unchanged after recursing into any
// contained elements.
func normaliseYAMLValue(v interface{}) interface{} {
	switch x := v.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			out[fmt.Sprintf("%v", k)] = normaliseYAMLValue(val)
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			out[k] = normaliseYAMLValue(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, val := range x {
			out[i] = normaliseYAMLValue(val)
		}
		return out
	default:
		return v
	}
}

// LoadDir loads all .json, .yaml, and .yml files from a directory.
// Other extensions are ignored. Subdirectories are not recursed into;
// callers that need recursive loading should compose their own
// walker on top of LoadFile.
func (b *Bank) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read bank directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
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
