// Package main provides the userflow-runner CLI entry point.
// It orchestrates multi-platform user flow testing by wiring
// TestEnvironment, challenge registration, and the runner.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.challenges/pkg/registry"
	"digital.vasic.challenges/pkg/report"
	"digital.vasic.challenges/pkg/runner"
	"digital.vasic.challenges/pkg/userflow"
)

// Exit codes.
const (
	exitSuccess  = 0
	exitFailures = 1
	exitError    = 2
)

// platformGroups holds the caller-supplied container service groups
// for each platform. The runner loads the groups at startup from the
// file named by --platform-groups; HelixQA's Challenges library
// carries no project-specific service list of its own. See
// loadPlatformGroups for the expected JSON schema.
var platformGroups map[string]userflow.PlatformGroup

// loadPlatformGroups decodes a JSON file mapping platform name →
// PlatformGroup so every project can describe its own service
// topology without modifying this binary. Example contents:
//
//	{
//	  "api":     {"name": "api",     "services": ["api", "postgres"],          "cpuLimit": 2, "memoryMB": 4096},
//	  "web":     {"name": "web",     "services": ["api", "web", "postgres"],    "cpuLimit": 3, "memoryMB": 6144},
//	  "android": {"name": "android", "services": ["api", "postgres"],          "cpuLimit": 2, "memoryMB": 4096}
//	}
//
// An empty path means "no groups configured"; the runner then treats
// every --platform request as unknown and exits with an actionable
// error instead of silently using Catalogizer-specific defaults.
func loadPlatformGroups(path string) (map[string]userflow.PlatformGroup, error) {
	if path == "" {
		return map[string]userflow.PlatformGroup{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read platform groups %q: %w", path, err)
	}
	groups := map[string]userflow.PlatformGroup{}
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, fmt.Errorf("parse platform groups %q: %w", path, err)
	}
	return groups, nil
}

// cliLogger adapts fmt.Printf-style logging to the
// challenge.Logger interface required by the runner.
type cliLogger struct {
	verbose bool
}

func (l *cliLogger) Info(msg string, args ...any) {
	fmt.Printf("[INFO]  %s", msg)
	l.printArgs(args)
	fmt.Println()
}

func (l *cliLogger) Warn(msg string, args ...any) {
	fmt.Printf("[WARN]  %s", msg)
	l.printArgs(args)
	fmt.Println()
}

func (l *cliLogger) Error(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR] %s", msg)
	l.printArgs(args)
	fmt.Fprintln(os.Stderr)
}

func (l *cliLogger) Debug(msg string, args ...any) {
	if l.verbose {
		fmt.Printf("[DEBUG] %s", msg)
		l.printArgs(args)
		fmt.Println()
	}
}

func (l *cliLogger) Close() error { return nil }

