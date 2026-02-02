package challenge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseChallenge_NewBaseChallenge(t *testing.T) {
	b := NewBaseChallenge(
		"base-001", "Base Test", "desc", "unit",
		[]ID{"dep-1"},
	)
	assert.Equal(t, ID("base-001"), b.ID())
	assert.Equal(t, "Base Test", b.Name())
	assert.Equal(t, "desc", b.Description())
	assert.Equal(t, "unit", b.Category())
	assert.Equal(t, []ID{"dep-1"}, b.Dependencies())
}

func TestBaseChallenge_NewBaseChallenge_NilDeps(t *testing.T) {
	b := NewBaseChallenge(
		"base-002", "No Deps", "desc", "unit", nil,
	)
	assert.NotNil(t, b.Dependencies())
	assert.Empty(t, b.Dependencies())
}

func TestBaseChallenge_Configure_Success(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"cfg-001", "Config Test", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID: "cfg-001",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}

	err := b.Configure(cfg)
	require.NoError(t, err)
	assert.NotNil(t, b.Config())

	// Directories should be created.
	_, err = os.Stat(b.ResultsDir())
	assert.NoError(t, err)
	_, err = os.Stat(b.LogsDir())
	assert.NoError(t, err)
}

func TestBaseChallenge_Configure_NilConfig(t *testing.T) {
	b := NewBaseChallenge(
		"cfg-002", "Nil Config", "desc", "unit", nil,
	)
	err := b.Configure(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must not be nil")
}

func TestBaseChallenge_Validate_NotConfigured(t *testing.T) {
	b := NewBaseChallenge(
		"val-001", "Validate Test", "desc", "unit", nil,
	)
	err := b.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestBaseChallenge_Validate_Configured(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"val-002", "Validate OK", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID: "val-002",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}
	require.NoError(t, b.Configure(cfg))
	assert.NoError(t, b.Validate(context.Background()))
}

func TestBaseChallenge_Cleanup_WithLogger(t *testing.T) {
	b := NewBaseChallenge(
		"clean-001", "Cleanup", "desc", "unit", nil,
	)
	ml := &mockLogger{}
	b.SetLogger(ml)

	err := b.Cleanup(context.Background())
	require.NoError(t, err)
	assert.True(t, ml.closed)
}

func TestBaseChallenge_Cleanup_NoLogger(t *testing.T) {
	b := NewBaseChallenge(
		"clean-002", "Cleanup No Log", "desc", "unit", nil,
	)
	err := b.Cleanup(context.Background())
	assert.NoError(t, err)
}

func TestBaseChallenge_ResultsDir_NoConfig(t *testing.T) {
	b := NewBaseChallenge(
		"dir-001", "Dir Test", "desc", "unit", nil,
	)
	assert.Equal(t, "results", b.ResultsDir())
}

func TestBaseChallenge_LogsDir_NoConfig(t *testing.T) {
	b := NewBaseChallenge(
		"dir-002", "Dir Test", "desc", "unit", nil,
	)
	assert.Equal(t, "logs", b.LogsDir())
}

func TestBaseChallenge_ResultsDir_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"dir-003", "Dir", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID: "dir-003",
		ResultsDir:  filepath.Join(tmpDir, "res"),
		LogsDir:     filepath.Join(tmpDir, "log"),
	}
	require.NoError(t, b.Configure(cfg))
	assert.Equal(t,
		filepath.Join(tmpDir, "res", "dir-003"),
		b.ResultsDir(),
	)
}

func TestBaseChallenge_GetEnv(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"env-001", "Env", "desc", "unit", nil,
	)

	// Without config, returns fallback.
	assert.Equal(t, "default", b.GetEnv("KEY", "default"))

	cfg := &Config{
		ChallengeID: "env-001",
		ResultsDir:  filepath.Join(tmpDir, "res"),
		LogsDir:     filepath.Join(tmpDir, "log"),
		Environment: map[string]string{"KEY": "value"},
	}
	require.NoError(t, b.Configure(cfg))
	assert.Equal(t, "value", b.GetEnv("KEY", "default"))
	assert.Equal(t, "fb", b.GetEnv("MISSING", "fb"))
}

func TestBaseChallenge_WriteJSONResult(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"json-001", "JSON Write", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID: "json-001",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}
	require.NoError(t, b.Configure(cfg))

	result := &Result{
		ChallengeID:   "json-001",
		ChallengeName: "JSON Write",
		Status:        StatusPassed,
		StartTime:     time.Now(),
		EndTime:       time.Now(),
		Duration:      50 * time.Millisecond,
	}

	err := b.WriteJSONResult(result)
	require.NoError(t, err)

	path := filepath.Join(b.ResultsDir(), "result.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var loaded Result
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, ID("json-001"), loaded.ChallengeID)
	assert.Equal(t, StatusPassed, loaded.Status)
}

func TestBaseChallenge_WriteMarkdownReport(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"md-001", "MD Write", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID: "md-001",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}
	require.NoError(t, b.Configure(cfg))

	result := &Result{
		ChallengeID:   "md-001",
		ChallengeName: "MD Write",
		Status:        StatusFailed,
		Duration:      200 * time.Millisecond,
		Assertions: []AssertionResult{
			{
				Target:  "status",
				Passed:  true,
				Message: "status ok",
			},
			{
				Target:  "body",
				Passed:  false,
				Message: "body mismatch",
			},
		},
		Error: "something went wrong",
	}

	err := b.WriteMarkdownReport(result)
	require.NoError(t, err)

	path := filepath.Join(b.ResultsDir(), "report.md")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# MD Write")
	assert.Contains(t, content, "**Status**: failed")
	assert.Contains(t, content, "[PASS] status")
	assert.Contains(t, content, "[FAIL] body")
	assert.Contains(t, content, "something went wrong")
}

