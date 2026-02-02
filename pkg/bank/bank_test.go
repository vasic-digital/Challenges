package bank

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.challenges/pkg/challenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestBankFile(t *testing.T, dir string, file BankFile) string {
	t.Helper()
	data, err := json.Marshal(file)
	require.NoError(t, err)
	path := filepath.Join(dir, "test_bank.json")
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}

func TestBank_LoadFile(t *testing.T) {
	dir := t.TempDir()
	path := createTestBankFile(t, dir, BankFile{
		Version: "1.0",
		Name:    "Test Bank",
		Challenges: []challenge.Definition{
			{ID: "ch-1", Name: "Challenge 1", Category: "test"},
			{ID: "ch-2", Name: "Challenge 2", Category: "test"},
		},
	})

	b := New()
	require.NoError(t, b.LoadFile(path))
	assert.Equal(t, 2, b.Count())

	def, ok := b.Get("ch-1")
	assert.True(t, ok)
	assert.Equal(t, "Challenge 1", def.Name)
}

func TestBank_LoadFile_NotFound(t *testing.T) {
	b := New()
	err := b.LoadFile("/nonexistent/bank.json")
	assert.Error(t, err)
}

func TestBank_LoadFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid"), 0644))

	b := New()
	err := b.LoadFile(path)
	assert.Error(t, err)
}

func TestBank_LoadFile_MissingID(t *testing.T) {
	dir := t.TempDir()
	path := createTestBankFile(t, dir, BankFile{
		Version:    "1.0",
		Challenges: []challenge.Definition{{Name: "No ID"}},
	})

	b := New()
	err := b.LoadFile(path)
	assert.Error(t, err)
}

func TestBank_LoadDir(t *testing.T) {
	dir := t.TempDir()
	for i, name := range []string{"a.json", "b.json"} {
		data, _ := json.Marshal(BankFile{
			Version: "1.0",
			Challenges: []challenge.Definition{
				{ID: challenge.ID(fmt.Sprintf("ch-%d", i)), Name: name},
			},
		})
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0644))
	}
	// Non-JSON file should be skipped
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("skip"), 0644))

	b := New()
	require.NoError(t, b.LoadDir(dir))
	assert.Equal(t, 2, b.Count())
}

func TestBank_ByCategory(t *testing.T) {
	b := New()
	b.definitions["ch-1"] = &challenge.Definition{ID: "ch-1", Category: "api"}
	b.definitions["ch-2"] = &challenge.Definition{ID: "ch-2", Category: "unit"}
	b.definitions["ch-3"] = &challenge.Definition{ID: "ch-3", Category: "api"}

	apis := b.ByCategory("api")
	assert.Len(t, apis, 2)
}

func TestBank_All(t *testing.T) {
	b := New()
	b.definitions["ch-1"] = &challenge.Definition{ID: "ch-1"}
	b.definitions["ch-2"] = &challenge.Definition{ID: "ch-2"}

	all := b.All()
	assert.Len(t, all, 2)
}

func TestBank_Sources(t *testing.T) {
	b := New()
	b.sources = []string{"a.json", "b.json"}
	sources := b.Sources()
	assert.Equal(t, []string{"a.json", "b.json"}, sources)
}
