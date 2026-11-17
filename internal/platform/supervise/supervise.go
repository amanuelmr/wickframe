// SPDX-License-Identifier: AGPL-3.0-only

// Package supervise runs long-lived components and restarts them when they fail
// or panic (02-components §5). A Supervisor is itself a component, so
// supervisors nest into the process's supervision tree.
//
// Restart waits use real timers, not the injected clock, so tests run the
// Supervisor inside a testing/synctest bubble with clock.System.
package supervise

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/amanuelmr/wickframe/internal/platform/clock"
	"github.com/amanuelmr/wickframe/internal/platform/logx"
	"github.com/amanuelmr/wickframe/internal/platform/retry"
)

var restartBackoff = retry.Backoff{Base: time.Second, Max: time.Minute}

// A child that ran at least this long before failing starts its backoff over,
// so a component that fails once a day is not restarted with the maximum delay.
const stableAfter = time.Minute

// Child is a supervised component. Run blocks until ctx is done or it fails; a
// nil return before ctx is done means the work is finished, so no restart.
type Child struct {
	Name string
	Run  func(ctx context.Context) error
}

// Supervisor runs children and restarts the ones that fail.
type Supervisor struct {
	log      *slog.Logger
	clk      clock.Clock
	children []Child
}

// New returns a Supervisor for children, started in the given order.
func New(log *slog.Logger, clk clock.Clock, children ...Child) *Supervisor {
	return &Supervisor{log: log, clk: clk, children: children}
}

// Run starts every child and blocks until ctx is done, then stops the children
// in reverse order, each one fully before the next.
func (s *Supervisor) Run(ctx context.Context) error {
	type running struct {
		cancel context.CancelFunc
		done   chan struct{}
	}
	children := make([]running, len(s.children))
	for i, c := range s.children {
		// Detached from ctx, so the children can be stopped one at a time.
		cctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		done := make(chan struct{})
		children[i] = running{cancel, done}
		go func() {
			defer close(done)
			s.keep(cctx, c)
		}()
	}
	<-ctx.Done()
	for i := len(children) - 1; i >= 0; i-- {
		children[i].cancel()
		<-children[i].done
	}
	return nil
}

func (s *Supervisor) keep(ctx context.Context, c Child) {
	log := s.log.With(logx.KeyComponent, c.Name)
	failures := 0
	for {
		start := s.clk.Now()
		stack, err := runRecovered(ctx, c)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			log.Info("component finished")
			return
		}
		if s.clk.Now().Sub(start) >= stableAfter {
			failures = 0
		}
		delay := restartBackoff.Delay(failures)
		failures++
		attrs := []any{logx.KeyErr, err, "restart_in", delay}
		if stack != nil {
			attrs = append(attrs, "stack", string(stack))
		}
		log.Error("component failed", attrs...)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func runRecovered(ctx context.Context, c Child) (stack []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack, err = debug.Stack(), fmt.Errorf("panic: %v", r)
		}
	}()
	return nil, c.Run(ctx)
}
