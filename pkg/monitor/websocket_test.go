package monitor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebSocketServer(t *testing.T) {
	tests := []struct {
		name      string
		addr      string
		collector *EventCollector
		dashboard *DashboardData
	}{
		{
			name:      "with default port",
			addr:      ":8080",
			collector: NewEventCollector(),
			dashboard: NewDashboardData("run-1"),
		},
		{
			name:      "with localhost and custom port",
			addr:      "localhost:9000",
			collector: NewEventCollector(),
			dashboard: NewDashboardData("run-2"),
		},
		{
			name:      "with empty address",
			addr:      "",
			collector: NewEventCollector(),
			dashboard: NewDashboardData("run-3"),
		},
		{
			name:      "with IP address",
			addr:      "127.0.0.1:3000",
			collector: NewEventCollector(),
			dashboard: NewDashboardData("run-4"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewWebSocketServer(tt.addr, tt.collector, tt.dashboard)

			assert.NotNil(t, server)
			assert.Equal(t, tt.addr, server.addr)
			assert.Equal(t, tt.collector, server.collector)
			assert.Equal(t, tt.dashboard, server.dashboard)
			assert.NotNil(t, server.clients)
			assert.Empty(t, server.clients)
		})
	}
}

func TestWebSocketServer_Start(t *testing.T) {
	t.Run("starts and serves endpoints", func(t *testing.T) {
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
		serverErr := make(chan error, 1)
		go func() {
			serverErr <- server.Start(ctx)
		}()

		// Wait for server to start
		var connected bool
		for i := 0; i < 50; i++ {
			conn, err := net.Dial("tcp", addr)
			if err == nil {
				conn.Close()
				connected = true
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		require.True(t, connected, "server should be listening")

		// Test health endpoint
		resp, err := http.Get("http://" + addr + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "ok", string(body))

		// Test dashboard endpoint
		resp, err = http.Get("http://" + addr + "/dashboard")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Cancel and wait for shutdown
		cancel()
		select {
		case err := <-serverErr:
			assert.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("server didn't shut down in time")
		}
	})

	t.Run("returns error for invalid address", func(t *testing.T) {
		collector := NewEventCollector()
		dashboard := NewDashboardData("run-1")
		// Use an invalid address to trigger an error
		server := NewWebSocketServer("invalid:address:format:99999", collector, dashboard)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := server.Start(ctx)
		// Expect an error due to invalid address format
		assert.Error(t, err)
	})
}

func TestWebSocketServer_Stop(t *testing.T) {
	t.Run("graceful shutdown via context cancellation", func(t *testing.T) {
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

		// Wait for server to be fully ready by making successful HTTP requests
		// This ensures Start() has fully initialized the server
		var ready bool
		for i := 0; i < 100; i++ {
			resp, err := http.Get("http://" + addr + "/health")
			if err == nil {
				resp.Body.Close()
				ready = true
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		require.True(t, ready, "server should be listening and responding")

		// Small delay to ensure Start() goroutine has completed all setup
		time.Sleep(50 * time.Millisecond)

		// Cancel context to trigger shutdown
		cancel()

		// Wait for server to stop
		select {
		case err := <-serverErr:
			// Server stopped with nil error (normal shutdown) or context cancelled
			// Both are acceptable
			if err != nil && err != context.Canceled {
				assert.NoError(t, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("server didn't shut down in time")
		}

		// Verify server is no longer accepting connections
		time.Sleep(100 * time.Millisecond) // Give time for port to be released
		_, err = net.DialTimeout("tcp", addr, 100*time.Millisecond)
		assert.Error(t, err, "server should no longer be accepting connections")
	})

	t.Run("stop before start returns nil", func(t *testing.T) {
		collector := NewEventCollector()
		dashboard := NewDashboardData("run-1")
		server := NewWebSocketServer(":0", collector, dashboard)

		ctx := context.Background()
		err := server.Stop(ctx)
		assert.NoError(t, err)
	})
}

func TestWebSocketServer_handleSSE(t *testing.T) {
	t.Run("streams events to client", func(t *testing.T) {
		collector := NewEventCollector()
		dashboard := NewDashboardData("run-1")
		server := NewWebSocketServer(":0", collector, dashboard)

		// Create request with context that can be cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req := httptest.NewRequest("GET", "/events", nil)
		req = req.WithContext(ctx)

		// Create a pipe so we can read the SSE stream
		pr, pw := io.Pipe()
		rec := &sseRecorder{
			header: make(http.Header),
			body:   pw,
		}

		// Handle SSE in goroutine
		done := make(chan struct{})
		go func() {
			server.handleSSE(rec, req)
			pw.Close()
			close(done)
		}()

		// Wait a bit for handler to start
		time.Sleep(50 * time.Millisecond)

		// Broadcast an event
		testEvent := []byte(`{"type":"test","message":"hello"}`)
		server.broadcast(testEvent)

		// Read the response
		reader := bufio.NewReader(pr)

		// Read initial dashboard event
		line, err := reader.ReadString('\n')
		require.NoError(t, err)
		assert.Contains(t, line, "event: dashboard")

		// Read data line
		line, err = reader.ReadString('\n')
		require.NoError(t, err)
		assert.Contains(t, line, "data:")

		// Skip empty line
		reader.ReadString('\n')

		// Read the broadcasted event
		line, err = reader.ReadString('\n')
		require.NoError(t, err)
		assert.Contains(t, line, "event: challenge")

		line, err = reader.ReadString('\n')
		require.NoError(t, err)
		assert.Contains(t, line, `"type":"test"`)

		// Cancel context to close connection
		cancel()

		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("handler didn't exit in time")
		}
	})

	t.Run("sets correct headers", func(t *testing.T) {
		collector := NewEventCollector()
		dashboard := NewDashboardData("run-1")
		server := NewWebSocketServer(":0", collector, dashboard)

		// Create request with context that can be cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		req := httptest.NewRequest("GET", "/events", nil)
		req = req.WithContext(ctx)

		// Create a pipe so we can read the SSE stream
		pr, pw := io.Pipe()
		rec := &sseRecorder{
			header: make(http.Header),
			body:   pw,
		}

		// Handle SSE in goroutine
		done := make(chan struct{})
		go func() {
			server.handleSSE(rec, req)
			pw.Close()
			close(done)
		}()

		// Wait for handler to start and read some data
		reader := bufio.NewReader(pr)
		line, err := reader.ReadString('\n')
		if err == nil {
			assert.Contains(t, line, "event: dashboard")
		}

		// Cancel context to close connection
		cancel()

		select {
		case <-done:
			// Handler exited cleanly
		case <-time.After(2 * time.Second):
			t.Fatal("handler didn't exit in time")
		}

		// Check headers were set
		assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
		assert.Equal(t, "keep-alive", rec.Header().Get("Connection"))
		assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("returns error when flusher not supported", func(t *testing.T) {
		collector := NewEventCollector()
		dashboard := NewDashboardData("run-1")
		server := NewWebSocketServer(":0", collector, dashboard)

		req := httptest.NewRequest("GET", "/events", nil)

		rec := &basicResponseWriter{
			header: make(http.Header),
			body:   &bufferWriter{},
			code:   0,
		}

		// Handler should return immediately when flusher is not supported
		// No goroutine needed since it returns synchronously
		server.handleSSE(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.code)
		assert.Contains(t, rec.body.String(), "streaming not supported")
	})
}

func TestWebSocketServer_handleDashboard(t *testing.T) {
	tests := []struct {
		name        string
		setupDash   func(*DashboardData)
		checkResult func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "returns empty dashboard",
			setupDash: func(d *DashboardData) {
				// No setup, empty dashboard
			},
			checkResult: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

				var data DashboardData
				err := json.Unmarshal(rec.Body.Bytes(), &data)
				require.NoError(t, err)
				assert.Equal(t, "running", data.Status)
				assert.Empty(t, data.Challenges)
			},
		},
		{
			name: "returns dashboard with challenges",
			setupDash: func(d *DashboardData) {
				d.UpdateFromEvent(ChallengeEvent{
					Type:        EventStarted,
					ChallengeID: "ch-1",
					Name:        "Test Challenge",
				})
				d.UpdateFromEvent(ChallengeEvent{
					Type:        EventCompleted,
					ChallengeID: "ch-1",
					Name:        "Test Challenge",
					Duration:    time.Second,
				})
			},
			checkResult: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Code)

				var data DashboardData
				err := json.Unmarshal(rec.Body.Bytes(), &data)
				require.NoError(t, err)
				assert.Len(t, data.Challenges, 1)
				assert.Equal(t, "passed", data.Challenges["ch-1"].Status)
				assert.Equal(t, 1, data.Summary.Passed)
			},
		},
		{
			name: "returns dashboard with mixed statuses",
			setupDash: func(d *DashboardData) {
				d.UpdateFromEvent(ChallengeEvent{
					Type: EventCompleted, ChallengeID: "ch-1", Name: "Pass",
				})
				d.UpdateFromEvent(ChallengeEvent{
					Type: EventFailed, ChallengeID: "ch-2", Name: "Fail", Message: "error",
				})
				d.UpdateFromEvent(ChallengeEvent{
					Type: EventSkipped, ChallengeID: "ch-3", Name: "Skip",
				})
			},
			checkResult: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var data DashboardData
				err := json.Unmarshal(rec.Body.Bytes(), &data)
				require.NoError(t, err)
				assert.Equal(t, 3, data.Summary.Total)
				assert.Equal(t, 1, data.Summary.Passed)
				assert.Equal(t, 1, data.Summary.Failed)
				assert.Equal(t, 1, data.Summary.Skipped)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewEventCollector()
			dashboard := NewDashboardData("run-1")
			tt.setupDash(dashboard)

			server := NewWebSocketServer(":0", collector, dashboard)

			req := httptest.NewRequest("GET", "/dashboard", nil)
			rec := httptest.NewRecorder()

			server.handleDashboard(rec, req)

			tt.checkResult(t, rec)
		})
	}
}

func TestWebSocketServer_broadcast(t *testing.T) {
	t.Run("broadcasts to all clients", func(t *testing.T) {
		collector := NewEventCollector()
		dashboard := NewDashboardData("run-1")
		server := NewWebSocketServer(":0", collector, dashboard)

		// Add multiple clients
		ch1 := make(chan []byte, 32)
		ch2 := make(chan []byte, 32)
		ch3 := make(chan []byte, 32)

		server.mu.Lock()
		server.clients[ch1] = struct{}{}
		server.clients[ch2] = struct{}{}
		server.clients[ch3] = struct{}{}
		server.mu.Unlock()

		// Broadcast data
		testData := []byte(`{"event":"test"}`)
		server.broadcast(testData)

		// Verify all clients received the data
		select {
		case data := <-ch1:
			assert.Equal(t, testData, data)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("client 1 didn't receive data")
		}

		select {
		case data := <-ch2:
			assert.Equal(t, testData, data)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("client 2 didn't receive data")
		}

		select {
		case data := <-ch3:
			assert.Equal(t, testData, data)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("client 3 didn't receive data")
		}
	})

	t.Run("skips slow clients", func(t *testing.T) {
		collector := NewEventCollector()
		dashboard := NewDashboardData("run-1")
		server := NewWebSocketServer(":0", collector, dashboard)

		// Add a slow client with full buffer
		slowCh := make(chan []byte) // Unbuffered - will block
		fastCh := make(chan []byte, 32)

		server.mu.Lock()
		server.clients[slowCh] = struct{}{}
		server.clients[fastCh] = struct{}{}
		server.mu.Unlock()

		// Broadcast should not block even if slow client can't receive
		done := make(chan struct{})
		go func() {
			server.broadcast([]byte(`{"test":"data"}`))
			close(done)
		}()

		select {
		case <-done:
			// Success - broadcast completed without blocking
		case <-time.After(100 * time.Millisecond):
			t.Fatal("broadcast blocked on slow client")
		}

		// Fast client should have received the data
		select {
		case data := <-fastCh:
			assert.Equal(t, []byte(`{"test":"data"}`), data)
		default:
			t.Fatal("fast client didn't receive data")
		}
	})

	t.Run("handles no clients", func(t *testing.T) {
		collector := NewEventCollector()
		dashboard := NewDashboardData("run-1")
		server := NewWebSocketServer(":0", collector, dashboard)

		// Should not panic with no clients
		assert.NotPanics(t, func() {
			server.broadcast([]byte(`{"test":"data"}`))
		})
	})

	t.Run("concurrent broadcast and client modification", func(t *testing.T) {
		collector := NewEventCollector()
		dashboard := NewDashboardData("run-1")
		server := NewWebSocketServer(":0", collector, dashboard)

		var wg sync.WaitGroup

		// Spawn broadcasters
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					server.broadcast([]byte(fmt.Sprintf(`{"id":%d}`, i*100+j)))
				}
			}(i)
		}

		// Spawn client adders/removers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					ch := make(chan []byte, 32)
					server.mu.Lock()
					server.clients[ch] = struct{}{}
					server.mu.Unlock()

					time.Sleep(time.Microsecond)

					server.mu.Lock()
					delete(server.clients, ch)
					server.mu.Unlock()
				}
			}()
		}

		wg.Wait()
	})
}

