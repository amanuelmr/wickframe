// SPDX-License-Identifier: AGPL-3.0-only

package fake_test

import (
	"context"
	"testing"
	"time"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/marketdata/connector/fake"
	"github.com/amanuelmr/wickframe/internal/platform/clock"
	"github.com/amanuelmr/wickframe/pkg/connectortest"
)

var (
	start = time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC)
	t0    = start.Truncate(time.Minute).UnixMilli()
	demo  = marketdata.StreamSub{Symbol: "DEMO", TF: marketdata.TF1m}
)

// closeCounter cancels its stream after n closed updates.
type closeCounter struct {
	n      int
	cancel context.CancelFunc
}

func (c *closeCounter) Candle(u marketdata.CandleUpdate) {
	if u.Closed {
		if c.n--; c.n == 0 {
			c.cancel()
		}
	}
}

func TestConformance(t *testing.T) {
	connectortest.Run(t, connectortest.Fixture{
		New: func(t *testing.T) marketdata.Connector {
			c := fake.New(fake.Scenario{Seed: 3, Price: 100, Loop: true, Steps: []fake.Step{
				{Kind: fake.Tick, Wait: time.Millisecond},
				{Kind: fake.Tick, Wait: time.Millisecond},
				{Kind: fake.Close, Wait: time.Millisecond},
			}}, clock.NewFake(start))
			// Six bars of history: five to serve and one opening at the range's end.
			ctx, cancel := context.WithCancel(t.Context())
			_ = c.Stream(ctx, []marketdata.StreamSub{demo}, &closeCounter{n: 6, cancel: cancel})
			return c
		},
		History: marketdata.HistoryRequest{Symbol: "DEMO", TF: marketdata.TF1m, From: t0, To: t0 + 5*60_000},
		Stream:  demo,
	})
}
