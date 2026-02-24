package userflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ BuildAdapter = (*NPMCLIAdapter)(nil)

func TestNPMCLIAdapter_Available_True(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "package.json")
	err := os.WriteFile(pkg, []byte("{}"), 0644)
	require.NoError(t, err)

	adapter := NewNPMCLIAdapter(dir)
	assert.True(t, adapter.Available(context.Background()))
}

func TestNPMCLIAdapter_Available_False(t *testing.T) {
	dir := t.TempDir()
	adapter := NewNPMCLIAdapter(dir)
	assert.False(t, adapter.Available(context.Background()))
}

func TestNPMCLIAdapter_Constructor(t *testing.T) {
	adapter := NewNPMCLIAdapter("/tmp/project")
	assert.NotNil(t, adapter)
	assert.Equal(t, "/tmp/project", adapter.projectRoot)
}

func TestNPMCLIAdapter_Build_NoNPM(t *testing.T) {
	dir := t.TempDir()
	adapter := NewNPMCLIAdapter(dir)

	result, err := adapter.Build(
		context.Background(),
		BuildTarget{Name: "web", Task: "build"},
	)

	// npm might not be installed or task may not exist.
	assert.NotNil(t, result)
	assert.Equal(t, "web", result.Target)
	if err != nil {
		assert.False(t, result.Success)
	}
}

func TestNPMCLIAdapter_RunTests_NoProject(t *testing.T) {
	dir := t.TempDir()
	adapter := NewNPMCLIAdapter(dir)

	result, err := adapter.RunTests(
		context.Background(),
		TestTarget{Name: "unit", Task: "test"},
	)

	assert.Error(t, err)
	assert.NotNil(t, result)
}

func TestNPMCLIAdapter_Lint_NoProject(t *testing.T) {
	dir := t.TempDir()
	adapter := NewNPMCLIAdapter(dir)

	result, err := adapter.Lint(
		context.Background(),
		LintTarget{Name: "lint", Task: "lint"},
	)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "eslint", result.Tool)
}

func TestNPMCLIAdapter_ESLintParsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantErr  int
		wantWarn int
	}{
		{
			name: "clean",
			json: `[{"messages":[],"errorCount":0,` +
				`"warningCount":0}]`,
			wantErr:  0,
			wantWarn: 0,
		},
		{
			name: "with_errors",
			json: `[{"messages":[{"severity":2}],` +
				`"errorCount":3,"warningCount":1}]`,
			wantErr:  3,
			wantWarn: 1,
		},
		{
			name: "multiple_files",
			json: `[{"messages":[],"errorCount":1,` +
				`"warningCount":2},` +
				`{"messages":[],"errorCount":0,` +
				`"warningCount":3}]`,
			wantErr:  1,
			wantWarn: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var results []eslintResult
			err := json.Unmarshal(
				[]byte(tt.json), &results,
			)
			require.NoError(t, err)

			var errs, warns int
			for _, r := range results {
				errs += r.ErrorCount
				warns += r.WarningCount
			}
			assert.Equal(t, tt.wantErr, errs)
			assert.Equal(t, tt.wantWarn, warns)
		})
	}
}
