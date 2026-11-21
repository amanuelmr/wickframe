// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/amanuelmr/wickframe/internal/bus"
)

// A batch is written when it holds batchRows candles or batchDelay after its
// first candle, whichever comes first. Up to readAhead more closed candles are
// read from the hub while a batch is being written.
const (
	batchRows  = 500
	batchDelay = 250 * time.Millisecond
	readAhead  = 8 * batchRows
)

// Persister stores the closed candles of a connector's series, in batches
// that can span series. Forming updates are not stored.
type Persister struct {
	repo   CandleRepository
	hub    *bus.Hub[CandleUpdate]
	series []Series
	subs   []*bus.Sub[CandleUpdate]
}

// Series is one instrument's candles in one timeframe.
type Series struct {
	InstrumentID int32
	TF           Timeframe
}

// NewPersister returns a Persister that is already subscribed to every series,
// so it misses no candle published after NewPersister returns.
func NewPersister(repo CandleRepository, hub *bus.Hub[CandleUpdate], series ...Series) *Persister {
	p := &Persister{repo: repo, hub: hub, series: series}
	p.subs = p.subscribe()
	return p
}

func (p *Persister) subscribe() []*bus.Sub[CandleUpdate] {
	subs := make([]*bus.Sub[CandleUpdate], len(p.series))
	for i, s := range p.series {
		subs[i] = p.hub.SubscribeLossless(CandleTopic(s.InstrumentID, s.TF))
	}
	return subs
}

// Run stores closed candles until ctx is done, then stores the ones already
// published and returns nil. It fails if a write fails or a subscription falls
// too far behind, after storing what it had read; calling Run again
// resubscribes.
func (p *Persister) Run(ctx context.Context) error {
	subs := p.subs
	if subs == nil {
		subs = p.subscribe()
	}
	p.subs = nil

	rows := make(chan Candle, readAhead)
	var (
		once    sync.Once
		readErr error
		readers sync.WaitGroup
	)
	// stop ends every subscription; each reader still reads what it has queued.
	stop := func(err error) {
		once.Do(func() {
			readErr = err
			for _, s := range subs {
				s.Close()
			}
		})
	}
	for i, sub := range subs {
		id := p.series[i].InstrumentID
		readers.Go(func() {
			for {
				u, err := sub.Next(ctx)
				if err != nil {
					if ctx.Err() == nil {
						stop(err)
					}
					return
				}
				if u.Closed {
					rows <- Candle{
						InstrumentID: id, TF: u.Sub.TF, T: u.T,
						O: u.O, H: u.H, L: u.L, C: u.C, V: u.V, Rev: u.Rev,
					}
				}
			}
		})
	}
	go func() {
		readers.Wait()
		close(rows)
	}()
	// Writes outlive ctx, so the candles read before shutdown are stored.
	err := p.write(context.WithoutCancel(ctx), rows)
	stop(nil)
	for range rows {
	}
	if err != nil {
		return err
	}
	if readErr != nil {
		return fmt.Errorf("persist: %w", readErr)
	}
	return nil
}

func (p *Persister) write(ctx context.Context, rows <-chan Candle) error {
	batch := make([]Candle, 0, batchRows)
	var due <-chan time.Time
	for {
		select {
		case c, ok := <-rows:
			if !ok {
				return p.upsert(ctx, batch)
			}
			batch = append(batch, c)
			if len(batch) == 1 {
				due = time.After(batchDelay)
			}
			if len(batch) < batchRows {
				continue
			}
		case <-due:
		}
		if err := p.upsert(ctx, batch); err != nil {
			return err
		}
		batch, due = batch[:0], nil
	}
}

func (p *Persister) upsert(ctx context.Context, batch []Candle) error {
	if len(batch) == 0 {
		return nil
	}
	if err := p.repo.UpsertCandles(ctx, batch); err != nil {
		return fmt.Errorf("persist %d candles: %w", len(batch), err)
	}
	return nil
}
