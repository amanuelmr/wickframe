// SPDX-License-Identifier: AGPL-3.0-only

// Package storetest is the contract test suite that every store adapter must
// pass, so the SQLite and PostgreSQL adapters behave identically (ADR-0004).
package storetest

import (
	"context"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/amanuelmr/wickframe/internal/marketdata"
)

// TestCandles runs the CandleRepository contract. newRepo must return an empty,
// migrated store each time it is called.
func TestCandles(t *testing.T, newRepo func(t *testing.T) marketdata.CandleRepository) {
	const hour = int64(3_600_000)
	t0 := int64(1_767_225_600_000) // 2026-01-01T00:00:00Z
	qv, n := 12_345.678, int32(987)
	bar := func(i int64) marketdata.Candle {
		return marketdata.Candle{
			InstrumentID: 42, TF: marketdata.TF1h, T: t0 + i*hour,
			O: 100, H: 110, L: 90, C: 105, V: 1.5,
		}
	}
	all := marketdata.CandleRange{InstrumentID: 42, TF: marketdata.TF1h, From: t0, To: t0 + 100*hour, Limit: 1000}

	upsert := func(t *testing.T, repo marketdata.CandleRepository, cs ...marketdata.Candle) {
		t.Helper()
		if err := repo.UpsertCandles(t.Context(), cs); err != nil {
			t.Fatalf("UpsertCandles: %v", err)
		}
	}
	read := func(t *testing.T, repo marketdata.CandleRepository, r marketdata.CandleRange) []marketdata.Candle {
		t.Helper()
		got, err := repo.Candles(t.Context(), r)
		if err != nil {
			t.Fatalf("Candles(%+v): %v", r, err)
		}
		return got
	}
	check := func(t *testing.T, got, want []marketdata.Candle) {
		t.Helper()
		if len(got) == 0 && len(want) == 0 {
			return
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("candles differ (-want +got):\n%s", diff)
		}
	}

	t.Run("round trips every field", func(t *testing.T) {
		repo := newRepo(t)
		full := bar(1)
		full.O, full.H, full.L, full.C, full.V = 0.1+0.2, math.MaxFloat64, 1e-8, 65_432.10987654321, 0
		full.QV, full.N, full.Rev, full.Derived = &qv, &n, 3, true
		upsert(t, repo, bar(0), full)
		check(t, read(t, repo, all), []marketdata.Candle{bar(0), full})
	})

	t.Run("reads a half-open range oldest first", func(t *testing.T) {
		repo := newRepo(t)
		upsert(t, repo, bar(3), bar(1), bar(0), bar(2), bar(4))
		r := all
		r.From, r.To = t0+hour, t0+4*hour
		check(t, read(t, repo, r), []marketdata.Candle{bar(1), bar(2), bar(3)})
	})

	t.Run("stops at the limit", func(t *testing.T) {
		repo := newRepo(t)
		upsert(t, repo, bar(0), bar(1), bar(2), bar(3))
		r := all
		r.Limit = 2
		check(t, read(t, repo, r), []marketdata.Candle{bar(0), bar(1)})
	})

	t.Run("keeps instruments and timeframes apart", func(t *testing.T) {
		repo := newRepo(t)
		other, daily := bar(0), bar(0)
		other.InstrumentID = 7
		daily.TF = marketdata.TF1D
		upsert(t, repo, bar(0), other, daily)
		check(t, read(t, repo, all), []marketdata.Candle{bar(0)})
	})

	t.Run("an equal or higher rev replaces and a lower rev is ignored", func(t *testing.T) {
		repo := newRepo(t)
		corrected := bar(0)
		corrected.C, corrected.Rev = 101, 2
		upsert(t, repo, bar(0), corrected)

		stale := bar(0)
		stale.C, stale.Rev = 999, 1
		upsert(t, repo, stale)
		check(t, read(t, repo, all), []marketdata.Candle{corrected})

		again := corrected
		again.V = 2.5
		upsert(t, repo, again)
		check(t, read(t, repo, all), []marketdata.Candle{again})
	})

	t.Run("accepts an empty batch and returns nothing for an empty range", func(t *testing.T) {
		repo := newRepo(t)
		upsert(t, repo)
		check(t, read(t, repo, all), nil)
	})

	t.Run("honours a cancelled context", func(t *testing.T) {
		repo := newRepo(t)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if err := repo.UpsertCandles(ctx, []marketdata.Candle{bar(0)}); err == nil {
			t.Error("UpsertCandles with a cancelled context succeeded")
		}
		check(t, read(t, repo, all), nil)
	})
}
