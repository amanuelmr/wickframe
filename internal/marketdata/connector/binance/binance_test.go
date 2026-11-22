// SPDX-License-Identifier: AGPL-3.0-only

package binance_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/google/go-cmp/cmp"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/marketdata/connector/binance"
	"github.com/amanuelmr/wickframe/internal/platform/clock"
	"github.com/amanuelmr/wickframe/pkg/connectortest"
)

const (
	t0     = int64(1_767_225_600_000) // 2026-01-01T00:00:00Z
	minute = int64(60_000)
)

var fixtures = filepath.Join("..", "..", "..", "..", "testdata", "connectors", "binance")

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixtures, name)) //nolint:gosec // fixture names are constants
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// source simulates Binance's REST host: GET /api/v3/klines over a synthetic
// 1m series of bars bars from t0, with endTime inclusive and limit capped at
// 1000, as Binance documents. Queued responses are served first.
type source struct {
	bars int

	mu       sync.Mutex
	queued   []*http.Response
	requests []string
	usedHdr  string // X-MBX-USED-WEIGHT-1M on the next 200, if set
}

func (s *source) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := req.Context().Err(); err != nil {
		return nil, fmt.Errorf("round trip: %w", err)
	}
	s.requests = append(s.requests, req.URL.RawQuery)
	if len(s.queued) > 0 {
		res := s.queued[0]
		s.queued = s.queued[1:]
		return res, nil
	}
	q := req.URL.Query()
	start, _ := strconv.ParseInt(q.Get("startTime"), 10, 64)
	end, _ := strconv.ParseInt(q.Get("endTime"), 10, 64)
	limit, _ := strconv.Atoi(q.Get("limit"))
	limit = min(limit, 1000)
	rows := []string{}
	for i := range s.bars {
		if t := t0 + int64(i)*minute; t >= start && t <= end && len(rows) < limit {
			rows = append(rows, row(i))
		}
	}
	res := response(http.StatusOK, "["+strings.Join(rows, ",")+"]")
	if s.usedHdr != "" {
		res.Header.Set("X-MBX-USED-WEIGHT-1M", s.usedHdr)
		s.usedHdr = ""
	}
	return res, nil
}

func (s *source) queue(status int, retryAfter, body string) {
	res := response(status, body) //nolint:bodyclose // the client under test closes it
	if retryAfter != "" {
		res.Header.Set("Retry-After", retryAfter)
	}
	s.queued = append(s.queued, res) //nolint:bodyclose // the client under test closes it
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

// row is bar i in the format of klines.json, with made-up prices.
func row(i int) string {
	t := t0 + int64(i)*minute
	o := 100 + float64(i%20)/4
	return fmt.Sprintf(`[%d,"%.8f","%.8f","%.8f","%.8f","%.8f",%d,"%.8f",%d,"0.00000000","0.00000000","0"]`,
		t, o, o+1, o-1, o+0.5, 2.0, t+minute-1, 2*o, i%50+1)
}

// later is the bubble's clock moved to 2030, so the 2026 test bars have closed.
type later struct{}

func (later) Now() time.Time { return time.Now().AddDate(30, 0, 0).UTC() }

func newConnector(src http.RoundTripper, now time.Time, streamURL string) *binance.Connector {
	return binance.New(binance.Options{
		HTTPClient: &http.Client{Transport: src},
		StreamURL:  streamURL,
		Clock:      clock.NewFake(now),
	})
}

// streamScript is what streamServer expects and sends.
type streamScript struct {
	streams []string         // names the client must subscribe to, in order
	reply   func(id int) any // answer to each SUBSCRIBE; nil acknowledges it
	then    [][]byte         // sent once every stream is subscribed
}

type subscribeCall struct {
	params []string
	at     time.Time
}

// streamServer accepts one client on /stream, reads SUBSCRIBE messages until
// it has every stream, answering each, then sends the script's messages and
// waits for the client to leave. It returns the URL and the calls it got.
func streamServer(t *testing.T, sc streamScript) (string, func() []subscribeCall) {
	t.Helper()
	var (
		mu    sync.Mutex
		calls []subscribeCall
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stream" || r.URL.RawQuery != "" {
			http.Error(w, "want /stream without streams", http.StatusBadRequest)
			return
		}
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow() //nolint:errcheck // test server
		var got []string
		for len(got) < len(sc.streams) {
			_, data, err := c.Read(r.Context())
			if err != nil {
				return
			}
			var m struct {
				Method string   `json:"method"`
				Params []string `json:"params"`
				ID     int      `json:"id"`
			}
			if json.Unmarshal(data, &m) != nil || m.Method != "SUBSCRIBE" {
				return
			}
			mu.Lock()
			calls = append(calls, subscribeCall{m.Params, time.Now()})
			mu.Unlock()
			got = append(got, m.Params...)
			var answer any = map[string]any{"result": nil, "id": m.ID}
			if sc.reply != nil {
				answer = sc.reply(m.ID)
			}
			b, _ := json.Marshal(answer)
			if c.Write(r.Context(), websocket.MessageText, b) != nil {
				return
			}
		}
		if !slices.Equal(got, sc.streams) {
			return
		}
		for _, m := range sc.then {
			if c.Write(r.Context(), websocket.MessageText, m) != nil {
				return
			}
		}
		_, _, _ = c.Read(r.Context())
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http"), func() []subscribeCall {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(calls)
	}
}

