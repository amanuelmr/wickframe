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

const (
	// Kline streams push every 2 s, so 10 s is five missed pushes: a dead
	// stream is caught well inside NFR-REL-02's 30 s.
	silenceTimeout = 10 * time.Second
	subscribeBatch = 200 // stream names per SUBSCRIBE message
	// Binance allows 5 incoming messages a second, pongs included.
	subscribePace = 250 * time.Millisecond
)

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
	conn, _, err := websocket.Dial(ctx, c.stream+"/stream", nil) //nolint:bodyclose // closed by the library on success
	if err != nil {
		return fmt.Errorf("binance stream: %w", err)
	}
	defer conn.CloseNow() //nolint:errcheck // the stream is over either way
	conn.SetReadLimit(1 << 20)
	if err := subscribe(ctx, conn, names); err != nil {
		return fmt.Errorf("binance stream: %w", err)
	}

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
		if msg.Error != nil {
			return fmt.Errorf("binance stream: subscribe %d: %s (code %d)", msg.ID, msg.Error.Msg, msg.Error.Code)
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

// subscribe sends SUBSCRIBE messages for names in paced batches. Binance
// answers each by its id, which Stream checks as it reads.
func subscribe(ctx context.Context, conn *websocket.Conn, names []string) error {
	for i := 0; i*subscribeBatch < len(names); i++ {
		if i > 0 {
			timer := time.NewTimer(subscribePace)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("subscribe: %w", ctx.Err())
			case <-timer.C:
			}
		}
		msg, err := json.Marshal(subscribeMessage{
			Method: "SUBSCRIBE", Params: names[i*subscribeBatch : min((i+1)*subscribeBatch, len(names))], ID: i + 1,
		})
		if err != nil {
			return fmt.Errorf("subscribe: %w", err)
		}
		if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
			return fmt.Errorf("subscribe: %w", err)
		}
	}
	return nil
}

type subscribeMessage struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int      `json:"id"`
}

// streamMessage is a combined-stream message carrying a kline event, or the
// answer to a SUBSCRIBE.
type streamMessage struct {
	ID    int `json:"id"`
	Error *struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	} `json:"error"`
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
