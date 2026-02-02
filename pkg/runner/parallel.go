package runner

import (
	"context"
	"fmt"
	"sync"

	"digital.vasic.challenges/pkg/challenge"
)

// parallelResult pairs a result with its original index so
// results can be returned in submission order.
type parallelResult struct {
	index  int
	result *challenge.Result
	err    error
}

// runParallel executes challenges concurrently with a semaphore
// limiting maxConcurrency goroutines. Results are returned in
// the same order as the input IDs.
func runParallel(
	ctx context.Context,
	r *DefaultRunner,
	ids []challenge.ID,
	config *challenge.Config,
	maxConcurrency int,
) ([]*challenge.Result, error) {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}

	sem := make(chan struct{}, maxConcurrency)
	resultsCh := make(chan parallelResult, len(ids))

	var wg sync.WaitGroup

	for i, id := range ids {
		wg.Add(1)
		go func(idx int, cID challenge.ID) {
			defer wg.Done()

			// Acquire semaphore slot.
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				resultsCh <- parallelResult{
					index: idx,
					err:   ctx.Err(),
				}
				return
			}

			c, err := r.registry.Get(cID)
			if err != nil {
				resultsCh <- parallelResult{
					index: idx,
					err: fmt.Errorf(
						"challenge %s: %w", cID, err,
					),
				}
				return
			}

			cfg := *config
			cfg.ChallengeID = cID

			result, execErr := r.executeChallenge(
				ctx, c, &cfg,
			)
			resultsCh <- parallelResult{
				index:  idx,
				result: result,
				err:    execErr,
			}
		}(i, id)
	}

	// Close channel after all goroutines complete.
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect results in submission order.
	ordered := make([]*challenge.Result, len(ids))
	var firstErr error

	for pr := range resultsCh {
		if pr.err != nil && firstErr == nil {
			firstErr = pr.err
		}
		ordered[pr.index] = pr.result
	}

	// Filter out nil entries if context was cancelled.
	results := make([]*challenge.Result, 0, len(ids))
	for _, r := range ordered {
		if r != nil {
			results = append(results, r)
		}
	}

	return results, firstErr
}
