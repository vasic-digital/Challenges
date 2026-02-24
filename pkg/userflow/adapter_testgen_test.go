package userflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface check.
var _ TestGenAdapter = (*PanopticTestGenAdapter)(nil)

func TestGeneratedTest_Fields(t *testing.T) {
	gt := GeneratedTest{
		Name:       "login_flow",
		Category:   "authentication",
		Priority:   "high",
		Confidence: 0.92,
		Steps: []TestStep{
			{
				Action: "click",
				Target: "button#login",
				Value:  "",
				Parameters: map[string]any{
					"timeout": "5s",
				},
			},
			{
				Action: "fill",
				Target: "input#email",
				Value:  "user@example.com",
			},
		},
	}

	assert.Equal(t, "login_flow", gt.Name)
	assert.Equal(t, "authentication", gt.Category)
	assert.Equal(t, "high", gt.Priority)
	assert.InDelta(t, 0.92, gt.Confidence, 0.001)
	assert.Len(t, gt.Steps, 2)

	assert.Equal(t, "click", gt.Steps[0].Action)
	assert.Equal(t, "button#login", gt.Steps[0].Target)
	assert.Equal(t, "", gt.Steps[0].Value)
	assert.Equal(
		t, "5s", gt.Steps[0].Parameters["timeout"],
	)

	assert.Equal(t, "fill", gt.Steps[1].Action)
	assert.Equal(t, "input#email", gt.Steps[1].Target)
	assert.Equal(
		t, "user@example.com", gt.Steps[1].Value,
	)
	assert.Nil(t, gt.Steps[1].Parameters)
}

func TestTestStep_ZeroValue(t *testing.T) {
	var s TestStep
	assert.Empty(t, s.Action)
	assert.Empty(t, s.Target)
	assert.Empty(t, s.Value)
	assert.Nil(t, s.Parameters)
}

func TestPanopticTestGenAdapter_Constructor(t *testing.T) {
	adapter := NewPanopticTestGenAdapter(
		"/usr/bin/panoptic",
	)
	assert.NotNil(t, adapter)
	assert.Equal(
		t, "/usr/bin/panoptic", adapter.binaryPath,
	)
}

func TestPanopticTestGenAdapter_Available_NotFound(
	t *testing.T,
) {
	adapter := NewPanopticTestGenAdapter(
		"/nonexistent/path/to/panoptic-binary-xyz",
	)
	assert.False(
		t, adapter.Available(context.Background()),
	)
}

func TestPanopticTestGenAdapter_Available_ExistingBinary(
	t *testing.T,
) {
	// /bin/sh exists on virtually all systems.
	adapter := NewPanopticTestGenAdapter("/bin/sh")
	assert.True(
		t, adapter.Available(context.Background()),
	)
}

func TestPanopticGeneratedTest_ToGeneratedTest(t *testing.T) {
	raw := panopticGeneratedTest{
		Name:       "submit_form",
		Category:   "forms",
		Priority:   "medium",
		Confidence: 0.85,
		Steps: []panopticTestStep{
			{
				Action: "fill",
				Target: "input#name",
				Value:  "Jane",
				Parameters: map[string]string{
					"clear": "true",
				},
			},
			{
				Action: "click",
				Target: "button#submit",
			},
		},
	}

	gt := raw.toGeneratedTest()
	assert.Equal(t, "submit_form", gt.Name)
	assert.Equal(t, "forms", gt.Category)
	assert.Equal(t, "medium", gt.Priority)
	assert.InDelta(t, 0.85, gt.Confidence, 0.001)
	assert.Len(t, gt.Steps, 2)

	assert.Equal(t, "fill", gt.Steps[0].Action)
	assert.Equal(t, "input#name", gt.Steps[0].Target)
	assert.Equal(t, "Jane", gt.Steps[0].Value)
	assert.Equal(
		t, "true", gt.Steps[0].Parameters["clear"],
	)

	assert.Equal(t, "click", gt.Steps[1].Action)
	assert.Equal(
		t, "button#submit", gt.Steps[1].Target,
	)
	assert.Nil(t, gt.Steps[1].Parameters)
}

func TestPanopticGeneratedTest_ToGeneratedTest_NoParams(
	t *testing.T,
) {
	raw := panopticGeneratedTest{
		Name:       "simple",
		Category:   "smoke",
		Priority:   "low",
		Confidence: 0.99,
		Steps: []panopticTestStep{
			{
				Action: "navigate",
				Target: "/home",
			},
		},
	}

	gt := raw.toGeneratedTest()
	assert.Len(t, gt.Steps, 1)
	assert.Nil(t, gt.Steps[0].Parameters)
}

// mockTestGenAdapter is a test double implementing
// TestGenAdapter with configurable responses.
type mockTestGenAdapter struct {
	tests     []GeneratedTest
	report    string
	available bool
}

var _ TestGenAdapter = (*mockTestGenAdapter)(nil)

func (m *mockTestGenAdapter) GenerateTests(
	_ context.Context, _ []byte,
) ([]GeneratedTest, error) {
	return m.tests, nil
}

func (m *mockTestGenAdapter) GenerateReport(
	_ context.Context, _ []byte,
) (string, error) {
	return m.report, nil
}

func (m *mockTestGenAdapter) Available(
	_ context.Context,
) bool {
	return m.available
}

func TestMockTestGenAdapter_GenerateTests(t *testing.T) {
	mock := &mockTestGenAdapter{
		tests: []GeneratedTest{
			{
				Name:       "test1",
				Category:   "smoke",
				Priority:   "high",
				Confidence: 0.95,
				Steps: []TestStep{
					{Action: "click", Target: "#btn"},
				},
			},
		},
		available: true,
	}

	ctx := context.Background()
	tests, err := mock.GenerateTests(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, tests, 1)
	assert.Equal(t, "test1", tests[0].Name)
}

func TestMockTestGenAdapter_GenerateReport(t *testing.T) {
	mock := &mockTestGenAdapter{
		report:    "# Test Report\n\nAll passed.",
		available: true,
	}

	ctx := context.Background()
	report, err := mock.GenerateReport(ctx, nil)
	assert.NoError(t, err)
	assert.Contains(t, report, "Test Report")
}

func TestMockTestGenAdapter_Available(t *testing.T) {
	mock := &mockTestGenAdapter{available: true}
	assert.True(
		t, mock.Available(context.Background()),
	)

	mock.available = false
	assert.False(
		t, mock.Available(context.Background()),
	)
}