func TestBuildDashboardData(t *testing.T) {
	tests := []struct {
		name      string
		events    []ChallengeEvent
		wantTotal int
		wantStats DashboardSummary
	}{
		{
			name:      "empty collector",
			events:    []ChallengeEvent{},
			wantTotal: 0,
			wantStats: DashboardSummary{},
		},
		{
			name: "single passed challenge",
			events: []ChallengeEvent{
				{Type: EventStarted, ChallengeID: "ch-1", Name: "Test"},
				{Type: EventCompleted, ChallengeID: "ch-1", Name: "Test", Duration: time.Second},
			},
			wantTotal: 1,
			wantStats: DashboardSummary{Total: 1, Passed: 1, PassRate: 100},
		},
		{
			name: "single failed challenge",
			events: []ChallengeEvent{
				{Type: EventStarted, ChallengeID: "ch-1", Name: "Test"},
				{Type: EventFailed, ChallengeID: "ch-1", Name: "Test", Message: "error"},
			},
			wantTotal: 1,
			wantStats: DashboardSummary{Total: 1, Failed: 1, PassRate: 0},
		},
		{
			name: "mixed results",
			events: []ChallengeEvent{
				{Type: EventCompleted, ChallengeID: "ch-1", Name: "Pass1"},
				{Type: EventCompleted, ChallengeID: "ch-2", Name: "Pass2"},
				{Type: EventFailed, ChallengeID: "ch-3", Name: "Fail1", Message: "err"},
				{Type: EventSkipped, ChallengeID: "ch-4", Name: "Skip1"},
				{Type: EventTimedOut, ChallengeID: "ch-5", Name: "Timeout1"},
			},
			wantTotal: 5,
			wantStats: DashboardSummary{
				Total:    5,
				Passed:   2,
				Failed:   1,
				Skipped:  1,
				PassRate: 200.0 / 3.0, // 2 passed out of 3 completed (passed+failed)
			},
		},
		{
			name: "challenge with multiple events",
			events: []ChallengeEvent{
				{Type: EventStarted, ChallengeID: "ch-1", Name: "Test"},
				{Type: EventCompleted, ChallengeID: "ch-1", Name: "Test"},
			},
			wantTotal: 1,
			wantStats: DashboardSummary{Total: 1, Passed: 1, PassRate: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewEventCollector()
			for _, event := range tt.events {
				collector.Emit(event)
			}

			result := BuildDashboardData(collector)

			assert.NotNil(t, result)
			assert.Equal(t, "snapshot", result.RunID)
			assert.Len(t, result.Challenges, tt.wantTotal)
			assert.Equal(t, tt.wantStats.Total, result.Summary.Total)
			assert.Equal(t, tt.wantStats.Passed, result.Summary.Passed)
			assert.Equal(t, tt.wantStats.Failed, result.Summary.Failed)
			assert.Equal(t, tt.wantStats.Skipped, result.Summary.Skipped)
			if tt.wantStats.Total > 0 {
				assert.InDelta(t, tt.wantStats.PassRate, result.Summary.PassRate, 0.01)
			}
		})
	}
}

