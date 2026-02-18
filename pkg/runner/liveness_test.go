package runner

import (
	"context"
	"testing"
	"time"

	"digital.vasic.challenges/pkg/challenge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLivenessMonitor_NilProgress_NoOp(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop, stuck := startLivenessMonitor(
		nil, 100*time.Millisecond, cancel,
		nil, "test-nil",
	)
	defer stop()

	assert.Nil(t, stuck,
		"stuck channel should be nil when progress is nil")
}

func TestLivenessMonitor_ZeroThreshold_NoOp(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	progress := challenge.NewProgressReporter()
	defer progress.Close()

	stop, stuck := startLivenessMonitor(
		progress, 0, cancel,
		nil, "test-zero",
	)
	defer stop()

	assert.Nil(t, stuck,
		"stuck channel should be nil when threshold is zero")
}

func TestLivenessMonitor_DetectsStuck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	progress := challenge.NewProgressReporter()
	defer progress.Close()

	stop, stuck := startLivenessMonitor(
		progress, 100*time.Millisecond, cancel,
		nil, "test-stuck",
	)
	defer stop()

	require.NotNil(t, stuck)

	// Don't report any progress — should detect stuck.
	select {
	case <-stuck:
		// Expected: stuck detected.
	case <-time.After(2 * time.Second):
		t.Fatal("expected stuck detection within 2s")
	}

	// Context should be cancelled.
	assert.Error(t, ctx.Err())
}

func TestLivenessMonitor_ProgressPreventsStuck(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	progress := challenge.NewProgressReporter()
	defer progress.Close()

	stop, stuck := startLivenessMonitor(
		progress, 200*time.Millisecond, cancel,
		nil, "test-alive",
	)

	require.NotNil(t, stuck)

	// Report progress every 50ms for 500ms — well within
	// the 200ms stale threshold, so should never trigger.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			progress.ReportProgress("alive", map[string]any{
				"tick": i,
			})
			time.Sleep(50 * time.Millisecond)
		}
	}()

	<-done
	stop()

	// Stuck should NOT have been signaled.
	select {
	case <-stuck:
		t.Fatal("should not detect stuck when progress is reported")
	default:
		// Expected: no stuck signal.
	}
}

func TestLivenessMonitor_StopPreventsStuck(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	progress := challenge.NewProgressReporter()
	defer progress.Close()

	stop, stuck := startLivenessMonitor(
		progress, 100*time.Millisecond, cancel,
		nil, "test-stop",
	)

	require.NotNil(t, stuck)

	// Stop immediately before threshold fires.
	stop()

	// Wait longer than threshold.
	time.Sleep(200 * time.Millisecond)

	// Should NOT have signaled stuck.
	select {
	case <-stuck:
		t.Fatal("should not detect stuck after stop()")
	default:
		// Expected.
	}
}

func TestLivenessMonitor_ProgressChannelClosed(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	progress := challenge.NewProgressReporter()

	stop, stuck := startLivenessMonitor(
		progress, 5*time.Second, cancel,
		nil, "test-close",
	)
	defer stop()

	require.NotNil(t, stuck)

	// Close progress channel — monitor should exit cleanly.
	progress.Close()

	// Wait a bit to ensure monitor goroutine exits.
	time.Sleep(100 * time.Millisecond)

	// Should NOT have signaled stuck.
	select {
	case <-stuck:
		t.Fatal("should not detect stuck when channel closed")
	default:
		// Expected: clean exit.
	}
}

func TestLivenessMonitor_WithLogger(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	progress := challenge.NewProgressReporter()
	defer progress.Close()
	logger := &stubLogger{}

	stop, stuck := startLivenessMonitor(
		progress, 100*time.Millisecond, cancel,
		logger, "test-log",
	)
	defer stop()

	require.NotNil(t, stuck)

	// Wait for stuck detection.
	select {
	case <-stuck:
		// Expected.
	case <-time.After(2 * time.Second):
		t.Fatal("expected stuck detection")
	}

	// Logger should have logged the stuck event.
	logger.mu.Lock()
	msgs := make([]string, len(logger.messages))
	copy(msgs, logger.messages)
	logger.mu.Unlock()

	found := false
	for _, msg := range msgs {
		if msg == "error:challenge_stuck" {
			found = true
			break
		}
	}
	assert.True(t, found,
		"expected 'challenge_stuck' log message, got: %v", msgs)
}
