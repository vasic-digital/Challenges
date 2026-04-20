package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.challenges/pkg/userflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// resolveGroups tests
// ---------------------------------------------------------------------------
//
// resolveGroups walks the caller-supplied platformGroups map. The tests here
// install a small synthetic map for the duration of each test and restore the
// original after — no project-specific names or service lists appear, keeping
// the binary 100% decoupled per CLAUDE.md §MANDATORY: Project-Agnostic.

// withPlatformGroups swaps the package-level platformGroups for the duration
// of a test. The cleanup restores whatever was there before so tests stay
// independent.
func withPlatformGroups(t *testing.T, groups map[string]userflow.PlatformGroup) {
	t.Helper()
	orig := platformGroups
	platformGroups = groups
	t.Cleanup(func() { platformGroups = orig })
}

// synthPlatformGroups returns a deterministic, intentionally-generic three-
// platform map used by the resolveGroups tests. The keys and service names
// are arbitrary strings — nothing project-specific leaks into the binary.
func synthPlatformGroups() map[string]userflow.PlatformGroup {
	return map[string]userflow.PlatformGroup{
		"alpha": {
			Name:     "alpha",
			Services: []string{"service-a"},
			CPULimit: 1.0, MemoryMB: 256,
		},
		"beta": {
			Name:     "beta",
			Services: []string{"service-a", "service-b"},
			CPULimit: 2.0, MemoryMB: 512,
		},
		"gamma": {
			Name:     "gamma",
			Services: []string{"service-c"},
			CPULimit: 0.5, MemoryMB: 128,
		},
	}
}

func TestResolveGroups_AllPlatformExpandsEveryKey(t *testing.T) {
	withPlatformGroups(t, synthPlatformGroups())

	groups, err := resolveGroups("all")
	require.NoError(t, err)
	assert.Len(t, groups, 3, "`all` must expand to every configured group")

	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.Name)
	}
	assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, names)
}

func TestResolveGroups_SinglePlatformRoundTrip(t *testing.T) {
	withPlatformGroups(t, synthPlatformGroups())

	for _, name := range []string{"alpha", "beta", "gamma"} {
		t.Run(name, func(t *testing.T) {
			groups, err := resolveGroups(name)
			require.NoError(t, err)
			require.Len(t, groups, 1)
			assert.Equal(t, name, groups[0].Name)
		})
	}
}

func TestResolveGroups_CaseInsensitive(t *testing.T) {
	withPlatformGroups(t, synthPlatformGroups())

	for _, input := range []string{"ALPHA", "Alpha", "BeTa"} {
		t.Run(input, func(t *testing.T) {
			groups, err := resolveGroups(input)
			require.NoError(t, err)
			require.Len(t, groups, 1)
			assert.Equal(t, strings.ToLower(input), groups[0].Name)
		})
	}
}

func TestResolveGroups_Whitespace(t *testing.T) {
	withPlatformGroups(t, synthPlatformGroups())

	for _, input := range []string{"  alpha  ", "\tbeta\t"} {
		t.Run(input, func(t *testing.T) {
			groups, err := resolveGroups(input)
			require.NoError(t, err)
			require.Len(t, groups, 1)
			assert.Equal(t, strings.TrimSpace(strings.ToLower(input)), groups[0].Name)
		})
	}
}

func TestResolveGroups_InvalidPlatform(t *testing.T) {
	withPlatformGroups(t, synthPlatformGroups())

	tests := []struct {
		name     string
		platform string
	}{
		{"empty string", ""},
		{"unknown platform", "delta"},
		{"typo", "alhpa"},
		{"numeric", "123"},
		{"special chars", "alpha!"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			groups, err := resolveGroups(tc.platform)
			assert.Error(t, err)
			assert.Nil(t, groups)
			assert.Contains(t, err.Error(), "unknown platform")
		})
	}
}

func TestResolveGroups_UnloadedPlatformGroupsErrors(t *testing.T) {
	withPlatformGroups(t, map[string]userflow.PlatformGroup{})

	groups, err := resolveGroups("alpha")
	assert.Error(t, err, "empty platformGroups must reject every request")
	assert.Nil(t, groups)
}

