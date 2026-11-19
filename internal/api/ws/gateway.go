// SPDX-License-Identifier: AGPL-3.0-only

package ws

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/amanuelmr/wickframe/internal/bus"
	"github.com/amanuelmr/wickframe/internal/marketdata"
)

// Protocol is the WebSocket sub-protocol the gateway speaks.
const Protocol = "wickframe.v1"

const (
	maxMessageSize = 64 << 10
	writeTimeout   = 10 * time.Second
)

// Gateway serves the candles channel of the protocol from a hub of candle
// updates.
type Gateway struct {
	hub         *bus.Hub[marketdata.CandleUpdate]
	instruments map[string]int32
	lastClosedT func(context.Context, int32, marketdata.Timeframe) (int64, error)
	conns       sync.WaitGroup
}

// New returns a Gateway that resolves instrument keys such as "fake:DEMO" to
// instrument IDs with instruments, and reads the open time of a series' newest
// stored candle with lastClosedT.
func New(
	hub *bus.Hub[marketdata.CandleUpdate],
	instruments map[string]int32,
	lastClosedT func(ctx context.Context, instrumentID int32, tf marketdata.Timeframe) (int64, error),
) *Gateway {
	return &Gateway{hub: hub, instruments: instruments, lastClosedT: lastClosedT}
}

// ServeHTTP upgrades the request and serves the connection until the client
// leaves or the request's context ends, which closes it with 1001.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Added before the upgrade, while http.Server.Shutdown still waits for the
	// request, so it happens before Wait.
	g.conns.Add(1)
	defer g.conns.Done()
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{Protocol}})
	if err != nil {
		return
	}
	defer c.CloseNow() //nolint:errcheck // the connection is done either way
	if c.Subprotocol() != Protocol {
		_ = c.Close(websocket.StatusPolicyViolation, "sub-protocol "+Protocol+" required")
		return
	}
	c.SetReadLimit(maxMessageSize)
	g.serve(r.Context(), c)
}

// Wait blocks until every connection has ended. http.Server.Shutdown does not
// wait for WebSockets, so call Wait after it.
func (g *Gateway) Wait() { g.conns.Wait() }

func (g *Gateway) serve(ctx context.Context, c *websocket.Conn) {
	// A Read whose context ends closes the connection without a close code, so
	// reads use a detached context and shutdown closes with 1001 here.
	closed := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(closed)
		_ = c.Close(websocket.StatusGoingAway, "server shutting down")
	})
	defer func() {
		if !stop() {
			<-closed
		}
	}()
	conn, cancel := context.WithCancel(context.WithoutCancel(ctx))
	var writers sync.WaitGroup
	defer writers.Wait()
	defer cancel()

	for {
		_, data, err := c.Read(conn)
		if err != nil {
			return
		}
		var m clientMessage
		if err := json.Unmarshal(data, &m); err != nil {
			_ = send(conn, c, problem{"error", "", "", "invalid_message", "message is not valid JSON"})
			continue
		}
		if m.Type != "subscribe" || m.Channel != "candles" || m.Sub == "" {
			_ = send(conn, c, problem{"error", m.ID, m.Sub, "invalid_message", "only subscribe to the candles channel is supported"})
			continue
		}
		id, ok := g.instruments[m.Instrument]
		if !ok {
			_ = send(conn, c, problem{"error", m.ID, m.Sub, "unknown_instrument", "unknown instrument " + m.Instrument})
			continue
		}
		tf, ok := marketdata.ParseTimeframe(m.TF)
		if !ok {
			_ = send(conn, c, problem{"error", m.ID, m.Sub, "invalid_message", "unknown timeframe " + m.TF})
			continue
		}

		// Subscribed before reading the store, so a bar that closes in between
		// still reaches the client.
		s := g.hub.Subscribe(marketdata.CandleTopic(id, tf))
		last, err := g.lastClosedT(conn, id, tf)
		if err != nil {
			s.Close()
			_ = c.Close(websocket.StatusInternalError, "could not read stored candles")
			return
		}
		if send(conn, c, subscribed{"subscribed", m.Sub, 0, last}) != nil {
			s.Close()
			return
		}
		writers.Go(func() {
			defer s.Close()
			for seq := int64(1); ; seq++ {
				u, err := s.Next(conn)
				if errors.Is(err, bus.ErrLagging) {
					_ = c.Close(websocket.StatusTryAgainLater, "client fell behind")
				}
				if err != nil || send(conn, c, newCandle(m.Sub, seq, u)) != nil {
					return
				}
			}
		})
	}
}

func send(ctx context.Context, c *websocket.Conn, msg any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err //nolint:wrapcheck // our own message types always marshal
	}
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return c.Write(ctx, websocket.MessageText, b) //nolint:wrapcheck // callers only stop on it
}

type clientMessage struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Sub        string `json:"sub"`
	Channel    string `json:"channel"`
	Instrument string `json:"instrument"`
	TF         string `json:"tf"`
}

type subscribed struct {
	Type        string `json:"type"`
	Sub         string `json:"sub"`
	Seq         int64  `json:"seq"`
	LastClosedT int64  `json:"lastClosedT"`
}

type candle struct {
	Type  string  `json:"type"`
	Sub   string  `json:"sub"`
	Seq   int64   `json:"seq"`
	T     int64   `json:"t"`
	O     float64 `json:"o"`
	H     float64 `json:"h"`
	L     float64 `json:"l"`
	C     float64 `json:"c"`
	V     float64 `json:"v"`
	State string  `json:"state"`
	Rev   int16   `json:"rev"`
}

func newCandle(sub string, seq int64, u marketdata.CandleUpdate) candle {
	state := "forming"
	if u.Closed {
		state = "closed"
	}
	return candle{"candle", sub, seq, u.T, u.O, u.H, u.L, u.C, u.V, state, u.Rev}
}

type problem struct {
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	Sub     string `json:"sub,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
