// SPDX-License-Identifier: AGPL-3.0-only

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/store/postgres/db"
)

var _ marketdata.CandleRepository = (*Store)(nil)

// UpsertCandles implements marketdata.CandleRepository.
func (s *Store) UpsertCandles(ctx context.Context, candles []marketdata.Candle) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		for _, c := range candles {
			err := q.UpsertCandle(ctx, db.UpsertCandleParams{
				InstrumentID: c.InstrumentID,
				Tf:           int16(c.TF),
				T:            c.T,
				O:            c.O,
				H:            c.H,
				L:            c.L,
				C:            c.C,
				V:            c.V,
				Qv:           c.QV,
				N:            c.N,
				Rev:          c.Rev,
				Derived:      c.Derived,
			})
			if err != nil {
				return fmt.Errorf("upsert candle %d/%d/%d: %w", c.InstrumentID, c.TF, c.T, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("upsert candles: %w", err)
	}
	return nil
}

// Candles implements marketdata.CandleRepository.
func (s *Store) Candles(ctx context.Context, r marketdata.CandleRange) ([]marketdata.Candle, error) {
	rows, err := s.q.Candles(ctx, db.CandlesParams{
		InstrumentID: r.InstrumentID,
		Tf:           int16(r.TF),
		FromT:        r.From,
		ToT:          r.To,
		RowLimit:     int32(min(r.Limit, 1<<31-1)), //nolint:gosec // clamped above
	})
	if err != nil {
		return nil, fmt.Errorf("candles %d/%d [%d, %d): %w", r.InstrumentID, r.TF, r.From, r.To, err)
	}
	out := make([]marketdata.Candle, len(rows))
	for i, row := range rows {
		out[i] = marketdata.Candle{
			InstrumentID: row.InstrumentID,
			TF:           marketdata.Timeframe(row.Tf),
			T:            row.T,
			O:            row.O,
			H:            row.H,
			L:            row.L,
			C:            row.C,
			V:            row.V,
			QV:           row.Qv,
			N:            row.N,
			Rev:          row.Rev,
			Derived:      row.Derived,
		}
	}
	return out, nil
}

// LastCandleT implements marketdata.CandleRepository.
func (s *Store) LastCandleT(ctx context.Context, instrumentID int32, tf marketdata.Timeframe) (int64, error) {
	t, err := s.q.LastCandleT(ctx, db.LastCandleTParams{InstrumentID: instrumentID, Tf: int16(tf)})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("last candle %d/%d: %w", instrumentID, tf, err)
	}
	return t, nil
}