// ---------------------------------------------------------------------------
// cliLogger tests
// ---------------------------------------------------------------------------

// captureStdout runs fn with stdout redirected to a pipe and
// returns whatever was printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// captureStderr runs fn with stderr redirected to a pipe and
// returns whatever was printed.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestCliLogger_Info_NoArgs(t *testing.T) {
	l := &cliLogger{verbose: false}
	out := captureStdout(t, func() {
		l.Info("hello world")
	})
	assert.Contains(t, out, "[INFO]")
	assert.Contains(t, out, "hello world")
}

func TestCliLogger_Info_WithKeyValueArgs(t *testing.T) {
	l := &cliLogger{verbose: false}
	out := captureStdout(t, func() {
		l.Info("starting", "port", 8080, "host", "localhost")
	})
	assert.Contains(t, out, "[INFO]")
	assert.Contains(t, out, "starting")
	assert.Contains(t, out, "port=8080")
	assert.Contains(t, out, "host=localhost")
}

func TestCliLogger_Info_WithOddArgs(t *testing.T) {
	l := &cliLogger{verbose: false}
	out := captureStdout(t, func() {
		l.Info("values", "a", "b", "c")
	})
	assert.Contains(t, out, "[INFO]")
	assert.Contains(t, out, "values")
	// Odd number of args: printed space-separated, not as k=v
	assert.Contains(t, out, " a")
	assert.Contains(t, out, " b")
	assert.Contains(t, out, " c")
}

func TestCliLogger_Warn(t *testing.T) {
	l := &cliLogger{verbose: false}
	out := captureStdout(t, func() {
		l.Warn("caution", "key", "val")
	})
	assert.Contains(t, out, "[WARN]")
	assert.Contains(t, out, "caution")
	assert.Contains(t, out, "key=val")
}

func TestCliLogger_Error_StderrPrefix(t *testing.T) {
	l := &cliLogger{verbose: false}
	stderr := captureStderr(t, func() {
		// printArgs writes to stdout (via fmt.Printf), so
		// we only verify the prefix and message on stderr.
		l.Error("something broke", "code", 500)
	})
	assert.Contains(t, stderr, "[ERROR]")
	assert.Contains(t, stderr, "something broke")
}

func TestCliLogger_Error_ArgsOnStdout(t *testing.T) {
	l := &cliLogger{verbose: false}
	// printArgs uses fmt.Printf which writes to stdout, even
	// when called from Error. Verify args appear on stdout.
	stdout := captureStdout(t, func() {
		l.Error("something broke", "code", 500)
	})
	assert.Contains(t, stdout, "code=500")
}

func TestCliLogger_Error_NoArgs(t *testing.T) {
	l := &cliLogger{verbose: false}
	stderr := captureStderr(t, func() {
		l.Error("bare error")
	})
	assert.Contains(t, stderr, "[ERROR]")
	assert.Contains(t, stderr, "bare error")
}

func TestCliLogger_Debug_VerboseEnabled(t *testing.T) {
	l := &cliLogger{verbose: true}
	out := captureStdout(t, func() {
		l.Debug("trace detail", "step", 3)
	})
	assert.Contains(t, out, "[DEBUG]")
	assert.Contains(t, out, "trace detail")
	assert.Contains(t, out, "step=3")
}

func TestCliLogger_Debug_VerboseDisabled(t *testing.T) {
	l := &cliLogger{verbose: false}
	out := captureStdout(t, func() {
		l.Debug("trace detail", "step", 3)
	})
	assert.Empty(t, out, "debug should not print when verbose is false")
}

func TestCliLogger_Close(t *testing.T) {
	l := &cliLogger{verbose: false}
	err := l.Close()
	assert.NoError(t, err)
}

func TestCliLogger_ImplementsLoggerInterface(t *testing.T) {
	// Compile-time check that cliLogger satisfies challenge.Logger.
	var _ challenge.Logger = (*cliLogger)(nil)
}

