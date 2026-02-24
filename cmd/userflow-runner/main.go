// Package main provides the userflow-runner CLI entry point.
// It orchestrates multi-platform user flow testing by wiring
// TestEnvironment, challenge registration, and the runner.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	platform := flag.String(
		"platform", "all",
		"Platform to test "+
			"(all, api, web, desktop, wizard, android, tv)",
	)
	report := flag.String(
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
	flag.Parse()

	ctx := context.Background()

	fmt.Printf("UserFlow Runner\n")
	fmt.Printf("Platform: %s\n", *platform)
	fmt.Printf("Report:   %s\n", *report)
	fmt.Printf("Compose:  %s\n", *composeFile)
	fmt.Printf("Root:     %s\n", *projectRoot)

	// TODO: Wire up TestEnvironment, challenge registration,
	// and runner. This will be connected in Phase 5.

	_ = ctx
	os.Exit(0)
}
