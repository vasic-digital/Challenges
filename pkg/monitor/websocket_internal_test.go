package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSocketServer_Start_ListenError tests the error path when ListenAndServe fails.
func TestWebSocketServer_Start_ListenError(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")

	// Use an invalid address to trigger an error
	server := NewWebSocketServer("invalid:99999:format", collector, dashboard)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := server.Start(ctx)
	// Should return an error for invalid address
	assert.Error(t, err)
}

// TestWebSocketServer_Start_PortInUse tests the error path when port is already in use.
func TestWebSocketServer_Start_PortInUse(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")

	// First, occupy a port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// Try to start server on the same port
	server := NewWebSocketServer(addr, collector, dashboard)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = server.Start(ctx)
	// Should return an error because port is in use
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "monitor server")
}

// TestWebSocketServer_Stop_BeforeStart tests Stop when server hasn't started.
func TestWebSocketServer_Stop_BeforeStart(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")
	server := NewWebSocketServer(":0", collector, dashboard)

	// Stop without starting - should return nil
	err := server.Stop(context.Background())
	assert.NoError(t, err)
}

// TestWebSocketServer_Stop_AfterStart tests graceful shutdown.
func TestWebSocketServer_Stop_AfterStart(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := NewWebSocketServer(addr, collector, dashboard)

	ctx, cancel := context.WithCancel(context.Background())

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	// Wait for server to be ready
	var ready bool
	for i := 0; i < 50; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, ready, "server should be listening")

	// Give time for server to fully initialize
	time.Sleep(50 * time.Millisecond)

	// Stop the server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	err = server.Stop(stopCtx)
	assert.NoError(t, err)

	// Cancel the start context
	cancel()

	// Wait for server to stop
	select {
	case err := <-serverErr:
		// Either nil or ErrServerClosed is fine
		if err != nil && err != http.ErrServerClosed {
			t.Logf("server returned: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server didn't shut down")
	}
}

// TestWebSocketServer_handleSSE_ChannelClose tests the SSE handler
// when the channel is closed.
func TestWebSocketServer_handleSSE_ChannelClose(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")
	server := NewWebSocketServer(":0", collector, dashboard)

	// Create request with context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req := httptest.NewRequest("GET", "/events", nil)
	req = req.WithContext(ctx)

	// Use the sseRecorder that implements http.Flusher
	rec := &testSSERecorder{
		header: make(http.Header),
		body:   make([]byte, 0, 1024),
	}

	// Handle SSE in goroutine
	done := make(chan struct{})
	go func() {
		server.handleSSE(rec, req)
		close(done)
	}()

	// Wait a bit, then cancel
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Handler exited correctly
	case <-time.After(2 * time.Second):
		t.Fatal("handler didn't exit")
	}
}

// TestWebSocketServer_Start_EventHandler tests that events are forwarded.
func TestWebSocketServer_Start_EventHandler(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := NewWebSocketServer(addr, collector, dashboard)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	go func() {
		server.Start(ctx)
	}()

	// Wait for server to be ready
	var ready bool
	for i := 0; i < 50; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, ready)

	// Give server time to register event handler
	time.Sleep(50 * time.Millisecond)

	// Emit an event
	collector.Emit(ChallengeEvent{
		Type:        EventStarted,
		ChallengeID: "test-challenge",
		Name:        "Test",
	})

	// Check dashboard was updated
	time.Sleep(50 * time.Millisecond)
	snap := dashboard.Snapshot()
	_, exists := snap.Challenges["test-challenge"]
	assert.True(t, exists, "dashboard should contain test-challenge")
}

// TestWebSocketServer_handleSSE_JsonMarshalError tests when dashboard
// snapshot can't be marshaled (edge case).
func TestWebSocketServer_handleSSE_JsonMarshalSuccess(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")
	server := NewWebSocketServer(":0", collector, dashboard)

	// Add some data to dashboard
	dashboard.UpdateFromEvent(ChallengeEvent{
		Type:        EventCompleted,
		ChallengeID: "ch-1",
		Name:        "Test",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest("GET", "/events", nil)
	req = req.WithContext(ctx)

	rec := &testSSERecorder{
		header: make(http.Header),
		body:   make([]byte, 0, 4096),
	}

	done := make(chan struct{})
	go func() {
		server.handleSSE(rec, req)
		close(done)
	}()

	// Wait for some data
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-done

	// Check that initial snapshot was sent
	assert.Contains(t, string(rec.body), "event: dashboard")
}

// TestBroadcast_EmitFromCollector tests that collector OnEvent triggers broadcast.
func TestBroadcast_EmitFromCollector(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")
	server := NewWebSocketServer(":0", collector, dashboard)

	// Add a client
	ch := make(chan []byte, 32)
	server.mu.Lock()
	server.clients[ch] = struct{}{}
	server.mu.Unlock()

	// Manually trigger broadcast (simulating what OnEvent does)
	event := ChallengeEvent{
		Type:        EventStarted,
		ChallengeID: "test",
		Name:        "Test Challenge",
	}
	data, _ := json.Marshal(event)
	server.broadcast(data)

	// Check client received data
	select {
	case received := <-ch:
		assert.Contains(t, string(received), "test")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("client didn't receive data")
	}
}

// testSSERecorder implements ResponseWriter and Flusher for testing.
type testSSERecorder struct {
	header http.Header
	body   []byte
}

func (r *testSSERecorder) Header() http.Header {
	return r.header
}

func (r *testSSERecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}

func (r *testSSERecorder) WriteHeader(statusCode int) {
	// No-op
}

func (r *testSSERecorder) Flush() {
	// No-op for testing
}
