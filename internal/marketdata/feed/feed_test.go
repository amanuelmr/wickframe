// SPDX-License-Identifier: AGPL-3.0-only

package feed

import (
	"math"
	"testing"
)

func TestParseDecimal(t *testing.T) {
	t.Parallel()
	ok := map[string]float64{
		"71407.85": 71407.85, "0": 0, "-0.5": -0.5, "+3": 3, "1e-8": 1e-8,
		"1.50000000": 1.5, ".5": 0.5, "5.": 5, "2.5E+3": 2500,
	}
	for s, want := range ok {
		got, err := ParseDecimal(s)
		if err != nil || got != want {
			t.Errorf("ParseDecimal(%q) = %v, %v; want %v", s, got, err, want)
		}
	}
	for _, s := range []string{
		"", " ", "NaN", "nan", "Inf", "-inf", "Infinity", "0x1p-2", "1_000",
		" 1.5", "1.5 ", "1e", "1e+", ".", "-", "1.2.3", "1e400", "-1e400", "١٢",
	} {
		if got, err := ParseDecimal(s); err == nil {
			t.Errorf("ParseDecimal(%q) = %v, want error", s, got)
		}
	}
}

func FuzzParseDecimal(f *testing.F) {
	for _, s := range []string{"1.5", "NaN", "0x10", "1e308", "-0"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		v, err := ParseDecimal(s)
		if err == nil && (math.IsNaN(v) || math.IsInf(v, 0)) {
			t.Fatalf("ParseDecimal(%q) returned non-finite %v", s, v)
		}
	})
}
