// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/amanuelmr/wickframe/internal/api/ws"
	"github.com/amanuelmr/wickframe/internal/bus"
	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/marketdata/connector/fake"
	"github.com/amanuelmr/wickframe/internal/platform/clock"
	"github.com/amanuelmr/wickframe/internal/platform/supervise"
	"github.com/amanuelmr/wickframe/internal/webui"
)

// DefaultAddr is where serve listens unless told otherwise.
const DefaultAddr = "127.0.0.1:7317"

// The fake connector's demo market is the only instrument for now.
const (
	demoKey = "fake:DEMO"
	demoID  = 1
)

const shutdownTimeout = 30 * time.Second

// Run serves the web UI and the WebSocket API on ln until ctx is done, with
// live candles from the fake connector's demo market.
func Run(ctx context.Context, ln net.Listener, log *slog.Logger) error {
	clk := clock.System{}
	hub := bus.New[marketdata.CandleUpdate]()
	connector := fake.New(fake.Demo(), clk)
	gateway := ws.New(hub, map[string]int32{demoKey: demoID})

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/ws", gateway)
	mux.Handle("GET /", webui.Handler())

	subs := []marketdata.StreamSub{{Symbol: "DEMO", TF: marketdata.TF1m}}
	return supervise.New(log, clk,
		supervise.Child{Name: "connector", Run: func(ctx context.Context) error {
			return connector.Stream(ctx, subs, publisher{hub, demoID}) //nolint:wrapcheck // the supervisor logs it with the child's name
		}},
		supervise.Child{Name: "http", Run: func(ctx context.Context) error {
			return serveHTTP(ctx, ln, mux, gateway)
		}},
	).Run(ctx)
}

type publisher struct {
	hub          *bus.Hub[marketdata.CandleUpdate]
	instrumentID int32
}

func (p publisher) Candle(u marketdata.CandleUpdate) {
	p.hub.Publish(marketdata.CandleTopic(p.instrumentID, u.Sub.TF), u)
}

func serveHTTP(ctx context.Context, ln net.Listener, h http.Handler, gateway *ws.Gateway) error {
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		return fmt.Errorf("serve http: %w", err)
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	err := srv.Shutdown(sctx)
	gateway.Wait()
	if err != nil {
		return fmt.Errorf("shut down http: %w", err)
	}
	return nil
}
