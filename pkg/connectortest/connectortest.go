// SPDX-License-Identifier: AGPL-3.0-only

// Package connectortest is the conformance kit every market data connector
// must pass against its fixtures (04-market-data §2).
package connectortest

import (
	"context"
	"errors"
	"net/url"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/amanuelmr/wickframe/internal/marketdata"
)

// Fixture describes a connector under test and what its fixtures cover.
type Fixture struct {
	// New returns a connector that serves from the fixtures, never the network.
	New func(t *testing.T) marketdata.Connector
	// History is a range the fixtures answer in full, with a bar at every
	// step. The source also has a bar opening at To, which History must leave
	// out. The range should need more than one request to the source.
	History marketdata.HistoryRequest
	// Stream is a series whose fixture sends forming updates and then closes
	// the bar.
	Stream marketdata.StreamSub
}

// Run runs the kit. Its subtests do not run in parallel, so the goroutine
// count can show leaks.
func Run(t *testing.T, f Fixture) {
	t.Run("metadata is complete", func(t *testing.T) { checkInfo(t, f) })
	t.Run("history covers the range in order", func(t *testing.T) { checkHistory(t, f) })
	t.Run("history stops with the context", func(t *testing.T) { checkHistoryCancel(t, f) })
	t.Run("stream goes from forming to closed and stops with the context", func(t *testing.T) { checkStream(t, f) })
}

func checkInfo(t *testing.T, f Fixture) {
	info := f.New(t).Info()
	if info.ID == "" || info.Name == "" || info.Version == "" {
		t.Errorf("ID, Name and Version must be set: %q, %q, %q", info.ID, info.Name, info.Version)
	}
	if u, err := url.Parse(info.TermsURL); err != nil || u.Scheme != "https" || u.Host == "" {
		t.Errorf("TermsURL %q is not an https URL", info.TermsURL)
	}
	if !slices.Contains([]marketdata.Auth{marketdata.AuthNone, marketdata.AuthAPIKey, marketdata.AuthAPIKeySecret, marketdata.AuthOAuth}, info.Auth) {
		t.Errorf("Auth %q is not a known mode", info.Auth)
	}
	if info.Freshness == "" {
		t.Error("Freshness is not set")
	}
	if info.SettingsSchema == "" {
		t.Error("SettingsSchema is not set")
	}
	if !slices.Contains(info.NativeTimeframes, f.History.TF) {
		t.Errorf("NativeTimeframes %v do not include the fixture's %v", info.NativeTimeframes, f.History.TF)
	}
	if !slices.Contains(info.StreamTimeframes, f.Stream.TF) {
		t.Errorf("StreamTimeframes %v do not include the fixture's %v", info.StreamTimeframes, f.Stream.TF)
	}
}

func checkHistory(t *testing.T, f Fixture) {
	req := f.History
	step, ok := marketdata.BarLength(req.TF)
	if !ok {
		t.Fatalf("the fixture's timeframe %v has no fixed bar length", req.TF)
	}
	got, err := f.New(t).History(t.Context(), req)
	if err != nil {
		t.Fatalf("History(%+v): %v", req, err)
	}
	want := int((req.To - req.From) / step.Milliseconds())
	if len(got) != want {
		t.Errorf("History returned %d bars, want %d for the fixture's range", len(got), want)
	}
	for i, c := range got {
		if wantT := req.From + int64(i)*step.Milliseconds(); c.T != wantT {
			t.Fatalf("bar %d opens at %d, want %d: overlap, hole or wrong order", i, c.T, wantT)
		}
		if c.TF != req.TF || c.InstrumentID != 0 {
			t.Errorf("bar %d has TF %v and InstrumentID %d, want %v and 0", i, c.TF, c.InstrumentID, req.TF)
		}
		u := marketdata.CandleUpdate{Sub: marketdata.StreamSub{Symbol: req.Symbol, TF: c.TF}, T: c.T, O: c.O, H: c.H, L: c.L, C: c.C, V: c.V, Closed: true}
		if err := marketdata.ValidateCandle(u); err != nil {
			t.Errorf("bar %d is invalid: %v", i, err)
		}
	}
}

func checkHistoryCancel(t *testing.T, f Fixture) {
	c := f.New(t)
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.History(ctx, f.History); !errors.Is(err, context.Canceled) {
		t.Errorf("History with a cancelled context = %v, want context.Canceled", err)
	}
	checkNoLeak(t, before)
}

type sink chan marketdata.CandleUpdate

func (s sink) Candle(u marketdata.CandleUpdate) { s <- u }

func checkStream(t *testing.T, f Fixture) {
	c := f.New(t)
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	updates := make(sink)
	done := make(chan error, 1)
	go func() { done <- c.Stream(ctx, []marketdata.StreamSub{f.Stream}, updates) }()

	formed := map[int64]bool{}
	closed := map[int64]bool{}
	timeout := time.After(10 * time.Second)
	for transition := false; !transition; {
		select {
		case u := <-updates:
			if u.Sub != f.Stream {
				t.Fatalf("update for %+v, want %+v", u.Sub, f.Stream)
			}
			if err := marketdata.ValidateCandle(u); err != nil {
				t.Fatalf("invalid update %+v: %v", u, err)
			}
			if !u.Closed {
				if closed[u.T] {
					t.Fatalf("forming update for bar %d after it closed", u.T)
				}
				formed[u.T] = true
				continue
			}
			closed[u.T] = true
			transition = formed[u.T]
		case err := <-done:
			t.Fatalf("Stream returned %v before any bar went from forming to closed", err)
		case <-timeout:
			t.Fatal("no bar went from forming to closed within 10s")
		}
	}

	cancel()
	for {
		select {
		case <-updates:
			continue
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Stream after cancel = %v, want context.Canceled", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Stream did not return within 5s of cancellation")
		}
		break
	}
	checkNoLeak(t, before)
}

// checkNoLeak waits up to 2 s for the goroutine count to fall back to before.
func checkNoLeak(t *testing.T, before int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for runtime.NumGoroutine() > before {
		select {
		case <-deadline:
			t.Errorf("%d goroutines left running, %d before", runtime.NumGoroutine()-before, before)
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}
