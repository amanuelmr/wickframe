// SPDX-License-Identifier: AGPL-3.0-only

// Package binance is the Binance spot connector (04-market-data §3). It reads
// public klines from Binance's market-data-only hosts, which need no account.
package binance

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/marketdata/feed"
	"github.com/amanuelmr/wickframe/internal/platform/clock"
	"github.com/amanuelmr/wickframe/internal/platform/retry"
)

// Market-data-only hosts, verified 2026-11-21.
const (
	DefaultRESTURL   = "https://data-api.binance.vision"
	DefaultStreamURL = "wss://data-stream.binance.vision"
)

var limits = marketdata.Limits{
	MaxBarsPerRequest:        1000,
	RequestWeight:            6000,
	RequestWeightWindow:      time.Minute,
	MaxStreamsPerConnection:  1024,
	ConnectionAttempts:       300,
	ConnectionAttemptsWindow: 5 * time.Minute,
	MaxConnectionLifetime:    24 * time.Hour,
}

var intervals = map[marketdata.Timeframe]string{
	marketdata.TF1m: "1m", marketdata.TF3m: "3m", marketdata.TF5m: "5m", marketdata.TF15m: "15m",
	marketdata.TF30m: "30m", marketdata.TF1h: "1h", marketdata.TF2h: "2h", marketdata.TF4h: "4h",
	marketdata.TF6h: "6h", marketdata.TF12h: "12h", marketdata.TF1D: "1d", marketdata.TF1W: "1w",
	marketdata.TF1M: "1M",
}

// Options configures a Connector. Zero fields take the defaults.
type Options struct {
	RESTURL, StreamURL string
	HTTPClient         *http.Client // for REST requests
	Clock              clock.Clock
	// Limiters shares rate limits with other connectors on the same hosts.
	Limiters *marketdata.HostLimiters
}

// Connector reads Binance spot klines.
type Connector struct {
	rest, stream string
	http         *http.Client
	clk          clock.Clock
	weight       *marketdata.Limiter // REST request weight per IP
	dials        *marketdata.Limiter // stream connection attempts per IP
}

var _ marketdata.Connector = (*Connector)(nil)

// New returns a Connector.
func New(o Options) *Connector {
	c := &Connector{rest: o.RESTURL, stream: o.StreamURL, http: o.HTTPClient, clk: o.Clock}
	if c.rest == "" {
		c.rest = DefaultRESTURL
	}
	if c.stream == "" {
		c.stream = DefaultStreamURL
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: 30 * time.Second}
	}
	if c.clk == nil {
		c.clk = clock.System{}
	}
	hl := o.Limiters
	if hl == nil {
		hl = marketdata.NewHostLimiters(c.clk)
	}
	c.weight = hl.Get(host(c.rest)+" weight", limits.RequestWeight, limits.RequestWeightWindow)
	c.dials = hl.Get(host(c.stream)+" connections", limits.ConnectionAttempts, limits.ConnectionAttemptsWindow)
	return c
}

func host(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return u.Host
	}
	return rawURL
}

// Info implements marketdata.Connector.
func (c *Connector) Info() marketdata.ConnectorInfo {
	tfs := make([]marketdata.Timeframe, 0, len(intervals))
	for tf := marketdata.TF1m; tf <= marketdata.TF1M; tf++ {
		if _, ok := intervals[tf]; ok {
			tfs = append(tfs, tf)
		}
	}
	return marketdata.ConnectorInfo{
		ID: "binance", Name: "Binance", Version: "1.0.0",
		TermsURL:         "https://developers.binance.com/en/docs/products/spot/PROD-TERMS-OF-USE",
		Auth:             marketdata.AuthNone,
		Freshness:        marketdata.FreshnessRealtime,
		NativeTimeframes: tfs,
		StreamTimeframes: tfs,
		Limits:           limits,
		SettingsSchema:   `{"type":"object","additionalProperties":false}`,
	}
}

const (
	klinesWeight    = 2
	maxAttempts     = 5
	maxResponseSize = 4 << 20
	defaultBan      = 2 * time.Minute // Binance's shortest ban
)

var limitBackoff = retry.Backoff{Base: time.Second, Max: 5 * time.Minute}

