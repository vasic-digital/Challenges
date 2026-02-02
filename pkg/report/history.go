package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// HistoricalEntry represents a single challenge run in the
// historical log.
type HistoricalEntry struct {
	Timestamp        time.Time `json:"timestamp"`
	ChallengeID      string    `json:"challenge_id"`
	Status           string    `json:"status"`
	Duration         string    `json:"duration"`
	AssertionsPassed int       `json:"assertions_passed"`
	AssertionsTotal  int       `json:"assertions_total"`
	ResultsPath      string    `json:"results_path"`
}

// AppendToHistory adds an entry to the historical log stored
// at historyPath. Each entry is a single JSON line.
func AppendToHistory(
	historyPath string,
	result *challenge.Result,
	resultsPath string,
) error {
	assertionsPassed := 0
	for _, a := range result.Assertions {
		if a.Passed {
			assertionsPassed++
		}
	}

	entry := HistoricalEntry{
		Timestamp:        result.EndTime,
		ChallengeID:      string(result.ChallengeID),
		Status:           result.Status,
		Duration:         result.Duration.String(),
		AssertionsPassed: assertionsPassed,
		AssertionsTotal:  len(result.Assertions),
		ResultsPath:      resultsPath,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf(
			"failed to marshal history entry: %w", err,
		)
	}

	file, err := os.OpenFile(
		historyPath,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to open history file: %w", err,
		)
	}
	defer func() { _ = file.Close() }()

	_, err = fmt.Fprintln(file, string(data))
	return err
}
