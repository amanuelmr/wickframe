// SPDX-License-Identifier: AGPL-3.0-only

// Package retry provides jittered exponential backoff and a retry loop that
// honours permanent errors and Retry-After delays (04-market-data §5).
package retry

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// Backoff is exponential backoff with full jitter: the delay before retry n is
// uniformly random in [0, min(Max, Base·2ⁿ)].
type Backoff struct {
	Base time.Duration
	Max  time.Duration
}

// Delay returns the delay before retry attempt (0 for the first retry).
func (b Backoff) Delay(attempt int) time.Duration {
	ceiling := b.Base
	for range attempt {
		if ceiling >= b.Max {
			break
		}
		ceiling *= 2
	}
	ceiling = min(ceiling, b.Max)
	return rand.N(ceiling + 1) //nolint:gosec // jitter, not security
}

type permanentError struct{ err error }

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

// Permanent marks err as not worth retrying; Do returns it at once.
func Permanent(err error) error { return &permanentError{err} }

type afterError struct {
	err   error
	delay time.Duration
}

func (e *afterError) Error() string { return e.err.Error() }
func (e *afterError) Unwrap() error { return e.err }

// After asks Do to wait exactly delay before the next attempt, instead of the
// backoff delay. Callers cap delay themselves.
func After(delay time.Duration, err error) error { return &afterError{err, delay} }

// Do calls fn until it succeeds, returns a Permanent error, has run attempts
// times, or ctx is done; then it returns fn's last error (joined with ctx.Err()).
func Do(ctx context.Context, b Backoff, attempts int, fn func(context.Context) error) error {
	for attempt := 0; ; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if _, ok := errors.AsType[*permanentError](err); ok {
			return err
		}
		if attempt+1 >= attempts {
			return err
		}
		delay := b.Delay(attempt)
		if a, ok := errors.AsType[*afterError](err); ok {
			delay = a.delay
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(ctx.Err(), err)
		case <-timer.C:
		}
	}
}
