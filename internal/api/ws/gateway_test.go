// SPDX-License-Identifier: AGPL-3.0-only

package ws

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"

	"github.com/amanuelmr/wickframe/internal/bus"
	"github.com/amanuelmr/wickframe/internal/marketdata"
)

var demo = marketdata.CandleTopic(1, marketdata.TF1m)

// start serves a Gateway whose request contexts end when stop is called.
func start(t *testing.T) (hub *bus.Hub[marketdata.CandleUpdate], url string, stop func()) {
	t.Helper()
	hub = bus.New[marketdata.CandleUpdate]()
	g := New(hub, map[string]int32{"fake:DEMO": 1, "fake:BROKEN": 2}, lastClosedT)
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	srv := httptest.NewUnstartedServer(g)
	srv.Config.BaseContext = func(net.Listener) context.Context { return ctx }
	srv.Start()
	t.Cleanup(func() {
		cancel()
		srv.Close()
		g.Wait()
	})
	return hub, "ws" + strings.TrimPrefix(srv.URL, "http"), cancel
}

// lastClosedT is a store holding the demo series up to 2 minutes, which fails
// for any other series.
func lastClosedT(_ context.Context, id int32, tf marketdata.Timeframe) (int64, error) {
	if id != 1 || tf != marketdata.TF1m {
		return 0, errors.New("disk I/O error")
	}
	return 120_000, nil
}

func dial(t *testing.T, url string, protocols ...string) *websocket.Conn {
	t.Helper()
	c, _, err := websocket.Dial(t.Context(), url, &websocket.DialOptions{Subprotocols: protocols}) //nolint:bodyclose // closed by the library on success
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.CloseNow() })
	return c
}

func write(t *testing.T, c *websocket.Conn, msg string) {
	t.Helper()
	if err := c.Write(t.Context(), websocket.MessageText, []byte(msg)); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func expect(t *testing.T, c *websocket.Conn, want string) {
	t.Helper()
	_, got, err := c.Read(t.Context())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestSubscribeRepliesThenStreamsCandles(t *testing.T) {
	hub, url, _ := start(t)
	hub.Publish(demo, marketdata.CandleUpdate{T: 60_000, O: 1, H: 3, L: 0.5, C: 2, V: 0})
	c := dial(t, url, Protocol)

	write(t, c, `{"type":"subscribe","id":"r1","sub":"c1","channel":"candles","instrument":"fake:DEMO","tf":"1m"}`)
	expect(t, c, `{"type":"subscribed","sub":"c1","seq":0,"lastClosedT":120000}`)
	expect(t, c, `{"type":"candle","sub":"c1","seq":1,"t":60000,"o":1,"h":3,"l":0.5,"c":2,"v":0,"state":"forming","rev":0}`)

	hub.Publish(demo, marketdata.CandleUpdate{T: 60_000, O: 1, H: 3, L: 0.5, C: 2.5, V: 7, Closed: true, Rev: 1})
	expect(t, c, `{"type":"candle","sub":"c1","seq":2,"t":60000,"o":1,"h":3,"l":0.5,"c":2.5,"v":7,"state":"closed","rev":1}`)
}

func TestInvalidMessagesGetAnErrorAndKeepTheConnection(t *testing.T) {
	_, url, _ := start(t)
	c := dial(t, url, Protocol)

	for _, tt := range []struct{ msg, want string }{
		{`not json`, `{"type":"error","code":"invalid_message","message":"message is not valid JSON"}`},
		{`{"type":"ping","id":"p"}`, `{"type":"error","id":"p","code":"invalid_message","message":"only subscribe to the candles channel is supported"}`},
		{`{"type":"subscribe","sub":"s","channel":"status"}`, `{"type":"error","sub":"s","code":"invalid_message","message":"only subscribe to the candles channel is supported"}`},
		{`{"type":"subscribe","sub":"s","channel":"candles","instrument":"x:Y","tf":"1m"}`, `{"type":"error","sub":"s","code":"unknown_instrument","message":"unknown instrument x:Y"}`},
		{`{"type":"subscribe","sub":"s","channel":"candles","instrument":"fake:DEMO","tf":"7m"}`, `{"type":"error","sub":"s","code":"invalid_message","message":"unknown timeframe 7m"}`},
	} {
		write(t, c, tt.msg)
		expect(t, c, tt.want)
	}
}

func TestConnectionWithoutTheSubprotocolIsClosedWith1008(t *testing.T) {
	_, url, _ := start(t)
	c := dial(t, url)

	_, _, err := c.Read(t.Context())
	if got := websocket.CloseStatus(err); got != websocket.StatusPolicyViolation {
		t.Fatalf("close status = %v (%v), want 1008", got, err)
	}
}

func TestShutdownClosesWith1001(t *testing.T) {
	_, url, stop := start(t)
	c := dial(t, url, Protocol)
	write(t, c, `{"type":"subscribe","sub":"c1","channel":"candles","instrument":"fake:DEMO","tf":"1m"}`)
	expect(t, c, `{"type":"subscribed","sub":"c1","seq":0,"lastClosedT":120000}`)

	stop()
	_, _, err := c.Read(t.Context())
	if got := websocket.CloseStatus(err); got != websocket.StatusGoingAway {
		t.Fatalf("close status = %v (%v), want 1001", got, err)
	}
}

func TestStoreFailureClosesWith1011(t *testing.T) {
	_, url, _ := start(t)
	c := dial(t, url, Protocol)
	write(t, c, `{"type":"subscribe","sub":"c1","channel":"candles","instrument":"fake:BROKEN","tf":"1m"}`)

	_, _, err := c.Read(t.Context())
	if got := websocket.CloseStatus(err); got != websocket.StatusInternalError {
		t.Fatalf("close status = %v (%v), want 1011", got, err)
	}
}

var _ http.Handler = (*Gateway)(nil)
