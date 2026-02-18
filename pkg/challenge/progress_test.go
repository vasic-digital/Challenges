package challenge

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressReporter_New(t *testing.T) {
	p := NewProgressReporter()
	require.NotNil(t, p)
	assert.NotNil(t, p.Channel())
	assert.Nil(t, p.LastUpdate())
}

func TestProgressReporter_ReportProgress(t *testing.T) {
	p := NewProgressReporter()
	defer p.Close()

	p.ReportProgress("scanning files", map[string]any{
		"files_scanned": 100,
	})

	// Should be available on channel.
	select {
	case update := <-p.Channel():
		assert.Equal(t, "scanning files", update.Message)
		assert.Equal(t, 100, update.Data["files_scanned"])
		assert.False(t, update.Timestamp.IsZero())
	case <-time.After(time.Second):
		t.Fatal("expected progress update on channel")
	}

	// LastUpdate should be set.
	last := p.LastUpdate()
	require.NotNil(t, last)
	assert.Equal(t, "scanning files", last.Message)
}

func TestProgressReporter_MultipleUpdates(t *testing.T) {
	p := NewProgressReporter()
	defer p.Close()

	for i := 0; i < 10; i++ {
		p.ReportProgress("update", map[string]any{
			"count": i,
		})
	}

	// Drain channel and count.
	count := 0
	for {
		select {
		case <-p.Channel():
			count++
		default:
			goto done
		}
	}
done:
	assert.Equal(t, 10, count)

	// LastUpdate should be the final one.
	last := p.LastUpdate()
	require.NotNil(t, last)
	assert.Equal(t, 9, last.Data["count"])
}

func TestProgressReporter_BufferFull_DropsUpdate(t *testing.T) {
	p := NewProgressReporter()
	defer p.Close()

	// Fill the buffer (capacity 64).
	for i := 0; i < 100; i++ {
		p.ReportProgress("fill", map[string]any{
			"i": i,
		})
	}

	// Should not block or panic. LastUpdate should be
	// the most recent regardless of buffer state.
	last := p.LastUpdate()
	require.NotNil(t, last)
	assert.Equal(t, 99, last.Data["i"])
}

func TestProgressReporter_Close_Idempotent(t *testing.T) {
	p := NewProgressReporter()

	// Close multiple times should not panic.
	assert.NotPanics(t, func() {
		p.Close()
		p.Close()
		p.Close()
	})
}

func TestProgressReporter_ReportAfterClose(t *testing.T) {
	p := NewProgressReporter()
	p.Close()

	// Should not panic when reporting after close.
	assert.NotPanics(t, func() {
		p.ReportProgress("after close", nil)
	})

	// LastUpdate should still be nil (no successful report).
	// Actually the update is recorded in last even after close.
	last := p.LastUpdate()
	require.NotNil(t, last)
	assert.Equal(t, "after close", last.Message)
}

func TestProgressReporter_ConcurrentAccess(t *testing.T) {
	p := NewProgressReporter()

	var wg sync.WaitGroup
	// Concurrent writers.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				p.ReportProgress("concurrent", map[string]any{
					"writer": n,
					"iter":   j,
				})
			}
		}(i)
	}

	// Concurrent reader.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = p.LastUpdate()
			select {
			case <-p.Channel():
			default:
			}
		}
	}()

	wg.Wait()
	p.Close()

	// Should not have panicked.
	last := p.LastUpdate()
	require.NotNil(t, last)
}

func TestProgressReporter_ChannelClosedOnClose(t *testing.T) {
	p := NewProgressReporter()
	p.ReportProgress("before close", nil)
	p.Close()

	// Channel should be closed — reading returns zero value.
	_, ok := <-p.Channel()
	if ok {
		// Might get the buffered "before close" update.
		_, ok = <-p.Channel()
	}
	assert.False(t, ok, "channel should be closed")
}

func TestBaseChallenge_ProgressReporter(t *testing.T) {
	b := NewBaseChallenge(
		"prog-001", "Progress Test", "desc", "test",
		nil,
	)

	// Initially nil.
	assert.Nil(t, b.Progress())

	// Set reporter.
	p := NewProgressReporter()
	b.SetProgressReporter(p)
	assert.Equal(t, p, b.Progress())

	// ReportProgress should work.
	b.ReportProgress("test progress", map[string]any{
		"step": 1,
	})

	select {
	case update := <-p.Channel():
		assert.Equal(t, "test progress", update.Message)
	case <-time.After(time.Second):
		t.Fatal("expected progress on channel")
	}

	p.Close()
}

func TestBaseChallenge_ReportProgress_NilReporter(t *testing.T) {
	b := NewBaseChallenge(
		"prog-002", "No Reporter", "desc", "test",
		nil,
	)

	// Should not panic when no reporter is set.
	assert.NotPanics(t, func() {
		b.ReportProgress("no-op", nil)
	})
}
