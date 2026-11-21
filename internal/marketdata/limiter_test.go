// SPDX-License-Identifier: AGPL-3.0-only

package marketdata_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/platform/clock"
)

// reserve reserves weight and returns how long it waited.
func reserve(t *testing.T, l *marketdata.Limiter, weight int) time.Duration {
	t.Helper()
	start := time.Now()
	if err := l.Reserve(t.Context(), weight); err != nil {
		t.Fatalf("Reserve(%d): %v", weight, err)
	}
	return time.Since(start)
}

// newLimiter returns a limiter of 10 weight per minute, at the start of a minute.
func newLimiter(t *testing.T) *marketdata.Limiter {
	t.Helper()
	time.Sleep(time.Until(time.Now().Truncate(time.Minute).Add(time.Minute)))
	return marketdata.NewLimiter(clock.System{}, 10, time.Minute)
}

func TestLimiterWaitsForTheNextWindowWhenTheBudgetIsSpent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimiter(t)
		time.Sleep(15 * time.Second)
		for range 5 {
			if waited := reserve(t, l, 2); waited != 0 {
				t.Fatalf("waited %v within the budget", waited)
			}
		}
		if waited := reserve(t, l, 2); waited != 45*time.Second {
			t.Errorf("waited %v, want the 45s left in the window", waited)
		}
	})
}

func TestLimiterTakesTheHostsHigherCount(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimiter(t)
		reserve(t, l, 2)
		l.Used(9)
		if waited := reserve(t, l, 2); waited != time.Minute {
			t.Errorf("waited %v after the host reported 9 of 10 used, want 1m", waited)
		}
		l.Used(1) // lower than what was reserved, so ignored
		reserve(t, l, 8)
		if waited := reserve(t, l, 1); waited != time.Minute {
			t.Errorf("waited %v, want 1m", waited)
		}
	})
}

func TestLimiterPauseHoldsEveryRequestBack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimiter(t)
		l.Pause(30 * time.Second)
		l.Pause(10 * time.Second) // a shorter pause does not cut the longer one
		waits := make(chan time.Duration, 2)
		for range 2 {
			go func() { waits <- reserve(t, l, 1) }()
		}
		for range 2 {
			if waited := <-waits; waited != 30*time.Second {
				t.Errorf("waited %v, want the 30s pause", waited)
			}
		}
	})
}

func TestLimiterRefusesRequestsDuringABan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimiter(t)
		l.Ban(2 * time.Minute)
		start := time.Now()
		if err := l.Reserve(t.Context(), 1); !errors.Is(err, marketdata.ErrBanned) || time.Since(start) != 0 {
			t.Fatalf("Reserve during a ban = %v after %v, want ErrBanned at once", err, time.Since(start))
		}
		time.Sleep(2 * time.Minute)
		reserve(t, l, 1)
	})
}

func TestLimiterReserveStopsWithTheContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLimiter(t)
		l.Pause(time.Hour)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		if err := l.Reserve(ctx, 1); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Reserve = %v, want the context's error", err)
		}
		free := marketdata.NewLimiter(clock.System{}, 1, time.Minute)
		if err := free.Reserve(ctx, 1); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Reserve with a done context = %v, want its error", err)
		}
		if waited := reserve(t, free, 1); waited != 0 {
			t.Errorf("waited %v: the cancelled Reserve took weight", waited)
		}
		if err := l.Reserve(t.Context(), 11); err == nil {
			t.Error("Reserve of more than the limit succeeded")
		}
	})
}
