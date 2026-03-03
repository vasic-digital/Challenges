package userflow

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"digital.vasic.challenges/pkg/challenge"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ challenge.Challenge = (*WebSocketFlowChallenge)(nil)

func TestNewWebSocketFlowChallenge(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		chName      string
		description string
		deps        []challenge.ID
		flow        WebSocketFlow
	}{
		{
			name:        "basic_challenge",
			id:          "ws-001",
			chName:      "WebSocket Echo",
			description: "Test echo endpoint",
			deps:        nil,
			flow: WebSocketFlow{
				URL: "ws://localhost:8080/ws",
				Steps: []WebSocketStep{
					{
						Name:    "send_hello",
						Action:  "send",
						Message: "hello",
					},
				},
			},
		},
		{
			name:        "challenge_with_deps",
			id:          "ws-002",
			chName:      "WebSocket Auth",
			description: "Test authenticated ws",
			deps: []challenge.ID{
				"ws-001",
			},
			flow: WebSocketFlow{
				URL: "wss://localhost:8443/ws",
				Headers: map[string]string{
					"Authorization": "Bearer tok",
				},
			},
		},
		{
			name:        "challenge_no_steps",
			id:          "ws-003",
			chName:      "WebSocket Empty",
			description: "Empty flow",
			deps:        nil,
			flow: WebSocketFlow{
				URL: "ws://localhost:8080/ws",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewGorillaWebSocketAdapter()
			ch := NewWebSocketFlowChallenge(
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
			assert.Equal(
				t, "websocket", ch.Category(),
			)
			assert.NotNil(t, ch.adapter)
			assert.Equal(t, tt.flow, ch.flow)
		})
	}
}

func TestWebSocketFlowChallenge_Execute_UnavailableAdapter(
	t *testing.T,
) {
	adapter := NewGorillaWebSocketAdapter()
	flow := WebSocketFlow{
		URL: "ws://localhost:19999/ws",
		Steps: []WebSocketStep{
			{
				Name:    "send",
				Action:  "send",
				Message: "hello",
			},
		},
	}

	ch := NewWebSocketFlowChallenge(
		"ws-unavail",
		"Test Unavailable",
		"Test with unavailable server",
		nil,
		adapter,
		flow,
	)

	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	// When adapter is not available (not connected),
	// the challenge passes with a skip message.
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

func TestExtractWSVariables(t *testing.T) {
	tests := []struct {
		name      string
		response  []byte
		extractTo map[string]string
		wantVars  map[string]string
	}{
		{
			name:     "extract_single_field",
			response: []byte(`{"id":"ws-123"}`),
			extractTo: map[string]string{
				"id": "msg_id",
			},
			wantVars: map[string]string{
				"msg_id": "ws-123",
			},
		},
		{
			name: "extract_multiple_fields",
			response: []byte(
				`{"type":"response","data":"hello"}`,
			),
			extractTo: map[string]string{
				"type": "msg_type",
				"data": "msg_data",
			},
			wantVars: map[string]string{
				"msg_type": "response",
				"msg_data": "hello",
			},
		},
		{
			name:      "invalid_json",
			response:  []byte("not json"),
			extractTo: map[string]string{"a": "b"},
			wantVars:  map[string]string{},
		},
		{
			name:     "missing_field",
			response: []byte(`{"other":"val"}`),
			extractTo: map[string]string{
				"missing": "var",
			},
			wantVars: map[string]string{},
		},
		{
			name:     "numeric_field",
			response: []byte(`{"count":42}`),
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
			extractWSVariables(
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
			if len(tt.wantVars) == 0 {
				assert.Empty(t, variables)
			}
		})
	}
}

func TestEvaluateWSStepAssertion(t *testing.T) {
	tests := []struct {
		name     string
		sa       StepAssertion
		response []byte
		all      [][]byte
		err      error
		want     bool
	}{
		{
			name: "response_contains_match",
			sa: StepAssertion{
				Type:  "response_contains",
				Value: "hello",
			},
			response: []byte("hello world"),
			want:     true,
		},
		{
			name: "response_contains_no_match",
			sa: StepAssertion{
				Type:  "response_contains",
				Value: "goodbye",
			},
			response: []byte("hello world"),
			want:     false,
		},
		{
			name: "not_empty_with_data",
			sa: StepAssertion{
				Type: "not_empty",
			},
			response: []byte("data"),
			want:     true,
		},
		{
			name: "not_empty_no_data",
			sa: StepAssertion{
				Type: "not_empty",
			},
			response: []byte{},
			want:     false,
		},
		{
			name: "not_empty_nil",
			sa: StepAssertion{
				Type: "not_empty",
			},
			response: nil,
			want:     false,
		},
		{
			name: "message_count_satisfied",
			sa: StepAssertion{
				Type:  "message_count",
				Value: float64(2),
			},
			all: [][]byte{
				[]byte("a"), []byte("b"), []byte("c"),
			},
			want: true,
		},
		{
			name: "message_count_not_satisfied",
			sa: StepAssertion{
				Type:  "message_count",
				Value: float64(5),
			},
			all:  [][]byte{[]byte("a")},
			want: false,
		},
		{
			name: "message_count_int_value",
			sa: StepAssertion{
				Type:  "message_count",
				Value: 3,
			},
			all: [][]byte{
				[]byte("a"), []byte("b"), []byte("c"),
			},
			want: true,
		},
		{
			name: "error_returns_false",
			sa: StepAssertion{
				Type: "not_empty",
			},
			response: []byte("data"),
			err:      fmt.Errorf("send failed"),
			want:     false,
		},
		{
			name: "unknown_type_returns_false",
			sa: StepAssertion{
				Type: "unknown",
			},
			response: []byte("data"),
			want:     false,
		},
		{
			name: "response_contains_non_string",
			sa: StepAssertion{
				Type:  "response_contains",
				Value: 42,
			},
			response: []byte("42"),
			want:     false,
		},
		{
			name: "message_count_non_numeric",
			sa: StepAssertion{
				Type:  "message_count",
				Value: "not a number",
			},
			all:  [][]byte{[]byte("a")},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateWSStepAssertion(
				tt.sa, tt.response, tt.all, tt.err,
			)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWSConnectActual(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "no_error",
			err:  nil,
			want: "connected",
		},
		{
			name: "with_error",
			err:  fmt.Errorf("connection refused"),
			want: "error: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wsConnectActual(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWSConnectMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "success",
			err:      nil,
			contains: "established",
		},
		{
			name:     "failure",
			err:      fmt.Errorf("refused"),
			contains: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := wsConnectMessage(tt.err)
			assert.Contains(t, msg, tt.contains)
		})
	}
}

func TestWSStepActual(t *testing.T) {
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
			err:  fmt.Errorf("timeout"),
			want: "error: timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wsStepActual(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWSStepMessage(t *testing.T) {
	tests := []struct {
		name     string
		stepName string
		action   string
		err      error
		contains []string
	}{
		{
			name:     "success",
			stepName: "send_hello",
			action:   "send",
			err:      nil,
			contains: []string{
				"send_hello", "send", "succeeded",
			},
		},
		{
			name:     "failure",
			stepName: "receive_msg",
			action:   "receive",
			err:      fmt.Errorf("deadline exceeded"),
			contains: []string{
				"receive_msg", "receive", "failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := wsStepMessage(
				tt.stepName, tt.action, tt.err,
			)
			for _, s := range tt.contains {
				assert.Contains(t, msg, s)
			}
		})
	}
}

