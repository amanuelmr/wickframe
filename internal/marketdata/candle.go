// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import "context"

// Timeframe is a candle period (03-domain-model §2.3). The values are stored in
// the candles table, so existing ones must never change.
type Timeframe int16

// Timeframes.
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

// Candle is a closed candle (03-domain-model §2.4). Forming candles are never
// stored, so it has no state.
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

// CandleRange selects candles of one instrument and timeframe with
// From <= T < To, oldest first, at most Limit of them.
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
}