func fixtureLines(t *testing.T) [][]byte {
	t.Helper()
	return bytes.Split(bytes.TrimSpace(fixture(t, "kline_stream.jsonl")), []byte("\n"))
}

var btc1m = marketdata.StreamSub{Symbol: "BTCUSDT", TF: marketdata.TF1m}

func TestConformance(t *testing.T) {
	connectortest.Run(t, connectortest.Fixture{
		New: func(t *testing.T) marketdata.Connector {
			url, _ := streamServer(t, streamScript{streams: []string{"btcusdt@kline_1m"}, then: fixtureLines(t)})
			return newConnector(&source{bars: 2500}, time.UnixMilli(t0+3000*minute), url)
		},
		// 2100 bars take three requests of at most 1000.
		History: marketdata.HistoryRequest{Symbol: "BTCUSDT", TF: marketdata.TF1m, From: t0, To: t0 + 2100*minute},
		Stream:  btc1m,
	})
}

func TestHistoryParsesTheKlinesFixtureAndLeavesOutTheFormingBar(t *testing.T) {
	src := &source{}
	src.queue(http.StatusOK, "", string(fixture(t, "klines.json")))
	// The third bar closes at t0+3m-1 ms, after this clock's time.
	c := newConnector(src, time.UnixMilli(t0+2*minute+30_000), "")

	got, err := c.History(t.Context(), marketdata.HistoryRequest{Symbol: "BTCUSDT", TF: marketdata.TF1m, From: t0, To: t0 + 3*minute})
	if err != nil {
		t.Fatal(err)
	}
	qv0, n0, qv1, n1 := 1256.25, int32(42), 806.0, int32(17)
	want := []marketdata.Candle{
		{TF: marketdata.TF1m, T: t0, O: 100, H: 101.5, L: 99.5, C: 101, V: 12.5, QV: &qv0, N: &n0},
		{TF: marketdata.TF1m, T: t0 + minute, O: 101, H: 102, L: 100.25, C: 100.75, V: 8, QV: &qv1, N: &n1},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("candles (-want +got):\n%s", diff)
	}
}

func TestHistoryPagesWithInclusiveEndTimes(t *testing.T) {
	src := &source{bars: 2500}
	c := newConnector(src, time.UnixMilli(t0+3000*minute), "")
	if _, err := c.History(t.Context(), marketdata.HistoryRequest{Symbol: "BTCUSDT", TF: marketdata.TF1m, From: t0, To: t0 + 2000*minute}); err != nil {
		t.Fatal(err)
	}
	end := t0 + 2000*minute - 1
	want := []string{
		fmt.Sprintf("endTime=%d&interval=1m&limit=1000&startTime=%d&symbol=BTCUSDT", end, t0),
		fmt.Sprintf("endTime=%d&interval=1m&limit=1000&startTime=%d&symbol=BTCUSDT", end, t0+999*minute+1),
		fmt.Sprintf("endTime=%d&interval=1m&limit=1000&startTime=%d&symbol=BTCUSDT", end, t0+1999*minute+1),
	}
	if diff := cmp.Diff(want, src.requests); diff != "" {
		t.Errorf("requests (-want +got):\n%s", diff)
	}
}

func TestA429PausesTheHostForRetryAfterThenRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := &source{bars: 10}
		src.queue(http.StatusTooManyRequests, "3", string(fixture(t, "error_429.json")))
		c := binance.New(binance.Options{HTTPClient: &http.Client{Transport: src}, Clock: later{}})
		req := marketdata.HistoryRequest{Symbol: "BTCUSDT", TF: marketdata.TF1m, From: t0, To: t0 + 10*minute}

		start := time.Now()
		got, err := c.History(t.Context(), req)
		if err != nil || len(got) != 10 {
			t.Fatalf("History = %d bars, %v; want 10 after the retry", len(got), err)
		}
		if waited := time.Since(start); waited != 3*time.Second {
			t.Errorf("waited %v, want the 3s Retry-After", waited)
		}
		if len(src.requests) != 2 {
			t.Errorf("sent %d requests, want 2", len(src.requests))
		}
	})
}

