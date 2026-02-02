package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// MasterSummary represents an aggregated summary of all
// challenge runs.
type MasterSummary struct {
	ID               string             `json:"id"`
	GeneratedAt      time.Time          `json:"generated_at"`
	Challenges       []ChallengeSummary `json:"challenges"`
	TotalChallenges  int                `json:"total_challenges"`
	PassedChallenges int                `json:"passed_challenges"`
	FailedChallenges int                `json:"failed_challenges"`
	TotalDuration    time.Duration      `json:"total_duration"`
	AveragePassRate  float64            `json:"average_pass_rate"`
}

// ChallengeSummary represents a summary of a single challenge.
type ChallengeSummary struct {
	ChallengeID      challenge.ID  `json:"challenge_id"`
	ChallengeName    string        `json:"challenge_name"`
	Status           string        `json:"status"`
	Duration         time.Duration `json:"duration"`
	AssertionsPassed int           `json:"assertions_passed"`
	AssertionsTotal  int           `json:"assertions_total"`
	ResultsPath      string        `json:"results_path"`
}

// BuildMasterSummary creates a master summary from challenge
// results.
func BuildMasterSummary(
	results []*challenge.Result,
) *MasterSummary {
	summary := &MasterSummary{
		ID: fmt.Sprintf(
			"summary_%s",
			time.Now().Format("20060102_150405"),
		),
		GeneratedAt: time.Now(),
		Challenges: make(
			[]ChallengeSummary, 0, len(results),
		),
	}

	for _, r := range results {
		assertionsPassed := 0
		for _, a := range r.Assertions {
			if a.Passed {
				assertionsPassed++
			}
		}

		cs := ChallengeSummary{
			ChallengeID:      r.ChallengeID,
			ChallengeName:    r.ChallengeName,
			Status:           r.Status,
			Duration:         r.Duration,
			AssertionsPassed: assertionsPassed,
			AssertionsTotal:  len(r.Assertions),
		}

		summary.Challenges = append(summary.Challenges, cs)
		summary.TotalChallenges++
		summary.TotalDuration += r.Duration

		if r.Status == challenge.StatusPassed {
			summary.PassedChallenges++
		} else {
			summary.FailedChallenges++
		}
	}

	if summary.TotalChallenges > 0 {
		summary.AveragePassRate =
			float64(summary.PassedChallenges) /
				float64(summary.TotalChallenges)
	}

	return summary
}

// SaveMasterSummary saves the master summary to both JSON and
// Markdown files in the given output directory.
func SaveMasterSummary(
	summary *MasterSummary,
	outputDir string,
) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf(
			"failed to create output directory: %w", err,
		)
	}

	ts := summary.GeneratedAt.Format("20060102_150405")

	jsonPath := filepath.Join(
		outputDir,
		fmt.Sprintf("master_summary_%s.json", ts),
	)
	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf(
			"failed to marshal summary: %w", err,
		)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf(
			"failed to write JSON summary: %w", err,
		)
	}

	mdPath := filepath.Join(
		outputDir,
		fmt.Sprintf("master_summary_%s.md", ts),
	)
	mdContent := generateSummaryMarkdown(summary)
	if err := os.WriteFile(
		mdPath, []byte(mdContent), 0644,
	); err != nil {
		return fmt.Errorf(
			"failed to write Markdown summary: %w", err,
		)
	}

	latestJSON := filepath.Join(outputDir, "latest_summary.json")
	latestMD := filepath.Join(outputDir, "latest_summary.md")

	_ = os.Remove(latestJSON)
	_ = os.Remove(latestMD)
	_ = os.Symlink(filepath.Base(jsonPath), latestJSON)
	_ = os.Symlink(filepath.Base(mdPath), latestMD)

	return nil
}

// generateSummaryMarkdown creates markdown from a master
// summary.
func generateSummaryMarkdown(summary *MasterSummary) string {
	var sb strings.Builder

	sb.WriteString(
		"# Challenges Framework - Master Summary\n\n",
	)
	sb.WriteString(
		fmt.Sprintf(
			"**Summary ID:** %s\n\n", summary.ID,
		),
	)
	sb.WriteString(
		fmt.Sprintf(
			"**Generated:** %s\n\n",
			summary.GeneratedAt.Format(time.RFC3339),
		),
	)

	sb.WriteString("## Overview\n\n")
	sb.WriteString(
		"| Challenge | Status | Duration " +
			"| Assertions |\n",
	)
	sb.WriteString(
		"|-----------|--------|----------" +
			"|------------|\n",
	)

	for _, c := range summary.Challenges {
		status := strings.ToUpper(c.Status)
		assertions := fmt.Sprintf(
			"%d/%d", c.AssertionsPassed, c.AssertionsTotal,
		)
		sb.WriteString(
			fmt.Sprintf(
				"| %s | %s | %v | %s |\n",
				c.ChallengeName, status,
				c.Duration, assertions,
			),
		)
	}

	sb.WriteString("\n## Statistics\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(
		fmt.Sprintf(
			"| Total Challenges | %d |\n",
			summary.TotalChallenges,
		),
	)
	sb.WriteString(
		fmt.Sprintf(
			"| Passed | %d |\n", summary.PassedChallenges,
		),
	)
	sb.WriteString(
		fmt.Sprintf(
			"| Failed | %d |\n", summary.FailedChallenges,
		),
	)
	sb.WriteString(
		fmt.Sprintf(
			"| Pass Rate | %.0f%% |\n",
			summary.AveragePassRate*100,
		),
	)
	sb.WriteString(
		fmt.Sprintf(
			"| Total Duration | %v |\n",
			summary.TotalDuration,
		),
	)

	sb.WriteString("\n---\n\n")
	sb.WriteString("*Generated by Challenges Framework*\n")

	return sb.String()
}