func TestCliLogger_PrintArgs_Empty(t *testing.T) {
	l := &cliLogger{verbose: false}
	// Info with no extra args should just print the message.
	out := captureStdout(t, func() {
		l.Info("bare message")
	})
	// Should not contain "=" (no key-value pairs).
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 1)
	assert.Equal(t, "[INFO]  bare message", lines[0])
}

func TestCliLogger_PrintArgs_EvenPairs(t *testing.T) {
	l := &cliLogger{verbose: false}
	out := captureStdout(t, func() {
		l.Info("msg", "k1", "v1", "k2", "v2")
	})
	assert.Contains(t, out, "k1=v1")
	assert.Contains(t, out, "k2=v2")
}

func TestCliLogger_PrintArgs_OddCount(t *testing.T) {
	l := &cliLogger{verbose: false}
	out := captureStdout(t, func() {
		l.Info("msg", "only-one")
	})
	// Odd count: args are space-separated, no "=" sign.
	assert.Contains(t, out, " only-one")
	assert.NotContains(t, out, "=")
}

// ---------------------------------------------------------------------------
// generateReport tests
// ---------------------------------------------------------------------------

func TestGenerateReport_EmptyResults(t *testing.T) {
	dir := t.TempDir()
	err := generateReport(nil, dir, "markdown")
	assert.NoError(t, err, "empty results should return nil immediately")
}

func TestGenerateReport_Markdown(t *testing.T) {
	dir := t.TempDir()
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-TEST-001",
			ChallengeName: "Test Challenge",
			Status:        challenge.StatusPassed,
			StartTime:     time.Now().Add(-1 * time.Second),
			EndTime:       time.Now(),
			Duration:      1 * time.Second,
			Assertions: []challenge.AssertionResult{
				{
					Type:   "not_empty",
					Target: "output",
					Passed: true,
				},
			},
		},
	}

	err := generateReport(results, dir, "markdown")
	require.NoError(t, err)

	// Check individual report file was created.
	reportPath := filepath.Join(dir, "CH-TEST-001.md")
	assert.FileExists(t, reportPath)

	// Check summary file was created.
	summaryPath := filepath.Join(dir, "summary.md")
	assert.FileExists(t, summaryPath)
}

func TestGenerateReport_JSON(t *testing.T) {
	dir := t.TempDir()
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-TEST-002",
			ChallengeName: "JSON Test",
			Status:        challenge.StatusFailed,
			Duration:      500 * time.Millisecond,
		},
	}

	err := generateReport(results, dir, "json")
	require.NoError(t, err)

	reportPath := filepath.Join(dir, "CH-TEST-002.json")
	assert.FileExists(t, reportPath)

	summaryPath := filepath.Join(dir, "summary.json")
	assert.FileExists(t, summaryPath)
}

func TestGenerateReport_HTML(t *testing.T) {
	dir := t.TempDir()
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-TEST-003",
			ChallengeName: "HTML Test",
			Status:        challenge.StatusPassed,
			Duration:      2 * time.Second,
		},
	}

	err := generateReport(results, dir, "html")
	require.NoError(t, err)

	reportPath := filepath.Join(dir, "CH-TEST-003.html")
	assert.FileExists(t, reportPath)

	summaryPath := filepath.Join(dir, "summary.html")
	assert.FileExists(t, summaryPath)
}

func TestGenerateReport_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-TEST-004",
			ChallengeName: "Bad Format",
			Status:        challenge.StatusPassed,
		},
	}

	err := generateReport(results, dir, "xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestGenerateReport_CaseInsensitiveFormat(t *testing.T) {
	tests := []struct {
		format string
		ext    string
	}{
		{"MARKDOWN", "md"},
		{"Markdown", "md"},
		{"JSON", "json"},
		{"Json", "json"},
		{"HTML", "html"},
		{"Html", "html"},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			dir := t.TempDir()
			results := []*challenge.Result{
				{
					ChallengeID:   "CH-CASE",
					ChallengeName: "Case Test",
					Status:        challenge.StatusPassed,
				},
			}

			err := generateReport(results, dir, tc.format)
			require.NoError(t, err)

			reportPath := filepath.Join(
				dir, fmt.Sprintf("CH-CASE.%s", tc.ext),
			)
			assert.FileExists(t, reportPath)
		})
	}
}

