package userflow

import "context"

// WebSocketConn defines the interface for WebSocket connections
// used in API testing.
type WebSocketConn interface {
	// WriteMessage sends a message over the WebSocket
	// connection.
	WriteMessage(data []byte) error

	// ReadMessage reads the next message from the WebSocket
	// connection.
	ReadMessage() ([]byte, error)

	// Close terminates the WebSocket connection.
	Close() error
}

// APIAdapter defines the interface for REST API and WebSocket
// testing. Implementations wrap HTTP clients with authentication,
// retry logic, and response parsing.
type APIAdapter interface {
	// Login authenticates with the given credentials and
	// returns a JWT token.
	Login(
		ctx context.Context, credentials Credentials,
	) (string, error)

	// LoginWithRetry attempts login with exponential backoff,
	// retrying up to the given number of times.
	LoginWithRetry(
		ctx context.Context,
		credentials Credentials,
		retries int,
	) (string, error)

	// Get performs an HTTP GET and returns the status code
	// and parsed JSON object.
	Get(
		ctx context.Context, path string,
	) (int, map[string]interface{}, error)

	// GetRaw performs an HTTP GET and returns the status code
	// and raw response body.
	GetRaw(
		ctx context.Context, path string,
	) (int, []byte, error)

	// GetArray performs an HTTP GET and returns the status
	// code and parsed JSON array.
	GetArray(
		ctx context.Context, path string,
	) (int, []interface{}, error)

	// PostJSON performs an HTTP POST with a JSON body and
	// returns the status code and raw response.
	PostJSON(
		ctx context.Context, path, body string,
	) (int, []byte, error)

	// PutJSON performs an HTTP PUT with a JSON body and
	// returns the status code and raw response.
	PutJSON(
		ctx context.Context, path, body string,
	) (int, []byte, error)

	// Delete performs an HTTP DELETE and returns the status
	// code and raw response.
	Delete(
		ctx context.Context, path string,
	) (int, []byte, error)

	// WebSocketConnect establishes a WebSocket connection to
	// the given path.
	WebSocketConnect(
		ctx context.Context, path string,
	) (WebSocketConn, error)

	// SetToken sets the JWT token for authenticated requests.
	SetToken(token string)

	// Available returns true if the API server is reachable.
	Available(ctx context.Context) bool
}
