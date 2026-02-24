// SPDX-FileCopyrightText: 2025 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"digital.vasic.challenges/pkg/assertion"
	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.challenges/pkg/plugin"
	"digital.vasic.challenges/pkg/registry"
	"digital.vasic.challenges/pkg/report"
	"digital.vasic.challenges/pkg/runner"
	"digital.vasic.challenges/pkg/yole"
)

func main() {
	var (
		platform = flag.String("platform", "all",
			"Platform to test: android, desktop, web, all")
		reportFmt = flag.String("report", "markdown",
			"Report format: markdown, json, html")
		outputDir = flag.String("output", "reports",
			"Output directory for reports")
		useDocker = flag.Bool("docker", false,
			"Run Gradle tasks in Docker containers")
		timeout = flag.Duration("timeout", 30*time.Minute,
			"Global timeout for all challenges")
	)
	flag.Parse()

	projectRoot := findProjectRoot()

	fmt.Println("Yole Challenges Runner")
	fmt.Printf("Project root: %s\n", projectRoot)
	fmt.Printf("Platform: %s\n", *platform)
	fmt.Printf("Docker: %v\n", *useDocker)
	fmt.Printf("Timeout: %v\n", *timeout)
	fmt.Println()

	// Initialize plugin system
	engine := assertion.NewEngine()
	yolePlugin := yole.NewYolePlugin(engine)
	pluginReg := plugin.NewRegistry()
	_ = pluginReg.Register(yolePlugin)
	_ = pluginReg.InitAll(&plugin.PluginContext{})

	// Create adapters
	gradle := yole.NewGradleCLIAdapter(
		projectRoot, *useDocker,
	)
	adb := yole.NewADBCLIAdapter()
	process := yole.NewProcessCLIAdapter()

	// Create challenge registry
	reg := registry.NewRegistry()

	ctx, cancel := context.WithTimeout(
		context.Background(), *timeout,
	)
	defer cancel()

	// Register infrastructure challenges (always)
	_ = reg.Register(yole.NewGradleBuildChallenge(gradle))
	_ = reg.Register(yole.NewGradleTestsChallenge(gradle))
	_ = reg.Register(yole.NewLintChallenge(gradle))

	// Register platform-specific challenges
	if *platform == "all" || *platform == "android" {
		_ = reg.Register(
			yole.NewRobolectricLaunchChallenge(gradle),
		)
		_ = reg.Register(
			yole.NewRobolectricFlowsChallenge(gradle),
		)
		_ = reg.Register(
			yole.NewUIAutomatorLaunchChallenge(adb, gradle),
		)
	}
	if *platform == "all" || *platform == "desktop" {
		_ = reg.Register(
			yole.NewDesktopLaunchChallenge(gradle, process),
		)
		_ = reg.Register(
			yole.NewDesktopUserFlowsChallenge(gradle),
		)
	}
	if *platform == "all" || *platform == "web" {
		_ = reg.Register(yole.NewWebLaunchChallenge(gradle))
	}

	fmt.Printf(
		"Registered %d challenges\n\n", reg.Count(),
	)

	// Create runner
	r := runner.NewRunner(
		runner.WithRegistry(reg),
		runner.WithTimeout(*timeout),
		runner.WithResultsDir(*outputDir),
	)

	// Run all challenges
	cfg := challenge.NewConfig("yole-challenges")
	results, err := r.RunAll(ctx, cfg)
	if err != nil {
		log.Printf("Runner error: %v\n", err)
	}

	// Generate reports
	if mkErr := os.MkdirAll(*outputDir, 0755); mkErr != nil {
		log.Printf(
			"Failed to create output dir: %v\n", mkErr,
		)
	}

	switch *reportFmt {
	case "json":
		reporter := report.NewJSONReporter(*outputDir, true)
		data, genErr := reporter.GenerateMasterSummary(
			results,
		)
		if genErr == nil {
			_ = os.WriteFile(
				filepath.Join(*outputDir, "results.json"),
				data, 0644,
			)
		}
	case "html":
		reporter := report.NewHTMLReporter(*outputDir)
		data, genErr := reporter.GenerateMasterSummary(
			results,
		)
		if genErr == nil {
			_ = os.WriteFile(
				filepath.Join(*outputDir, "results.html"),
				data, 0644,
			)
		}
	default:
		reporter := report.NewMarkdownReporter(*outputDir)
		_ = reporter.SaveMasterSummary(
			results, "results.md",
		)
	}

	fmt.Printf("\nReport written to %s/\n", *outputDir)

	// Exit with non-zero if any challenge failed
	for _, r := range results {
		if r.Status == challenge.StatusFailed ||
			r.Status == challenge.StatusError {
			os.Exit(1)
		}
	}
}

func findProjectRoot() string {
	cwd, _ := os.Getwd()
	// Try walking up to find settings.gradle.kts
	dir := cwd
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(
			filepath.Join(dir, "settings.gradle.kts"),
		); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return cwd
}
