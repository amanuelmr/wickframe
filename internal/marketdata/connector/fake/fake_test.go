// SPDX-License-Identifier: AGPL-3.0-only

package fake

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/platform/clock"
)

type event struct {
	At time.Duration
	U  marketdata.CandleUpdate
}

type recorder struct {
	start  time.Time
	events []event
}

func (r *recorder) Candle(u marketdata.CandleUpdate) {
	r.events = append(r.events, event{time.Since(r.start), u})
}

var btc = marketdata.StreamSub{Symbol: "BTCUSD", TF: marketdata.TF1m}

func TestScenarioPlaysExactlyForASeed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sec := time.Second
		f := New(Scenario{Seed: 7, Price: 100, Steps: []Step{
			{Tick, sec},
			{Tick, sec},
			{Close, sec},
			{Tick, sec},
			{Disconnect, 0},
			{Tick, 5 * sec},
			{Close, sec},
			{Correct, sec},
		}}, clock.NewFake(time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC)))
		r := &recorder{start: time.Now()}
		subs := []marketdata.StreamSub{btc}

		if err := f.Stream(t.Context(), subs, r); !errors.Is(err, ErrDisconnected) {
			t.Fatalf("first Stream = %v, want ErrDisconnected", err)
		}
		if err := f.Stream(t.Context(), subs, r); err != nil {
			t.Fatalf("second Stream = %v, want nil at the end of the scenario", err)
		}

		t0 := int64(1_767_225_600_000) // 2026-01-01T00:00:00Z, the bar holding the clock's time
		t1 := t0 + 60_000
		u := func(t int64, o, h, l, c, v float64, closed bool, rev int16) marketdata.CandleUpdate {
			return marketdata.CandleUpdate{Sub: btc, T: t, O: o, H: h, L: l, C: c, V: v, Closed: closed, Rev: rev}
		}
		want := []event{
			{1 * sec, u(t0, 100, 100, 99.99, 99.99, 0.861, false, 0)},
			{2 * sec, u(t0, 100, 100, 99.89, 99.89, 1.8082, false, 0)},
			{3 * sec, u(t0, 100, 100, 99.89, 99.89, 1.8082, true, 0)},
			{4 * sec, u(t1, 99.89, 100, 99.89, 100, 0.134, false, 0)},
			{9 * sec, u(t1, 99.89, 100.15, 99.89, 100.15, 0.3202, false, 0)},
			{10 * sec, u(t1, 99.89, 100.15, 99.89, 100.15, 0.3202, true, 0)},
			{11 * sec, u(t1, 99.89, 100.21, 99.89, 100.21, 0.3202, true, 1)},
		}
		if diff := cmp.Diff(want, r.events); diff != "" {
			t.Errorf("updates (-want +got):\n%s", diff)
		}
	})
}

func TestDemoRunsUntilCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute+time.Second/2)
		defer cancel()
		r := &recorder{start: time.Now()}

		err := New(Demo(), clock.System{}).Stream(ctx, []marketdata.StreamSub{btc}, r)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Stream = %v, want context.DeadlineExceeded", err)
		}

		if len(r.events) != 180 {
			t.Fatalf("got %d updates, want 180 in 3 minutes", len(r.events))
		}
		var closes []int64
		for _, e := range r.events {
			if u := e.U; u.L > min(u.O, u.C) || u.H < max(u.O, u.C) || u.V < 0 {
				t.Fatalf("update at %v breaks the candle invariants: %+v", e.At, u)
			}
			if e.U.Closed {
				closes = append(closes, e.U.T)
			}
		}
		want := []int64{946_684_800_000, 946_684_860_000, 946_684_920_000}
		if diff := cmp.Diff(want, closes); diff != "" {
			t.Errorf("closed bar times (-want +got):\n%s", diff)
		}
	})
}