// History implements marketdata.Connector.
func (c *Connector) History(ctx context.Context, req marketdata.HistoryRequest) ([]marketdata.Candle, error) {
	interval, ok := intervals[req.TF]
	if !ok {
		return nil, fmt.Errorf("binance history: timeframe %v is not available", req.TF)
	}
	var out []marketdata.Candle
	for from := req.From; from < req.To; {
		q := url.Values{
			"symbol": {req.Symbol}, "interval": {interval},
			"startTime": {strconv.FormatInt(from, 10)},
			// endTime includes a bar opening at it; the request's To does not.
			"endTime": {strconv.FormatInt(req.To-1, 10)},
			"limit":   {strconv.Itoa(limits.MaxBarsPerRequest)},
		}
		body, err := c.get(ctx, "/api/v3/klines", q, klinesWeight)
		if err != nil {
			return nil, fmt.Errorf("binance history %s %s: %w", req.Symbol, interval, err)
		}
		var page []kline
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("binance history %s %s: %w", req.Symbol, interval, err)
		}
		now := c.clk.Now().UnixMilli()
		for _, k := range page {
			if k.closeT >= now {
				break // the forming bar
			}
			out = append(out, k.candle(req.TF))
		}
		if len(page) < limits.MaxBarsPerRequest {
			break
		}
		from = page[len(page)-1].t + 1
	}
	return out, nil
}

// get sends a weighted GET to the REST host. A 429 pauses every request to
// the host for its Retry-After, then this one is sent again; a 418 refuses
// every request until the ban ends.
func (c *Connector) get(ctx context.Context, path string, q url.Values, weight int) ([]byte, error) {
	for attempt := 1; ; attempt++ {
		if err := c.weight.Reserve(ctx, weight); err != nil {
			return nil, err //nolint:wrapcheck // the caller adds the request
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.rest+path+"?"+q.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send request: %w", err)
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
		_ = res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		if used, err := strconv.Atoi(res.Header.Get("X-Mbx-Used-Weight-1m")); err == nil {
			c.weight.Used(used)
		}
		switch res.StatusCode {
		case http.StatusOK:
			return body, nil
		case http.StatusTooManyRequests:
			c.weight.Pause(retryAfter(res, limitBackoff.Delay(attempt-1)))
			if attempt == maxAttempts {
				return nil, fmt.Errorf("rate limited %d times: %s", attempt, apiError(body))
			}
		case http.StatusTeapot:
			c.weight.Ban(retryAfter(res, defaultBan))
			return nil, fmt.Errorf("%w: %s", marketdata.ErrBanned, apiError(body))
		default:
			return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, apiError(body))
		}
	}
}

func retryAfter(res *http.Response, fallback time.Duration) time.Duration {
	if s, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && s >= 0 {
		return time.Duration(s) * time.Second
	}
	return fallback
}

// apiError returns the message of Binance's {"code":…,"msg":…} error body.
func apiError(body []byte) string {
	var e struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if json.Unmarshal(body, &e) != nil || e.Msg == "" {
		return fmt.Sprintf("%.200q", body)
	}
	return fmt.Sprintf("%s (code %d)", e.Msg, e.Code)
}

// kline is one row of /api/v3/klines: [open time, open, high, low, close,
// volume, close time, quote volume, trades, …].
type kline struct {
	t, closeT  int64
	o, h, l, c float64
	v, qv      float64
	n          int32
}

func (k *kline) UnmarshalJSON(b []byte) error {
	var row []jsontext.Value
	if err := json.Unmarshal(b, &row); err != nil {
		return err //nolint:wrapcheck // json adds the position
	}
	if len(row) < 9 {
		return fmt.Errorf("kline has %d fields, want at least 9", len(row))
	}
	var nums [6]string
	err := errors.Join(
		json.Unmarshal(row[0], &k.t), json.Unmarshal(row[6], &k.closeT), json.Unmarshal(row[8], &k.n),
		json.Unmarshal(row[1], &nums[0]), json.Unmarshal(row[2], &nums[1]), json.Unmarshal(row[3], &nums[2]),
		json.Unmarshal(row[4], &nums[3]), json.Unmarshal(row[5], &nums[4]), json.Unmarshal(row[7], &nums[5]),
	)
	if err != nil {
		return fmt.Errorf("kline: %w", err)
	}
	return parseDecimals(nums[:], &k.o, &k.h, &k.l, &k.c, &k.v, &k.qv)
}

func (k kline) candle(tf marketdata.Timeframe) marketdata.Candle {
	qv, n := k.qv, k.n
	return marketdata.Candle{TF: tf, T: k.t, O: k.o, H: k.h, L: k.l, C: k.c, V: k.v, QV: &qv, N: &n}
}

func parseDecimals(s []string, dst ...*float64) error {
	for i, d := range dst {
		v, err := feed.ParseDecimal(s[i])
		if err != nil {
			return err //nolint:wrapcheck // feed names the value
		}
		*d = v
	}
	return nil
}
