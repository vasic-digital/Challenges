package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefinitionsFromFile_JSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bank.json")

	content := `{
		"version": "1.0",
		"challenges": [
			{
				"id": "test-1",
				"name": "Test One",
				"description": "First test",
				"category": "core",
				"dependencies": [],
				"assertions": [],
				"metrics": []
			},
			{
				"id": "test-2",
				"name": "Test Two",
				"description": "Second test",
				"category": "e2e",
				"dependencies": ["test-1"],
				"assertions": [
					{
						"type": "not_empty",
						"target": "response",
						"message": "response must not be empty"
					}
				],
				"metrics": ["latency"]
			}
		]
	}`

	require.NoError(t, os.WriteFile(p, []byte(content), 0644))

	r := NewRegistry()
	require.NoError(t, LoadDefinitionsFromFile(r, p))

	defs := r.ListDefinitions()
	require.Len(t, defs, 2)
	assert.Equal(t, "Test One", defs[0].Name)
	assert.Equal(t, "Test Two", defs[1].Name)
	assert.Len(t, defs[1].Assertions, 1)
}

func TestLoadDefinitionsFromFile_NotFound(t *testing.T) {
	r := NewRegistry()
	err := LoadDefinitionsFromFile(r, "/nonexistent.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestLoadDefinitionsFromFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(p, []byte("{bad"), 0644))

	r := NewRegistry()
	err := LoadDefinitionsFromFile(r, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestLoadDefinitionsFromFile_DuplicateID(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "dup.json")

	content := `{
		"version": "1.0",
		"challenges": [
			{"id": "same", "name": "A"},
			{"id": "same", "name": "B"}
		]
	}`
	require.NoError(t, os.WriteFile(p, []byte(content), 0644))

	r := NewRegistry()
	err := LoadDefinitionsFromFile(r, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestLoadDefinitionsFromDir(t *testing.T) {
	dir := t.TempDir()

	f1 := `{
		"version":"1.0",
		"challenges":[{"id":"a","name":"A"}]
	}`
	f2 := `{
		"version":"1.0",
		"challenges":[{"id":"b","name":"B"}]
	}`
	// This file should be ignored (wrong extension).
	f3 := "not a challenge"

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "a.json"), []byte(f1), 0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "b.json"), []byte(f2), 0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "readme.txt"), []byte(f3), 0644,
	))

	r := NewRegistry()
	require.NoError(t, LoadDefinitionsFromDir(r, dir))

	defs := r.ListDefinitions()
	require.Len(t, defs, 2)
}

func TestLoadDefinitionsFromDir_NotFound(t *testing.T) {
	r := NewRegistry()
	err := LoadDefinitionsFromDir(r, "/nonexistent_dir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read directory")
}

func TestLoadDefinitionsFromDir_BadFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "bad.json"),
		[]byte("{invalid}"),
		0644,
	))

	r := NewRegistry()
	err := LoadDefinitionsFromDir(r, dir)
	require.Error(t, err)
}