func TestBaseChallenge_ReadDependencyResult(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a dependency result.
	depResult := &Result{
		ChallengeID:   "dep-001",
		ChallengeName: "Dependency",
		Status:        StatusPassed,
	}
	depData, err := json.Marshal(depResult)
	require.NoError(t, err)
	depPath := filepath.Join(tmpDir, "dep-result.json")
	require.NoError(t, os.WriteFile(depPath, depData, 0o644))

	b := NewBaseChallenge(
		"read-dep-001", "Read Dep", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID: "read-dep-001",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Dependencies: map[ID]string{
			"dep-001": depPath,
		},
	}
	require.NoError(t, b.Configure(cfg))

	loaded, err := b.ReadDependencyResult("dep-001")
	require.NoError(t, err)
	assert.Equal(t, ID("dep-001"), loaded.ChallengeID)
	assert.Equal(t, StatusPassed, loaded.Status)
}

func TestBaseChallenge_ReadDependencyResult_NotConfigured(
	t *testing.T,
) {
	b := NewBaseChallenge(
		"rd-002", "Not Configured", "desc", "unit", nil,
	)
	_, err := b.ReadDependencyResult("dep-001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestBaseChallenge_ReadDependencyResult_MissingDep(
	t *testing.T,
) {
	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"rd-003", "Missing Dep", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID:  "rd-003",
		ResultsDir:   filepath.Join(tmpDir, "results"),
		LogsDir:      filepath.Join(tmpDir, "logs"),
		Dependencies: map[ID]string{},
	}
	require.NoError(t, b.Configure(cfg))

	_, err := b.ReadDependencyResult("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path not found")
}

func TestBaseChallenge_EvaluateAssertions_NoEngine(
	t *testing.T,
) {
	b := NewBaseChallenge(
		"eval-001", "No Engine", "desc", "unit", nil,
	)
	defs := []AssertionDef{
		{Type: "equals", Target: "x"},
	}
	results := b.EvaluateAssertions(defs, nil)
	assert.Len(t, results, 1)
	assert.False(t, results[0].Passed)
	assert.Contains(
		t, results[0].Message,
		"no assertion engine configured",
	)
}

func TestBaseChallenge_EvaluateAssertions_WithEngine(
	t *testing.T,
) {
	b := NewBaseChallenge(
		"eval-002", "With Engine", "desc", "unit", nil,
	)
	b.SetAssertionEngine(&mockAssertionEngine{})
	defs := []AssertionDef{
		{Type: "equals", Target: "x"},
		{Type: "not_empty", Target: "y"},
	}
	results := b.EvaluateAssertions(
		defs, map[string]any{"x": 1, "y": "hello"},
	)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.True(t, r.Passed)
	}
}

func TestBaseChallenge_CreateResult(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseChallenge(
		"cr-001", "Create Result", "desc", "unit", nil,
	)
	cfg := &Config{
		ChallengeID: "cr-001",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}
	require.NoError(t, b.Configure(cfg))

	start := time.Now().Add(-100 * time.Millisecond)
	metrics := map[string]MetricValue{
		"latency": {Name: "latency", Value: 42.5, Unit: "ms"},
	}
	outputs := map[string]string{"body": "ok"}

	r := b.CreateResult(
		StatusPassed, start,
		[]AssertionResult{{Passed: true}},
		metrics, outputs, "",
	)

	assert.Equal(t, ID("cr-001"), r.ChallengeID)
	assert.Equal(t, "Create Result", r.ChallengeName)
	assert.Equal(t, StatusPassed, r.Status)
	assert.False(t, r.StartTime.IsZero())
	assert.False(t, r.EndTime.IsZero())
	assert.True(t, r.Duration > 0)
	assert.Len(t, r.Assertions, 1)
	assert.Contains(t, r.Metrics, "latency")
	assert.Equal(t, "ok", r.Outputs["body"])
	assert.Empty(t, r.Error)
	assert.NotEmpty(t, r.Logs.ChallengeLog)
	assert.NotEmpty(t, r.Logs.OutputLog)
}

func TestBaseChallenge_CreateResult_WithError(t *testing.T) {
	b := NewBaseChallenge(
		"cr-002", "Error Result", "desc", "unit", nil,
	)
	start := time.Now()
	r := b.CreateResult(
		StatusError, start, nil, nil, nil, "boom",
	)
	assert.Equal(t, StatusError, r.Status)
	assert.Equal(t, "boom", r.Error)
}

func TestBaseChallenge_LogInfo_WithLogger(t *testing.T) {
	b := NewBaseChallenge(
		"log-001", "Log Test", "desc", "unit", nil,
	)
	ml := &mockLogger{}
	b.SetLogger(ml)
	b.logInfo("test message")
	assert.Equal(t, []string{"test message"}, ml.infos)
}

func TestBaseChallenge_LogInfo_NoLogger(t *testing.T) {
	b := NewBaseChallenge(
		"log-002", "No Logger", "desc", "unit", nil,
	)
	// Should not panic.
	b.logInfo("test message")
}

func TestBaseChallenge_LogError_WithLogger(t *testing.T) {
	b := NewBaseChallenge(
		"log-003", "Log Error", "desc", "unit", nil,
	)
	ml := &mockLogger{}
	b.SetLogger(ml)
	b.logError("error message")
	assert.Equal(t, []string{"error message"}, ml.errors)
}
