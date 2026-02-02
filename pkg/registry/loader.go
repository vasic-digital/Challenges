package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"digital.vasic.challenges/pkg/challenge"
)

// bankFile is the on-disk structure for a challenge definition
// bank (JSON or YAML).
type bankFile struct {
	Version    string                 `json:"version"`
	Challenges []challenge.Definition `json:"challenges"`
}

// LoadDefinitionsFromFile reads a JSON file containing a bank
// of challenge definitions and registers each one into the
// given registry. YAML support uses the same struct tags
// because gopkg.in/yaml.v3 honours json tags.
func LoadDefinitionsFromFile(
	reg Registry,
	path string,
) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf(
			"failed to read definitions file %s: %w",
			path, err,
		)
	}

	return loadDefinitionsFromBytes(reg, data, path)
}

// LoadDefinitionsFromDir loads all .json and .yaml/.yml
// definition bank files from a directory. It does not recurse
// into subdirectories.
func LoadDefinitionsFromDir(
	reg Registry,
	dir string,
) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf(
			"failed to read directory %s: %w", dir, err,
		)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			continue
		}

		p := filepath.Join(dir, entry.Name())
		if err := LoadDefinitionsFromFile(reg, p); err != nil {
			return fmt.Errorf(
				"failed to load %s: %w", p, err,
			)
		}
	}

	return nil
}

// loadDefinitionsFromBytes unmarshals a bank file and
// registers its definitions.
func loadDefinitionsFromBytes(
	reg Registry,
	data []byte,
	source string,
) error {
	var bank bankFile
	if err := json.Unmarshal(data, &bank); err != nil {
		return fmt.Errorf(
			"failed to parse definitions from %s: %w",
			source, err,
		)
	}

	for i := range bank.Challenges {
		def := &bank.Challenges[i]
		if err := reg.RegisterDefinition(def); err != nil {
			return fmt.Errorf(
				"definition %s from %s: %w",
				def.ID, source, err,
			)
		}
	}

	return nil
}
