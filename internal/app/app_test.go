// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/amanuelmr/wickframe/internal/api/ws"
	"github.com/amanuelmr/wickframe/internal/bus"
	"github.com/amanuelmr/wickframe/internal/marketdata"
)

func TestServeStreamsDemoCandlesAndShutsDown(t *testing.T) {
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, ln, slog.New(slog.DiscardHandler)) }()
	addr := ln.Addr().String()

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+addr+"/", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", res.StatusCode)
	}

	c, _, err := websocket.Dial(t.Context(), "ws://"+addr+"/api/v1/ws", &websocket.DialOptions{Subprotocols: []string{ws.Protocol}}) //nolint:bodyclose // closed by the library on success
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.CloseNow() //nolint:errcheck // already closed by the server when the test passes
	msg := `{"type":"subscribe","sub":"c1","channel":"candles","instrument":"fake:DEMO","tf":"1m"}`
	if err := c.Write(t.Context(), websocket.MessageText, []byte(msg)); err != nil {
		t.Fatalf("write: %v", err)
	}
	for _, want := range []string{"subscribed", "candle"} {
		var got struct {
			Type  string `json:"type"`
			Sub   string `json:"sub"`
			Seq   int64  `json:"seq"`
			State string `json:"state"`
		}
		_, data, err := c.Read(t.Context())
		if err != nil {
			t.Fatalf("read %s: %v", want, err)
		}
		if err := json.Unmarshal(data, &got); err != nil || got.Type != want || got.Sub != "c1" {
			t.Fatalf("got %s, want a %s message for c1", data, want)
		}
		if want == "candle" && (got.Seq != 1 || got.State != "forming") {
			t.Fatalf("got %s, want the first forming update", data)
		}
	}

	cancel()
	if _, _, err := c.Read(t.Context()); websocket.CloseStatus(err) != websocket.StatusGoingAway {
		t.Errorf("connection ended with %v, want close code 1001", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return within 5 s of cancellation")
	}
}

func TestPublisherDropsInvalidUpdatesWithAWarning(t *testing.T) {
	hub := bus.New[marketdata.CandleUpdate]()
	s := hub.Subscribe(marketdata.CandleTopic(demoID, marketdata.TF1m))
	defer s.Close()
	var logs bytes.Buffer
	p := publisher{hub, demoID, slog.New(slog.NewTextHandler(&logs, nil))}
	sub := marketdata.StreamSub{Symbol: "DEMO", TF: marketdata.TF1m}

	p.Candle(marketdata.CandleUpdate{Sub: sub, T: 0, O: 1, H: 2, L: 1, C: 2, Closed: true})
	p.Candle(marketdata.CandleUpdate{Sub: sub, T: 60_000, O: 2, H: 1, L: 1, C: 2}) // high below open
	p.Candle(marketdata.CandleUpdate{Sub: sub, T: 60_000, O: 2, H: 3, L: 2, C: 3})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var got []int64
	for {
		u, err := s.Next(ctx)
		if err != nil {
			break
		}
		got = append(got, u.T)
	}
	if want := []int64{0, 60_000}; !slices.Equal(got, want) {
		t.Errorf("published bar times %v, want %v", got, want)
	}
	if !strings.Contains(logs.String(), "level=WARN msg=\"dropped an invalid candle update\" err=\"high and low do not contain open and close\"") {
		t.Errorf("missing the warning for the invalid update, logs:\n%s", logs.String())
	}
}
