// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import (
	"errors"
	"math"
)

// Reasons a candle update is invalid.
var (
	ErrNotFinite  = errors.New("a value is not finite")
	ErrOutOfRange = errors.New("high and low do not contain open and close")
	ErrNegativeV  = errors.New("volume is negative")
	ErrMisaligned = errors.New("open time is not on a bar boundary")
)

// ValidateCandle checks the invariants that hold for a single update: finite
// values, a consistent range, and an aligned open time. Stale updates are left
// to consumers, which compare t and rev.
func ValidateCandle(u CandleUpdate) error {
	for _, x := range []float64{u.O, u.H, u.L, u.C, u.V} {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return ErrNotFinite
		}
	}
	if u.L > min(u.O, u.C) || u.H < max(u.O, u.C) {
		return ErrOutOfRange
	}
	if u.V < 0 {
		return ErrNegativeV
	}
	// Session markets align intraday bars to the session open; M1 has only 24x7.
	if d, ok := BarLength(u.Sub.TF); ok && u.T%d.Milliseconds() != 0 {
		return ErrMisaligned
	}
	return nil
}