func (l *cliLogger) printArgs(args []any) {
	if len(args) == 0 {
		return
	}
	// Print key=value pairs when args come in pairs.
	if len(args)%2 == 0 {
		for i := 0; i < len(args); i += 2 {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	} else {
		for _, a := range args {
			fmt.Printf(" %v", a)
		}
	}
}

func main() {
	os.Exit(run())
}

func run() int {
	platform := flag.String(
		"platform", "all",
		"Platform to test "+
			"(all, api, web, desktop, wizard, android, tv)",
	)
	reportFmt := flag.String(
		"report", "markdown",
		"Report format (markdown, json, html)",
	)
	composeFile := flag.String(
		"compose", "docker-compose.test.yml",
		"Compose file path",
	)
	projectRoot := flag.String(
		"root", ".",
		"Project root directory",
	)
	timeout := flag.Duration(
		"timeout", 1*time.Hour,
		"Overall timeout for all challenge execution",
	)
	outputDir := flag.String(
		"output", "results",
		"Output directory for reports and results",
	)
	verbose := flag.Bool(
		"verbose", false,
		"Enable verbose debug logging",
	)
	platformGroupsFile := flag.String(
		"platform-groups", "",
		"Path to a JSON file mapping platform name → PlatformGroup "+
			"(services list, cpu/memory limits). Required unless "+
			"--platform is a group the caller registered at build time.",
	)
	// Phase 23.6 — Constitution §11.4 default-on for production runs.
	// The runner package honors the CHALLENGE_ANTIBLUFF_STRICT env var
	// for cross-process gating; this CLI flag is the operator-facing
	// switch that defaults the env var to "1" (strict) for every CLI
	// invocation. Library code paths (unit tests, embedded callers)
	// bypass this CLI so they stay legacy default-OFF for backward
	// compat — see runner.Run + antibluff_runner_test.go.
	strictAntiBluff := flag.Bool(
		"strict-anti-bluff", true,
		"Downgrade Status=Passed challenge results to StatusFailed "+
			"when no RecordedActions / no passing Assertions are "+
			"present (Constitution §11.4 captured-evidence rule). "+
			"Sets CHALLENGE_ANTIBLUFF_STRICT=1 in process env.",
	)
	flag.Parse()

	// Phase 23.6 — propagate the flag to the env var the runner reads.
	// Honor an explicit env override only if it's already a non-empty
	// non-default value: a caller who exported CHALLENGE_ANTIBLUFF_STRICT=0
	// before invoking this CLI is asking to opt out (e.g. for a one-off
	// soak test against legacy fixtures).
	if envCur, ok := os.LookupEnv("CHALLENGE_ANTIBLUFF_STRICT"); !ok || envCur == "" {
		if *strictAntiBluff {
			_ = os.Setenv("CHALLENGE_ANTIBLUFF_STRICT", "1")
		}
	}

	logger := &cliLogger{verbose: *verbose}

	loaded, loadErr := loadPlatformGroups(*platformGroupsFile)
	if loadErr != nil {
		logger.Error("platform groups", "error", loadErr)
		return exitError
	}
	platformGroups = loaded

	logger.Info("UserFlow Runner starting")
	logger.Info("Configuration",
		"platform", *platform,
		"report", *reportFmt,
		"compose", *composeFile,
		"root", *projectRoot,
		"timeout", *timeout,
		"output", *outputDir,
		"platform_groups_file", *platformGroupsFile,
		"platform_groups_loaded", len(platformGroups),
	)

	// Validate report format.
	switch strings.ToLower(*reportFmt) {
	case "markdown", "json", "html":
		// valid
	default:
		logger.Error("unsupported report format",
			"format", *reportFmt,
		)
		fmt.Fprintf(os.Stderr,
			"Error: unsupported report format: %s "+
				"(use markdown, json, or html)\n",
			*reportFmt,
		)
		return exitError
	}

	// Resolve platform groups.
	groups, err := resolveGroups(*platform)
	if err != nil {
		logger.Error("invalid platform", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return exitError
	}

	// Ensure output directory exists.
	absOutput, err := filepath.Abs(*outputDir)
	if err != nil {
		logger.Error("resolve output dir",
			"error", err,
		)
		fmt.Fprintf(os.Stderr,
			"Error: cannot resolve output dir: %v\n", err,
		)
		return exitError
	}
	if err := os.MkdirAll(absOutput, 0o755); err != nil {
		logger.Error("create output dir", "error", err)
		fmt.Fprintf(os.Stderr,
			"Error: cannot create output dir: %v\n", err,
		)
		return exitError
	}

	// Set up context with timeout and signal handling.
	ctx, cancel := context.WithTimeout(
		context.Background(), *timeout,
	)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			logger.Warn("received signal, shutting down",
				"signal", sig,
			)
			cancel()
		case <-ctx.Done():
		}
	}()

	// Create test environment.
	env, err := userflow.NewTestEnvironment(
		userflow.WithComposeFile(*composeFile),
		userflow.WithProjectName("userflow-runner"),
		userflow.WithPlatformGroups(groups),
	)
	if err != nil {
		logger.Error("create test environment",
			"error", err,
		)
		fmt.Fprintf(os.Stderr,
			"Error: failed to create test environment: %v\n",
			err,
		)
		return exitError
	}

	// Setup all platform groups.
	logger.Info("setting up platform groups",
		"count", len(groups),
	)
	if err := env.SetupAll(ctx); err != nil {
		logger.Error("environment setup failed",
			"error", err,
		)
		fmt.Fprintf(os.Stderr,
			"Error: environment setup failed: %v\n", err,
		)
		// Attempt teardown even on setup failure.
		teardownErr := env.TeardownAll(
			context.Background(),
		)
		if teardownErr != nil {
			logger.Warn("teardown after setup failure",
				"error", teardownErr,
			)
		}
		return exitError
	}

	// Ensure teardown runs regardless of outcome.
	defer func() {
		logger.Info("tearing down test environment")
		teardownCtx, teardownCancel := context.WithTimeout(
			context.Background(), 2*time.Minute,
		)
		defer teardownCancel()
		if tdErr := env.TeardownAll(teardownCtx); tdErr != nil {
			logger.Error("teardown failed",
				"error", tdErr,
			)
		}
	}()

	// Create runner with configured timeouts.
	reg := registry.Default
	r := runner.NewRunner(
		runner.WithRegistry(reg),
		runner.WithLogger(logger),
		runner.WithTimeout(*timeout),
		runner.WithStaleThreshold(5*time.Minute),
		runner.WithResultsDir(absOutput),
	)

	// Run all registered challenges in dependency order.
	logger.Info("running challenges",
		"registered", reg.Count(),
	)

	cfg := &challenge.Config{
		ResultsDir:  absOutput,
		LogsDir:     filepath.Join(absOutput, "logs"),
		Timeout:     0, // Use runner timeout.
		Verbose:     *verbose,
		Environment: make(map[string]string),
		Dependencies: make(
			map[challenge.ID]string,
		),
	}

	// Inject project root and compose file into environment
	// so challenges can discover them.
	cfg.Environment["PROJECT_ROOT"] = *projectRoot
	cfg.Environment["COMPOSE_FILE"] = *composeFile
	cfg.Environment["PLATFORM"] = *platform

	results, runErr := r.RunAll(ctx, cfg)

	// Check for context cancellation (signal or timeout).
	if ctx.Err() != nil && runErr != nil {
		logger.Warn("run interrupted",
			"reason", ctx.Err(),
			"completed", len(results),
		)
	} else if runErr != nil {
		logger.Error("run error",
			"error", runErr,
			"completed", len(results),
		)
	}

	// Generate report even if some challenges failed.
	logger.Info("generating report",
		"format", *reportFmt,
		"results", len(results),
	)

	if err := generateReport(
		results, absOutput, *reportFmt,
	); err != nil {
		logger.Error("report generation failed",
			"error", err,
		)
		fmt.Fprintf(os.Stderr,
			"Error: report generation failed: %v\n", err,
		)
		// Continue to exit code logic; partial results
		// are still useful.
	}

	// Save structured master summary.
	summary := report.BuildMasterSummary(results)
	if err := report.SaveMasterSummary(
		summary, absOutput,
	); err != nil {
		logger.Warn("save master summary failed",
			"error", err,
		)
	}

	// Print summary to stdout.
	printSummary(results, logger)

	// Determine exit code.
	if runErr != nil {
		return exitError
	}
	for _, res := range results {
		if res.Status != challenge.StatusPassed &&
			res.Status != challenge.StatusSkipped {
			return exitFailures
		}
	}

	logger.Info("all challenges passed")
	return exitSuccess
}