func TestGenerateReport_MultipleResults(t *testing.T) {
	dir := t.TempDir()
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-MULTI-001",
			ChallengeName: "First",
			Status:        challenge.StatusPassed,
			Duration:      1 * time.Second,
		},
		{
			ChallengeID:   "CH-MULTI-002",
			ChallengeName: "Second",
			Status:        challenge.StatusFailed,
			Duration:      2 * time.Second,
			Error:         "assertion failed",
		},
		{
			ChallengeID:   "CH-MULTI-003",
			ChallengeName: "Third",
			Status:        challenge.StatusSkipped,
		},
	}

	err := generateReport(results, dir, "json")
	require.NoError(t, err)

	for _, r := range results {
		path := filepath.Join(
			dir, fmt.Sprintf("%s.json", r.ChallengeID),
		)
		assert.FileExists(t, path,
			"report for %s should exist", r.ChallengeID)
	}

	summaryPath := filepath.Join(dir, "summary.json")
	assert.FileExists(t, summaryPath)
}

func TestGenerateReport_InvalidOutputDir(t *testing.T) {
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-ERR",
			ChallengeName: "Error Test",
			Status:        challenge.StatusPassed,
		},
	}

	// Use a path that cannot be written to.
	err := generateReport(
		results,
		"/dev/null/impossible/path",
		"json",
	)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// printSummary tests
// ---------------------------------------------------------------------------

func TestPrintSummary_NoResults(t *testing.T) {
	l := &cliLogger{verbose: false}
	out := captureStdout(t, func() {
		printSummary(nil, l)
	})
	assert.Contains(t, out, "no challenges were executed")
}

func TestPrintSummary_AllPassed(t *testing.T) {
	l := &cliLogger{verbose: false}
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-001",
			ChallengeName: "Alpha",
			Status:        challenge.StatusPassed,
			Duration:      1 * time.Second,
			Assertions: []challenge.AssertionResult{
				{Passed: true},
				{Passed: true},
			},
		},
		{
			ChallengeID:   "CH-002",
			ChallengeName: "Beta",
			Status:        challenge.StatusPassed,
			Duration:      2 * time.Second,
			Assertions: []challenge.AssertionResult{
				{Passed: true},
			},
		},
	}

	out := captureStdout(t, func() {
		printSummary(results, l)
	})

	assert.Contains(t, out, "Total:      2 challenges")
	assert.Contains(t, out, "Passed:     2")
	assert.Contains(t, out, "Failed:     0")
	assert.Contains(t, out, "Skipped:    0")
	assert.Contains(t, out, "Errors:     0")
	assert.Contains(t, out, "Assertions: 3/3")
	assert.NotContains(t, out, "Failed/Errored Challenges:")
}

func TestPrintSummary_MixedResults(t *testing.T) {
	l := &cliLogger{verbose: false}
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-001",
			ChallengeName: "Pass",
			Status:        challenge.StatusPassed,
			Duration:      1 * time.Second,
			Assertions: []challenge.AssertionResult{
				{Passed: true},
			},
		},
		{
			ChallengeID:   "CH-002",
			ChallengeName: "Fail",
			Status:        challenge.StatusFailed,
			Duration:      2 * time.Second,
			Error:         "timeout exceeded",
			Assertions: []challenge.AssertionResult{
				{Passed: false},
				{Passed: true},
			},
		},
		{
			ChallengeID:   "CH-003",
			ChallengeName: "Skip",
			Status:        challenge.StatusSkipped,
		},
		{
			ChallengeID:   "CH-004",
			ChallengeName: "Error",
			Status:        challenge.StatusError,
			Error:         "connection refused",
		},
	}

	out := captureStdout(t, func() {
		printSummary(results, l)
	})

	assert.Contains(t, out, "Total:      4 challenges")
	assert.Contains(t, out, "Passed:     1")
	assert.Contains(t, out, "Failed:     1")
	assert.Contains(t, out, "Skipped:    1")
	assert.Contains(t, out, "Errors:     1")
	assert.Contains(t, out, "Assertions: 2/3")

	// Failed/errored section should appear.
	assert.Contains(t, out, "Failed/Errored Challenges:")
	assert.Contains(t, out, "CH-002")
	assert.Contains(t, out, "Fail")
	assert.Contains(t, out, "timeout exceeded")
	assert.Contains(t, out, "CH-004")
	assert.Contains(t, out, "Error")
	assert.Contains(t, out, "connection refused")
}

