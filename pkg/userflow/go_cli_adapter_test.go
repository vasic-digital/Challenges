package userflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ BuildAdapter = (*GoCLIAdapter)(nil)

func TestGoCLIAdapter_Available_True(t *testing.T) {
	dir := t.TempDir()
	gomod := filepath.Join(dir, "go.mod")
	err := os.WriteFile(
		gomod, []byte("module test\n"), 0644,
	)
	require.NoError(t, err)

	adapter := NewGoCLIAdapter(dir)
	assert.True(t, adapter.Available(context.Background()))
}

func TestGoCLIAdapter_Available_False(t *testing.T) {
	dir := t.TempDir()
	adapter := NewGoCLIAdapter(dir)
	assert.False(t, adapter.Available(context.Background()))
}

func TestGoCLIAdapter_Constructor(t *testing.T) {
	adapter := NewGoCLIAdapter("/tmp/mygo")
	assert.NotNil(t, adapter)
	assert.Equal(t, "/tmp/mygo", adapter.projectRoot)
}

func TestGoCLIAdapter_ParseGoTestJSON(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantTotal   int
		wantFailed  int
		wantSkipped int
		wantSuites  int
	}{
		{
			name: "all_pass",
			output: `{"Time":"2024-01-01T00:00:00Z",` +
				`"Action":"pass","Package":"pkg/a",` +
				`"Test":"TestFoo","Elapsed":0.5}` +
				"\n" +
				`{"Time":"2024-01-01T00:00:00Z",` +
				`"Action":"pass","Package":"pkg/a",` +
				`"Test":"TestBar","Elapsed":0.3}`,
			wantTotal:   2,
			wantFailed:  0,
			wantSkipped: 0,
			wantSuites:  1,
		},
		{
			name: "mixed_results",
			output: `{"Action":"pass","Package":"pkg/a",` +
				`"Test":"TestOK","Elapsed":0.1}` +
				"\n" +
				`{"Action":"fail","Package":"pkg/a",` +
				`"Test":"TestBad","Elapsed":0.2}` +
				"\n" +
				`{"Action":"skip","Package":"pkg/b",` +
				`"Test":"TestSkip","Elapsed":0}`,
			wantTotal:   3,
			wantFailed:  1,
			wantSkipped: 1,
			wantSuites:  2,
		},
		{
			name: "package_level_events_ignored",
			output: `{"Action":"pass","Package":"pkg/a",` +
				`"Elapsed":1.0}` +
				"\n" +
				`{"Action":"pass","Package":"pkg/a",` +
				`"Test":"TestX","Elapsed":0.5}`,
			wantTotal:   1,
			wantFailed:  0,
			wantSkipped: 0,
			wantSuites:  1,
		},
		{
			name:        "empty_output",
			output:      "",
			wantTotal:   0,
			wantFailed:  0,
			wantSkipped: 0,
			wantSuites:  0,
		},
		{
			name:        "invalid_json",
			output:      "not json\n{broken",
			wantTotal:   0,
			wantFailed:  0,
			wantSkipped: 0,
			wantSuites:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGoTestJSON(
				tt.output, time.Second,
			)
			assert.Equal(
				t, tt.wantTotal, result.TotalTests,
			)
			assert.Equal(
				t, tt.wantFailed, result.TotalFailed,
			)
			assert.Equal(
				t, tt.wantSkipped, result.TotalSkipped,
			)
			assert.Equal(
				t, tt.wantSuites, len(result.Suites),
			)
		})
	}
}

func TestGoCLIAdapter_Build_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	adapter := NewGoCLIAdapter(dir)

	result, err := adapter.Build(
		context.Background(),
		BuildTarget{Name: "api", Task: "./..."},
	)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestGoCLIAdapter_Lint_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	adapter := NewGoCLIAdapter(dir)

	result, err := adapter.Lint(
		context.Background(),
		LintTarget{Name: "vet", Task: "./..."},
	)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, "go vet", result.Tool)
}
