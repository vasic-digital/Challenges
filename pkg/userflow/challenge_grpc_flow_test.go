package userflow

import (
	"context"
	"fmt"
	"testing"

	"digital.vasic.challenges/pkg/challenge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ challenge.Challenge = (*GRPCFlowChallenge)(nil)

func TestNewGRPCFlowChallenge(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		chName      string
		description string
		deps        []challenge.ID
		flow        GRPCFlow
	}{
		{
			name:        "basic_challenge",
			id:          "grpc-001",
			chName:      "gRPC Health Check",
			description: "Verify gRPC health endpoint",
			deps:        nil,
			flow: GRPCFlow{
				ServerAddr: "localhost:50051",
				Steps: []GRPCStep{
					{
						Name:   "health",
						Method: "grpc.health.v1.Health/Check",
					},
				},
			},
		},
		{
			name:        "challenge_with_deps",
			id:          "grpc-002",
			chName:      "gRPC List Services",
			description: "List all gRPC services",
			deps: []challenge.ID{
				"grpc-001",
			},
			flow: GRPCFlow{
				ServerAddr: "localhost:50051",
			},
		},
		{
			name:        "challenge_with_options",
			id:          "grpc-003",
			chName:      "gRPC With Auth",
			description: "Test with auth headers",
			deps:        nil,
			flow: GRPCFlow{
				ServerAddr: "localhost:50051",
				Options: GRPCFlowOptions{
					Insecure: true,
					Headers: map[string]string{
						"Authorization": "Bearer tk",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewGRPCCLIAdapter(
				tt.flow.ServerAddr,
			)
			ch := NewGRPCFlowChallenge(
				tt.id,
				tt.chName,
				tt.description,
				tt.deps,
				adapter,
				tt.flow,
			)
			require.NotNil(t, ch)
			assert.Equal(
				t,
				challenge.ID(tt.id),
				ch.ID(),
			)
			assert.Equal(t, tt.chName, ch.Name())
			assert.Equal(
				t, tt.description, ch.Description(),
			)
			assert.Equal(t, "grpc", ch.Category())
			assert.NotNil(t, ch.adapter)
			assert.Equal(t, tt.flow, ch.flow)
		})
	}
}

func TestGRPCFlowChallenge_Execute_UnavailableAdapter(
	t *testing.T,
) {
	adapter := NewGRPCCLIAdapter("localhost:19999")
	flow := GRPCFlow{
		ServerAddr: "localhost:19999",
		Steps: []GRPCStep{
			{
				Name:   "test",
				Method: "test.Service/Method",
			},
		},
	}

	ch := NewGRPCFlowChallenge(
		"grpc-unavail",
		"Test Unavailable",
		"Test with unavailable server",
		nil,
		adapter,
		flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	// When adapter is not available, the challenge
	// passes with a skip message.
	assert.Equal(
		t, challenge.StatusPassed, result.Status,
	)
	assert.Len(t, result.Assertions, 1)
	assert.Equal(
		t,
		"infrastructure",
		result.Assertions[0].Type,
	)
	assert.True(t, result.Assertions[0].Passed)
}

func TestValidateGRPCFields(t *testing.T) {
	tests := []struct {
		name     string
		stepName string
		response string
		expected map[string]interface{}
		wantPass int
		wantFail int
	}{
		{
			name:     "field_exists_and_matches",
			stepName: "step1",
			response: `{"status":"SERVING"}`,
			expected: map[string]interface{}{
				"status": "SERVING",
			},
			wantPass: 1,
		},
		{
			name:     "field_exists_nil_expected",
			stepName: "step2",
			response: `{"id":"123"}`,
			expected: map[string]interface{}{
				"id": nil,
			},
			wantPass: 1,
		},
		{
			name:     "field_missing",
			stepName: "step3",
			response: `{"other":"value"}`,
			expected: map[string]interface{}{
				"missing_field": "expected",
			},
			wantFail: 1,
		},
		{
			name:     "field_value_mismatch",
			stepName: "step4",
			response: `{"status":"NOT_SERVING"}`,
			expected: map[string]interface{}{
				"status": "SERVING",
			},
			wantFail: 1,
		},
		{
			name:     "invalid_json",
			stepName: "step5",
			response: "not-json",
			expected: map[string]interface{}{
				"key": "val",
			},
			wantFail: 1,
		},
		{
			name:     "multiple_fields",
			stepName: "step6",
			response: `{"a":"1","b":"2","c":"3"}`,
			expected: map[string]interface{}{
				"a": "1",
				"b": "2",
				"d": "4",
			},
			wantPass: 2,
			wantFail: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateGRPCFields(
				tt.stepName,
				tt.response,
				tt.expected,
			)
			passed := 0
			failed := 0
			for _, r := range results {
				if r.Passed {
					passed++
				} else {
					failed++
				}
			}
			assert.Equal(t, tt.wantPass, passed)
			assert.Equal(t, tt.wantFail, failed)
		})
	}
}

func TestExtractGRPCVariables(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		extractTo map[string]string
		wantVars  map[string]string
	}{
		{
			name:     "extract_single_field",
			response: `{"id":"abc-123","name":"test"}`,
			extractTo: map[string]string{
				"id": "user_id",
			},
			wantVars: map[string]string{
				"user_id": "abc-123",
			},
		},
		{
			name:     "extract_multiple_fields",
			response: `{"token":"tk1","session":"s1"}`,
			extractTo: map[string]string{
				"token":   "auth_token",
				"session": "session_id",
			},
			wantVars: map[string]string{
				"auth_token": "tk1",
				"session_id": "s1",
			},
		},
		{
			name:      "invalid_json_no_extract",
			response:  "not json",
			extractTo: map[string]string{"a": "b"},
			wantVars:  map[string]string{},
		},
		{
			name:     "field_not_found",
			response: `{"other":"value"}`,
			extractTo: map[string]string{
				"missing": "var",
			},
			wantVars: map[string]string{},
		},
		{
			name:     "numeric_field",
			response: `{"count":42}`,
			extractTo: map[string]string{
				"count": "total",
			},
			wantVars: map[string]string{
				"total": "42",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variables := make(map[string]string)
			extractGRPCVariables(
				tt.response,
				tt.extractTo,
				variables,
			)
			for k, v := range tt.wantVars {
				assert.Equal(
					t, v, variables[k],
					"variable %s", k,
				)
			}
			// Ensure no extra variables for empty
			// expectations.
			if len(tt.wantVars) == 0 {
				assert.Empty(t, variables)
			}
		})
	}
}

func TestEvaluateGRPCStepAssertion(t *testing.T) {
	tests := []struct {
		name     string
		sa       StepAssertion
		response string
		stream   []string
		err      error
		want     bool
	}{
		{
			name: "response_contains_present",
			sa: StepAssertion{
				Type:  "response_contains",
				Value: "SERVING",
			},
			response: `{"status":"SERVING"}`,
			want:     true,
		},
		{
			name: "response_contains_absent",
			sa: StepAssertion{
				Type:  "response_contains",
				Value: "SERVING",
			},
			response: `{"status":"UNKNOWN"}`,
			want:     false,
		},
		{
			name: "not_empty_with_response",
			sa: StepAssertion{
				Type: "not_empty",
			},
			response: `{"data":"value"}`,
			want:     true,
		},
		{
			name: "not_empty_empty_response",
			sa: StepAssertion{
				Type: "not_empty",
			},
			response: "",
			want:     false,
		},
		{
			name: "stream_count_satisfied",
			sa: StepAssertion{
				Type:  "stream_count",
				Value: float64(2),
			},
			stream: []string{"r1", "r2", "r3"},
			want:   true,
		},
		{
			name: "stream_count_not_satisfied",
			sa: StepAssertion{
				Type:  "stream_count",
				Value: float64(5),
			},
			stream: []string{"r1", "r2"},
			want:   false,
		},
		{
			name: "stream_count_int_value",
			sa: StepAssertion{
				Type:  "stream_count",
				Value: 2,
			},
			stream: []string{"r1", "r2"},
			want:   true,
		},
		{
			name: "error_returns_false",
			sa: StepAssertion{
				Type: "not_empty",
			},
			response: `{"data":"value"}`,
			err:      fmt.Errorf("invocation error"),
			want:     false,
		},
		{
			name: "unknown_type_returns_false",
			sa: StepAssertion{
				Type: "unknown_assertion",
			},
			response: `{"data":"value"}`,
			want:     false,
		},
		{
			name: "response_contains_non_string_value",
			sa: StepAssertion{
				Type:  "response_contains",
				Value: 123,
			},
			response: `{"status":"SERVING"}`,
			want:     false,
		},
		{
			name: "stream_count_non_numeric_value",
			sa: StepAssertion{
				Type:  "stream_count",
				Value: "not a number",
			},
			stream: []string{"r1"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateGRPCStepAssertion(
				tt.sa,
				tt.response,
				tt.stream,
				tt.err,
			)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGRPCInvokeActual(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "no_error",
			err:  nil,
			want: "success",
		},
		{
			name: "with_error",
			err:  fmt.Errorf("connection refused"),
			want: "error: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := grpcInvokeActual(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGRPCInvokeMessage(t *testing.T) {
	tests := []struct {
		name     string
		stepName string
		err      error
		contains string
	}{
		{
			name:     "success",
			stepName: "health",
			err:      nil,
			contains: "succeeded",
		},
		{
			name:     "failure",
			stepName: "invoke",
			err:      fmt.Errorf("deadline exceeded"),
			contains: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := grpcInvokeMessage(
				tt.stepName, tt.err,
			)
			assert.Contains(t, msg, tt.stepName)
			assert.Contains(t, msg, tt.contains)
		})
	}
}

func TestGRPCStreamMessage(t *testing.T) {
	tests := []struct {
		name     string
		stepName string
		count    int
		contains string
	}{
		{
			name:     "zero_responses",
			stepName: "stream1",
			count:    0,
			contains: "no responses",
		},
		{
			name:     "multiple_responses",
			stepName: "stream2",
			count:    5,
			contains: "5 response(s)",
		},
		{
			name:     "single_response",
			stepName: "stream3",
			count:    1,
			contains: "1 response(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := grpcStreamMessage(
				tt.stepName, tt.count,
			)
			assert.Contains(t, msg, tt.stepName)
			assert.Contains(t, msg, tt.contains)
		})
	}
}

func TestGRPCFieldMessage(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		passed   bool
		expected string
		actual   string
		contains string
	}{
		{
			name:     "match",
			field:    "status",
			passed:   true,
			expected: "SERVING",
			actual:   "SERVING",
			contains: "matches",
		},
		{
			name:     "mismatch",
			field:    "status",
			passed:   false,
			expected: "SERVING",
			actual:   "NOT_SERVING",
			contains: "expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := grpcFieldMessage(
				tt.field,
				tt.passed,
				tt.expected,
				tt.actual,
			)
			assert.Contains(t, msg, tt.field)
			assert.Contains(t, msg, tt.contains)
		})
	}
}