func TestPrintSummary_SkippedOnly(t *testing.T) {
	l := &cliLogger{verbose: false}
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-001",
			ChallengeName: "Skipped One",
			Status:        challenge.StatusSkipped,
		},
	}

	out := captureStdout(t, func() {
		printSummary(results, l)
	})

	assert.Contains(t, out, "Total:      1 challenges")
	assert.Contains(t, out, "Skipped:    1")
	assert.Contains(t, out, "Passed:     0")
	assert.NotContains(t, out, "Failed/Errored Challenges:")
}

func TestPrintSummary_DurationAggregation(t *testing.T) {
	l := &cliLogger{verbose: false}
	results := []*challenge.Result{
		{
			ChallengeID: "CH-001",
			Status:      challenge.StatusPassed,
			Duration:    3 * time.Second,
		},
		{
			ChallengeID: "CH-002",
			Status:      challenge.StatusPassed,
			Duration:    7 * time.Second,
		},
	}

	out := captureStdout(t, func() {
		printSummary(results, l)
	})

	assert.Contains(t, out, "Duration:   10s")
}

func TestPrintSummary_StuckAndTimedOutCountAsErrors(t *testing.T) {
	l := &cliLogger{verbose: false}
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-001",
			ChallengeName: "Stuck",
			Status:        challenge.StatusStuck,
			Error:         "no progress for 5m",
		},
		{
			ChallengeID:   "CH-002",
			ChallengeName: "TimedOut",
			Status:        challenge.StatusTimedOut,
			Error:         "exceeded 1h deadline",
		},
	}

	out := captureStdout(t, func() {
		printSummary(results, l)
	})

	// stuck and timed_out are not passed/failed/skipped, so
	// they should fall into the errored bucket.
	assert.Contains(t, out, "Errors:     2")
	assert.Contains(t, out, "Failed/Errored Challenges:")
	assert.Contains(t, out, "CH-001")
	assert.Contains(t, out, "CH-002")
}

// ---------------------------------------------------------------------------
// loadPlatformGroups contract
// ---------------------------------------------------------------------------
//
// The module's constitution (CLAUDE.md §MANDATORY: Project-Agnostic) forbids
// baking any project-specific service data into this binary. Platform groups
// are loaded at runtime from a caller-supplied JSON file; the tests here
// exercise that loader contract without asserting on any specific service
// name, platform key, or resource number — those all belong to the caller.

func TestLoadPlatformGroups_EmptyPathReturnsEmptyMap(t *testing.T) {
	groups, err := loadPlatformGroups("")
	assert.NoError(t, err)
	assert.NotNil(t, groups, "empty-path loader must return a non-nil map")
	assert.Empty(t, groups, "empty-path loader must yield no groups")
}

func TestLoadPlatformGroups_MissingFileErrors(t *testing.T) {
	_, err := loadPlatformGroups("/nonexistent/path/platform-groups.json")
	assert.Error(t, err, "missing file must surface an error")
	assert.Contains(t, err.Error(), "read platform groups")
}

func TestLoadPlatformGroups_MalformedJSONErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{ not: json"), 0o600))

	_, err := loadPlatformGroups(path)
	assert.Error(t, err, "malformed JSON must surface an error")
	assert.Contains(t, err.Error(), "parse platform groups")
}