func TestBuildDashboardData_ChallengeStates(t *testing.T) {
	collector := NewEventCollector()
	collector.Emit(ChallengeEvent{
		Type:        EventStarted,
		ChallengeID: "ch-1",
		Name:        "Running Challenge",
	})
	collector.Emit(ChallengeEvent{
		Type:        EventCompleted,
		ChallengeID: "ch-2",
		Name:        "Passed Challenge",
		Duration:    2 * time.Second,
	})
	collector.Emit(ChallengeEvent{
		Type:        EventFailed,
		ChallengeID: "ch-3",
		Name:        "Failed Challenge",
		Message:     "assertion failed",
	})
	collector.Emit(ChallengeEvent{
		Type:        EventSkipped,
		ChallengeID: "ch-4",
		Name:        "Skipped Challenge",
	})
	collector.Emit(ChallengeEvent{
		Type:        EventTimedOut,
		ChallengeID: "ch-5",
		Name:        "Timed Out Challenge",
	})

	result := BuildDashboardData(collector)

	assert.Equal(t, "running", result.Challenges["ch-1"].Status)
	assert.Equal(t, "passed", result.Challenges["ch-2"].Status)
	assert.Equal(t, "failed", result.Challenges["ch-3"].Status)
	assert.Equal(t, "assertion failed", result.Challenges["ch-3"].Message)
	assert.Equal(t, "skipped", result.Challenges["ch-4"].Status)
	assert.Equal(t, "timed_out", result.Challenges["ch-5"].Status)
}

