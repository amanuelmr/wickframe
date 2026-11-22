// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/amanuelmr/wickframe/internal/platform/clock"
)

// ErrBanned is returned while a source refuses requests from this IP.
var ErrBanned = errors.New("source has banned this IP")

// Limiter paces the requests to one host by weight per fixed window, and
// holds them back after the host reports a limit.
type Limiter struct {
	clk    clock.Clock
	limit  int
	window time.Duration

	mu          sync.Mutex
	start       time.Time // of the current window
	used        int
	pausedUntil time.Time
	bannedUntil time.Time
}

// NewLimiter returns a Limiter allowing limit weight per window, with windows
// aligned to multiples of window since the Unix epoch.
func NewLimiter(clk clock.Clock, limit int, window time.Duration) *Limiter {
	return &Limiter{clk: clk, limit: limit, window: window}
}

// Reserve waits until weight fits in the current window and no pause is in
// force, then takes it. It fails at once during a ban.
func (l *Limiter) Reserve(ctx context.Context, weight int) error {
	if weight > l.limit {
		return fmt.Errorf("reserve %d weight: more than the limit of %d", weight, l.limit)
	}
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("reserve %d weight: %w", weight, err)
		}
		l.mu.Lock()
		now := l.roll()
		if now.Before(l.bannedUntil) {
			until := l.bannedUntil
			l.mu.Unlock()
			return fmt.Errorf("%w until %s", ErrBanned, until.Format(time.RFC3339))
		}
		var wait time.Duration
		switch {
		case now.Before(l.pausedUntil):
			wait = l.pausedUntil.Sub(now)
		case l.used+weight > l.limit:
			wait = l.start.Add(l.window).Sub(now)
		default:
			l.used += weight
			l.mu.Unlock()
			return nil
		}
		l.mu.Unlock()
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

// Used corrects the current window's weight from the host's own count, which
// includes other clients on the same IP.
func (l *Limiter) Used(weight int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.roll()
	l.used = max(l.used, weight)
}

// Pause holds every request back for d.
func (l *Limiter) Pause(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if until := l.clk.Now().Add(d); until.After(l.pausedUntil) {
		l.pausedUntil = until
	}
}

// Ban refuses every request for d.
func (l *Limiter) Ban(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if until := l.clk.Now().Add(d); until.After(l.bannedUntil) {
		l.bannedUntil = until
	}
}

// roll starts a new window if the current one has ended, and returns the time.
// It is called with l.mu held.
func (l *Limiter) roll() time.Time {
	now := l.clk.Now()
	if start := now.Truncate(l.window); start.After(l.start) {
		l.start, l.used = start, 0
	}
	return now
}

// HostLimiters hands out one Limiter per host and limit, so every connector
// that reaches a host shares its budget.
type HostLimiters struct {
	clk clock.Clock
	mu  sync.Mutex
	m   map[string]*Limiter
}

// NewHostLimiters returns an empty HostLimiters.
func NewHostLimiters(clk clock.Clock) *HostLimiters {
	return &HostLimiters{clk: clk, m: make(map[string]*Limiter)}
}

// Get returns the Limiter for a host and limit name, such as
// "data-api.binance.vision weight", creating it with limit per window on first
// use.
func (h *HostLimiters) Get(key string, limit int, window time.Duration) *Limiter {
	h.mu.Lock()
	defer h.mu.Unlock()
	l := h.m[key]
	if l == nil {
		l = NewLimiter(h.clk, limit, window)
		h.m[key] = l
	}
	return l
}
