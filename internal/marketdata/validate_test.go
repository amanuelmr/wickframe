// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import (
	"errors"
	"math"
	"testing"
)

func TestValidateCandle(t *testing.T) {
	ok := CandleUpdate{Sub: StreamSub{TF: TF5m}, T: 300_000, O: 10, H: 12, L: 9, C: 11, V: 3}
	with := func(f func(*CandleUpdate)) CandleUpdate {
		u := ok
		f(&u)
		return u
	}
	tests := []struct {
		name string
		u    CandleUpdate
		want error
	}{
		{"valid", ok, nil},
		{"flat bar with no volume", with(func(u *CandleUpdate) { u.O, u.H, u.L, u.C, u.V = 10, 10, 10, 10, 0 }), nil},
		{"daily bar is not checked for alignment", with(func(u *CandleUpdate) { u.Sub.TF, u.T = TF1D, 1 }), nil},
		{"NaN close", with(func(u *CandleUpdate) { u.C = math.NaN() }), ErrNotFinite},
		{"infinite volume", with(func(u *CandleUpdate) { u.V = math.Inf(1) }), ErrNotFinite},
		{"high below close", with(func(u *CandleUpdate) { u.C = 12.5 }), ErrOutOfRange},
		{"high below open", with(func(u *CandleUpdate) { u.O = 12.5 }), ErrOutOfRange},
		{"low above open", with(func(u *CandleUpdate) { u.L = 10.5 }), ErrOutOfRange},
		{"low above high", with(func(u *CandleUpdate) { u.L, u.H = 13, 8 }), ErrOutOfRange},
		{"negative volume", with(func(u *CandleUpdate) { u.V = -1 }), ErrNegativeV},
		{"open time inside a bar", with(func(u *CandleUpdate) { u.T = 360_000 }), ErrMisaligned},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateCandle(tt.u); !errors.Is(err, tt.want) {
				t.Errorf("ValidateCandle = %v, want %v", err, tt.want)
			}
		})
	}
}
