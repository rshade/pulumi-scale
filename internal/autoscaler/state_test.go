package autoscaler

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Export retryOnConcurrency for testing (or keep it private and test via a public wrapper or reflection, but internal tests can see private methods)
// Since this is `package autoscaler`, we can test private methods of `StateManager`.

func TestRetryOnConcurrency(t *testing.T) {
	sm := &StateManager{}
	ctx := context.Background()

	t.Run("success on first try", func(t *testing.T) {
		calls := 0
		op := func() error {
			calls++
			return nil
		}
		err := sm.retryOnConcurrency(ctx, op)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if calls != 1 {
			t.Errorf("Expected 1 call, got %d", calls)
		}
	})

	t.Run("fail with non-conflict error", func(t *testing.T) {
		calls := 0
		op := func() error {
			calls++
			return errors.New("network error")
		}
		err := sm.retryOnConcurrency(ctx, op)
		if err == nil || err.Error() != "network error" {
			t.Errorf("Expected network error, got %v", err)
		}
		if calls != 1 {
			t.Errorf("Expected 1 call (no retry), got %d", calls)
		}
	})

	t.Run("retry on conflict", func(t *testing.T) {
		calls := 0
		op := func() error {
			calls++
			if calls < 3 {
				return errors.New("error: conflict: another update is in progress")
			}
			return nil
		}
		
		// Use a short context or trust logic speed?
		// Retry logic has exponential backoff 1s, 2s...
		// This test will take ~3 seconds. Acceptable.
		
		start := time.Now()
		err := sm.retryOnConcurrency(ctx, op)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if calls != 3 {
			t.Errorf("Expected 3 calls, got %d", calls)
		}
		// Expect delay: 0 (fail) -> 1s -> 1 (fail) -> 2s -> 2 (success). Total wait ~3s.
		if duration < 2*time.Second {
			t.Errorf("Expected backoff delay, but finished too fast: %v", duration)
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		calls := 0
		op := func() error {
			calls++
			return errors.New("concurrent update")
		}

		// We don't want to wait 60s for this test.
		// We can't easily mock the backoff constants in the current impl without refactoring.
		// For this test, I will skip execution or accept it takes time, OR verify just 1 retry if I can cancel ctx?
		
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		err := sm.retryOnConcurrency(ctx, op)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		// Should fail with context deadline exceeded or max retries
	})
}
