// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import (
	"context"
	"fmt"
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

// Persister stores the closed candles of one series. Forming updates are not
// stored.
type Persister struct {
	repo         CandleRepository
	hub          *bus.Hub[CandleUpdate]
	topic        string
	instrumentID int32
	sub          *bus.Sub[CandleUpdate]
}

// NewPersister returns a Persister that is already subscribed, so it misses no
// candle published after NewPersister returns.
func NewPersister(repo CandleRepository, hub *bus.Hub[CandleUpdate], instrumentID int32, tf Timeframe) *Persister {
	topic := CandleTopic(instrumentID, tf)
	return &Persister{repo: repo, hub: hub, topic: topic, instrumentID: instrumentID, sub: hub.SubscribeLossless(topic)}
}

// Run stores closed candles until ctx is done, then stores the ones already
// published and returns nil. It fails if a write fails or the subscription
// falls too far behind; calling Run again resubscribes.
func (p *Persister) Run(ctx context.Context) error {
	sub := p.sub
	if sub == nil {
		sub = p.hub.SubscribeLossless(p.topic)
	}
	p.sub = nil

	rows := make(chan Candle, readAhead)
	var readErr error
	go func() {
		defer close(rows)
		for {
			u, err := sub.Next(ctx)
			if err != nil {
				readErr = err
				return
			}
			if u.Closed {
				rows <- Candle{
					InstrumentID: p.instrumentID, TF: u.Sub.TF, T: u.T,
					O: u.O, H: u.H, L: u.L, C: u.C, V: u.V, Rev: u.Rev,
				}
			}
		}
	}()
	// Writes outlive ctx, so the candles read before shutdown are stored.
	err := p.write(context.WithoutCancel(ctx), rows)
	sub.Close()
	for range rows {
	}
	if err != nil {
		return err
	}
	if ctx.Err() == nil {
		return fmt.Errorf("persist %s: %w", p.topic, readErr)
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
		return fmt.Errorf("persist %s: %w", p.topic, err)
	}
	return nil
}
