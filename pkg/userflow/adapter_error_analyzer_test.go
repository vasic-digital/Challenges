package userflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface check.
var _ ErrorAnalyzerAdapter = (*PanopticErrorAnalyzerAdapter)(nil)

func TestErrorAnalysis_Fields(t *testing.T) {
	analysis := ErrorAnalysis{
		TotalErrors: 5,
		Categories: map[string]int{
			"network":    2,
			"validation": 3,
		},
		Severity: map[string]int{
			"critical": 1,
			"warning":  4,
		},
		Recommendations: []ErrorRecommendation{
			{
				Type:     "retry",
				Priority: "high",
				Message:  "Add retry logic for network calls",
				Impact:   "Reduces transient failures by 80%",
			},
			{
				Type:     "validation",
				Priority: "medium",
				Message:  "Add input sanitization",
				Impact:   "Prevents 3 validation errors",
			},
		},
	}

	assert.Equal(t, 5, analysis.TotalErrors)
	assert.Equal(t, 2, analysis.Categories["network"])
	assert.Equal(
		t, 3, analysis.Categories["validation"],
	)
	assert.Equal(t, 1, analysis.Severity["critical"])
	assert.Equal(t, 4, analysis.Severity["warning"])
	assert.Len(t, analysis.Recommendations, 2)

	assert.Equal(
		t, "retry", analysis.Recommendations[0].Type,
	)
	assert.Equal(
		t, "high", analysis.Recommendations[0].Priority,
	)
	assert.Contains(
		t,
		analysis.Recommendations[0].Message,
		"retry logic",
	)
	assert.Contains(
		t,
		analysis.Recommendations[0].Impact,
		"80%",
	)

	assert.Equal(
		t,
		"validation",
		analysis.Recommendations[1].Type,
	)
	assert.Equal(
		t,
		"medium",
		analysis.Recommendations[1].Priority,
	)
}

func TestErrorRecommendation_ZeroValue(t *testing.T) {
	var r ErrorRecommendation
	assert.Empty(t, r.Type)
	assert.Empty(t, r.Priority)
	assert.Empty(t, r.Message)
	assert.Empty(t, r.Impact)
}

func TestPanopticErrorAnalyzerAdapter_Constructor(
	t *testing.T,
) {
	adapter := NewPanopticErrorAnalyzerAdapter(
		"/usr/bin/panoptic",
	)
	assert.NotNil(t, adapter)
	assert.Equal(
		t, "/usr/bin/panoptic", adapter.binaryPath,
	)
}

func TestPanopticErrorAnalyzerAdapter_Available_NotFound(
	t *testing.T,
) {
	adapter := NewPanopticErrorAnalyzerAdapter(
		"/nonexistent/path/to/panoptic-binary-xyz",
	)
	assert.False(
		t, adapter.Available(context.Background()),
	)
}

func TestPanopticErrorAnalyzerAdapter_Available_ExistingBinary(
	t *testing.T,
) {
	// /bin/sh exists on virtually all systems.
	adapter := NewPanopticErrorAnalyzerAdapter("/bin/sh")
	assert.True(
		t, adapter.Available(context.Background()),
	)
}

func TestPanopticErrorAnalysis_ToErrorAnalysis(t *testing.T) {
	raw := panopticErrorAnalysis{
		TotalErrors: 3,
		Categories: map[string]int{
			"timeout": 2,
			"auth":    1,
		},
		Severity: map[string]int{
			"error":   2,
			"warning": 1,
		},
		Recommendations: []panopticErrorRecommendation{
			{
				Type:     "timeout",
				Priority: "high",
				Message:  "Increase connection timeout",
				Impact:   "Fixes 2 timeout errors",
			},
		},
	}

	analysis := raw.toErrorAnalysis()
	assert.Equal(t, 3, analysis.TotalErrors)
	assert.Equal(t, 2, analysis.Categories["timeout"])
	assert.Equal(t, 1, analysis.Categories["auth"])
	assert.Equal(t, 2, analysis.Severity["error"])
	assert.Equal(t, 1, analysis.Severity["warning"])
	assert.Len(t, analysis.Recommendations, 1)
	assert.Equal(
		t, "timeout", analysis.Recommendations[0].Type,
	)
	assert.Equal(
		t, "high", analysis.Recommendations[0].Priority,
	)
	assert.Equal(
		t,
		"Increase connection timeout",
		analysis.Recommendations[0].Message,
	)
	assert.Equal(
		t,
		"Fixes 2 timeout errors",
		analysis.Recommendations[0].Impact,
	)
}

func TestPanopticErrorAnalysis_ToErrorAnalysis_Empty(
	t *testing.T,
) {
	raw := panopticErrorAnalysis{
		TotalErrors: 0,
		Categories:  map[string]int{},
		Severity:    map[string]int{},
	}

	analysis := raw.toErrorAnalysis()
	assert.Equal(t, 0, analysis.TotalErrors)
	assert.Empty(t, analysis.Categories)
	assert.Empty(t, analysis.Severity)
	assert.Empty(t, analysis.Recommendations)
}

// mockErrorAnalyzerAdapter is a test double implementing
// ErrorAnalyzerAdapter with configurable responses.
type mockErrorAnalyzerAdapter struct {
	analysis  *ErrorAnalysis
	available bool
}

var _ ErrorAnalyzerAdapter = (*mockErrorAnalyzerAdapter)(nil)

func (m *mockErrorAnalyzerAdapter) AnalyzeErrors(
	_ context.Context, _ string,
) (*ErrorAnalysis, error) {
	return m.analysis, nil
}

func (m *mockErrorAnalyzerAdapter) Available(
	_ context.Context,
) bool {
	return m.available
}

func TestMockErrorAnalyzerAdapter_AnalyzeErrors(
	t *testing.T,
) {
	mock := &mockErrorAnalyzerAdapter{
		analysis: &ErrorAnalysis{
			TotalErrors: 2,
			Categories: map[string]int{
				"network": 2,
			},
			Severity: map[string]int{
				"error": 2,
			},
			Recommendations: []ErrorRecommendation{
				{
					Type:     "retry",
					Priority: "high",
					Message:  "Add retries",
					Impact:   "Fixes flaky calls",
				},
			},
		},
		available: true,
	}

	ctx := context.Background()
	analysis, err := mock.AnalyzeErrors(
		ctx, "ERROR: connection refused",
	)
	assert.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.Equal(t, 2, analysis.TotalErrors)
	assert.Len(t, analysis.Recommendations, 1)
}

func TestMockErrorAnalyzerAdapter_Available(t *testing.T) {
	mock := &mockErrorAnalyzerAdapter{available: true}
	assert.True(
		t, mock.Available(context.Background()),
	)

	mock.available = false
	assert.False(
		t, mock.Available(context.Background()),
	)
}
