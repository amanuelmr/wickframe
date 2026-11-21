// SPDX-License-Identifier: AGPL-3.0-only

package binance

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/amanuelmr/wickframe/internal/marketdata"
)

// Binance pings every 20 s and kline streams push every 2 s, so a connection
// silent for twice the ping interval is dead.
const silenceTimeout = 40 * time.Second

// Stream implements marketdata.Connector. It uses one connection for all subs.
func (c *Connector) Stream(ctx context.Context, subs []marketdata.StreamSub, sink marketdata.StreamSink) error {
	if len(subs) > limits.MaxStreamsPerConnection {
		return fmt.Errorf("binance stream: %d subscriptions, at most %d per connection", len(subs), limits.MaxStreamsPerConnection)
	}
	byName := make(map[string]marketdata.StreamSub, len(subs))
	names := make([]string, 0, len(subs))
	for _, s := range subs {
		interval, ok := intervals[s.TF]
		if !ok {
			return fmt.Errorf("binance stream: timeframe %v is not available", s.TF)
		}
		name := strings.ToLower(s.Symbol) + "@kline_" + interval
		byName[name] = s
		names = append(names, name)
	}
	if err := c.dials.Reserve(ctx, 1); err != nil {
		return fmt.Errorf("binance stream: %w", err)
	}
	conn, _, err := websocket.Dial(ctx, c.stream+"/stream?streams="+strings.Join(names, "/"), nil) //nolint:bodyclose // closed by the library on success
	if err != nil {
		return fmt.Errorf("binance stream: %w", err)
	}
	defer conn.CloseNow() //nolint:errcheck // the stream is over either way
	conn.SetReadLimit(1 << 20)

	for {
		rctx, cancel := context.WithTimeout(ctx, silenceTimeout)
		_, data, err := conn.Read(rctx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				err = ctx.Err()
			}
			return fmt.Errorf("binance stream: %w", err)
		}
		var msg streamMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("binance stream: %w", err)
		}
		sub, ok := byName[msg.Stream]
		if !ok || msg.Data.Event != "kline" {
			continue
		}
		u, err := msg.Data.K.update(sub)
		if err != nil {
			return fmt.Errorf("binance stream %s: %w", msg.Stream, err)
		}
		sink.Candle(u)
	}
}

// streamMessage is a combined-stream message carrying a kline event.
type streamMessage struct {
	Stream string `json:"stream"`
	Data   struct {
		Event string      `json:"e"`
		K     klineUpdate `json:"k"`
	} `json:"data"`
}

type klineUpdate struct {
	T      int64  `json:"t"`
	O      string `json:"o"`
	H      string `json:"h"`
	L      string `json:"l"`
	C      string `json:"c"`
	V      string `json:"v"`
	Closed bool   `json:"x"`
}

func (k klineUpdate) update(sub marketdata.StreamSub) (marketdata.CandleUpdate, error) {
	u := marketdata.CandleUpdate{Sub: sub, T: k.T, Closed: k.Closed}
	err := parseDecimals([]string{k.O, k.H, k.L, k.C, k.V}, &u.O, &u.H, &u.L, &u.C, &u.V)
	return u, err
}