func TestA418StopsEveryRequestUntilTheBanEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := &source{bars: 10}
		src.queue(http.StatusTeapot, "120", string(fixture(t, "error_418.json")))
		c := binance.New(binance.Options{HTTPClient: &http.Client{Transport: src}, Clock: later{}})
		req := marketdata.HistoryRequest{Symbol: "BTCUSDT", TF: marketdata.TF1m, From: t0, To: t0 + 10*minute}

		_, err := c.History(t.Context(), req)
		if !errors.Is(err, marketdata.ErrBanned) || !strings.Contains(err.Error(), "IP banned until") {
			t.Fatalf("History = %v, want ErrBanned with Binance's message", err)
		}
		if _, err := c.History(t.Context(), req); !errors.Is(err, marketdata.ErrBanned) || len(src.requests) != 1 {
			t.Fatalf("History during the ban = %v after %d requests, want ErrBanned without sending", err, len(src.requests))
		}
		time.Sleep(2 * time.Minute)
		if got, err := c.History(t.Context(), req); err != nil || len(got) != 10 {
			t.Fatalf("History after the ban = %d bars, %v", len(got), err)
		}
	})
}

func TestUsedWeightHeaderHoldsTheNextRequestBack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(time.Until(time.Now().Truncate(time.Minute).Add(time.Minute + 10*time.Second)))
		src := &source{bars: 2000, usedHdr: "5999"} // another client on this IP used most of the budget
		c := binance.New(binance.Options{HTTPClient: &http.Client{Transport: src}, Clock: later{}})

		start := time.Now()
		if _, err := c.History(t.Context(), marketdata.HistoryRequest{Symbol: "BTCUSDT", TF: marketdata.TF1m, From: t0, To: t0 + 2000*minute}); err != nil {
			t.Fatal(err)
		}
		if waited := time.Since(start); waited != 50*time.Second {
			t.Errorf("waited %v, want the 50s left in the minute", waited)
		}
	})
}

func TestHistoryRejectsAMalformedKline(t *testing.T) {
	for _, body := range []string{
		`[[1767225600000,"NaN","1","1","1","1",1767225659999,"1",1,"0","0","0"]]`,
		`[[1767225600000,"1","1","1","1","1"]]`,
		`{"bars":[]}`,
	} {
		src := &source{}
		src.queue(http.StatusOK, "", body)
		c := newConnector(src, time.UnixMilli(t0+10*minute), "")
		if got, err := c.History(t.Context(), marketdata.HistoryRequest{Symbol: "BTCUSDT", TF: marketdata.TF1m, From: t0, To: t0 + minute}); err == nil {
			t.Errorf("History of %s = %v, want an error", body, got)
		}
	}
}

type sinkFunc func(marketdata.CandleUpdate)

func (f sinkFunc) Candle(u marketdata.CandleUpdate) { f(u) }

func TestStreamTranslatesTheFixtureInOrder(t *testing.T) {
	url, _ := streamServer(t, streamScript{streams: []string{"btcusdt@kline_1m", "ethusdt@kline_1h"}, then: fixtureLines(t)})
	c := newConnector(&source{}, time.UnixMilli(t0), url)
	updates := make(chan marketdata.CandleUpdate, 10)
	done := make(chan error, 1)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	go func() {
		done <- c.Stream(ctx, []marketdata.StreamSub{btc1m, {Symbol: "ETHUSDT", TF: marketdata.TF1h}}, sinkFunc(func(u marketdata.CandleUpdate) { updates <- u }))
	}()
	var got []marketdata.CandleUpdate
	for range 4 {
		select {
		case u := <-updates:
			got = append(got, u)
		case err := <-done:
			t.Fatalf("Stream returned %v after %d updates", err, len(got))
		case <-time.After(10 * time.Second):
			t.Fatalf("got %d updates in 10s, want 4", len(got))
		}
	}
	stop()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("Stream after cancel = %v, want context.Canceled", err)
	}
	want := []marketdata.CandleUpdate{
		{Sub: btc1m, T: t0, O: 100, H: 100.5, L: 100, C: 100.5, V: 1.25},
		{Sub: btc1m, T: t0, O: 100, H: 100.5, L: 99.5, C: 99.75, V: 3},
		{Sub: btc1m, T: t0, O: 100, H: 100.5, L: 99.5, C: 100.25, V: 4, Closed: true},
		{Sub: btc1m, T: t0 + minute, O: 100.25, H: 100.25, L: 100.25, C: 100.25, V: 0.1},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("updates (-want +got):\n%s", diff)
	}
}

