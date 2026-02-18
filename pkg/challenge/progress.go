package challenge

import (
	"sync"
	"time"
)

// ProgressUpdate represents a single progress report from a
// running challenge. Challenges emit these periodically to
// signal that they are alive and making forward progress.
type ProgressUpdate struct {
	// Timestamp is when the progress was reported.
	Timestamp time.Time `json:"timestamp"`

	// Message is a human-readable description of what
	// the challenge is currently doing.
	Message string `json:"message"`

	// Data holds arbitrary key-value progress metrics
	// (e.g., "files_scanned": 5000, "bytes_processed": 1e9).
	Data map[string]any `json:"data,omitempty"`
}

// ProgressReporter allows challenges to signal that they are
// alive and making forward progress. The runner's liveness
// monitor watches for these updates and cancels execution
// only when no progress has been reported within the
// configured stale threshold.
//
// Unlike timeouts, which limit total duration, the stale
// threshold limits idle duration. A challenge scanning 100k
// files over NAS can run for hours — as long as it keeps
// reporting progress, it will never be killed.
type ProgressReporter struct {
	ch     chan ProgressUpdate
	mu     sync.Mutex
	last   *ProgressUpdate
	closed bool
}

// NewProgressReporter creates a buffered progress channel.
// The buffer size prevents slow consumers from blocking the
// challenge — older updates are dropped if the buffer fills.
func NewProgressReporter() *ProgressReporter {
	return &ProgressReporter{
		ch: make(chan ProgressUpdate, 64),
	}
}

// ReportProgress emits a progress update. This is safe to
// call from any goroutine. If the buffer is full, the update
// is dropped (the liveness monitor still sees the most recent
// buffered update).
func (p *ProgressReporter) ReportProgress(
	msg string,
	data map[string]any,
) {
	update := ProgressUpdate{
		Timestamp: time.Now(),
		Message:   msg,
		Data:      data,
	}

	p.mu.Lock()
	p.last = &update
	closed := p.closed
	p.mu.Unlock()

	if closed {
		return
	}

	// Non-blocking send; drop if buffer is full.
	select {
	case p.ch <- update:
	default:
	}
}

// Channel returns the read-only channel for consuming
// progress updates. The runner's liveness monitor reads
// from this channel.
func (p *ProgressReporter) Channel() <-chan ProgressUpdate {
	return p.ch
}

// LastUpdate returns the most recent progress update, or nil
// if no progress has been reported yet.
func (p *ProgressReporter) LastUpdate() *ProgressUpdate {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.last
}

// Close signals that no more progress updates will be sent.
// Safe to call multiple times.
func (p *ProgressReporter) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.closed {
		p.closed = true
		close(p.ch)
	}
}