func TestGRPCFlowTypes(t *testing.T) {
	t.Run("GRPCFlow_structure", func(t *testing.T) {
		flow := GRPCFlow{
			ServerAddr: "localhost:50051",
			Options: GRPCFlowOptions{
				Insecure: true,
				Headers: map[string]string{
					"X-Key": "val",
				},
			},
			Steps: []GRPCStep{
				{
					Name:    "step1",
					Method:  "svc/Method",
					Request: `{"key":"val"}`,
					Stream:  false,
					ExpectedFields: map[string]interface{}{
						"result": "ok",
					},
					Assertions: []StepAssertion{
						{
							Type:   "not_empty",
							Target: "response",
						},
					},
					ExtractTo: map[string]string{
						"id": "session_id",
					},
				},
			},
		}

		assert.Equal(
			t, "localhost:50051", flow.ServerAddr,
		)
		assert.True(t, flow.Options.Insecure)
		assert.Len(t, flow.Steps, 1)
		assert.Equal(t, "step1", flow.Steps[0].Name)
		assert.False(t, flow.Steps[0].Stream)
	})

	t.Run("GRPCStep_streaming", func(t *testing.T) {
		step := GRPCStep{
			Name:   "stream_test",
			Method: "svc/StreamMethod",
			Stream: true,
			Assertions: []StepAssertion{
				{
					Type:  "stream_count",
					Value: float64(3),
				},
			},
		}
		assert.True(t, step.Stream)
		assert.Equal(
			t, "svc/StreamMethod", step.Method,
		)
	})
}