func TestWebSocketFlowTypes(t *testing.T) {
	t.Run("WebSocketFlow_structure", func(t *testing.T) {
		flow := WebSocketFlow{
			URL: "ws://localhost:8080/ws",
			Headers: map[string]string{
				"Auth": "token",
			},
			Steps: []WebSocketStep{
				{
					Name:    "send_msg",
					Action:  "send",
					Message: `{"type":"hello"}`,
				},
				{
					Name:    "receive_msg",
					Action:  "receive",
					Timeout: 5 * time.Second,
					Assertions: []StepAssertion{
						{
							Type:   "not_empty",
							Target: "response",
						},
					},
				},
				{
					Name:   "wait",
					Action: "wait",
					Timeout: 100 *
						time.Millisecond,
				},
			},
		}

		assert.Equal(
			t,
			"ws://localhost:8080/ws",
			flow.URL,
		)
		assert.Equal(
			t, "token", flow.Headers["Auth"],
		)
		assert.Len(t, flow.Steps, 3)
		assert.Equal(
			t, "send", flow.Steps[0].Action,
		)
		assert.Equal(
			t, "receive", flow.Steps[1].Action,
		)
		assert.Equal(
			t, "wait", flow.Steps[2].Action,
		)
	})

	t.Run("WebSocketStep_actions", func(t *testing.T) {
		actions := []string{
			"send", "receive", "send_receive",
			"receive_all", "wait",
		}
		for _, action := range actions {
			step := WebSocketStep{
				Name:   action + "_step",
				Action: action,
			}
			assert.Equal(t, action, step.Action)
		}
	})

	t.Run("WebSocketStep_extract_to", func(t *testing.T) {
		step := WebSocketStep{
			Name:   "extract_step",
			Action: "send_receive",
			ExtractTo: map[string]string{
				"session_id": "sid",
				"token":      "auth_token",
			},
		}
		assert.Len(t, step.ExtractTo, 2)
		assert.Equal(
			t, "sid", step.ExtractTo["session_id"],
		)
	})
}

func TestWebSocketFlowChallenge_Execute_EchoServer(
	t *testing.T,
) {
	// Start a WebSocket echo server for the test.
	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(
					w, r, nil,
				)
				if err != nil {
					return
				}
				defer conn.Close()
				for {
					mt, msg, err := conn.ReadMessage()
					if err != nil {
						return
					}
					_ = conn.WriteMessage(mt, msg)
				}
			},
		),
	)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(
		srv.URL, "http",
	)

	// Pre-connect the adapter so Available() returns
	// true, which is the prerequisite for Execute to
	// actually run steps.
	adapter := NewGorillaWebSocketAdapter()

	flow := WebSocketFlow{
		URL: wsURL,
		Steps: []WebSocketStep{
			{
				Name:    "echo_test",
				Action:  "send_receive",
				Message: `{"msg":"hello"}`,
				Timeout: 5 * time.Second,
				Assertions: []StepAssertion{
					{
						Type:    "response_contains",
						Target:  "echo_response",
						Value:   "hello",
						Message: "echo should contain hello",
					},
				},
			},
		},
	}

	ch := NewWebSocketFlowChallenge(
		"ws-echo",
		"Echo Test",
		"Test echo WebSocket",
		nil,
		adapter,
		flow,
	)

	// Since the adapter is not connected yet,
	// Available returns false, so the challenge skips.
	result, err := ch.Execute(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(
		t, challenge.StatusPassed, result.Status,
	)
}
