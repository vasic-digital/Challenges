package userflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.challenges/pkg/challenge"
)

// mockAPIAdapter implements APIAdapter for testing.
type mockAPIAdapter struct {
	loginToken string
	loginErr   error
	tokenSet   string

	getRawResponses map[string]mockHTTPResponse
	postResponses   map[string]mockHTTPResponse
	putResponses    map[string]mockHTTPResponse
	deleteResponses map[string]mockHTTPResponse
}

type mockHTTPResponse struct {
	code int
	body []byte
	err  error
}

func newMockAPIAdapter() *mockAPIAdapter {
	return &mockAPIAdapter{
		getRawResponses: make(map[string]mockHTTPResponse),
		postResponses:   make(map[string]mockHTTPResponse),
		putResponses:    make(map[string]mockHTTPResponse),
		deleteResponses: make(map[string]mockHTTPResponse),
	}
}

func (m *mockAPIAdapter) Login(
	_ context.Context, _ Credentials,
) (string, error) {
	return m.loginToken, m.loginErr
}

func (m *mockAPIAdapter) LoginWithRetry(
	_ context.Context, _ Credentials, _ int,
) (string, error) {
	return m.loginToken, m.loginErr
}

func (m *mockAPIAdapter) Get(
	ctx context.Context, path string,
) (int, map[string]interface{}, error) {
	code, body, err := m.GetRaw(ctx, path)
	if err != nil {
		return code, nil, err
	}
	var result map[string]interface{}
	_ = json.Unmarshal(body, &result)
	return code, result, nil
}

func (m *mockAPIAdapter) GetRaw(
	_ context.Context, path string,
) (int, []byte, error) {
	if resp, ok := m.getRawResponses[path]; ok {
		return resp.code, resp.body, resp.err
	}
	return 404, nil, nil
}

func (m *mockAPIAdapter) GetArray(
	_ context.Context, _ string,
) (int, []interface{}, error) {
	return 200, nil, nil
}

func (m *mockAPIAdapter) PostJSON(
	_ context.Context, path, _ string,
) (int, []byte, error) {
	if resp, ok := m.postResponses[path]; ok {
		return resp.code, resp.body, resp.err
	}
	return 404, nil, nil
}

func (m *mockAPIAdapter) PutJSON(
	_ context.Context, path, _ string,
) (int, []byte, error) {
	if resp, ok := m.putResponses[path]; ok {
		return resp.code, resp.body, resp.err
	}
	return 404, nil, nil
}

func (m *mockAPIAdapter) Delete(
	_ context.Context, path string,
) (int, []byte, error) {
	if resp, ok := m.deleteResponses[path]; ok {
		return resp.code, resp.body, resp.err
	}
	return 404, nil, nil
}

func (m *mockAPIAdapter) WebSocketConnect(
	_ context.Context, _ string,
) (WebSocketConn, error) {
	return nil, nil
}

func (m *mockAPIAdapter) SetToken(token string) {
	m.tokenSet = token
}

func (m *mockAPIAdapter) Available(
	_ context.Context,
) bool {
	return true
}

// --- APIHealthChallenge tests ---

func TestNewAPIHealthChallenge(t *testing.T) {
	adapter := newMockAPIAdapter()
	ch := NewAPIHealthChallenge(
		"HEALTH-001", adapter, "/health", 200, nil,
	)

	assert.Equal(
		t, challenge.ID("HEALTH-001"), ch.ID(),
	)
	assert.Equal(t, "API Health Check", ch.Name())
	assert.Equal(t, "api", ch.Category())
}

func TestAPIHealthChallenge_Execute_Success(t *testing.T) {
	adapter := newMockAPIAdapter()
	adapter.getRawResponses["/health"] = mockHTTPResponse{
		code: 200,
		body: []byte(`{"status":"ok"}`),
	}

	ch := NewAPIHealthChallenge(
		"HEALTH-002", adapter, "/health", 200, nil,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 1)
	assert.True(t, result.Assertions[0].Passed)
	assert.Equal(t, "200", result.Assertions[0].Expected)
	assert.Equal(t, "200", result.Assertions[0].Actual)
	assert.Contains(
		t, result.Assertions[0].Message, "as expected",
	)

	rt, ok := result.Metrics["response_time"]
	require.True(t, ok)
	assert.Equal(t, "s", rt.Unit)
}