func TestLoadPlatformGroups_ValidFixtureRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "groups.json")
	// Caller-supplied fixture: service names, cpu/memory values, and
	// platform keys are all arbitrary — the loader treats them as opaque
	// user data. This is exactly what the constitution requires.
	fixture := []byte(`{
	  "test-platform": {
	    "name": "test-platform",
	    "services": ["service-one", "service-two"],
	    "cpu_limit": 1.5,
	    "memory_mb": 512
	  }
	}`)
	require.NoError(t, os.WriteFile(path, fixture, 0o600))

	groups, err := loadPlatformGroups(path)
	require.NoError(t, err)
	require.Contains(t, groups, "test-platform")

	g := groups["test-platform"]
	assert.Equal(t, "test-platform", g.Name)
	assert.Equal(t, []string{"service-one", "service-two"}, g.Services)
	assert.InDelta(t, 1.5, g.CPULimit, 0.0001)
	assert.Equal(t, 512, g.MemoryMB)
}

// ---------------------------------------------------------------------------
// Exit code constants
// ---------------------------------------------------------------------------

func TestExitCodes_Values(t *testing.T) {
	assert.Equal(t, 0, exitSuccess)
	assert.Equal(t, 1, exitFailures)
	assert.Equal(t, 2, exitError)
}

// ---------------------------------------------------------------------------
// Report format validation (as done in run())
// ---------------------------------------------------------------------------

func TestReportFormatValidation(t *testing.T) {
	tests := []struct {
		name  string
		fmt   string
		valid bool
	}{
		{"markdown lowercase", "markdown", true},
		{"json lowercase", "json", true},
		{"html lowercase", "html", true},
		{"Markdown mixed case", "Markdown", true},
		{"JSON uppercase", "JSON", true},
		{"HTML uppercase", "HTML", true},
		{"xml invalid", "xml", false},
		{"csv invalid", "csv", false},
		{"empty invalid", "", false},
		{"pdf invalid", "pdf", false},
		{"text invalid", "text", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			valid := isValidReportFormat(tc.fmt)
			assert.Equal(t, tc.valid, valid)
		})
	}
}

// isValidReportFormat mirrors the validation logic in run().
// We define it here as a helper so we can test the validation
// logic in isolation without invoking run() (which has side
// effects like flag.Parse and os.Exit).
func isValidReportFormat(format string) bool {
	switch strings.ToLower(format) {
	case "markdown", "json", "html":
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// Output directory creation
// ---------------------------------------------------------------------------

func TestOutputDirectoryCreation(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "a", "b", "c")

	// Verify it does not exist yet.
	_, err := os.Stat(nested)
	assert.True(t, os.IsNotExist(err))

	// MkdirAll should create the full chain.
	err = os.MkdirAll(nested, 0o755)
	require.NoError(t, err)

	info, err := os.Stat(nested)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestOutputDirectoryCreation_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	// Creating an already-existing directory should not fail.
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Timeout parsing (flag.Duration behavior verification)
// ---------------------------------------------------------------------------

func TestTimeoutParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"1 hour", "1h", 1 * time.Hour, false},
		{"30 minutes", "30m", 30 * time.Minute, false},
		{"90 seconds", "90s", 90 * time.Second, false},
		{"mixed", "1h30m", 90 * time.Minute, false},
		{"zero", "0s", 0, false},
		{"invalid", "abc", 0, true},
		{"negative", "-5m", -5 * time.Minute, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := time.ParseDuration(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, d)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// generateReport with assertions in results
// ---------------------------------------------------------------------------

func TestGenerateReport_WithAssertions(t *testing.T) {
	dir := t.TempDir()
	results := []*challenge.Result{
		{
			ChallengeID:   "CH-ASSERT",
			ChallengeName: "Assertion Test",
			Status:        challenge.StatusFailed,
			Duration:      100 * time.Millisecond,
			Assertions: []challenge.AssertionResult{
				{
					Type:     "not_empty",
					Target:   "response_body",
					Expected: "non-empty",
					Actual:   "",
					Passed:   false,
					Message:  "expected non-empty response",
				},
				{
					Type:     "contains",
					Target:   "content_type",
					Expected: "application/json",
					Actual:   "application/json; charset=utf-8",
					Passed:   true,
					Message:  "content type matches",
				},
			},
			Error: "1 assertion failed",
		},
	}

	err := generateReport(results, dir, "json")
	require.NoError(t, err)

	reportPath := filepath.Join(dir, "CH-ASSERT.json")
	assert.FileExists(t, reportPath)

	data, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}
