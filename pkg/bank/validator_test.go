package bank

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.challenges/pkg/challenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFile_Valid(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(BankFile{
		Version: "1.0",
		Challenges: []challenge.Definition{
			{ID: "ch-1", Name: "Test 1"},
			{ID: "ch-2", Name: "Test 2"},
		},
	})
	path := filepath.Join(dir, "valid.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	errors := ValidateFile(path)
	assert.Empty(t, errors)
}

func TestValidateFile_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(BankFile{
		Challenges: []challenge.Definition{
			{ID: "ch-1", Name: "Test"},
		},
	})
	path := filepath.Join(dir, "no_version.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	errors := ValidateFile(path)
	assert.Len(t, errors, 1)
	assert.Equal(t, "version", errors[0].Field)
}

func TestValidateFile_DuplicateIDs(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(BankFile{
		Version: "1.0",
		Challenges: []challenge.Definition{
			{ID: "ch-1", Name: "First"},
			{ID: "ch-1", Name: "Duplicate"},
		},
	})
	path := filepath.Join(dir, "dupes.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	errors := ValidateFile(path)
	assert.NotEmpty(t, errors)
}

func TestValidateFile_FileNotFound(t *testing.T) {
	errors := ValidateFile("/nonexistent/file.json")
	assert.Len(t, errors, 1)
	assert.Equal(t, "file", errors[0].Field)
}

func TestValidateFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0644))

	errors := ValidateFile(path)
	assert.Len(t, errors, 1)
	assert.Equal(t, "json", errors[0].Field)
}

func TestValidationError_Error(t *testing.T) {
	e1 := ValidationError{Field: "id", Message: "required", Index: 0}
	assert.Contains(t, e1.Error(), "challenges[0]")

	e2 := ValidationError{Field: "version", Message: "missing", Index: -1}
	assert.NotContains(t, e2.Error(), "challenges")
}
