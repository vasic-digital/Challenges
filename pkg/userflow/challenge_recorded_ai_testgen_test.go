package userflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

// mockTestGenForRecording implements TestGenAdapter for
// recorded AI test generation tests with configurable
// availability and error responses.
type mockTestGenForRecording struct {
	available bool
	tests     []GeneratedTest
	err       error
}

func (m *mockTestGenForRecording) GenerateTests(
	_ context.Context, _ []byte,
) ([]GeneratedTest, error) {
	return m.tests, m.err
}

func (m *mockTestGenForRecording) GenerateReport(
	_ context.Context, _ []byte,
) (string, error) {
	return "", nil
}

func (m *mockTestGenForRecording) Available(
	_ context.Context,
) bool {
	return m.available
}

// --- RecordedAITestGenChallenge tests ---

func TestNewRecordedAITestGenChallenge_Constructor(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	testgen := &mockTestGenForRecording{
		available: true,
	}
	deps := []challenge.ID{"SETUP-001", "HEALTH-001"}

	ch := NewRecordedAITestGenChallenge(
		"RECAI-001", "Recorded AI TestGen",
		"Recorded AI test generation",
		deps, browser, recorder, testgen,
		"http://localhost:3000", 10, "/tmp/output",
	)

	assert.Equal(
		t, challenge.ID("RECAI-001"), ch.ID(),
	)
	assert.Equal(
		t, "Recorded AI TestGen", ch.Name(),
	)
	assert.Equal(
		t, "Recorded AI test generation",
		ch.Description(),
	)
	assert.Equal(t, deps, ch.Dependencies())
	assert.NotNil(t, ch.browser)
	assert.NotNil(t, ch.recorder)
	assert.NotNil(t, ch.testgen)
	assert.Equal(
		t, "http://localhost:3000", ch.targetURL,
	)
	assert.Equal(t, 10, ch.maxTests)
	assert.Equal(t, "/tmp/output", ch.outputDir)
}

func TestRecordedAITestGenChallenge_Category(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	testgen := &mockTestGenForRecording{
		available: true,
	}

	ch := NewRecordedAITestGenChallenge(
		"RECAI-002", "Category Test",
		"Check category",
		nil, browser, recorder, testgen,
		"http://localhost:3000", 5, "",
	)

	assert.Equal(t, "ai", ch.Category())
}

