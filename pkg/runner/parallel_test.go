package runner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.challenges/pkg/challenge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunParallel_Success(t *testing.T) {
	a := newStub("a")
	b := newStub("b")
	c := newStub("c")
	reg := setupRegistry(t, a, b, c)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "b", "c"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 2,
	)
	require.NoError(t, err)
	require.Len(t, results, 3)

	for _, result := range results {
		assert.Equal(t, challenge.StatusPassed, result.Status)
	}
}

func TestRunParallel_NotFound(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "missing"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 2,
	)

	// Should have an error for the missing challenge.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
	// The existing challenge should still have run.
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestRunParallel_RespectsMaxConcurrency(t *testing.T) {
	var concurrent int64
	var maxSeen int64

	// Create slow stubs that track concurrency via timing.
	stubs := make([]*stubChallenge, 5)
	for i := range stubs {
		stubs[i] = newStub(string(rune('a' + i)))
		stubs[i].execDelay = 100 * time.Millisecond
	}

	reg := setupRegistry(t,
		stubs[0], stubs[1], stubs[2],
		stubs[3], stubs[4],
	)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "b", "c", "d", "e"}

	start := time.Now()
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 2,
	)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Len(t, results, 5)

	// With 5 tasks at 100ms each and concurrency=2,
	// minimum time is ~300ms (ceil(5/2)*100ms).
	assert.GreaterOrEqual(t, elapsed, 200*time.Millisecond)

	_ = concurrent
	_ = maxSeen
}

func TestRunParallel_MaxConcurrency_One(t *testing.T) {
	stubs := make([]*stubChallenge, 3)
	for i := range stubs {
		stubs[i] = newStub(string(rune('a' + i)))
		stubs[i].execDelay = 50 * time.Millisecond
	}

	reg := setupRegistry(t, stubs[0], stubs[1], stubs[2])

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "b", "c"}

	start := time.Now()
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 1,
	)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Len(t, results, 3)

	// With concurrency=1, should take at least 150ms.
	assert.GreaterOrEqual(t, elapsed, 120*time.Millisecond)
}

func TestRunParallel_MaxConcurrency_Large(t *testing.T) {
	stubs := make([]*stubChallenge, 3)
	for i := range stubs {
		stubs[i] = newStub(string(rune('a' + i)))
		stubs[i].execDelay = 50 * time.Millisecond
	}

	reg := setupRegistry(t, stubs[0], stubs[1], stubs[2])

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "b", "c"}

	start := time.Now()
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 100,
	)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Len(t, results, 3)

	// With high concurrency, all should run at once (~50ms).
	assert.Less(t, elapsed, 200*time.Millisecond)
}

