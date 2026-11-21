// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/amanuelmr/wickframe/internal/bus"
	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/marketdata/connector/fake"
	"github.com/amanuelmr/wickframe/internal/platform/clock"
)

// recorder keeps the closed updates it passes on to the publisher.
type recorder struct {
	publisher
	closed []marketdata.CandleUpdate
}

func (r *recorder) Candle(u marketdata.CandleUpdate) {
	if u.Closed {
		r.closed = append(r.closed, u)
	}
	r.publisher.Candle(u)
}

func TestClosedBarsFromTheFakeConnectorAreStoredAcrossARestart(t *testing.T) {
	dataDir := t.TempDir()
	demo := marketdata.StreamSub{Symbol: "DEMO", TF: marketdata.TF1m}
	conn := fake.New(fake.Scenario{Seed: 7, Price: 100, Steps: []fake.Step{
		{Kind: fake.Tick},
		{Kind: fake.Close},
		{Kind: fake.Tick},
		{Kind: fake.Close},
		{Kind: fake.Correct},
		{Kind: fake.Disconnect},
		{Kind: fake.Tick},
		{Kind: fake.Close},
		{Kind: fake.Tick},
		{Kind: fake.Correct},
	}}, clock.NewFake(time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC)))

	// serve is one run of the server: stream until the scenario stops at a
	// disconnect or its end, then shut down and read what was stored.
	var closed []marketdata.CandleUpdate
	serve := func() []marketdata.Candle {
		store := openStore(t, dataDir)
		defer func() { _ = store.Close() }()
		hub := bus.New[marketdata.CandleUpdate]()
		persister := marketdata.NewPersister(store, hub, marketdata.Series{InstrumentID: demoID, TF: marketdata.TF1m})
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- persister.Run(ctx) }()

		r := &recorder{publisher: publisher{hub, demoID, slog.New(slog.DiscardHandler)}}
		_ = conn.Stream(t.Context(), []marketdata.StreamSub{demo}, r)
		cancel()
		if err := <-done; err != nil {
			t.Fatalf("persister Run = %v, want nil", err)
		}
		closed = append(closed, r.closed...)

		got, err := store.Candles(t.Context(), marketdata.CandleRange{InstrumentID: demoID, TF: marketdata.TF1m, To: 1 << 62, Limit: 100})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	// want is the newest closed update of each bar, which corrections replace.
	want := func() []marketdata.Candle {
		var out []marketdata.Candle
		for _, u := range closed {
			c := marketdata.Candle{InstrumentID: demoID, TF: u.Sub.TF, T: u.T, O: u.O, H: u.H, L: u.L, C: u.C, V: u.V, Rev: u.Rev}
			if len(out) > 0 && out[len(out)-1].T == u.T {
				out[len(out)-1] = c
			} else {
				out = append(out, c)
			}
		}
		return out
	}

	got := serve()
	if diff := cmp.Diff(want(), got); diff != "" {
		t.Errorf("after the first run (-want +got):\n%s", diff)
	}
	got = serve()
	if diff := cmp.Diff(want(), got); diff != "" {
		t.Errorf("after the restart (-want +got):\n%s", diff)
	}
	revs := make([]int16, 0, len(got))
	for _, c := range got {
		revs = append(revs, c.Rev)
	}
	if !slices.Equal(revs, []int16{0, 1, 1}) {
		t.Errorf("stored bars with revs %v, want 3 bars, the last two corrected", revs)
	}
}