// sseRecorder is a custom recorder that implements http.Flusher
type sseRecorder struct {
	header http.Header
	body   io.Writer
}

func (r *sseRecorder) Header() http.Header {
	return r.header
}

func (r *sseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *sseRecorder) WriteHeader(statusCode int) {
	// No-op for SSE
}

func (r *sseRecorder) Flush() {
	// No-op for testing
}


// bufferWriter is a simple buffer for writing
type bufferWriter struct {
	buf []byte
}

func (b *bufferWriter) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *bufferWriter) String() string {
	return string(b.buf)
}

// basicResponseWriter is a minimal ResponseWriter that does NOT implement http.Flusher
type basicResponseWriter struct {
	header http.Header
	body   *bufferWriter
	code   int
}

func (r *basicResponseWriter) Header() http.Header {
	return r.header
}

func (r *basicResponseWriter) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *basicResponseWriter) WriteHeader(statusCode int) {
	r.code = statusCode
}

// Ensure basicResponseWriter does NOT implement http.Flusher
var _ http.ResponseWriter = (*basicResponseWriter)(nil)

func TestDashboardData_UpdateFromEvent_AllTypes(t *testing.T) {
	tests := []struct {
		name       string
		eventType  EventType
		wantStatus string
	}{
		{
			name:       "started event",
			eventType:  EventStarted,
			wantStatus: "running",
		},
		{
			name:       "completed event",
			eventType:  EventCompleted,
			wantStatus: "passed",
		},
		{
			name:       "failed event",
			eventType:  EventFailed,
			wantStatus: "failed",
		},
		{
			name:       "skipped event",
			eventType:  EventSkipped,
			wantStatus: "skipped",
		},
		{
			name:       "timed_out event",
			eventType:  EventTimedOut,
			wantStatus: "timed_out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dashboard := NewDashboardData("run-1")
			dashboard.UpdateFromEvent(ChallengeEvent{
				Type:        tt.eventType,
				ChallengeID: "ch-1",
				Name:        "Test",
			})

			assert.Equal(t, tt.wantStatus, dashboard.Challenges["ch-1"].Status)
		})
	}
}

