// SPDX-License-Identifier: AGPL-3.0-only

package marketdata_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/amanuelmr/wickframe/internal/bus"
	"github.com/amanuelmr/wickframe/internal/marketdata"
)

var (
	demo       = marketdata.StreamSub{Symbol: "DEMO", TF: marketdata.TF1m}
	demoSeries = marketdata.Series{InstrumentID: 1, TF: marketdata.TF1m}
)

// recordingRepo records each batch with the time it was written. While block
// is open, writes wait for it to close.
type recordingRepo struct {
	marketdata.CandleRepository
	block   chan struct{}
	mu      sync.Mutex
	batches [][]marketdata.Candle
	at      []time.Time
}

func (r *recordingRepo) UpsertCandles(_ context.Context, cs []marketdata.Candle) error {
	if r.block != nil {
		<-r.block
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.batches = append(r.batches, slices.Clone(cs))
	r.at = append(r.at, time.Now())
	return nil
}

func closedBar(t int64) marketdata.CandleUpdate {
	return marketdata.CandleUpdate{Sub: demo, T: t * 60_000, O: 1, H: 1, L: 1, C: 1, Closed: true}
}

// runPersister runs p until the returned stop is called, which waits for Run
// to return nil.
func runPersister(t *testing.T, p *marketdata.Persister) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()
	return func() {
		t.Helper()
		cancel()
		if err := <-done; err != nil {
			t.Fatalf("Run = %v, want nil", err)
		}
	}
}

func TestSlowWriteDoesNotHoldThePublisherBack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hub := bus.New[marketdata.CandleUpdate]()
		repo := &recordingRepo{block: make(chan struct{})}
		stop := runPersister(t, marketdata.NewPersister(repo, hub, demoSeries))
		topic := marketdata.CandleTopic(1, marketdata.TF1m)

		hub.Publish(topic, closedBar(0))
		time.Sleep(time.Second)
		synctest.Wait() // the first batch is being written

		// More than the hub's queue holds, so it only fits if the persister
		// keeps reading during the write.
		n := bus.LosslessQueue + 1000
		start := time.Now()
		for i := 1; i <= n; i++ {
			hub.Publish(topic, closedBar(int64(i)))
		}
		if waited := time.Since(start); waited != 0 {
			t.Errorf("publisher waited %v during a slow write", waited)
		}
		close(repo.block)
		stop()

		var got []int64
		for _, b := range repo.batches {
			for _, c := range b {
				got = append(got, c.T/60_000)
			}
		}
		want := make([]int64, n+1)
		for i := range want {
			want[i] = int64(i)
		}
		if !slices.Equal(got, want) {
			t.Errorf("stored %d bars, want bars 0 to %d once each, in order", len(got), n)
		}
	})
}

func TestPersisterWritesFullBatchesAtOnceAndTheRestAfterTheDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hub := bus.New[marketdata.CandleUpdate]()
		repo := &recordingRepo{}
		stop := runPersister(t, marketdata.NewPersister(repo, hub, demoSeries))
		start := time.Now()
		for i := range 501 {
			hub.Publish(marketdata.CandleTopic(1, marketdata.TF1m), closedBar(int64(i)))
		}
		time.Sleep(time.Second)
		stop()

		sizes := make([]int, 0, len(repo.batches))
		for _, b := range repo.batches {
			sizes = append(sizes, len(b))
		}
		if !slices.Equal(sizes, []int{500, 1}) {
			t.Fatalf("wrote batches of %v candles, want [500 1]", sizes)
		}
		if at := []time.Duration{repo.at[0].Sub(start), repo.at[1].Sub(start)}; !slices.Equal(at, []time.Duration{0, 250 * time.Millisecond}) {
			t.Errorf("batches written at %v, want [0s 250ms]", at)
		}
	})
}

func TestOneBatchSpansSeries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hub := bus.New[marketdata.CandleUpdate]()
		repo := &recordingRepo{}
		hourly := marketdata.Series{InstrumentID: 2, TF: marketdata.TF1h}
		stop := runPersister(t, marketdata.NewPersister(repo, hub, demoSeries, hourly))
		for i := range 300 {
			hub.Publish(marketdata.CandleTopic(1, marketdata.TF1m), closedBar(int64(i)))
			u := closedBar(int64(i) * 60)
			u.Sub.TF = marketdata.TF1h
			hub.Publish(marketdata.CandleTopic(2, marketdata.TF1h), u)
		}
		time.Sleep(time.Second)
		stop()

		perSeries := map[marketdata.Series]int{}
		sizes := make([]int, 0, len(repo.batches))
		for _, b := range repo.batches {
			sizes = append(sizes, len(b))
			for _, c := range b {
				perSeries[marketdata.Series{InstrumentID: c.InstrumentID, TF: c.TF}]++
			}
		}
		if !slices.Equal(sizes, []int{500, 100}) {
			t.Errorf("wrote batches of %v candles, want [500 100]", sizes)
		}
		if want := map[marketdata.Series]int{demoSeries: 300, hourly: 300}; !maps.Equal(perSeries, want) {
			t.Errorf("stored %v candles per series, want %v", perSeries, want)
		}
	})
}

func TestLaggingSeriesEndsRunAfterStoringWhatWasRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hub := bus.New[marketdata.CandleUpdate]()
		repo := &recordingRepo{block: make(chan struct{})}
		p := marketdata.NewPersister(repo, hub, demoSeries)
		done := make(chan error, 1)
		go func() { done <- p.Run(t.Context()) }()

		// Enough to fill the persister's buffer and the hub's queue while the
		// first write is stuck, so the hub cuts the subscription off.
		n := 2*bus.LosslessQueue + 2*500 + 10
		for i := range n {
			hub.Publish(marketdata.CandleTopic(1, marketdata.TF1m), closedBar(int64(i)))
		}
		close(repo.block)
		err := <-done
		if !errors.Is(err, bus.ErrLagging) {
			t.Fatalf("Run = %v, want ErrLagging", err)
		}
		var got []int64
		for _, b := range repo.batches {
			for _, c := range b {
				got = append(got, c.T/60_000)
			}
		}
		for i, bar := range got {
			if bar != int64(i) {
				t.Fatalf("stored bar %d at position %d, want bars 0, 1, 2… in order", bar, i)
			}
		}
		if len(got) < bus.LosslessQueue || len(got) >= n {
			t.Errorf("stored %d of %d bars, want everything read before the cut-off", len(got), n)
		}
	})
}
