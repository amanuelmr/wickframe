// SPDX-License-Identifier: AGPL-3.0-only

package retry_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/amanuelmr/wickframe/internal/platform/retry"
)

var errFlaky = errors.New("flaky")

func TestDelayStaysWithinTheExponentialCeiling(t *testing.T) {
	t.Parallel()
	b := retry.Backoff{Base: time.Second, Max: 5 * time.Minute}
	ceilings := []time.Duration{1, 2, 4, 8, 16, 32, 64, 128, 256, 300, 300}
	for attempt, c := range ceilings {
		ceiling := c * time.Second
		var maxSeen time.Duration
		for range 2000 {
			d := b.Delay(attempt)
			if d < 0 || d > ceiling {
				t.Fatalf("Delay(%d) = %v, want within [0, %v]", attempt, d, ceiling)
			}
			maxSeen = max(maxSeen, d)
		}
		if maxSeen < ceiling/2 {
			t.Errorf("Delay(%d) never exceeded %v in 2000 draws; jitter looks wrong", attempt, maxSeen)
		}
	}
	if d := b.Delay(10_000); d > 5*time.Minute {
		t.Fatalf("Delay(10000) = %v, want at most the 5m cap", d)
	}
}

func TestDoRetriesUntilSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := retry.Backoff{Base: time.Second, Max: time.Minute}
		calls := 0
		start := time.Now()
		err := retry.Do(t.Context(), b, 5, func(context.Context) error {
			if calls++; calls < 3 {
				return errFlaky
			}
			return nil
		})
		if err != nil || calls != 3 {
			t.Fatalf("Do() = %v after %d calls, want nil after 3", err, calls)
		}
		if waited := time.Since(start); waited > 3*time.Second {
			t.Fatalf("waited %v, want at most 1s + 2s of backoff", waited)
		}
	})
}

func TestDoReturnsTheLastErrorWhenAttemptsRunOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		err := retry.Do(t.Context(), retry.Backoff{Base: time.Second, Max: time.Second}, 4, func(context.Context) error {
			calls++
			return errFlaky
		})
		if !errors.Is(err, errFlaky) || calls != 4 {
			t.Fatalf("Do() = %v after %d calls, want errFlaky after 4", err, calls)
		}
	})
}

func TestDoStopsAtAPermanentError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		errConfig := errors.New("400 bad request")
		calls := 0
		err := retry.Do(t.Context(), retry.Backoff{Base: time.Second, Max: time.Second}, 9, func(context.Context) error {
			calls++
			return retry.Permanent(errConfig)
		})
		if !errors.Is(err, errConfig) || calls != 1 {
			t.Fatalf("Do() = %v after %d calls, want errConfig after 1", err, calls)
		}
	})
}

func TestDoWaitsExactlyTheRequestedDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		calls := 0
		err := retry.Do(t.Context(), retry.Backoff{Base: time.Millisecond, Max: time.Millisecond}, 2, func(context.Context) error {
			if calls++; calls == 1 {
				return retry.After(42*time.Second, errFlaky)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if waited := time.Since(start); waited != 42*time.Second {
			t.Fatalf("waited %v, want exactly the 42s Retry-After", waited)
		}
	})
}

func TestDoGivesUpWhenTheContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		start := time.Now()
		err := retry.Do(ctx, retry.Backoff{Base: time.Hour, Max: time.Hour}, 3, func(context.Context) error {
			return retry.After(time.Hour, errFlaky)
		})
		if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, errFlaky) {
			t.Fatalf("Do() = %v, want both the deadline and the last error", err)
		}
		if waited := time.Since(start); waited != 10*time.Second {
			t.Fatalf("returned after %v, want at the 10s deadline", waited)
		}
	})
}
