package gocrc

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Result represents the outcome of a worker's execution.
type Result[T any] struct {
	Value T
	Err   error
	Index int
}

// Worker is a function that performs a task and returns a value of type T.
type Worker[T any] func(ctx context.Context) (T, error)

// MultiError is a collection of errors with their corresponding worker indices.
type MultiError[T any] struct {
	Results []Result[T]
}

func (m *MultiError[T]) Error() string {
	var sb strings.Builder
	sb.WriteString("multiple errors occurred:")
	for _, res := range m.Results {
		if res.Err != nil {
			sb.WriteString(fmt.Sprintf("\n - Worker [%d]: %v", res.Index, res.Err))
		}
	}
	return sb.String()
}

// Race runs multiple workers concurrently. The first worker to complete (successfully or with error)
// will cause all other workers to be cancelled immediately.
// Returns the result of the first worker to complete.
func Race[T any](ctx context.Context, workers ...Worker[T]) (Result[T], error) {
	if len(workers) == 0 {
		return Result[T]{}, nil
	}

	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan Result[T], 1)

	for i := range workers {
		index := i
		worker := workers[i]
		go func() {
			val, err := worker(raceCtx)
			res := Result[T]{Value: val, Err: err, Index: index}
			select {
			case resultCh <- res:
				cancel() // Signal others to stop
			case <-raceCtx.Done():
				// Another worker already won
			}
		}()
	}

	select {
	case res := <-resultCh:
		return res, res.Err
	case <-ctx.Done():
		return Result[T]{Index: -1, Err: ctx.Err()}, ctx.Err()
	}
}

// NoRace runs multiple workers concurrently and waits for all of them to complete.
// Returns a slice of all results (in order) and a MultiError if any workers failed.
func NoRace[T any](ctx context.Context, workers ...Worker[T]) ([]Result[T], error) {
	if len(workers) == 0 {
		return nil, nil
	}

	results := make([]Result[T], len(workers))
	var wg sync.WaitGroup
	var hasError bool
	var mu sync.Mutex

	wg.Add(len(workers))
	for i := range workers {
		index := i
		worker := workers[i]
		go func() {
			defer wg.Done()
			val, err := worker(ctx)

			mu.Lock()
			results[index] = Result[T]{
				Value: val,
				Err:   err,
				Index: index,
			}
			if err != nil {
				hasError = true
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	if hasError {
		var errResults []Result[T]
		for _, r := range results {
			if r.Err != nil {
				errResults = append(errResults, r)
			}
		}
		return results, &MultiError[T]{Results: errResults}
	}
	return results, nil
}
