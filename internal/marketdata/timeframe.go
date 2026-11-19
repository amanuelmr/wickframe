// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import (
	"fmt"
	"time"
)

var tfCodes = [...]string{
	TF1m: "1m", TF3m: "3m", TF5m: "5m", TF15m: "15m", TF30m: "30m",
	TF1h: "1h", TF2h: "2h", TF4h: "4h", TF6h: "6h", TF12h: "12h",
	TF1D: "1D", TF1W: "1W", TF1M: "1M",
}

// String returns the timeframe's code, such as "1m" or "1D".
func (tf Timeframe) String() string {
	if tf < TF1m || tf > TF1M {
		return fmt.Sprintf("Timeframe(%d)", tf)
	}
	return tfCodes[tf]
}

// ParseTimeframe returns the timeframe for a code such as "1m" or "1D".
func ParseTimeframe(code string) (Timeframe, bool) {
	for tf := TF1m; tf <= TF1M; tf++ {
		if tfCodes[tf] == code {
			return tf, true
		}
	}
	return 0, false
}

// CandleTopic is the bus topic for an instrument's candles in one timeframe.
func CandleTopic(instrumentID int32, tf Timeframe) string {
	return fmt.Sprintf("candle.%d.%s", instrumentID, tf)
}

var barLengths = map[Timeframe]time.Duration{
	TF1m: time.Minute, TF3m: 3 * time.Minute, TF5m: 5 * time.Minute,
	TF15m: 15 * time.Minute, TF30m: 30 * time.Minute, TF1h: time.Hour,
	TF2h: 2 * time.Hour, TF4h: 4 * time.Hour, TF6h: 6 * time.Hour, TF12h: 12 * time.Hour,
}

// BarLength returns the fixed length of an intraday timeframe, whose bars are
// aligned to the Unix epoch in 24x7 markets. Daily and longer bars follow the
// instrument's daily boundary, so they have none.
func BarLength(tf Timeframe) (time.Duration, bool) {
	d, ok := barLengths[tf]
	return d, ok
}