func TestDashboardData_UpdateFromEvent_UpdatesSummary(t *testing.T) {
	dashboard := NewDashboardData("run-1")

	// Add passed challenge
	dashboard.UpdateFromEvent(ChallengeEvent{
		Type: EventCompleted, ChallengeID: "ch-1", Name: "Pass1",
	})
	assert.Equal(t, 1, dashboard.Summary.Total)
	assert.Equal(t, 1, dashboard.Summary.Passed)

	// Add failed challenge
	dashboard.UpdateFromEvent(ChallengeEvent{
		Type: EventFailed, ChallengeID: "ch-2", Name: "Fail1",
	})
	assert.Equal(t, 2, dashboard.Summary.Total)
	assert.Equal(t, 1, dashboard.Summary.Failed)

	// Add skipped challenge
	dashboard.UpdateFromEvent(ChallengeEvent{
		Type: EventSkipped, ChallengeID: "ch-3", Name: "Skip1",
	})
	assert.Equal(t, 3, dashboard.Summary.Total)
	assert.Equal(t, 1, dashboard.Summary.Skipped)
}

func TestWebSocketServer_Start_MarshalError(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := NewWebSocketServer(addr, collector, dashboard)

	// Save original and restore after test
	originalMarshal := jsonMarshal
	t.Cleanup(func() { jsonMarshal = originalMarshal })

	// Inject a failing marshaler
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, assert.AnError
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	go func() {
		_ = server.Start(ctx)
	}()

	// Wait for server to start
	var connected bool
	for i := 0; i < 50; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			connected = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.True(t, connected, "server should be listening")

	// Emit an event - the marshal error should be handled gracefully
	collector.Emit(ChallengeEvent{
		Type:        EventStarted,
		ChallengeID: "ch-1",
		Name:        "Test Challenge",
	})

	// Give time for event to be processed
	time.Sleep(50 * time.Millisecond)

	// Server should still be running
	resp, err := http.Get("http://" + addr + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()
}

func TestWebSocketServer_handleSSE_MarshalError(t *testing.T) {
	collector := NewEventCollector()
	dashboard := NewDashboardData("run-1")
	server := NewWebSocketServer(":0", collector, dashboard)

	// Save original and restore after test
	originalMarshal := jsonMarshal
	t.Cleanup(func() { jsonMarshal = originalMarshal })

	// Inject a failing marshaler
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, assert.AnError
	}

	// Create request with context that can be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest("GET", "/events", nil)
	req = req.WithContext(ctx)

	// Create a pipe so we can read the SSE stream
	pr, pw := io.Pipe()
	rec := &sseRecorder{
		header: make(http.Header),
		body:   pw,
	}

	// Handle SSE in goroutine
	done := make(chan struct{})
	go func() {
		server.handleSSE(rec, req)
		pw.Close()
		close(done)
	}()

	// Read what we can from the response
	reader := bufio.NewReader(pr)
	_, _ = reader.ReadString('\n') // May fail due to marshal error

	cancel()

	select {
	case <-done:
		// Handler exited cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("handler didn't exit in time")
	}
}

// Note: The `!ok` branch in handleSSE (lines 122-125) is defensive code
// that cannot be triggered in normal operation because:
// 1. The channel is created by handleSSE itself (line 96)
// 2. The channel is only closed in the defer block (line 108)
// 3. The defer only runs after the function returns
// This branch protects against theoretical race conditions but is
// unreachable in the current implementation.
