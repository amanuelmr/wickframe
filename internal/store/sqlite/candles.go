// SPDX-License-Identifier: AGPL-3.0-only

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/store/sqlite/db"
)

var _ marketdata.CandleRepository = (*Store)(nil)

// UpsertCandles implements marketdata.CandleRepository.
func (s *Store) UpsertCandles(ctx context.Context, candles []marketdata.Candle) error {
	tx, err := s.write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("upsert candles: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := s.w.WithTx(tx)
	for _, c := range candles {
		p := db.UpsertCandleParams{
			InstrumentID: int64(c.InstrumentID),
			Tf:           int64(c.TF),
			T:            c.T,
			O:            c.O,
			H:            c.H,
			L:            c.L,
			C:            c.C,
			V:            c.V,
			Rev:          int64(c.Rev),
		}
		if c.QV != nil {
			p.Qv = sql.NullFloat64{Float64: *c.QV, Valid: true}
		}
		if c.N != nil {
			p.N = sql.NullInt64{Int64: int64(*c.N), Valid: true}
		}
		if c.Derived {
			p.Derived = 1
		}
		if err := q.UpsertCandle(ctx, p); err != nil {
			return fmt.Errorf("upsert candle %d/%d/%d: %w", c.InstrumentID, c.TF, c.T, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("upsert candles: %w", err)
	}
	return nil
}

// Candles implements marketdata.CandleRepository.
func (s *Store) Candles(ctx context.Context, r marketdata.CandleRange) ([]marketdata.Candle, error) {
	rows, err := s.r.Candles(ctx, db.CandlesParams{
		InstrumentID: int64(r.InstrumentID),
		Tf:           int64(r.TF),
		FromT:        r.From,
		ToT:          r.To,
		RowLimit:     int64(r.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("candles %d/%d [%d, %d): %w", r.InstrumentID, r.TF, r.From, r.To, err)
	}
	out := make([]marketdata.Candle, len(rows))
	for i, row := range rows {
		c := marketdata.Candle{
			InstrumentID: int32(row.InstrumentID),      //nolint:gosec // stored from an int32
			TF:           marketdata.Timeframe(row.Tf), //nolint:gosec // stored from an int16
			T:            row.T,
			O:            row.O,
			H:            row.H,
			L:            row.L,
			C:            row.C,
			V:            row.V,
			Rev:          int16(row.Rev), //nolint:gosec // stored from an int16
			Derived:      row.Derived != 0,
		}
		if row.Qv.Valid {
			c.QV = &row.Qv.Float64
		}
		if row.N.Valid {
			n := int32(row.N.Int64) //nolint:gosec // stored from an int32
			c.N = &n
		}
		out[i] = c
	}
	slices.Reverse(out) // read newest first, so the limit keeps the newest
	return out, nil
}

// LastCandleT implements marketdata.CandleRepository.
func (s *Store) LastCandleT(ctx context.Context, instrumentID int32, tf marketdata.Timeframe) (int64, error) {
	t, err := s.r.LastCandleT(ctx, db.LastCandleTParams{InstrumentID: int64(instrumentID), Tf: int64(tf)})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("last candle %d/%d: %w", instrumentID, tf, err)
	}
	return t, nil
}
