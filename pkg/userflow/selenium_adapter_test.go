package userflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ BrowserAdapter = (*SeleniumAdapter)(nil)

func TestNewSeleniumAdapter(t *testing.T) {
	tests := []struct {
		name      string
		serverURL string
		wantURL   string
	}{
		{
			name:      "basic_url",
			serverURL: "http://localhost:4444",
			wantURL:   "http://localhost:4444",
		},
		{
			name:      "trailing_slash_stripped",
			serverURL: "http://localhost:4444/",
			wantURL:   "http://localhost:4444",
		},
		{
			name:      "custom_port",
			serverURL: "http://selenium-hub:9515",
			wantURL:   "http://selenium-hub:9515",
		},
		{
			name:      "empty_url",
			serverURL: "",
			wantURL:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewSeleniumAdapter(tt.serverURL)
			require.NotNil(t, adapter)
			assert.Equal(t, tt.wantURL, adapter.serverURL)
			assert.NotNil(t, adapter.httpClient)
			assert.Empty(t, adapter.sessionID)
		})
	}
}

func TestSeleniumAdapter_Available_NoServer(
	t *testing.T,
) {
	adapter := NewSeleniumAdapter(
		"http://localhost:19998",
	)
	// No Selenium server running on this port.
	available := adapter.Available(context.Background())
	assert.False(t, available)
}