func TestFixturesAreValidJSON(t *testing.T) {
	for _, name := range []string{"klines.json", "error_429.json", "error_418.json"} {
		if !jsontext.Value(fixture(t, name)).IsValid() {
			t.Errorf("%s is not valid JSON", name)
		}
	}
	sc := bufio.NewScanner(bytes.NewReader(fixture(t, "kline_stream.jsonl")))
	for sc.Scan() {
		if !jsontext.Value(sc.Bytes()).IsValid() {
			t.Errorf("kline_stream.jsonl has an invalid line: %s", sc.Bytes())
		}
	}
}

// runStream runs Stream until want updates arrive or it returns, then stops it
// and returns the updates and Stream's error.
func runStream(t *testing.T, c *binance.Connector, subs []marketdata.StreamSub, want int) ([]marketdata.CandleUpdate, error) {
	t.Helper()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	updates := make(chan marketdata.CandleUpdate, want)
	done := make(chan error, 1)
	go func() { done <- c.Stream(ctx, subs, sinkFunc(func(u marketdata.CandleUpdate) { updates <- u })) }()
	var got []marketdata.CandleUpdate
	for len(got) < want {
		select {
		case u := <-updates:
			got = append(got, u)
		case err := <-done:
			return got, err
		case <-time.After(10 * time.Second):
			t.Fatalf("got %d updates in 10s, want %d", len(got), want)
		}
	}
	stop()
	return got, <-done
}

func TestStreamSubscribesTheMaximumStreamsInPacedBatches(t *testing.T) {
	subs := make([]marketdata.StreamSub, 1024)
	names := make([]string, len(subs))
	for i := range subs {
		subs[i] = marketdata.StreamSub{Symbol: fmt.Sprintf("SYM%04dUSDT", i), TF: marketdata.TF1m}
		names[i] = fmt.Sprintf("sym%04dusdt@kline_1m", i)
	}
	last := fmt.Sprintf(`{"stream":%q,"data":{"e":"kline","k":{"t":%d,"o":"1","h":"1","l":"1","c":"1","v":"0","x":false}}}`, names[1023], t0)
	url, calls := streamServer(t, streamScript{streams: names, then: [][]byte{[]byte(last)}})
	c := newConnector(&source{}, time.UnixMilli(t0), url)

	got, err := runStream(t, c, subs, 1)
	if !errors.Is(err, context.Canceled) || len(got) != 1 || got[0].Sub != subs[1023] {
		t.Fatalf("Stream = %+v, %v; want the last stream's update, then context.Canceled", got, err)
	}
	sent := calls()
	sizes := make([]int, 0, len(sent))
	for i, call := range sent {
		sizes = append(sizes, len(call.params))
		// A timer never fires early, so a gap shorter than the pace is a bug.
		if gap := call.at.Sub(sent[max(i-1, 0)].at); i > 0 && gap < 240*time.Millisecond {
			t.Errorf("SUBSCRIBE %d came %v after the previous one, want at most 4 a second", i+1, gap)
		}
	}
	if !slices.Equal(sizes, []int{200, 200, 200, 200, 200, 24}) {
		t.Errorf("SUBSCRIBE batches of %v names, want [200 200 200 200 200 24]", sizes)
	}
	if err := c.Stream(t.Context(), make([]marketdata.StreamSub, 1025), nil); err == nil {
		t.Error("Stream of 1025 subscriptions succeeded")
	}
}

func TestStreamEndsWhenASubscribeFails(t *testing.T) {
	url, _ := streamServer(t, streamScript{streams: []string{"btcusdt@kline_1m"}, reply: func(id int) any {
		return map[string]any{"error": map[string]any{"code": 2, "msg": "Invalid request: unknown stream"}, "id": id}
	}})
	_, err := runStream(t, newConnector(&source{}, time.UnixMilli(t0), url), []marketdata.StreamSub{btc1m}, 1)
	if err == nil || !strings.Contains(err.Error(), "Invalid request: unknown stream (code 2)") {
		t.Errorf("Stream = %v, want Binance's subscribe error", err)
	}
}
