package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIClient_Defaults(t *testing.T) {
	c := NewAPIClient("http://localhost:8080")
	assert.Equal(t, "http://localhost:8080", c.BaseURL())
	assert.Equal(t, "", c.Token())
	assert.Equal(t, "/api/v1/auth/login", c.loginPath)
	assert.Equal(t, "session_token", c.tokenField)
	assert.Equal(t, "username", c.userField)
	assert.Equal(t, "password", c.passField)
}

func TestNewAPIClient_TrailingSlash(t *testing.T) {
	c := NewAPIClient("http://localhost:8080/")
	assert.Equal(t, "http://localhost:8080", c.BaseURL())
}

func TestNewAPIClient_Options(t *testing.T) {
	c := NewAPIClient("http://example.com",
		WithLoginPath("/auth/login"),
		WithTokenField("access_token"),
		WithUsernameField("email"),
		WithPasswordField("pass"),
		WithTimeout(5*time.Second),
	)
	assert.Equal(t, "/auth/login", c.loginPath)
	assert.Equal(t, "access_token", c.tokenField)
	assert.Equal(t, "email", c.userField)
	assert.Equal(t, "pass", c.passField)
	assert.Equal(t, 5*time.Second, c.httpClient.Timeout)
}

func TestAPIClient_SetToken(t *testing.T) {
	c := NewAPIClient("http://localhost")
	c.SetToken("my-token")
	assert.Equal(t, "my-token", c.Token())
}

func TestAPIClient_Login(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/auth/login", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "admin", body["username"])
		assert.Equal(t, "secret", body["password"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session_token": "jwt-abc-123",
			"expires_at":    "2030-01-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	result, err := c.Login(context.Background(), "admin", "secret")
	require.NoError(t, err)
	assert.Equal(t, "jwt-abc-123", c.Token())
	assert.Equal(t, "jwt-abc-123", result["session_token"])
}

func TestAPIClient_Login_CustomTokenField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "custom-token",
		})
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL, WithTokenField("access_token"))
	_, err := c.Login(context.Background(), "u", "p")
	require.NoError(t, err)
	assert.Equal(t, "custom-token", c.Token())
}

func TestAPIClient_Login_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"bad credentials"}`))
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	_, err := c.Login(context.Background(), "u", "p")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestAPIClient_Login_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	_, err := c.Login(context.Background(), "u", "p")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse login response")
}

func TestAPIClient_Login_MissingToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user": "admin",
		})
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	_, err := c.Login(context.Background(), "u", "p")
	require.NoError(t, err)
	assert.Equal(t, "", c.Token())
}

func TestAPIClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
		assert.Equal(t, "/api/v1/health", r.URL.Path)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
		})
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	c.SetToken("my-token")
	code, result, err := c.Get(context.Background(), "/api/v1/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "healthy", result["status"])
}

func TestAPIClient_Get_NoAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	code, _, err := c.Get(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestAPIClient_GetArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode([]interface{}{"a", "b", "c"})
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	c.SetToken("tok")
	code, arr, err := c.GetArray(context.Background(), "/items")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Len(t, arr, 3)
}

func TestAPIClient_GetRaw(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("raw bytes"))
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	code, data, err := c.GetRaw(context.Background(), "/raw")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "raw bytes", string(data))
}

func TestAPIClient_PostJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "test", body["name"])

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	c.SetToken("tok")
	code, data, err := c.PostJSON(
		context.Background(), "/create", `{"name":"test"}`,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, code)
	assert.Contains(t, string(data), `"id"`)
}

func TestAPIClient_Get_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	_, _, err := c.Get(context.Background(), "/bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}

func TestAPIClient_GetArray_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL)
	_, _, err := c.GetArray(context.Background(), "/bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}
