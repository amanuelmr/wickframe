// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/store/sqlite"
	"github.com/amanuelmr/wickframe/internal/store/storetest"
)

// serve returns a handler for the API over a store holding storetest bars 0
// to 9 of instrument test:X.
func serve(t *testing.T) http.Handler {
	t.Helper()
	store, err := sqlite.Open(t.Context(), filepath.Join(t.TempDir(), "wickframe.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	bars := make([]marketdata.Candle, 10)
	for i := range bars {
		bars[i] = storetest.Bar(int64(i))
	}
	if err := store.UpsertCandles(t.Context(), bars); err != nil {
		t.Fatal(err)
	}
	return handler(store)
}

func handler(repo marketdata.CandleRepository) http.Handler {
	mux := http.NewServeMux()
	New(mux, repo, map[string]int32{"test:X": 42}, slog.New(slog.DiscardHandler))
	return mux
}

type page struct {
	Bars []struct {
		T int64 `json:"t"`
	} `json:"bars"`
	NextTo          *int64 `json:"nextTo"`
	HistoryComplete bool   `json:"historyComplete"`
}

func get(t *testing.T, h http.Handler, query string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/candles?"+query, nil))
	return rec
}

func getPage(t *testing.T, h http.Handler, query string) page {
	t.Helper()
	rec := get(t, h, query)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET ?%s = %d %s", query, rec.Code, rec.Body)
	}
	var p page
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

// summary describes a page by bar numbers: "6-9 next=6" or "0-1 complete".
func summary(p page) string {
	n := func(t int64) int64 { return (t - storetest.T0) / 3_600_000 }
	s := "empty"
	if len(p.Bars) > 0 {
		s = fmt.Sprintf("%d-%d", n(p.Bars[0].T), n(p.Bars[len(p.Bars)-1].T))
	}
	if p.NextTo != nil {
		s += fmt.Sprintf(" next=%d", n(*p.NextTo+1))
	}
	if p.HistoryComplete {
		s += " complete"
	}
	return s
}

func TestCandlesPageBackwardsFromTheNewestBar(t *testing.T) {
	h := serve(t)
	var got []string
	query := "instrument=test:X&tf=1h&limit=4"
	for range 4 {
		p := getPage(t, h, query)
		got = append(got, summary(p))
		if p.NextTo == nil {
			break
		}
		query = fmt.Sprintf("instrument=test:X&tf=1h&limit=4&to=%d", *p.NextTo)
	}
	if diff := cmp.Diff([]string{"6-9 next=6", "2-5 next=2", "0-1 complete"}, got); diff != "" {
		t.Errorf("pages (-want +got):\n%s", diff)
	}
}

func TestCandlesRangeIncludesFromAndTo(t *testing.T) {
	h := serve(t)
	for _, tt := range []struct{ query, want string }{
		{fmt.Sprintf("instrument=test:X&tf=1h&to=%d", storetest.Bar(5).T), "0-5 complete"},
		{fmt.Sprintf("instrument=test:X&tf=1h&from=%d&to=%d", storetest.Bar(3).T, storetest.Bar(5).T), "3-5"},
		{fmt.Sprintf("instrument=test:X&tf=1h&from=%d&to=%d&limit=2", storetest.Bar(3).T, storetest.Bar(5).T), "4-5 next=4"},
		{fmt.Sprintf("instrument=test:X&tf=1h&to=%d", storetest.T0-1), "empty complete"},
		{"instrument=test:X&tf=1h&to=9223372036854775807", "0-9 complete"},
		{"instrument=test:X&tf=1D", "empty complete"},
	} {
		if got := summary(getPage(t, h, tt.query)); got != tt.want {
			t.Errorf("?%s = %s, want %s", tt.query, got, tt.want)
		}
	}
}

type failingRepo struct{ marketdata.CandleRepository }

func (failingRepo) Candles(context.Context, marketdata.CandleRange) ([]marketdata.Candle, error) {
	return nil, errors.New("disk I/O error at /secret/path")
}

func TestCandlesErrorsAreProblemDetails(t *testing.T) {
	h := serve(t)
	for _, tt := range []struct {
		h          http.Handler
		query      string
		status     int
		typ        string
		withDetail bool
	}{
		{h, "instrument=test:Y&tf=1h", 404, "urn:wickframe:problem:not-found", true},
		{h, "instrument=test:X&tf=1h&limit=5001", 422, "urn:wickframe:problem:validation-error", true},
		{h, "instrument=test:X&tf=1h&limit=0", 422, "urn:wickframe:problem:validation-error", true},
		{h, "instrument=test:X&tf=7m", 422, "urn:wickframe:problem:validation-error", true},
		{h, "instrument=test:X", 422, "urn:wickframe:problem:validation-error", true},
		{h, "instrument=test:X&tf=1h&to=soon", 422, "urn:wickframe:problem:validation-error", true},
		{handler(failingRepo{}), "instrument=test:X&tf=1h", 500, "urn:wickframe:problem:internal-error", false},
	} {
		rec := get(t, tt.h, tt.query)
		var p Problem
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatalf("?%s: %v in %s", tt.query, err, rec.Body)
		}
		if rec.Code != tt.status || p.Status != tt.status || p.Type != tt.typ || (p.Detail != "") != tt.withDetail {
			t.Errorf("?%s = %d %s, want %d with type %s", tt.query, rec.Code, rec.Body, tt.status, tt.typ)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
			t.Errorf("?%s Content-Type = %q", tt.query, ct)
		}
	}
}
