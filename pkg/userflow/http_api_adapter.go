package userflow

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"digital.vasic.challenges/pkg/httpclient"

	"github.com/gorilla/websocket"
)

// HTTPAPIAdapter implements APIAdapter by wrapping the
// existing httpclient.APIClient and adding WebSocket
// support via gorilla/websocket.
type HTTPAPIAdapter struct {
	client  *httpclient.APIClient
	baseURL string
}

// Compile-time interface check.
var _ APIAdapter = (*HTTPAPIAdapter)(nil)

// NewHTTPAPIAdapter creates an HTTPAPIAdapter targeting the
// given base URL with optional httpclient.ClientOption values.
func NewHTTPAPIAdapter(
	baseURL string, opts ...httpclient.ClientOption,
) *HTTPAPIAdapter {
	return &HTTPAPIAdapter{
		client:  httpclient.NewAPIClient(baseURL, opts...),
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// Login authenticates with the given credentials and returns
// the JWT token.
func (a *HTTPAPIAdapter) Login(
	ctx context.Context, creds Credentials,
) (string, error) {
	resp, err := a.client.Login(
		ctx, creds.Username, creds.Password,
	)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}

	token := a.client.Token()
	if token == "" {
		return "", fmt.Errorf(
			"login succeeded but no token returned: %v",
			resp,
		)
	}
	return token, nil
}

// LoginWithRetry attempts login with exponential backoff.
func (a *HTTPAPIAdapter) LoginWithRetry(
	ctx context.Context,
	creds Credentials,
	retries int,
) (string, error) {
	_, err := a.client.LoginWithRetry(
		ctx, creds.Username, creds.Password, retries,
	)
	if err != nil {
		return "", fmt.Errorf(
			"login with retry: %w", err,
		)
	}

	token := a.client.Token()
	if token == "" {
		return "", fmt.Errorf(
			"login succeeded but no token returned",
		)
	}
	return token, nil
}

// Get performs an HTTP GET and returns the status code and
// parsed JSON object.
func (a *HTTPAPIAdapter) Get(
	ctx context.Context, path string,
) (int, map[string]interface{}, error) {
	return a.client.Get(ctx, path)
}

// GetRaw performs an HTTP GET and returns the status code and
// raw response body.
func (a *HTTPAPIAdapter) GetRaw(
	ctx context.Context, path string,
) (int, []byte, error) {
	return a.client.GetRaw(ctx, path)
}

// GetArray performs an HTTP GET and returns the status code
// and parsed JSON array.
func (a *HTTPAPIAdapter) GetArray(
	ctx context.Context, path string,
) (int, []interface{}, error) {
	return a.client.GetArray(ctx, path)
}

// PostJSON performs an HTTP POST with a JSON body.
func (a *HTTPAPIAdapter) PostJSON(
	ctx context.Context, path, body string,
) (int, []byte, error) {
	return a.client.PostJSON(ctx, path, body)
}

// PutJSON performs an HTTP PUT with a JSON body.
func (a *HTTPAPIAdapter) PutJSON(
	ctx context.Context, path, body string,
) (int, []byte, error) {
	return a.client.PutJSON(ctx, path, body)
}

// Delete performs an HTTP DELETE.
func (a *HTTPAPIAdapter) Delete(
	ctx context.Context, path string,
) (int, []byte, error) {
	return a.client.Delete(ctx, path)
}

// WebSocketConnect establishes a WebSocket connection to the
// given path using gorilla/websocket.
func (a *HTTPAPIAdapter) WebSocketConnect(
	ctx context.Context, path string,
) (WebSocketConn, error) {
	// Convert http:// to ws:// for WebSocket.
	wsURL := a.baseURL + path
	wsURL = strings.Replace(
		wsURL, "http://", "ws://", 1,
	)
	wsURL = strings.Replace(
		wsURL, "https://", "wss://", 1,
	)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	headers := http.Header{}
	token := a.client.Token()
	if token != "" {
		headers.Set(
			"Authorization", "Bearer "+token,
		)
	}

	conn, _, err := dialer.DialContext(
		ctx, wsURL, headers,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"websocket connect %s: %w", path, err,
		)
	}

	return &gorillaWSConn{conn: conn}, nil
}

// SetToken sets the JWT token for authenticated requests.
func (a *HTTPAPIAdapter) SetToken(token string) {
	a.client.SetToken(token)
}

// Available returns true if the API server responds to a
// GET /health request with a status code below 500.
func (a *HTTPAPIAdapter) Available(
	ctx context.Context,
) bool {
	status, _, err := a.client.GetRaw(ctx, "/health")
	if err != nil {
		return false
	}
	return status < 500
}

// gorillaWSConn wraps a gorilla/websocket.Conn to implement
// the WebSocketConn interface.
type gorillaWSConn struct {
	conn *websocket.Conn
}

// Compile-time interface check.
var _ WebSocketConn = (*gorillaWSConn)(nil)

// WriteMessage sends a text message over the WebSocket.
func (c *gorillaWSConn) WriteMessage(data []byte) error {
	return c.conn.WriteMessage(
		websocket.TextMessage, data,
	)
}

// ReadMessage reads the next message from the WebSocket.
func (c *gorillaWSConn) ReadMessage() ([]byte, error) {
	_, data, err := c.conn.ReadMessage()
	return data, err
}

// Close terminates the WebSocket connection.
func (c *gorillaWSConn) Close() error {
	return c.conn.Close()
}
