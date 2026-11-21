// SPDX-License-Identifier: AGPL-3.0-only

// Package feed holds parsing helpers for untrusted market-data payloads
// (trust boundary TB2, docs/architecture/11-security.md).
package feed

import (
	"fmt"
	"math"
	"strconv"
)

// ParseDecimal parses a plain decimal string from a source ("71407.85",
// "-0.5", "1e-8") into a finite float64.
//
// strconv.ParseFloat alone is too permissive for feeds: it accepts "NaN",
// "Inf", "infinity", hex floats ("0x1p-2") and underscores ("1_000"), and
// returns ±Inf with an error on overflow. ParseDecimal accepts only
// [sign] digits [. digits] [e|E [sign] digits] and rejects non-finite results.
func ParseDecimal(s string) (float64, error) {
	if !isPlainDecimal(s) {
		return 0, fmt.Errorf("feed: not a plain decimal: %q", s)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("feed: parse %q: %w", s, err)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("feed: non-finite value %q", s)
	}
	return f, nil
}

func isPlainDecimal(s string) bool {
	i, n := 0, len(s)
	if i < n && (s[i] == '+' || s[i] == '-') {
		i++
	}
	digits := 0
	for i < n && isDigit(s[i]) {
		i++
		digits++
	}
	if i < n && s[i] == '.' {
		i++
		for i < n && isDigit(s[i]) {
			i++
			digits++
		}
	}
	if digits == 0 {
		return false
	}
	if i < n && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < n && (s[i] == '+' || s[i] == '-') {
			i++
		}
		exp := 0
		for i < n && isDigit(s[i]) {
			i++
			exp++
		}
		if exp == 0 {
			return false
		}
	}
	return i == n
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