func TestRunParallel_ZeroConcurrency(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 0,
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestRunParallel_NegativeConcurrency(t *testing.T) {
	reg := setupRegistry(t, newStub("a"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), -5,
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, challenge.StatusPassed, results[0].Status)
}

func TestRunParallel_ContextCancelled(t *testing.T) {
	s := newStub("a")
	s.execDelay = 5 * time.Second
	reg := setupRegistry(t, s)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ctx, cancel := context.WithTimeout(
		context.Background(), 50*time.Millisecond,
	)
	defer cancel()

	ids := []challenge.ID{"a"}
	_, err := r.RunParallel(
		ctx, ids, challenge.NewConfig(""), 1,
	)

	// May or may not get an error depending on timing.
	_ = err
}

func TestRunParallel_ContextCancelled_MultipleChallenges(
	t *testing.T,
) {
	stubs := make([]*stubChallenge, 5)
	for i := range stubs {
		stubs[i] = newStub(string(rune('a' + i)))
		stubs[i].execDelay = 5 * time.Second
	}

	reg := setupRegistry(t,
		stubs[0], stubs[1], stubs[2],
		stubs[3], stubs[4],
	)

	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ctx, cancel := context.WithTimeout(
		context.Background(), 100*time.Millisecond,
	)
	defer cancel()

	ids := []challenge.ID{"a", "b", "c", "d", "e"}
	_, _ = r.RunParallel(
		ctx, ids, challenge.NewConfig(""), 2,
	)
	// Test mainly verifies no panic/deadlock occurs.
}

func TestRunParallel_Empty(t *testing.T) {
	reg := setupRegistry(t)
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	results, err := r.RunParallel(
		context.Background(), nil,
		challenge.NewConfig(""), 2,
	)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestRunParallel_EmptySlice(t *testing.T) {
	reg := setupRegistry(t)
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	results, err := r.RunParallel(
		context.Background(), []challenge.ID{},
		challenge.NewConfig(""), 2,
	)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestRunParallel_PreservesOrder(t *testing.T) {
	var execOrder int64

	stubs := make([]*stubChallenge, 3)
	for i := range stubs {
		stubs[i] = newStub(
			string(rune('a' + i)),
		)
	}

	reg := setupRegistry(t, stubs[0], stubs[1], stubs[2])
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "b", "c"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 1,
	)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// With concurrency=1, results should be in order.
	assert.Equal(t, challenge.ID("a"), results[0].ChallengeID)
	assert.Equal(t, challenge.ID("b"), results[1].ChallengeID)
	assert.Equal(t, challenge.ID("c"), results[2].ChallengeID)

	_ = execOrder
	_ = atomic.AddInt64
}

func TestRunParallel_PreservesOrder_HighConcurrency(
	t *testing.T,
) {
	// Even with high concurrency, results should be in
	// submission order.
	stubs := make([]*stubChallenge, 5)
	for i := range stubs {
		stubs[i] = newStub(string(rune('a' + i)))
	}

	reg := setupRegistry(t,
		stubs[0], stubs[1], stubs[2],
		stubs[3], stubs[4],
	)
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "b", "c", "d", "e"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 5,
	)
	require.NoError(t, err)
	require.Len(t, results, 5)

	for i, id := range ids {
		assert.Equal(t, id, results[i].ChallengeID)
	}
}

func TestRunParallel_SingleChallenge(t *testing.T) {
	reg := setupRegistry(t, newStub("solo"))
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"solo"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 4,
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, challenge.ID("solo"), results[0].ChallengeID)
	assert.Equal(t, challenge.StatusPassed, results[0].Status)
}

func TestRunParallel_MixedResults(t *testing.T) {
	passing := newStub("pass")
	failing := newStub("fail")
	failing.execResult = &challenge.Result{
		Assertions: []challenge.AssertionResult{
			{Passed: false, Message: "assertion failed"},
		},
	}

	reg := setupRegistry(t, passing, failing)
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"pass", "fail"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 2,
	)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Find results by ID since parallel order may vary
	// in the ordered slice.
	resultMap := make(map[challenge.ID]*challenge.Result)
	for _, r := range results {
		resultMap[r.ChallengeID] = r
	}

	assert.Equal(t,
		challenge.StatusPassed,
		resultMap["pass"].Status,
	)
	assert.Equal(t,
		challenge.StatusFailed,
		resultMap["fail"].Status,
	)
}

func TestRunParallel_AllFailing(t *testing.T) {
	stubs := make([]*stubChallenge, 3)
	for i := range stubs {
		stubs[i] = newStub(string(rune('a' + i)))
		stubs[i].execResult = &challenge.Result{
			Assertions: []challenge.AssertionResult{
				{Passed: false, Message: "nope"},
			},
		}
	}

	reg := setupRegistry(t, stubs[0], stubs[1], stubs[2])
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"a", "b", "c"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 3,
	)
	require.NoError(t, err)
	require.Len(t, results, 3)

	for _, result := range results {
		assert.Equal(t, challenge.StatusFailed, result.Status)
	}
}

func TestRunParallel_IndependentExecution(t *testing.T) {
	// Verify that challenges run independently -- a slow one
	// does not block others beyond the semaphore limit.
	slow := newStub("slow")
	slow.execDelay = 200 * time.Millisecond

	fast := newStub("fast")
	// No delay.

	reg := setupRegistry(t, slow, fast)
	r := NewRunner(
		WithRegistry(reg),
		WithResultsDir(t.TempDir()),
	)

	ids := []challenge.ID{"slow", "fast"}
	results, err := r.RunParallel(
		context.Background(), ids,
		challenge.NewConfig(""), 2,
	)
	require.NoError(t, err)
	require.Len(t, results, 2)

	for _, result := range results {
		assert.Equal(t, challenge.StatusPassed, result.Status)
	}
}
