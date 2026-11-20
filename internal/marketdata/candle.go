// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import "context"

// Timeframe is a candle period. The values are stored, so existing ones must
// never change.
type Timeframe int16

// Timeframes from shortest to longest.
const (
	TF1m Timeframe = iota + 1
	TF3m
	TF5m
	TF15m
	TF30m
	TF1h
	TF2h
	TF4h
	TF6h
	TF12h
	TF1D
	TF1W
	TF1M
)

// Candle is a closed candle; forming candles are never stored.
type Candle struct {
	InstrumentID int32
	TF           Timeframe
	T            int64 // open time, Unix ms UTC
	O, H, L, C   float64
	V            float64
	QV           *float64 // quote volume, if the source provides it
	N            *int32   // trade count, if the source provides it
	Rev          int16
	Derived      bool
}

// CandleRange selects the newest Limit candles of one instrument and timeframe
// with From <= T < To.
type CandleRange struct {
	InstrumentID int32
	TF           Timeframe
	From, To     int64
	Limit        int
}

// CandleRepository stores closed candles.
type CandleRepository interface {
	// UpsertCandles stores candles in one transaction. A stored candle is
	// replaced only by one with an equal or higher Rev, so a late, stale write
	// never undoes a correction.
	UpsertCandles(ctx context.Context, candles []Candle) error
	Candles(ctx context.Context, r CandleRange) ([]Candle, error)
	// LastCandleT returns the open time of the series' newest candle, or 0 if
	// there is none.
	LastCandleT(ctx context.Context, instrumentID int32, tf Timeframe) (int64, error)
}