func TestSeleniumAdapter_Available_ServerRunning(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"value":{"ready":true}}`)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	available := adapter.Available(context.Background())
	assert.True(t, available)
}

func TestSeleniumAdapter_Available_ServerError(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(
					http.StatusInternalServerError,
				)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	available := adapter.Available(context.Background())
	assert.False(t, available)
}

func TestSeleniumAdapter_Navigate_NoSession(
	t *testing.T,
) {
	adapter := NewSeleniumAdapter(
		"http://localhost:19998",
	)
	err := adapter.Navigate(
		context.Background(), "http://example.com",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "navigate to")
}

func TestSeleniumAdapter_Click_NoSession(
	t *testing.T,
) {
	adapter := NewSeleniumAdapter(
		"http://localhost:19998",
	)
	err := adapter.Click(
		context.Background(), "#button",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "click find")
}

func TestSeleniumAdapter_Close_NoSession(
	t *testing.T,
) {
	adapter := NewSeleniumAdapter(
		"http://localhost:4444",
	)
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
}

func TestSeleniumAdapter_NetworkIntercept_NoOp(
	t *testing.T,
) {
	adapter := NewSeleniumAdapter(
		"http://localhost:4444",
	)
	err := adapter.NetworkIntercept(
		context.Background(),
		"**/api/**",
		func(_ *InterceptedRequest) {},
	)
	assert.NoError(t, err)
}

func TestSeleniumAdapter_FindElement_MockServer(
	t *testing.T,
) {
	const sessID = "test-session-abc"
	const elemKey = "element-6066-11e4-a52e-4f735466cecf"
	const elemID = "elem-42"

	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				wantPath := fmt.Sprintf(
					"/session/%s/element", sessID,
				)
				if r.URL.Path == wantPath &&
					r.Method == http.MethodPost {
					resp := map[string]interface{}{
						"value": map[string]interface{}{
							elemKey: elemID,
						},
					}
					w.Header().Set(
						"Content-Type",
						"application/json",
					)
					json.NewEncoder(w).Encode(resp)
					return
				}
				w.WriteHeader(
					http.StatusNotFound,
				)
				fmt.Fprint(w, `{"value":{`+
					`"message":"not found"}}`)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	adapter.sessionID = sessID

	id, err := adapter.findElement(
		context.Background(), "#test",
	)
	require.NoError(t, err)
	assert.Equal(t, elemID, id)
}

func TestSeleniumAdapter_FindElement_LegacyKey(
	t *testing.T,
) {
	const sessID = "test-session-legacy"
	const elemID = "legacy-elem-1"

	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"ELEMENT": elemID,
					},
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	adapter.sessionID = sessID

	id, err := adapter.findElement(
		context.Background(), ".btn",
	)
	require.NoError(t, err)
	assert.Equal(t, elemID, id)
}

func TestSeleniumAdapter_FindElement_NoID(
	t *testing.T,
) {
	const sessID = "test-session-noid"

	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"unknown_key": "abc",
					},
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	adapter.sessionID = sessID

	_, err := adapter.findElement(
		context.Background(), ".missing",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no element ID")
}

func TestSeleniumBrowserName(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantResult string
	}{
		{
			name:       "chrome",
			input:      "chrome",
			wantResult: "chrome",
		},
		{
			name:       "chromium",
			input:      "chromium",
			wantResult: "chrome",
		},
		{
			name:       "firefox",
			input:      "firefox",
			wantResult: "firefox",
		},
		{
			name:       "gecko",
			input:      "gecko",
			wantResult: "firefox",
		},
		{
			name:       "edge",
			input:      "edge",
			wantResult: "MicrosoftEdge",
		},
		{
			name:       "msedge",
			input:      "msedge",
			wantResult: "MicrosoftEdge",
		},
		{
			name:       "safari",
			input:      "safari",
			wantResult: "safari",
		},
		{
			name:       "unknown_defaults_to_chrome",
			input:      "opera",
			wantResult: "chrome",
		},
		{
			name:       "empty_defaults_to_chrome",
			input:      "",
			wantResult: "chrome",
		},
		{
			name:       "case_insensitive",
			input:      "Firefox",
			wantResult: "firefox",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := seleniumBrowserName(tt.input)
			assert.Equal(t, tt.wantResult, got)
		})
	}
}

func TestSeleniumOptionsKey(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantResult string
	}{
		{
			name:       "chrome",
			input:      "chrome",
			wantResult: "goog:chromeOptions",
		},
		{
			name:       "chromium",
			input:      "chromium",
			wantResult: "goog:chromeOptions",
		},
		{
			name:       "firefox",
			input:      "firefox",
			wantResult: "moz:firefoxOptions",
		},
		{
			name:       "gecko",
			input:      "gecko",
			wantResult: "moz:firefoxOptions",
		},
		{
			name:       "edge",
			input:      "edge",
			wantResult: "ms:edgeOptions",
		},
		{
			name:       "msedge",
			input:      "msedge",
			wantResult: "ms:edgeOptions",
		},
		{
			name:       "unknown_defaults_to_chrome",
			input:      "brave",
			wantResult: "goog:chromeOptions",
		},
		{
			name:       "safari_defaults_to_chrome",
			input:      "safari",
			wantResult: "goog:chromeOptions",
		},
		{
			name:       "case_insensitive",
			input:      "Edge",
			wantResult: "ms:edgeOptions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := seleniumOptionsKey(tt.input)
			assert.Equal(t, tt.wantResult, got)
		})
	}
}

func TestEscapeJSSingle(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no_escaping",
			in:   "button.submit",
			want: "button.submit",
		},
		{
			name: "single_quote",
			in:   "div[data-id='test']",
			want: "div[data-id=\\'test\\']",
		},
		{
			name: "backslash",
			in:   "path\\to\\element",
			want: "path\\\\to\\\\element",
		},
		{
			name: "mixed",
			in:   "a[href='/page']",
			want: "a[href=\\'/page\\']",
		},
		{
			name: "empty_string",
			in:   "",
			want: "",
		},
		{
			name: "no_special_chars",
			in:   "div.container",
			want: "div.container",
		},
		{
			name: "multiple_quotes",
			in:   "input[name='user'][type='text']",
			want: "input[name=\\'user\\'][type=\\'text\\']",
		},
		{
			name: "backslash_then_quote",
			in:   "a\\'b",
			want: "a\\\\\\'b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeJSSingle(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSeleniumAdapter_SessionPath(t *testing.T) {
	adapter := NewSeleniumAdapter(
		"http://localhost:4444",
	)
	adapter.sessionID = "abc-123"

	tests := []struct {
		name   string
		suffix string
		want   string
	}{
		{
			name:   "empty_suffix",
			suffix: "",
			want:   "/session/abc-123",
		},
		{
			name:   "url_suffix",
			suffix: "/url",
			want:   "/session/abc-123/url",
		},
		{
			name:   "element_suffix",
			suffix: "/element",
			want:   "/session/abc-123/element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.sessionPath(tt.suffix)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSeleniumAdapter_Initialize_MockServer(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"sessionId": "sess-xyz-789",
					},
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	err := adapter.Initialize(
		context.Background(),
		BrowserConfig{
			BrowserType: "chrome",
			Headless:    true,
			WindowSize:  [2]int{1920, 1080},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "sess-xyz-789", adapter.sessionID)
}

func TestSeleniumAdapter_Initialize_FlatSessionID(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"sessionId": "flat-sess-001",
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	err := adapter.Initialize(
		context.Background(),
		BrowserConfig{BrowserType: "firefox"},
	)
	require.NoError(t, err)
	assert.Equal(t, "flat-sess-001", adapter.sessionID)
}

func TestSeleniumAdapter_Initialize_NoSessionID(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"other": "data",
					},
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	err := adapter.Initialize(
		context.Background(),
		BrowserConfig{BrowserType: "chrome"},
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no sessionId")
}

func TestSeleniumAdapter_Close_WithSession(
	t *testing.T,
) {
	var deleteCalled bool
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deleteCalled = true
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				fmt.Fprint(w, `{"value":null}`)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	adapter.sessionID = "sess-to-close"

	err := adapter.Close(context.Background())
	assert.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Empty(t, adapter.sessionID)
}

func TestSeleniumAdapter_EvaluateJS_MockServer(
	t *testing.T,
) {
	const sessID = "eval-session"

	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": "hello world",
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	adapter.sessionID = sessID

	result, err := adapter.EvaluateJS(
		context.Background(),
		"return document.title;",
	)
	require.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestSeleniumAdapter_EvaluateJS_NilResult(
	t *testing.T,
) {
	const sessID = "eval-nil"

	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				resp := map[string]interface{}{
					"value": nil,
				}
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	adapter.sessionID = sessID

	result, err := adapter.EvaluateJS(
		context.Background(),
		"return null;",
	)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestSeleniumAdapter_WebDriverError(
	t *testing.T,
) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				w.WriteHeader(http.StatusBadRequest)
				resp := map[string]interface{}{
					"value": map[string]interface{}{
						"message": "invalid argument",
					},
				}
				json.NewEncoder(w).Encode(resp)
			},
		),
	)
	defer srv.Close()

	adapter := NewSeleniumAdapter(srv.URL)
	adapter.sessionID = "error-session"

	err := adapter.Navigate(
		context.Background(), "not-a-url",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid argument")
}

func TestSeleniumAdapter_ContextCancellation(
	t *testing.T,
) {
	adapter := NewSeleniumAdapter(
		"http://localhost:19998",
	)
	adapter.sessionID = "cancel-session"

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel() // Cancel immediately.

	err := adapter.Navigate(ctx, "http://example.com")
	assert.Error(t, err)
}
