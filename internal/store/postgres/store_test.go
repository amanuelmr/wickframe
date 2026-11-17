// SPDX-License-Identifier: AGPL-3.0-only

//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/store/postgres"
	"github.com/amanuelmr/wickframe/internal/store/storetest"
)

// PostgreSQL 16 is the oldest supported server (ADR-0004).
const image = "postgres:16-alpine"

// startPostgres starts one server and returns a function that creates a fresh,
// empty database on it for each caller.
func startPostgres(t *testing.T) func(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	provider, err := testcontainers.NewDockerProvider()
	if err == nil {
		defer func() { _ = provider.Close() }()
		err = provider.Health(ctx)
	}
	if err != nil {
		t.Skipf("Docker is not available, so the PostgreSQL store tests are skipped (%v)", err)
	}
	ctr, err := tcpostgres.Run(ctx, image, tcpostgres.BasicWaitStrategies())
	testcontainers.CleanupContainer(t, ctr)
	if err != nil {
		t.Fatalf("start %s: %v", image, err)
	}
	admin, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	var n atomic.Int64
	return func(t *testing.T) string {
		t.Helper()
		name := fmt.Sprintf("test_%d", n.Add(1))
		conn, err := pgx.Connect(t.Context(), admin)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close(context.Background()) }()
		if _, err := conn.Exec(t.Context(), "CREATE DATABASE "+name); err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(admin)
		if err != nil {
			t.Fatal(err)
		}
		u.Path = "/" + name
		return u.String()
	}
}

func TestCandleContract(t *testing.T) {
	newDatabase := startPostgres(t)
	storetest.TestCandles(t, func(t *testing.T) marketdata.CandleRepository {
		s, err := postgres.Open(t.Context(), newDatabase(t))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(s.Close)
		if err := s.Migrate(t.Context()); err != nil {
			t.Fatal(err)
		}
		return s
	})
}
