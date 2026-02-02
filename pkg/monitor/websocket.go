package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// WebSocketServer provides a WebSocket endpoint for live dashboard updates.
// This is a simplified implementation that uses Server-Sent Events (SSE)
// to avoid external dependencies. For full WebSocket support, users can
// wrap the EventCollector with gorilla/websocket.
type WebSocketServer struct {
	mu        sync.RWMutex
	collector *EventCollector
	dashboard *DashboardData
	clients   map[chan []byte]struct{}
	addr      string
	server    *http.Server
}

// NewWebSocketServer creates a new SSE server for live monitoring.
func NewWebSocketServer(addr string, collector *EventCollector, dashboard *DashboardData) *WebSocketServer {
	return &WebSocketServer{
		addr:      addr,
		collector: collector,
		dashboard: dashboard,
		clients:   make(map[chan []byte]struct{}),
	}
}

// Start begins serving the SSE endpoint.
func (s *WebSocketServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.handleSSE)
	mux.HandleFunc("/dashboard", s.handleDashboard)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// Register event handler to broadcast to clients
	s.collector.OnEvent(func(event ChallengeEvent) {
		s.dashboard.UpdateFromEvent(event)
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		s.broadcast(data)
	})

	go func() {
		<-ctx.Done()
		s.server.Close()
	}()

	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("monitor server: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the server.
func (s *WebSocketServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *WebSocketServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan []byte, 32)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
		close(ch)
	}()

	// Send initial dashboard state
	snap := s.dashboard.Snapshot()
	if data, err := json.Marshal(snap); err == nil {
		fmt.Fprintf(w, "event: dashboard\ndata: %s\n\n", data)
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: challenge\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *WebSocketServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	snap := s.dashboard.Snapshot()
	json.NewEncoder(w).Encode(snap)
}

func (s *WebSocketServer) broadcast(data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- data:
		default:
			// Client too slow, skip
		}
	}
}
