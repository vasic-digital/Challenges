package userflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"digital.vasic.challenges/pkg/httpclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface checks.
var (
	_ APIAdapter    = (*HTTPAPIAdapter)(nil)
	_ WebSocketConn = (*gorillaWSConn)(nil)
)

func TestHTTPAPIAdapter_Constructor(t *testing.T) {
	adapter := NewHTTPAPIAdapter(
		"http://localhost:8080",
	)
	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.client)
	assert.Equal(
		t, "http://localhost:8080", adapter.baseURL,
	)
}

func TestHTTPAPIAdapter_Constructor_TrailingSlash(
	t *testing.T,
) {
	adapter := NewHTTPAPIAdapter(
		"http://localhost:8080/",
	)
	assert.Equal(
		t, "http://localhost:8080", adapter.baseURL,
	)
}

func TestHTTPAPIAdapter_Constructor_WithOptions(
	t *testing.T,
) {
	adapter := NewHTTPAPIAdapter(
		"http://localhost:8080",
		httpclient.WithLoginPath("/auth/login"),
		httpclient.WithTokenField("token"),
	)
	assert.NotNil(t, adapter)
}

func TestHTTPAPIAdapter_Available_HealthyServer(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	assert.True(t, adapter.Available(context.Background()))
}

func TestHTTPAPIAdapter_Available_UnhealthyServer(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(
					http.StatusInternalServerError,
				)
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	assert.False(t, adapter.Available(context.Background()))
}

func TestHTTPAPIAdapter_Available_NoServer(t *testing.T) {
	adapter := NewHTTPAPIAdapter(
		"http://localhost:19999",
	)
	assert.False(t, adapter.Available(context.Background()))
}

func TestHTTPAPIAdapter_SetToken(t *testing.T) {
	adapter := NewHTTPAPIAdapter(
		"http://localhost:8080",
	)
	adapter.SetToken("test-token-123")
	assert.Equal(t, "test-token-123", adapter.client.Token())
}

func TestHTTPAPIAdapter_Login(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/auth/login" {
					w.Header().Set(
						"Content-Type", "application/json",
					)
					resp := map[string]string{
						"session_token": "jwt-token-abc",
					}
					_ = json.NewEncoder(w).Encode(resp)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	token, err := adapter.Login(
		context.Background(),
		Credentials{
			Username: "admin",
			Password: "admin123",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "jwt-token-abc", token)
}

func TestHTTPAPIAdapter_Login_Failure(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write(
					[]byte(`{"error":"invalid credentials"}`),
				)
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	_, err := adapter.Login(
		context.Background(),
		Credentials{
			Username: "bad",
			Password: "bad",
		},
	)
	assert.Error(t, err)
}

func TestHTTPAPIAdapter_Get(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(
					"Content-Type", "application/json",
				)
				_, _ = w.Write(
					[]byte(`{"key":"value"}`),
				)
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	status, body, err := adapter.Get(
		context.Background(), "/test",
	)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "value", body["key"])
}

func TestHTTPAPIAdapter_GetRaw(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("raw data"))
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	status, body, err := adapter.GetRaw(
		context.Background(), "/raw",
	)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "raw data", string(body))
}

func TestHTTPAPIAdapter_PostJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(
					t, "application/json",
					r.Header.Get("Content-Type"),
				)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":1}`))
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	status, body, err := adapter.PostJSON(
		context.Background(),
		"/create",
		`{"name":"test"}`,
	)
	require.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.Contains(t, string(body), "id")
}

func TestHTTPAPIAdapter_PutJSON(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(
					t, http.MethodPut, r.Method,
				)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"updated":true}`))
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	status, body, err := adapter.PutJSON(
		context.Background(),
		"/update",
		`{"name":"updated"}`,
	)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Contains(t, string(body), "updated")
}

func TestHTTPAPIAdapter_Delete(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(
					t, http.MethodDelete, r.Method,
				)
				w.WriteHeader(http.StatusNoContent)
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	status, _, err := adapter.Delete(
		context.Background(), "/item/1",
	)
	require.NoError(t, err)
	assert.Equal(t, 204, status)
}

func TestHTTPAPIAdapter_GetArray(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(
					"Content-Type", "application/json",
				)
				_, _ = w.Write(
					[]byte(`[{"id":1},{"id":2}]`),
				)
			},
		),
	)
	defer server.Close()

	adapter := NewHTTPAPIAdapter(server.URL)
	status, arr, err := adapter.GetArray(
		context.Background(), "/list",
	)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Len(t, arr, 2)
}
