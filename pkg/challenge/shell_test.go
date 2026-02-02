package challenge

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellChallenge_NewShellChallenge(t *testing.T) {
	sc := NewShellChallenge(
		"shell-001", "Shell Test", "desc", "e2e",
		[]ID{"dep-1"},
		"/path/to/script.sh",
		[]string{"--verbose"},
		"/work/dir",
	)
	assert.Equal(t, ID("shell-001"), sc.ID())
	assert.Equal(t, "Shell Test", sc.Name())
	assert.Equal(t, "/path/to/script.sh", sc.ScriptPath)
	assert.Equal(t, []string{"--verbose"}, sc.Args)
	assert.Equal(t, "/work/dir", sc.WorkDir)
}

func TestShellChallenge_NewShellChallenge_NilArgs(t *testing.T) {
	sc := NewShellChallenge(
		"shell-002", "No Args", "desc", "e2e",
		nil, "/path/script.sh", nil, "",
	)
	assert.NotNil(t, sc.Args)
	assert.Empty(t, sc.Args)
}

func TestShellChallenge_Validate_ScriptNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	sc := NewShellChallenge(
		"shell-val-001", "Missing Script", "desc", "e2e",
		nil, "/nonexistent/script.sh", nil, "",
	)
	cfg := &Config{
		ChallengeID: "shell-val-001",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}
	require.NoError(t, sc.Configure(cfg))

	err := sc.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "script")
}

func TestShellChallenge_Validate_ScriptIsDirectory(
	t *testing.T,
) {
	tmpDir := t.TempDir()
	scriptDir := filepath.Join(tmpDir, "script.sh")
	require.NoError(t, os.MkdirAll(scriptDir, 0o755))

	sc := NewShellChallenge(
		"shell-val-002", "Dir Script", "desc", "e2e",
		nil, scriptDir, nil, "",
	)
	cfg := &Config{
		ChallengeID: "shell-val-002",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
	}
	require.NoError(t, sc.Configure(cfg))

	err := sc.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is a directory")
}

func TestShellChallenge_Validate_NotConfigured(t *testing.T) {
	sc := NewShellChallenge(
		"shell-val-003", "Not Configured", "desc", "e2e",
		nil, "/some/script.sh", nil, "",
	)
	err := sc.Validate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestShellChallenge_Execute_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a passing script.
	scriptPath := filepath.Join(tmpDir, "pass.sh")
	script := "#!/bin/bash\necho 'hello world'\nexit 0\n"
	require.NoError(t,
		os.WriteFile(scriptPath, []byte(script), 0o755),
	)

	sc := NewShellChallenge(
		"shell-exec-001", "Pass Script", "desc", "e2e",
		nil, scriptPath, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "shell-exec-001",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     10 * time.Second,
	}
	require.NoError(t, sc.Configure(cfg))

	result, err := sc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StatusPassed, result.Status)
	assert.Equal(t, "hello world", result.Outputs["stdout"])
	assert.Equal(t, "0", result.Outputs["exit_code"])
	assert.Empty(t, result.Error)
}

func TestShellChallenge_Execute_Failure(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "fail.sh")
	script := "#!/bin/bash\necho 'error output' >&2\nexit 1\n"
	require.NoError(t,
		os.WriteFile(scriptPath, []byte(script), 0o755),
	)

	sc := NewShellChallenge(
		"shell-exec-002", "Fail Script", "desc", "e2e",
		nil, scriptPath, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "shell-exec-002",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     10 * time.Second,
	}
	require.NoError(t, sc.Configure(cfg))

	result, err := sc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, result.Status)
	assert.Equal(t, "1", result.Outputs["exit_code"])
	assert.Equal(t, "error output", result.Outputs["stderr"])
	assert.Contains(t, result.Error, "exited with code 1")
}

func TestShellChallenge_Execute_Timeout(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "slow.sh")
	script := "#!/bin/bash\nsleep 10\n"
	require.NoError(t,
		os.WriteFile(scriptPath, []byte(script), 0o755),
	)

	sc := NewShellChallenge(
		"shell-exec-003", "Slow Script", "desc", "e2e",
		nil, scriptPath, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "shell-exec-003",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     500 * time.Millisecond,
	}
	require.NoError(t, sc.Configure(cfg))

	result, err := sc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StatusTimedOut, result.Status)
	assert.Contains(t, result.Error, "timed out")
}

