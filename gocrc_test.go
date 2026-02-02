package gocrc

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRace(t *testing.T) {
	t.Run("first_worker_wins", func(t *testing.T) {
		ctx := context.Background()
		var cancelled int32

		w1 := func(ctx context.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "win", nil
		}
		w2 := func(ctx context.Context) (string, error) {
			select {
			case <-time.After(500 * time.Millisecond):
				return "slow", nil
			case <-ctx.Done():
				atomic.AddInt32(&cancelled, 1)
				return "", ctx.Err()
			}
		}

		res, err := Race(ctx, w1, w2)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if res.Value != "win" {
			t.Errorf("expected 'win', got %v", res.Value)
		}

		// Wait a bit to ensure cancellation propagates
		time.Sleep(100 * time.Millisecond)
		if atomic.LoadInt32(&cancelled) != 1 {
			t.Errorf("expected w2 to be cancelled, but it wasn't")
		}
	})

	t.Run("error_from_first_worker", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("boom")

		w1 := func(ctx context.Context) (int, error) {
			time.Sleep(50 * time.Millisecond)
			return 0, expectedErr
		}
		w2 := func(ctx context.Context) (int, error) {
			time.Sleep(200 * time.Millisecond)
			return 1, nil
		}

		res, err := Race(ctx, w1, w2)
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
		if res.Index != 0 {
			t.Errorf("expected index 0, got %d", res.Index)
		}
	})
}

func TestNoRace(t *testing.T) {
	t.Run("all_succeed_with_values", func(t *testing.T) {
		ctx := context.Background()
		w1 := func(ctx context.Context) (int, error) { return 10, nil }
		w2 := func(ctx context.Context) (int, error) { return 20, nil }

		results, err := NoRace(ctx, w1, w2)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
		if results[0].Value != 10 || results[1].Value != 20 {
			t.Errorf("values mismatch: %v", results)
		}
		if results[0].Index != 0 || results[1].Index != 1 {
			t.Errorf("index mismatch: %v", results)
		}
	})

	t.Run("some_fail_detailed", func(t *testing.T) {
		ctx := context.Background()
		err1 := errors.New("err1")

		w1 := func(ctx context.Context) (string, error) { return "ok", nil }
		w2 := func(ctx context.Context) (string, error) { return "", err1 }

		results, err := NoRace(ctx, w1, w2)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		merr, ok := err.(*MultiError[string])
		if !ok {
			t.Fatalf("expected *MultiError[string], got %T", err)
		}

		if len(merr.Results) != 1 {
			t.Errorf("expected 1 error result, got %d", len(merr.Results))
		}
		if merr.Results[0].Index != 1 {
			t.Errorf("expected index 1 to fail, got %d", merr.Results[0].Index)
		}

		// Check that we still got the successful result
		if results[0].Value != "ok" {
			t.Errorf("expected index 0 to have value 'ok', got %v", results[0].Value)
		}
	})

	t.Run("grouped_tasks_abc", func(t *testing.T) {
		ctx := context.Background()

		// Task A: 2 workers
		workersA := []Worker[string]{
			func(ctx context.Context) (string, error) { return "A1", nil },
			func(ctx context.Context) (string, error) { return "A2", nil },
		}
		// Task B: 1 worker
		workersB := []Worker[string]{
			func(ctx context.Context) (string, error) { return "B1", nil },
		}

		// Higher level orchestration
		results, err := NoRace(ctx,
			func(ctx context.Context) ([]Result[string], error) {
				return NoRace(ctx, workersA...)
			},
			func(ctx context.Context) ([]Result[string], error) {
				return NoRace(ctx, workersB...)
			},
		)

		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 group results, got %d", len(results))
		}

		// results[0].Value should contain the results of Group A
		groupA := results[0].Value
		if len(groupA) != 2 {
			t.Errorf("expected Group A to have 2 results, got %d", len(groupA))
		}
		if groupA[0].Value != "A1" || groupA[1].Value != "A2" {
			t.Errorf("Group A values mismatch")
		}
	})
}
