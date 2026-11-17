// SPDX-License-Identifier: AGPL-3.0-only

package clock_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/amanuelmr/wickframe/internal/platform/clock"
)

func TestSystemFollowsTimeInUTC(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var clk clock.Clock = clock.System{}
		start := clk.Now()
		if start.Location() != time.UTC {
			t.Fatalf("Now() location = %v, want UTC", start.Location())
		}
		time.Sleep(90 * time.Second)
		if got := clk.Now().Sub(start); got != 90*time.Second {
			t.Fatalf("Now() advanced by %v, want 90s", got)
		}
	})
}

func TestFakeMovesOnlyWhenAdvanced(t *testing.T) {
	start := time.Date(2026, 3, 29, 0, 59, 0, 0, time.FixedZone("CET", 3600))
	clk := clock.NewFake(start)
	if got := clk.Now(); !got.Equal(start) || got.Location() != time.UTC {
		t.Fatalf("Now() = %v, want %v in UTC", got, start)
	}
	clk.Advance(time.Minute)
	clk.Advance(time.Hour)
	if got, want := clk.Now(), start.Add(61*time.Minute); !got.Equal(want) {
		t.Fatalf("Now() = %v, want %v", got, want)
	}
}
