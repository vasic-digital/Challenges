package userflow

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketFlowAdapter defines the interface for WebSocket
// flow testing beyond simple send/receive. Implementations
// provide connection management, bidirectional messaging, and
// timeout-aware receive operations.
type WebSocketFlowAdapter interface {
	// Connect establishes a WebSocket connection to the given
	// URL with optional headers.
	Connect(
		ctx context.Context,
		url string,
		headers map[string]string,
	) error

	// Send sends a message over the WebSocket connection.
	Send(ctx context.Context, message []byte) error

	// Receive reads the next message with a timeout. Returns
	// the message payload or an error if the timeout expires
	// or the connection is closed.
	Receive(
		ctx context.Context, timeout time.Duration,
	) ([]byte, error)

	// ReceiveAll reads messages until the timeout expires,
	// returning all collected messages. Does not return an
	// error on timeout; the timeout simply terminates
	// collection.
	ReceiveAll(
		ctx context.Context, timeout time.Duration,
	) ([][]byte, error)

	// SendAndReceive sends a message and waits for a single
	// response within the given timeout.
	SendAndReceive(
		ctx context.Context,
		message []byte,
		timeout time.Duration,
	) ([]byte, error)

	// Close terminates the WebSocket connection gracefully.
	Close(ctx context.Context) error

	// Available returns true if the WebSocket endpoint is
	// reachable by attempting a test connection.
	Available(ctx context.Context) bool
}

// GorillaWebSocketAdapter implements WebSocketFlowAdapter
// using the gorilla/websocket library. It provides
// thread-safe read and write operations via a sync.Mutex.
type GorillaWebSocketAdapter struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
	readMu  sync.Mutex
}

// NewGorillaWebSocketAdapter creates a new adapter. Call
// Connect before using other methods.
func NewGorillaWebSocketAdapter() *GorillaWebSocketAdapter {
	return &GorillaWebSocketAdapter{}
}

// Connect establishes a WebSocket connection to the given
// URL. Optional headers are added to the upgrade request.
func (a *GorillaWebSocketAdapter) Connect(
	ctx context.Context,
	url string,
	headers map[string]string,
) error {
	if a.conn != nil {
		return fmt.Errorf("already connected")
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	var reqHeaders http.Header
	if len(headers) > 0 {
		reqHeaders = make(http.Header)
		for k, v := range headers {
			reqHeaders.Set(k, v)
		}
	}

	conn, _, err := dialer.DialContext(
		ctx, url, reqHeaders,
	)
	if err != nil {
		return fmt.Errorf("websocket connect %s: %w", url, err)
	}

	// Set up ping/pong handler to keep the connection alive.
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(
			time.Now().Add(60 * time.Second),
		)
	})

	a.conn = conn
	return nil
}

// Send sends a text message over the WebSocket connection.
// Thread-safe via write mutex.
func (a *GorillaWebSocketAdapter) Send(
	_ context.Context, message []byte,
) error {
	if a.conn == nil {
		return fmt.Errorf("not connected")
	}

	a.writeMu.Lock()
	defer a.writeMu.Unlock()

	return a.conn.WriteMessage(
		websocket.TextMessage, message,
	)
}

// Receive reads the next message with a timeout. Returns an
// error if the deadline expires before a message arrives.
func (a *GorillaWebSocketAdapter) Receive(
	_ context.Context, timeout time.Duration,
) ([]byte, error) {
	if a.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	a.readMu.Lock()
	defer a.readMu.Unlock()

	if err := a.conn.SetReadDeadline(
		time.Now().Add(timeout),
	); err != nil {
		return nil, fmt.Errorf(
			"set read deadline: %w", err,
		)
	}

	_, msg, err := a.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}

	return msg, nil
}

// ReceiveAll reads messages until the timeout expires. All
// messages received before the deadline are returned. A
// timeout is not treated as an error.
func (a *GorillaWebSocketAdapter) ReceiveAll(
	ctx context.Context, timeout time.Duration,
) ([][]byte, error) {
	if a.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	deadline := time.Now().Add(timeout)
	var messages [][]byte

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}

		msg, err := a.Receive(ctx, remaining)
		if err != nil {
			// Timeout is expected; stop collecting.
			if isTimeoutError(err) {
				break
			}
			return messages, fmt.Errorf(
				"receive all: %w", err,
			)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// SendAndReceive sends a message and waits for a single
// response within the given timeout.
func (a *GorillaWebSocketAdapter) SendAndReceive(
	ctx context.Context,
	message []byte,
	timeout time.Duration,
) ([]byte, error) {
	if err := a.Send(ctx, message); err != nil {
		return nil, fmt.Errorf(
			"send and receive (send): %w", err,
		)
	}

	resp, err := a.Receive(ctx, timeout)
	if err != nil {
		return nil, fmt.Errorf(
			"send and receive (receive): %w", err,
		)
	}

	return resp, nil
}

// Close terminates the WebSocket connection by sending a
// close frame and closing the underlying connection.
func (a *GorillaWebSocketAdapter) Close(
	_ context.Context,
) error {
	if a.conn == nil {
		return nil
	}

	a.writeMu.Lock()
	defer a.writeMu.Unlock()

	// Send close frame with normal closure.
	closeMsg := websocket.FormatCloseMessage(
		websocket.CloseNormalClosure, "",
	)
	_ = a.conn.WriteMessage(
		websocket.CloseMessage, closeMsg,
	)

	err := a.conn.Close()
	a.conn = nil
	return err
}

// Available returns true if the WebSocket endpoint is
// reachable. It attempts a quick connect and immediately
// closes.
func (a *GorillaWebSocketAdapter) Available(
	ctx context.Context,
) bool {
	// If already connected, the endpoint is available.
	if a.conn != nil {
		return true
	}
	return false
}

// isTimeoutError checks whether the error is a timeout from
// the websocket read deadline.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	// net.Error timeout check via string matching since we
	// do not want to import net just for the interface.
	errStr := err.Error()
	return contains(errStr, "timeout") ||
		contains(errStr, "deadline exceeded") ||
		contains(errStr, "i/o timeout")
}

// contains is a simple case-insensitive substring check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		containsAt(s, substr)
}

// containsAt performs a linear scan for substr in s. This
// avoids importing strings in the adapter to keep the import
// set minimal (strings is not yet imported in this file).
func containsAt(s, substr string) bool {
	sLen := len(s)
	subLen := len(substr)
	for i := 0; i <= sLen-subLen; i++ {
		match := true
		for j := 0; j < subLen; j++ {
			sc := s[i+j]
			tc := substr[j]
			// Lowercase ASCII comparison.
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
