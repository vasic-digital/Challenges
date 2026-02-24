package userflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

func TestAITestGenerationChallenge_Execute_Success(
	t *testing.T,
) {
	browser := newMockBrowserAdapter()
	testgen := &mockTestGenAdapter{
		tests: []GeneratedTest{
			{
				Name:       "login_flow",
				Category:   "authentication",
				Priority:   "high",
				Confidence: 0.95,
				Steps: []TestStep{
					{Action: "click", Target: "#login"},
				},
			},
			{
				Name:       "navigation_test",
				Category:   "navigation",
				Priority:   "medium",
				Confidence: 0.88,
				Steps: []TestStep{
					{Action: "navigate", Target: "/home"},
				},
			},
			{
				Name:       "form_validation",
				Category:   "forms",
				Priority:   "high",
				Confidence: 0.91,
				Steps: []TestStep{
					{
						Action: "fill",
						Target: "#name",
						Value:  "Jane",
					},
				},
			},
		},
		available: true,
	}

	tmpDir := t.TempDir()

	ch := NewAITestGenerationChallenge(
		"AIGEN-001", "AI Test Gen",
		"Generate tests from screenshot",
		nil, browser, testgen,
		"http://localhost:3000",
		10, tmpDir,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)

	// Check metrics.
	testsMetric, ok := result.Metrics["tests_generated"]
	require.True(t, ok)
	assert.Equal(t, 3.0, testsMetric.Value)

	catMetric, ok := result.Metrics["test_categories"]
	require.True(t, ok)
	assert.Equal(t, 3.0, catMetric.Value)

	covMetric, ok := result.Metrics["test_coverage"]
	require.True(t, ok)
	// Average: (0.95 + 0.88 + 0.91) / 3 ≈ 0.9133.
	assert.InDelta(t, 0.9133, covMetric.Value, 0.001)

	durMetric, ok := result.Metrics["total_duration"]
	require.True(t, ok)
	assert.Greater(t, durMetric.Value, 0.0)

	// Check outputs.
	assert.Equal(t, "3", result.Outputs["tests_generated"])
	assert.Equal(
		t, "3", result.Outputs["test_categories"],
	)

	// Verify output file was written.
	outPath := filepath.Join(
		tmpDir, "generated_tests.json",
	)
	assert.Equal(t, outPath, result.Outputs["output_file"])

	data, readErr := os.ReadFile(outPath)
	require.NoError(t, readErr)

	var saved []GeneratedTest
	require.NoError(t, json.Unmarshal(data, &saved))
	assert.Len(t, saved, 3)
	assert.Equal(t, "login_flow", saved[0].Name)
	assert.Equal(
		t, "navigation_test", saved[1].Name,
	)
	assert.Equal(
		t, "form_validation", saved[2].Name,
	)

	// Check assertions.
	require.Len(t, result.Assertions, 2)
	assert.True(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Message, "3 test(s)",
	)
	assert.True(t, result.Assertions[1].Passed)

	// Browser was used.
	assert.True(t, browser.closed)
	require.Len(t, browser.navigatedURLs, 1)
	assert.Equal(
		t, "http://localhost:3000",
		browser.navigatedURLs[0],
	)
	assert.GreaterOrEqual(t, browser.screenshotCount, 1)
}

func TestAITestGenerationChallenge_Execute_BrowserUnavailable(
	t *testing.T,
) {
	browser := &mockBrowserUnavailable{}
	testgen := &mockTestGenAdapter{
		available: true,
	}

	ch := NewAITestGenerationChallenge(
		"AIGEN-002", "AI Unavailable",
		"Browser unavailable - skipped",
		nil, browser, testgen,
		"http://localhost:3000", 5, "",
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.Contains(
		t, result.Assertions[0].Message,
		"Browser not available",
	)
}

func TestAITestGenerationChallenge_Execute_TestGenUnavailable(
	t *testing.T,
) {
	browser := newMockBrowserAdapter()
	testgen := &mockTestGenAdapter{
		available: false,
	}

	ch := NewAITestGenerationChallenge(
		"AIGEN-003", "TestGen Unavailable",
		"TestGen unavailable - skipped",
		nil, browser, testgen,
		"http://localhost:3000", 5, "",
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.Contains(
		t, result.Assertions[0].Message,
		"TestGen not available",
	)
}

func TestNewAITestGenerationChallenge_Constructor(
	t *testing.T,
) {
	browser := newMockBrowserAdapter()
	testgen := &mockTestGenAdapter{available: true}
	deps := []challenge.ID{"BROWSER-SETUP"}

	ch := NewAITestGenerationChallenge(
		"AIGEN-004", "AI Constructor",
		"Verify constructor fields",
		deps, browser, testgen,
		"http://localhost:8080/app",
		20, "/tmp/output",
	)

	assert.Equal(
		t, challenge.ID("AIGEN-004"), ch.ID(),
	)
	assert.Equal(t, "AI Constructor", ch.Name())
	assert.Equal(t, "ai", ch.Category())
	assert.Equal(t, deps, ch.Dependencies())
	assert.NotNil(t, ch.browser)
	assert.NotNil(t, ch.testgen)
	assert.Equal(
		t, "http://localhost:8080/app", ch.targetURL,
	)
	assert.Equal(t, 20, ch.maxTests)
	assert.Equal(t, "/tmp/output", ch.outputDir)
}

func TestAITestGenerationChallenge_Execute_MaxTestsCap(
	t *testing.T,
) {
	browser := newMockBrowserAdapter()
	testgen := &mockTestGenAdapter{
		tests: []GeneratedTest{
			{
				Name:       "test1",
				Category:   "smoke",
				Confidence: 0.9,
			},
			{
				Name:       "test2",
				Category:   "smoke",
				Confidence: 0.85,
			},
			{
				Name:       "test3",
				Category:   "regression",
				Confidence: 0.8,
			},
		},
		available: true,
	}

	tmpDir := t.TempDir()

	ch := NewAITestGenerationChallenge(
		"AIGEN-005", "AI MaxTests",
		"Cap generated tests to maxTests",
		nil, browser, testgen,
		"http://localhost:3000",
		2, tmpDir,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)

	// Only 2 tests should be saved.
	testsMetric := result.Metrics["tests_generated"]
	assert.Equal(t, 2.0, testsMetric.Value)

	assert.Equal(t, "2", result.Outputs["tests_generated"])

	// Read saved file and verify cap.
	outPath := filepath.Join(
		tmpDir, "generated_tests.json",
	)
	data, readErr := os.ReadFile(outPath)
	require.NoError(t, readErr)

	var saved []GeneratedTest
	require.NoError(t, json.Unmarshal(data, &saved))
	assert.Len(t, saved, 2)
}

func TestAITestGenerationChallenge_Execute_NoOutputDir(
	t *testing.T,
) {
	browser := newMockBrowserAdapter()
	testgen := &mockTestGenAdapter{
		tests: []GeneratedTest{
			{
				Name:       "test1",
				Category:   "smoke",
				Confidence: 0.9,
			},
		},
		available: true,
	}

	ch := NewAITestGenerationChallenge(
		"AIGEN-006", "AI No Output",
		"No output dir - skip file write",
		nil, browser, testgen,
		"http://localhost:3000",
		10, "",
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	assert.Equal(t, "1", result.Outputs["tests_generated"])
	// No output_file should be set.
	_, hasFile := result.Outputs["output_file"]
	assert.False(t, hasFile)
}
