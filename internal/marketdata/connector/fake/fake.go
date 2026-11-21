// SPDX-License-Identifier: AGPL-3.0-only

// Package fake is a deterministic market for tests and demos (14-quality §2.1).
// It plays a Scenario on a seeded random walk, so the same seed always gives
// the same updates, and it needs no network.
package fake

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/platform/clock"
)

// ErrDisconnected ends a Stream at a Disconnect step.
var ErrDisconnected = errors.New("fake connector disconnected")

// Kind is what a Step does to every subscribed series.
type Kind int

const (
	// Tick moves the price and sends a forming update.
	Tick Kind = iota + 1
	// Close sends the bar as closed and opens the next one at its close.
	Close
	// Disconnect ends Stream with ErrDisconnected. The next Stream continues
	// with the following step.
	Disconnect
	// Correct sends the last closed bar again with a new close and a higher Rev.
	Correct
)

// Step is one scenario event, Wait after the previous one.
type Step struct {
	Kind Kind
	Wait time.Duration
}

// Scenario is a script for the fake market. Bar times start at the first
// Stream's time, aligned to the timeframe, and advance one bar per Close.
type Scenario struct {
	Seed  uint64
	Price float64
	Steps []Step
	Loop  bool
}

// Demo runs forever, with a forming update every second and a close every
// minute.
func Demo() Scenario {
	steps := make([]Step, 60)
	for i := range steps {
		steps[i] = Step{Kind: Tick, Wait: time.Second}
	}
	steps[59].Kind = Close
	return Scenario{Seed: 1, Price: 100, Steps: steps, Loop: true}
}

type bar struct {
	t             int64
	o, h, l, c, v float64
	rev           int16
}

type series struct {
	length int64
	cur    bar
	closed []bar
}

// Connector plays a Scenario. Calls to Stream and History must not overlap.
type Connector struct {
	sc     Scenario
	clk    clock.Clock
	rng    *rand.Rand
	next   int
	series map[marketdata.StreamSub]*series
}

var _ marketdata.Connector = (*Connector)(nil)

// New returns a Connector that plays sc.
func New(sc Scenario, clk clock.Clock) *Connector {
	return &Connector{
		sc:     sc,
		clk:    clk,
		rng:    rand.New(rand.NewPCG(sc.Seed, sc.Seed)), //nolint:gosec // a reproducible walk, not a secret
		series: make(map[marketdata.StreamSub]*series),
	}
}

// Info implements marketdata.Connector.
func (f *Connector) Info() marketdata.ConnectorInfo {
	tfs := []marketdata.Timeframe{
		marketdata.TF1m, marketdata.TF3m, marketdata.TF5m, marketdata.TF15m, marketdata.TF30m,
		marketdata.TF1h, marketdata.TF2h, marketdata.TF4h, marketdata.TF6h, marketdata.TF12h,
	}
	return marketdata.ConnectorInfo{
		ID: "fake", Name: "Fake market", Version: "1.0.0",
		// The data is generated locally, so the project's own license covers it.
		TermsURL:         "https://github.com/amanuelmr/wickframe/blob/main/LICENSE",
		Auth:             marketdata.AuthNone,
		Freshness:        marketdata.FreshnessRealtime,
		NativeTimeframes: tfs,
		StreamTimeframes: tfs,
		SettingsSchema:   `{"type":"object","additionalProperties":false}`,
	}
}

// History implements marketdata.Connector. The fake market's history is the
// bars its streams have closed, with their corrections.
func (f *Connector) History(ctx context.Context, req marketdata.HistoryRequest) ([]marketdata.Candle, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("fake history: %w", err)
	}
	var out []marketdata.Candle
	if s := f.series[marketdata.StreamSub{Symbol: req.Symbol, TF: req.TF}]; s != nil {
		for _, b := range s.closed {
			if b.t >= req.From && b.t < req.To {
				out = append(out, marketdata.Candle{TF: req.TF, T: b.t, O: b.o, H: b.h, L: b.l, C: b.c, V: b.v, Rev: b.rev})
			}
		}
	}
	return out, nil
}

// Stream implements marketdata.Connector. It returns nil when a scenario that
// does not loop ends.
func (f *Connector) Stream(ctx context.Context, subs []marketdata.StreamSub, sink marketdata.StreamSink) error {
	for _, sub := range subs {
		if f.series[sub] != nil {
			continue
		}
		d, ok := marketdata.BarLength(sub.TF)
		if !ok {
			return fmt.Errorf("fake connector does not stream timeframe %d", sub.TF)
		}
		length := d.Milliseconds()
		now := f.clk.Now().UnixMilli()
		p := f.sc.Price
		f.series[sub] = &series{length: length, cur: bar{t: now - now%length, o: p, h: p, l: p, c: p}}
	}
	for {
		if f.next == len(f.sc.Steps) {
			if !f.sc.Loop || f.next == 0 {
				return nil
			}
			f.next = 0
		}
		step := f.sc.Steps[f.next]
		select {
		case <-ctx.Done():
			return fmt.Errorf("fake stream: %w", ctx.Err())
		case <-time.After(step.Wait):
		}
		f.next++
		if step.Kind == Disconnect {
			return ErrDisconnected
		}
		for _, sub := range subs {
			f.apply(step.Kind, sub, f.series[sub], sink)
		}
	}
}

func (f *Connector) apply(kind Kind, sub marketdata.StreamSub, s *series, sink marketdata.StreamSink) {
	switch kind {
	case Tick:
		f.walk(&s.cur)
		s.cur.v = round(s.cur.v+f.rng.Float64(), 1e4)
		sink.Candle(update(sub, s.cur, false))
	case Close:
		sink.Candle(update(sub, s.cur, true))
		s.closed = append(s.closed, s.cur)
		c := s.cur.c
		s.cur = bar{t: s.cur.t + s.length, o: c, h: c, l: c, c: c}
	case Correct:
		if len(s.closed) == 0 {
			return
		}
		last := &s.closed[len(s.closed)-1]
		f.walk(last)
		last.rev++
		sink.Candle(update(sub, *last, true))
	case Disconnect:
		// Stream returns before applying it.
	}
}

func (f *Connector) walk(b *bar) {
	b.c = round(b.c*(1+0.001*f.rng.NormFloat64()), 100)
	b.h = max(b.h, b.c)
	b.l = min(b.l, b.c)
}

func round(x, scale float64) float64 { return math.Round(x*scale) / scale }

func update(sub marketdata.StreamSub, b bar, closed bool) marketdata.CandleUpdate {
	return marketdata.CandleUpdate{
		Sub: sub, T: b.t, O: b.o, H: b.h, L: b.l, C: b.c, V: b.v, Closed: closed, Rev: b.rev,
	}
}