// resolveGroups converts the --platform flag value into a list of
// PlatformGroup structs. The "all" keyword expands to every group the
// caller registered via --platform-groups, sorted by name for
// deterministic ordering. Unknown platforms surface an error whose
// message lists the configured keys so operators can self-correct
// without having to read the source. No project-specific names are
// baked in — keys are whatever the caller supplied.
func resolveGroups(
	platform string,
) ([]userflow.PlatformGroup, error) {
	p := strings.ToLower(strings.TrimSpace(platform))

	configuredNames := make([]string, 0, len(platformGroups))
	for name := range platformGroups {
		configuredNames = append(configuredNames, name)
	}
	sort.Strings(configuredNames)

	if p == "all" {
		all := make(
			[]userflow.PlatformGroup,
			0, len(platformGroups),
		)
		for _, name := range configuredNames {
			all = append(all, platformGroups[name])
		}
		return all, nil
	}

	group, ok := platformGroups[p]
	if !ok {
		configured := strings.Join(configuredNames, ", ")
		if configured == "" {
			configured = "(none — pass --platform-groups)"
		}
		return nil, fmt.Errorf(
			"unknown platform: %s (valid: all, %s)",
			platform, configured,
		)
	}
	return []userflow.PlatformGroup{group}, nil
}

// generateReport creates a report file in the requested format
// using the appropriate reporter implementation.
func generateReport(
	results []*challenge.Result,
	outputDir string,
	format string,
) error {
	if len(results) == 0 {
		return nil
	}

	var reporter report.Reporter
	var ext string

	switch strings.ToLower(format) {
	case "markdown":
		reporter = report.NewMarkdownReporter(outputDir)
		ext = "md"
	case "json":
		reporter = report.NewJSONReporter(outputDir, true)
		ext = "json"
	case "html":
		reporter = report.NewHTMLReporter(outputDir)
		ext = "html"
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Generate individual challenge reports.
	for _, res := range results {
		data, err := reporter.GenerateReport(res)
		if err != nil {
			return fmt.Errorf(
				"generate report for %s: %w",
				res.ChallengeID, err,
			)
		}

		filename := fmt.Sprintf(
			"%s.%s", res.ChallengeID, ext,
		)
		path := filepath.Join(outputDir, filename)
		if err := os.WriteFile(
			path, data, 0o644,
		); err != nil {
			return fmt.Errorf(
				"write report %s: %w", path, err,
			)
		}
	}

	// Generate master summary.
	summaryData, err := reporter.GenerateMasterSummary(
		results,
	)
	if err != nil {
		return fmt.Errorf(
			"generate master summary: %w", err,
		)
	}

	summaryPath := filepath.Join(
		outputDir,
		fmt.Sprintf("summary.%s", ext),
	)
	if err := os.WriteFile(
		summaryPath, summaryData, 0o644,
	); err != nil {
		return fmt.Errorf(
			"write summary %s: %w", summaryPath, err,
		)
	}

	return nil
}

// printSummary writes a human-readable summary to stdout.
func printSummary(
	results []*challenge.Result,
	logger *cliLogger,
) {
	if len(results) == 0 {
		logger.Info("no challenges were executed")
		return
	}

	passed := 0
	failed := 0
	skipped := 0
	errored := 0
	totalAssertions := 0
	passedAssertions := 0
	var totalDuration time.Duration

	for _, res := range results {
		totalDuration += res.Duration
		for _, a := range res.Assertions {
			totalAssertions++
			if a.Passed {
				passedAssertions++
			}
		}

		switch res.Status {
		case challenge.StatusPassed:
			passed++
		case challenge.StatusFailed:
			failed++
		case challenge.StatusSkipped:
			skipped++
		default:
			errored++
		}
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  UserFlow Runner - Results Summary")
	fmt.Println("========================================")
	fmt.Printf("  Total:      %d challenges\n",
		len(results),
	)
	fmt.Printf("  Passed:     %d\n", passed)
	fmt.Printf("  Failed:     %d\n", failed)
	fmt.Printf("  Skipped:    %d\n", skipped)
	fmt.Printf("  Errors:     %d\n", errored)
	fmt.Printf("  Assertions: %d/%d\n",
		passedAssertions, totalAssertions,
	)
	fmt.Printf("  Duration:   %v\n", totalDuration)
	fmt.Println("========================================")
	fmt.Println()

	// List failures for quick diagnosis.
	if failed+errored > 0 {
		fmt.Println("Failed/Errored Challenges:")
		for _, res := range results {
			if res.Status != challenge.StatusPassed &&
				res.Status != challenge.StatusSkipped {
				fmt.Printf(
					"  - [%s] %s: %s\n",
					strings.ToUpper(res.Status),
					res.ChallengeID,
					res.ChallengeName,
				)
				if res.Error != "" {
					fmt.Printf(
						"    Error: %s\n", res.Error,
					)
				}
			}
		}
		fmt.Println()
	}
}