func TestRecordedAITestGenChallenge_Execute_Success(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	testgen := &mockTestGenForRecording{
		available: true,
		tests: []GeneratedTest{
			{
				Name:       "login_flow",
				Category:   "authentication",
				Priority:   "high",
				Confidence: 0.95,
				Steps: []TestStep{
					{
						Action: "click",
						Target: "#login",
					},
				},
			},
			{
				Name:       "navigation_test",
				Category:   "navigation",
				Priority:   "medium",
				Confidence: 0.88,
				Steps: []TestStep{
					{
						Action: "navigate",
						Target: "/home",
					},
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
	}

	tmpDir := t.TempDir()

	ch := NewRecordedAITestGenChallenge(
		"RECAI-003", "Recorded AI Gen",
		"Full flow with recording and AI",
		nil, browser, recorder, testgen,
		"http://localhost:3000",
		10, tmpDir,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusPassed, result.Status,
	)

	// Verify browser interactions.
	assert.True(t, browser.initialized)
	assert.Equal(
		t, "http://localhost:3000",
		browser.navigated,
	)
	assert.True(t, browser.closed)

	// Verify recorder interactions.
	assert.True(t, recorder.started)
	assert.True(t, recorder.stopped)
	assert.Equal(
		t, "http://localhost:3000",
		recorder.config.URL,
	)

	// Verify recording metrics.
	vidDur, ok := result.Metrics["video_duration"]
	assert.True(t, ok)
	assert.Equal(t, 5.0, vidDur.Value)

	vidFrames, ok :=
		result.Metrics["video_frame_count"]
	assert.True(t, ok)
	assert.Equal(t, 150.0, vidFrames.Value)

	vidSize, ok := result.Metrics["video_file_size"]
	assert.True(t, ok)
	assert.Equal(t, 1024000.0, vidSize.Value)

	// Verify recording outputs.
	assert.Equal(
		t, "/tmp/recording.webm",
		result.Outputs["video_path"],
	)

	// Verify test generation metrics.
	testsMetric, ok :=
		result.Metrics["tests_generated"]
	require.True(t, ok)
	assert.Equal(t, 3.0, testsMetric.Value)

	catMetric, ok :=
		result.Metrics["test_categories"]
	require.True(t, ok)
	assert.Equal(t, 3.0, catMetric.Value)

	covMetric, ok :=
		result.Metrics["test_coverage"]
	require.True(t, ok)
	// Average: (0.95 + 0.88 + 0.91) / 3 ~ 0.9133.
	assert.InDelta(
		t, 0.9133, covMetric.Value, 0.001,
	)

	durMetric, ok :=
		result.Metrics["total_duration"]
	require.True(t, ok)
	assert.Greater(t, durMetric.Value, 0.0)

	// Verify outputs.
	assert.Equal(
		t, "3", result.Outputs["tests_generated"],
	)
	assert.Equal(
		t, "3", result.Outputs["test_categories"],
	)

	// Verify output file was written.
	outPath := filepath.Join(
		tmpDir, "generated_tests.json",
	)
	assert.Equal(
		t, outPath, result.Outputs["output_file"],
	)

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

	// Verify recording assertions are present.
	hasRecStart := false
	hasRecIntegrity := false
	for _, a := range result.Assertions {
		if a.Target == "start_recording" {
			hasRecStart = true
			assert.True(t, a.Passed)
		}
		if a.Target == "video_integrity" {
			hasRecIntegrity = true
			assert.True(t, a.Passed)
		}
	}
	assert.True(
		t, hasRecStart,
		"missing recording start assertion",
	)
	assert.True(
		t, hasRecIntegrity,
		"missing recording integrity assertion",
	)
}

func TestRecordedAITestGenChallenge_Execute_BrowserUnavailable(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	browser.available = false
	recorder := newMockRecorderForFlow()
	testgen := &mockTestGenForRecording{
		available: true,
	}

	ch := NewRecordedAITestGenChallenge(
		"RECAI-004", "Browser Unavail",
		"Browser not available",
		nil, browser, recorder, testgen,
		"http://localhost:3000", 5, "",
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error, "browser not available",
	)

	// Recorder should not have been started.
	assert.False(t, recorder.started)
	assert.False(t, browser.initialized)
}

func TestRecordedAITestGenChallenge_Execute_RecorderUnavailable(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	recorder.available = false
	testgen := &mockTestGenForRecording{
		available: true,
	}

	ch := NewRecordedAITestGenChallenge(
		"RECAI-005", "Recorder Unavail",
		"Recorder not available",
		nil, browser, recorder, testgen,
		"http://localhost:3000", 5, "",
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error, "recorder not available",
	)

	// Browser should not have been initialized.
	assert.False(t, browser.initialized)
	assert.False(t, recorder.started)
}

func TestRecordedAITestGenChallenge_Execute_TestGenUnavailable(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	testgen := &mockTestGenForRecording{
		available: false,
	}

	ch := NewRecordedAITestGenChallenge(
		"RECAI-006", "TestGen Unavail",
		"TestGen not available",
		nil, browser, recorder, testgen,
		"http://localhost:3000", 5, "",
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusSkipped, result.Status,
	)
	assert.Contains(
		t, result.Error, "testgen not available",
	)

	// Browser and recorder should not have been started.
	assert.False(t, browser.initialized)
	assert.False(t, recorder.started)
}

func TestRecordedAITestGenChallenge_Execute_RecordingStartFailure(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	recorder.startErr = fmt.Errorf(
		"ffmpeg not installed",
	)
	testgen := &mockTestGenForRecording{
		available: true,
	}

	ch := NewRecordedAITestGenChallenge(
		"RECAI-007", "Rec Start Fail",
		"Recording start fails",
		nil, browser, recorder, testgen,
		"http://localhost:3000", 5, "",
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusFailed, result.Status,
	)
	assert.Contains(
		t, result.Error, "ffmpeg not installed",
	)
}

func TestRecordedAITestGenChallenge_Execute_ZeroIntegrity(
	t *testing.T,
) {
	browser := newMockBrowserForRecording()
	recorder := newMockRecorderForFlow()
	recorder.result = &RecordingResult{
		FilePath:   "/tmp/empty.webm",
		Duration:   0,
		FrameCount: 0,
		FileSize:   0,
	}
	testgen := &mockTestGenForRecording{
		available: true,
		tests: []GeneratedTest{
			{
				Name:       "test1",
				Category:   "smoke",
				Confidence: 0.9,
			},
		},
	}

	ch := NewRecordedAITestGenChallenge(
		"RECAI-008", "Zero Integrity",
		"Recording with zero integrity",
		nil, browser, recorder, testgen,
		"http://localhost:3000", 5, "",
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(
		t, challenge.StatusFailed, result.Status,
	)

	// Integrity assertion should fail.
	hasIntegrity := false
	for _, a := range result.Assertions {
		if a.Target == "video_integrity" {
			hasIntegrity = true
			assert.False(t, a.Passed)
		}
	}
	assert.True(
		t, hasIntegrity,
		"missing video_integrity assertion",
	)
}
