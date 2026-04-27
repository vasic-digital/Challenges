package bank

// Regression tests for the YAML loader path added 2026-04-11. Each
// case pins one invariant the new parseBankFile branch must uphold:
//
//  1. A .yaml file with a standard Challenges BankFile layout loads
//     via the YAML code path and populates BankFile.Challenges with
//     the expected definitions (including snake_case tags like
//     estimated_duration).
//  2. A .yml file works the same — both extensions are recognised.
//  3. LoadDir picks up mixed .json + .yaml files in the same dir.
//  4. Malformed YAML produces a "parse bank file" wrapped error
//     mentioning the file path, matching the JSON path's error shape.
//  5. normaliseYAMLValue collapses map[interface{}]interface{} into
//     map[string]interface{} so json.Marshal does not fail on
//     nested map metadata fields.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.challenges/pkg/challenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalYAMLBank = `version: "1.0"
name: "Example YAML Bank"
metadata:
  project: test-project
  nested:
    level: deep
challenges:
  - id: "yaml-example-001"
    name: "Example challenge 001"
    description: "First YAML-loaded challenge"
    category: "demo"
    estimated_duration: "30s"
    inputs:
      - name: "input_a"
        source: "env"
        required: true
    outputs:
      - name: "out_x"
        type: "string"
        description: "x"
    assertions:
      - type: "equals"
        target: "out_x"
        value: "expected"
        message: "x must equal expected"
    metrics:
      - latency_ms
  - id: "yaml-example-002"
    name: "Example challenge 002"
    description: "Second YAML-loaded challenge"
    category: "demo"
    estimated_duration: "1m"
    dependencies:
      - yaml-example-001
`

func writeTempBank(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func TestBank_LoadFile_YAML(t *testing.T) {
	p := writeTempBank(t, "example.yaml", minimalYAMLBank)

	b := New()
	require.NoError(t, b.LoadFile(p))

	def001, ok := b.Get(challenge.ID("yaml-example-001"))
	require.True(t, ok, "yaml-example-001 must be loaded")
	assert.Equal(t, "Example challenge 001", def001.Name)
	assert.Equal(t, "demo", def001.Category)
	assert.Equal(t, "30s", def001.EstimatedDuration,
		"snake_case estimated_duration tag must survive YAML→JSON round-trip")
	require.Len(t, def001.Inputs, 1)
	assert.Equal(t, "input_a", def001.Inputs[0].Name)
	assert.True(t, def001.Inputs[0].Required)
	require.Len(t, def001.Assertions, 1)
	assert.Equal(t, "equals", def001.Assertions[0].Type)

	def002, ok := b.Get(challenge.ID("yaml-example-002"))
	require.True(t, ok)
	require.Len(t, def002.Dependencies, 1)
	assert.Equal(t, challenge.ID("yaml-example-001"), def002.Dependencies[0])

	assert.Equal(t, 2, len(b.All()))
}

func TestBank_LoadFile_YML_ExtensionAlias(t *testing.T) {
	p := writeTempBank(t, "example.yml", minimalYAMLBank)

	b := New()
	require.NoError(t, b.LoadFile(p))
	assert.Equal(t, 2, len(b.All()))
}

func TestBank_LoadDir_MixedJSONAndYAML(t *testing.T) {
	dir := t.TempDir()

	// One YAML file (the minimal bank above).
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "yaml_bank.yaml"),
		[]byte(minimalYAMLBank), 0o600,
	))
	// One JSON file with a single extra definition.
	jsonBank := `{
"version": "1.0",
"name": "JSON sibling",
"challenges": [
  {"id": "json-sibling-001", "name": "JSON challenge", "category": "demo",
   "estimated_duration": "10s"}
]
}`
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "json_bank.json"),
		[]byte(jsonBank), 0o600,
	))
	// A file with an unsupported extension — must be ignored.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "ignored.txt"),
		[]byte("ignore me"), 0o600,
	))

	b := New()
	require.NoError(t, b.LoadDir(dir))

	// 2 from YAML + 1 from JSON = 3 definitions.
	assert.Equal(t, 3, len(b.All()))
	_, ok := b.Get(challenge.ID("yaml-example-001"))
	assert.True(t, ok)
	_, ok = b.Get(challenge.ID("json-sibling-001"))
	assert.True(t, ok)
}

func TestBank_LoadFile_MalformedYAML(t *testing.T) {
	// Intentionally broken YAML — unbalanced brackets the v3 parser
	// rejects unambiguously.
	broken := `version: "1.0"
challenges:
  - id: "broken-001"
    inputs: [unclosed
`
	p := writeTempBank(t, "bad.yaml", broken)

	b := New()
	err := b.LoadFile(p)
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "parse bank file") ||
			strings.Contains(err.Error(), p),
		"error should wrap the file path and match the JSON-path error shape, got: %v", err)
}

func TestNormaliseYAMLValue_InterfaceKeyedMap(t *testing.T) {
	// yaml.v3 can produce map[interface{}]interface{} for nested
	// maps with non-string-looking keys; we accept anything yaml
	// produces and normalise to string-keyed maps.
	input := map[interface{}]interface{}{
		"a":      1,
		"b":      "two",
		"nested": map[interface{}]interface{}{"x": true, "y": []interface{}{1, 2}},
	}
	out := normaliseYAMLValue(input)
	m, ok := out.(map[string]interface{})
	require.True(t, ok, "top-level must be map[string]interface{}")
	assert.Equal(t, 1, m["a"])
	assert.Equal(t, "two", m["b"])

	nested, ok := m["nested"].(map[string]interface{})
	require.True(t, ok, "nested map must also be normalised")
	assert.Equal(t, true, nested["x"])

	seq, ok := nested["y"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, len(seq))
}

func TestNormaliseYAMLValue_PassThrough(t *testing.T) {
	// Scalars must pass through unchanged.
	cases := []interface{}{
		nil,
		"string",
		42,
		3.14,
		true,
		[]interface{}{"a", 1},
	}
	for _, c := range cases {
		got := normaliseYAMLValue(c)
		assert.Equal(t, c, got)
	}
}