func TestAPIHealthChallenge_Execute_WrongStatus(
	t *testing.T,
) {
	adapter := newMockAPIAdapter()
	adapter.getRawResponses["/health"] = mockHTTPResponse{
		code: 503,
		body: []byte(`{"status":"unavailable"}`),
	}

	ch := NewAPIHealthChallenge(
		"HEALTH-003", adapter, "/health", 200, nil,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
	assert.Equal(t, "503", result.Assertions[0].Actual)
}

func TestAPIHealthChallenge_Execute_Error(t *testing.T) {
	adapter := newMockAPIAdapter()
	adapter.getRawResponses["/health"] = mockHTTPResponse{
		err: fmt.Errorf("connection refused"),
	}

	ch := NewAPIHealthChallenge(
		"HEALTH-004", adapter, "/health", 200, nil,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Message,
		"connection refused",
	)
}

func TestAPIHealthChallenge_Execute_WithDeps(t *testing.T) {
	adapter := newMockAPIAdapter()
	deps := []challenge.ID{"ENV-SETUP"}
	ch := NewAPIHealthChallenge(
		"HEALTH-005", adapter, "/health", 200, deps,
	)

	assert.Equal(
		t, []challenge.ID{"ENV-SETUP"},
		ch.Dependencies(),
	)
}

// --- APIFlowChallenge tests ---

func TestNewAPIFlowChallenge(t *testing.T) {
	adapter := newMockAPIAdapter()
	flow := APIFlow{
		Name: "test-flow",
		Steps: []APIStep{
			{Name: "step1", Method: "GET", Path: "/api"},
		},
	}
	ch := NewAPIFlowChallenge(
		"FLOW-001", "Test Flow", "A test flow",
		nil, adapter, flow,
	)

	assert.Equal(
		t, challenge.ID("FLOW-001"), ch.ID(),
	)
	assert.Equal(t, "Test Flow", ch.Name())
	assert.Equal(t, "api", ch.Category())
}

func TestAPIFlowChallenge_Execute_SimpleFlow(
	t *testing.T,
) {
	adapter := newMockAPIAdapter()
	adapter.loginToken = "test-jwt-token"
	adapter.getRawResponses["/api/v1/users"] = mockHTTPResponse{
		code: 200,
		body: []byte(`[{"id":1,"name":"admin"}]`),
	}

	flow := APIFlow{
		Name: "list-users",
		Credentials: Credentials{
			Username: "admin",
			Password: "pass",
		},
		Steps: []APIStep{
			{
				Name:           "list users",
				Method:         "GET",
				Path:           "/api/v1/users",
				ExpectedStatus: 200,
			},
		},
	}

	ch := NewAPIFlowChallenge(
		"FLOW-002", "List Users", "List users flow",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)

	// Login assertion + status code assertion.
	require.Len(t, result.Assertions, 2)
	assert.True(t, result.Assertions[0].Passed)
	assert.Equal(t, "login", result.Assertions[0].Type)
	assert.True(t, result.Assertions[1].Passed)
	assert.Equal(
		t, "status_code", result.Assertions[1].Type,
	)

	assert.Equal(t, "test-jwt-token", adapter.tokenSet)
}

func TestAPIFlowChallenge_Execute_LoginFailure(
	t *testing.T,
) {
	adapter := newMockAPIAdapter()
	adapter.loginErr = fmt.Errorf("invalid credentials")

	flow := APIFlow{
		Name: "fail-login",
		Credentials: Credentials{
			Username: "admin",
			Password: "wrong",
		},
		Steps: []APIStep{
			{
				Name:   "step1",
				Method: "GET",
				Path:   "/api/v1/data",
			},
		},
	}

	ch := NewAPIFlowChallenge(
		"FLOW-003", "Login Fail", "Login fails",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
	assert.False(t, result.Assertions[0].Passed)
	assert.Contains(
		t, result.Assertions[0].Message,
		"invalid credentials",
	)
}

func TestAPIFlowChallenge_Execute_VariableExtraction(
	t *testing.T,
) {
	adapter := newMockAPIAdapter()

	createResp := map[string]interface{}{
		"id":   42,
		"name": "test-item",
	}
	createBody, _ := json.Marshal(createResp)
	adapter.postResponses["/api/v1/items"] = mockHTTPResponse{
		code: 201, body: createBody,
	}

	// The second step uses the extracted variable.
	adapter.getRawResponses["/api/v1/items/42"] = mockHTTPResponse{
		code: 200,
		body: []byte(`{"id":42,"name":"test-item"}`),
	}

	flow := APIFlow{
		Name: "create-and-get",
		Steps: []APIStep{
			{
				Name:           "create item",
				Method:         "POST",
				Path:           "/api/v1/items",
				Body:           `{"name":"test-item"}`,
				ExpectedStatus: 201,
				ExtractTo: map[string]string{
					"id": "item_id",
				},
			},
			{
				Name:           "get item",
				Method:         "GET",
				Path:           "/api/v1/items/{{item_id}}",
				ExpectedStatus: 200,
				Assertions: []StepAssertion{
					{
						Type:    "response_contains",
						Target:  "body",
						Value:   "test-item",
						Message: "should contain item name",
					},
				},
			},
		},
	}

	ch := NewAPIFlowChallenge(
		"FLOW-004", "CRUD", "Create and read",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)

	// 2 status_code assertions + 1 response_contains.
	require.Len(t, result.Assertions, 3)
	for _, a := range result.Assertions {
		assert.True(t, a.Passed, "failed: %s", a.Message)
	}

	stepsMetric := result.Metrics["steps_executed"]
	assert.Equal(t, 2.0, stepsMetric.Value)
}

func TestAPIFlowChallenge_Execute_NoCredentials(
	t *testing.T,
) {
	adapter := newMockAPIAdapter()
	adapter.getRawResponses["/api/public"] = mockHTTPResponse{
		code: 200, body: []byte(`{"public":true}`),
	}

	flow := APIFlow{
		Name: "public-api",
		Steps: []APIStep{
			{
				Name:           "public endpoint",
				Method:         "GET",
				Path:           "/api/public",
				ExpectedStatus: 200,
			},
		},
	}

	ch := NewAPIFlowChallenge(
		"FLOW-005", "Public", "Public API",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	// No login assertion, just status_code.
	require.Len(t, result.Assertions, 1)
	assert.Equal(
		t, "status_code", result.Assertions[0].Type,
	)
}

func TestAPIFlowChallenge_Execute_PutAndDelete(
	t *testing.T,
) {
	adapter := newMockAPIAdapter()
	adapter.putResponses["/api/v1/items/1"] = mockHTTPResponse{
		code: 200, body: []byte(`{"updated":true}`),
	}
	adapter.deleteResponses["/api/v1/items/1"] = mockHTTPResponse{
		code: 204, body: nil,
	}

	flow := APIFlow{
		Name: "update-delete",
		Steps: []APIStep{
			{
				Name:           "update",
				Method:         "PUT",
				Path:           "/api/v1/items/1",
				Body:           `{"name":"updated"}`,
				ExpectedStatus: 200,
			},
			{
				Name:           "delete",
				Method:         "DELETE",
				Path:           "/api/v1/items/1",
				ExpectedStatus: 204,
			},
		},
	}

	ch := NewAPIFlowChallenge(
		"FLOW-006", "CRUD", "Update and delete",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusPassed, result.Status)
	require.Len(t, result.Assertions, 2)
}

func TestAPIFlowChallenge_Execute_StepAssertionFailure(
	t *testing.T,
) {
	adapter := newMockAPIAdapter()
	adapter.getRawResponses["/api/data"] = mockHTTPResponse{
		code: 200, body: []byte(`{"items":[]}`),
	}

	flow := APIFlow{
		Name: "assert-fail",
		Steps: []APIStep{
			{
				Name:           "get data",
				Method:         "GET",
				Path:           "/api/data",
				ExpectedStatus: 200,
				Assertions: []StepAssertion{
					{
						Type:    "response_contains",
						Target:  "body",
						Value:   "expected-value",
						Message: "should contain value",
					},
				},
			},
		},
	}

	ch := NewAPIFlowChallenge(
		"FLOW-007", "Assert Fail", "Assertion fails",
		nil, adapter, flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, challenge.StatusFailed, result.Status)
}

func TestSubstituteVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]string
		expected string
	}{
		{
			name:  "single variable",
			input: "/api/items/{{id}}",
			vars:  map[string]string{"id": "42"},
			expected: "/api/items/42",
		},
		{
			name:  "multiple variables",
			input: "/api/{{version}}/items/{{id}}",
			vars: map[string]string{
				"version": "v1",
				"id":      "99",
			},
			expected: "/api/v1/items/99",
		},
		{
			name:     "no variables",
			input:    "/api/items",
			vars:     map[string]string{},
			expected: "/api/items",
		},
		{
			name:     "missing variable",
			input:    "/api/items/{{id}}",
			vars:     map[string]string{},
			expected: "/api/items/{{id}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteVars(tt.input, tt.vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateStepAssertion(t *testing.T) {
	tests := []struct {
		name     string
		sa       StepAssertion
		code     int
		body     []byte
		err      error
		expected bool
	}{
		{
			name: "status_code match int",
			sa: StepAssertion{
				Type: "status_code", Value: 200,
			},
			code:     200,
			expected: true,
		},
		{
			name: "status_code match float64",
			sa: StepAssertion{
				Type: "status_code", Value: float64(200),
			},
			code:     200,
			expected: true,
		},
		{
			name: "status_code mismatch",
			sa: StepAssertion{
				Type: "status_code", Value: 200,
			},
			code:     404,
			expected: false,
		},
		{
			name: "response_contains match",
			sa: StepAssertion{
				Type: "response_contains", Value: "ok",
			},
			code:     200,
			body:     []byte(`{"status":"ok"}`),
			expected: true,
		},
		{
			name: "response_contains mismatch",
			sa: StepAssertion{
				Type:  "response_contains",
				Value: "missing",
			},
			code:     200,
			body:     []byte(`{"status":"ok"}`),
			expected: false,
		},
		{
			name: "not_empty pass",
			sa:   StepAssertion{Type: "not_empty"},
			code: 200,
			body: []byte(`data`),
			expected: true,
		},
		{
			name: "not_empty fail",
			sa:   StepAssertion{Type: "not_empty"},
			code: 200,
			body: nil,
			expected: false,
		},
		{
			name: "error returns false",
			sa: StepAssertion{
				Type: "status_code", Value: 200,
			},
			code:     200,
			err:      fmt.Errorf("timeout"),
			expected: false,
		},
		{
			name:     "unknown type",
			sa:       StepAssertion{Type: "unknown"},
			code:     200,
			body:     []byte(`ok`),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateStepAssertion(
				tt.sa, tt.code, tt.body, tt.err,
			)
			assert.Equal(t, tt.expected, result)
		})
	}
}