func TestShellChallenge_Execute_WithArgs(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "args.sh")
	script := "#!/bin/bash\necho \"arg1=$1 arg2=$2\"\n"
	require.NoError(t,
		os.WriteFile(scriptPath, []byte(script), 0o755),
	)

	sc := NewShellChallenge(
		"shell-exec-004", "Args Script", "desc", "e2e",
		nil, scriptPath, []string{"hello", "world"}, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "shell-exec-004",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     10 * time.Second,
	}
	require.NoError(t, sc.Configure(cfg))

	result, err := sc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StatusPassed, result.Status)
	assert.Equal(t,
		"arg1=hello arg2=world",
		result.Outputs["stdout"],
	)
}

func TestShellChallenge_Execute_WithEnvironment(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "env.sh")
	script := "#!/bin/bash\necho \"val=$MY_VAR\"\n"
	require.NoError(t,
		os.WriteFile(scriptPath, []byte(script), 0o755),
	)

	sc := NewShellChallenge(
		"shell-exec-005", "Env Script", "desc", "e2e",
		nil, scriptPath, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "shell-exec-005",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     10 * time.Second,
		Environment: map[string]string{
			"MY_VAR": "injected",
		},
	}
	require.NoError(t, sc.Configure(cfg))

	result, err := sc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StatusPassed, result.Status)
	assert.Equal(t, "val=injected", result.Outputs["stdout"])
}

func TestShellChallenge_Execute_WritesOutputLog(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "output.sh")
	script := "#!/bin/bash\necho 'stdout line'\n" +
		"echo 'stderr line' >&2\n"
	require.NoError(t,
		os.WriteFile(scriptPath, []byte(script), 0o755),
	)

	sc := NewShellChallenge(
		"shell-exec-006", "Output Log", "desc", "e2e",
		nil, scriptPath, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "shell-exec-006",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     10 * time.Second,
	}
	require.NoError(t, sc.Configure(cfg))

	_, err := sc.Execute(context.Background())
	require.NoError(t, err)

	logPath := filepath.Join(sc.LogsDir(), "output.log")
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "=== STDOUT ===")
	assert.Contains(t, content, "stdout line")
	assert.Contains(t, content, "=== STDERR ===")
	assert.Contains(t, content, "stderr line")
}

func TestShellChallenge_Execute_WritesResultJSON(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "result.sh")
	script := "#!/bin/bash\necho 'done'\n"
	require.NoError(t,
		os.WriteFile(scriptPath, []byte(script), 0o755),
	)

	sc := NewShellChallenge(
		"shell-exec-007", "Result JSON", "desc", "e2e",
		nil, scriptPath, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "shell-exec-007",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     10 * time.Second,
	}
	require.NoError(t, sc.Configure(cfg))

	_, err := sc.Execute(context.Background())
	require.NoError(t, err)

	resultPath := filepath.Join(
		sc.ResultsDir(), "result.json",
	)
	_, err = os.Stat(resultPath)
	assert.NoError(t, err)
}

func TestShellChallenge_Execute_ExitCode2(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "exit2.sh")
	script := "#!/bin/bash\nexit 2\n"
	require.NoError(t,
		os.WriteFile(scriptPath, []byte(script), 0o755),
	)

	sc := NewShellChallenge(
		"shell-exec-008", "Exit Code 2", "desc", "e2e",
		nil, scriptPath, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "shell-exec-008",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     10 * time.Second,
	}
	require.NoError(t, sc.Configure(cfg))

	result, err := sc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, result.Status)
	assert.Equal(t, "2", result.Outputs["exit_code"])
	assert.Contains(t, result.Error, "exited with code 2")
}

func TestShellChallenge_Execute_NoTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "quick.sh")
	script := "#!/bin/bash\necho 'fast'\n"
	require.NoError(t,
		os.WriteFile(scriptPath, []byte(script), 0o755),
	)

	sc := NewShellChallenge(
		"shell-exec-009", "No Timeout", "desc", "e2e",
		nil, scriptPath, nil, tmpDir,
	)
	cfg := &Config{
		ChallengeID: "shell-exec-009",
		ResultsDir:  filepath.Join(tmpDir, "results"),
		LogsDir:     filepath.Join(tmpDir, "logs"),
		Timeout:     0, // No timeout.
	}
	require.NoError(t, sc.Configure(cfg))

	result, err := sc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StatusPassed, result.Status)
}
