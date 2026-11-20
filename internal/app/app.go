// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/amanuelmr/wickframe/internal/api/httpapi"
	"github.com/amanuelmr/wickframe/internal/api/ws"
	"github.com/amanuelmr/wickframe/internal/bus"
	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/marketdata/connector/fake"
	"github.com/amanuelmr/wickframe/internal/platform/clock"
	"github.com/amanuelmr/wickframe/internal/platform/logx"
	"github.com/amanuelmr/wickframe/internal/platform/supervise"
	"github.com/amanuelmr/wickframe/internal/store/sqlite"
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
// live candles from the fake connector's demo market. Closed candles are stored
// in a SQLite database in dataDir.
func Run(ctx context.Context, ln net.Listener, dataDir string, log *slog.Logger) (err error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	store, err := sqlite.Open(ctx, filepath.Join(dataDir, "wickframe.db"))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, store.Close()) }()
	if err := store.Migrate(ctx); err != nil {
		return err
	}

	clk := clock.System{}
	hub := bus.New[marketdata.CandleUpdate]()
	persister := marketdata.NewPersister(store, hub, demoID, marketdata.TF1m)
	connector := fake.New(fake.Demo(), clk)
	instruments := map[string]int32{demoKey: demoID}
	gateway := ws.New(hub, instruments, store.LastCandleT)

	mux := http.NewServeMux()
	httpapi.New(mux, store, instruments, log)
	mux.Handle("GET /api/v1/ws", gateway)
	mux.Handle("GET /", webui.Handler())

	subs := []marketdata.StreamSub{{Symbol: "DEMO", TF: marketdata.TF1m}}
	return supervise.New(log, clk,
		supervise.Child{Name: "persister", Run: persister.Run},
		supervise.Child{Name: "connector", Run: func(ctx context.Context) error {
			return connector.Stream(ctx, subs, publisher{hub, demoID, log}) //nolint:wrapcheck // the supervisor logs it with the child's name
		}},
		supervise.Child{Name: "http", Run: func(ctx context.Context) error {
			return serveHTTP(ctx, ln, mux, gateway)
		}},
	).Run(ctx)
}

// publisher validates a connector's updates and publishes the valid ones.
type publisher struct {
	hub          *bus.Hub[marketdata.CandleUpdate]
	instrumentID int32
	log          *slog.Logger
}

func (p publisher) Candle(u marketdata.CandleUpdate) {
	if err := marketdata.ValidateCandle(u); err != nil {
		p.log.Warn("dropped an invalid candle update", logx.KeyErr, err, "symbol", u.Sub.Symbol, "tf", u.Sub.TF.String(), "update", u)
		return
	}
	topic := marketdata.CandleTopic(p.instrumentID, u.Sub.TF)
	if u.Closed {
		p.hub.Publish(topic, u)
	} else {
		p.hub.PublishConflatable(topic, u)
	}
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
