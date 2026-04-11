package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ClientOption configures an APIClient via functional options.
type ClientOption func(*APIClient)

// APIClient wraps net/http.Client with JWT authentication support
// for calling REST APIs. Defaults match common conventions so
// callers can use NewAPIClient(url) with zero options.
type APIClient struct {
	baseURL     string
	token       string
	loginPath   string
	tokenField  string
	tokenHeader string
	userField   string
	passField   string
	httpClient  *http.Client
}

// NewAPIClient creates an API client targeting the given base URL.
// Pass ClientOption values to override defaults.
func NewAPIClient(baseURL string, opts ...ClientOption) *APIClient {
	c := &APIClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		loginPath:   "/api/v1/auth/login",
		tokenField:  "session_token",
		tokenHeader: "Authorization",
		userField:   "username",
		passField:   "password",
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// WithLoginPath overrides the default login endpoint path.
func WithLoginPath(path string) ClientOption {
	return func(c *APIClient) { c.loginPath = path }
}

// WithTokenField overrides the JSON field name used to extract
// the token from the login response.
func WithTokenField(field string) ClientOption {
	return func(c *APIClient) { c.tokenField = field }
}

// WithTokenHeader overrides the header name used to send the token
// (e.g., "X-Access-Token" for custom authentication).
func WithTokenHeader(header string) ClientOption {
	return func(c *APIClient) { c.tokenHeader = header }
}

// WithUsernameField overrides the JSON field name for the username
// in the login request body.
func WithUsernameField(field string) ClientOption {
	return func(c *APIClient) { c.userField = field }
}

// WithPasswordField overrides the JSON field name for the password
// in the login request body.
func WithPasswordField(field string) ClientOption {
	return func(c *APIClient) { c.passField = field }
}

// WithTimeout overrides the default HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *APIClient) { c.httpClient.Timeout = d }
}

// Login authenticates with the API and stores the JWT token
// for subsequent requests. Returns the parsed login response.
// AuthError is returned by Login when the server rejects the credentials
// (HTTP 4xx). It's a non-retryable failure: no amount of re-requesting
// will turn bad credentials into good ones. LoginWithRetry short-circuits
// on this error via errors.As.
type AuthError struct {
	StatusCode int
	Body       string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("login returned HTTP %d: %s", e.StatusCode, e.Body)
}

func (c *APIClient) Login(
	ctx context.Context, username, password string,
) (map[string]interface{}, error) {
	body := fmt.Sprintf(
		`{%q:%q,%q:%q}`, c.userField, username, c.passField, password,
	)
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.baseURL+c.loginPath,
		strings.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// 4xx is a definitive client error: bad credentials, missing
		// fields, rate limited, etc. Surfacing AuthError lets
		// LoginWithRetry short-circuit instead of burning ~150 seconds
		// of exponential backoff.
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, &AuthError{StatusCode: resp.StatusCode, Body: string(data)}
		}
		return nil, fmt.Errorf(
			"login returned HTTP %d: %s", resp.StatusCode, string(data),
		)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse login response: %w", err)
	}

	if token, ok := result[c.tokenField].(string); ok && token != "" {
		c.token = token
	}

	return result, nil
}

// LoginWithRetry calls Login with exponential backoff retry.
// Useful when the server may be under heavy load (e.g., post-scan aggregation).
// Short-circuits on AuthError (HTTP 4xx) because retrying bad credentials
// is pointless and wastes ~150 seconds of backoff.
func (c *APIClient) LoginWithRetry(
	ctx context.Context, username, password string, maxRetries int,
) (map[string]interface{}, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * 5 * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
		resp, err := c.Login(ctx, username, password)
		if err == nil {
			return resp, nil
		}
		// Don't retry on authentication failures — bad credentials
		// stay bad no matter how many times we ask.
		var authErr *AuthError
		if errors.As(err, &authErr) {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("login failed after %d retries: %w", maxRetries, lastErr)
}

// Get performs an authenticated GET request and returns the
// status code and parsed JSON object response.
func (c *APIClient) Get(
	ctx context.Context, path string,
) (int, map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.baseURL+path, nil,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	if c.token != "" {
		if c.tokenHeader == "Authorization" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		} else {
			req.Header.Set(c.tokenHeader, c.token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("parse response: %w", err)
	}

	return resp.StatusCode, result, nil
}

// GetArray performs an authenticated GET request and returns the
// status code and parsed JSON array response.
func (c *APIClient) GetArray(
	ctx context.Context, path string,
) (int, []interface{}, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.baseURL+path, nil,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	if c.token != "" {
		if c.tokenHeader == "Authorization" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		} else {
			req.Header.Set(c.tokenHeader, c.token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response: %w", err)
	}

	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("parse response: %w", err)
	}

	return resp.StatusCode, result, nil
}

// GetRaw performs an authenticated GET and returns status code
// and raw body bytes. Used when the response could be either
// an object or array.
func (c *APIClient) GetRaw(
	ctx context.Context, path string,
) (int, []byte, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.baseURL+path, nil,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	if c.token != "" {
		if c.tokenHeader == "Authorization" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		} else {
			req.Header.Set(c.tokenHeader, c.token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response: %w", err)
	}

	return resp.StatusCode, data, nil
}

// PostJSON performs an authenticated POST request with a JSON body
// and returns the status code and raw response bytes.
func (c *APIClient) PostJSON(
	ctx context.Context, path string, body string,
) (int, []byte, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+path, strings.NewReader(body),
	)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		if c.tokenHeader == "Authorization" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		} else {
			req.Header.Set(c.tokenHeader, c.token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response: %w", err)
	}

	return resp.StatusCode, data, nil
}

// PutJSON performs an authenticated PUT request with a JSON body
// and returns the status code and raw response bytes.
func (c *APIClient) PutJSON(
	ctx context.Context, path string, body string,
) (int, []byte, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, c.baseURL+path, strings.NewReader(body),
	)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		if c.tokenHeader == "Authorization" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		} else {
			req.Header.Set(c.tokenHeader, c.token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response: %w", err)
	}

	return resp.StatusCode, data, nil
}

// Delete performs an authenticated DELETE request and returns
// the status code and raw response bytes.
func (c *APIClient) Delete(
	ctx context.Context, path string,
) (int, []byte, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete, c.baseURL+path, nil,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	if c.token != "" {
		if c.tokenHeader == "Authorization" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		} else {
			req.Header.Set(c.tokenHeader, c.token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response: %w", err)
	}

	return resp.StatusCode, data, nil
}

// DeleteWithBody performs an authenticated DELETE request with a body
// and returns the status code and raw response bytes.
func (c *APIClient) DeleteWithBody(
	ctx context.Context, path string, body string,
) (int, []byte, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete, c.baseURL+path, strings.NewReader(body),
	)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		if c.tokenHeader == "Authorization" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		} else {
			req.Header.Set(c.tokenHeader, c.token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response: %w", err)
	}

	return resp.StatusCode, data, nil
}

// Token returns the stored JWT token.
func (c *APIClient) Token() string {
	return c.token
}

// SetToken sets the JWT token directly (e.g. when obtained externally).
func (c *APIClient) SetToken(token string) {
	c.token = token
}

// BaseURL returns the configured base URL.
func (c *APIClient) BaseURL() string {
	return c.baseURL
}
