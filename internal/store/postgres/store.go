// SPDX-License-Identifier: AGPL-3.0-only

// Package postgres is the PostgreSQL store adapter for server mode (ADR-0004,
// 08-storage §4). It implements the domain repository ports with
// sqlc-generated queries on pgx.
package postgres

import (
	"context"
	"fmt"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/amanuelmr/wickframe/internal/store/postgres/db"
	"github.com/amanuelmr/wickframe/migrations"
)

// Store is a PostgreSQL connection pool.
type Store struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// Open connects to the database at dsn. Call Migrate before use.
func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	cfg.MaxConns = int32(min(4*runtime.GOMAXPROCS(0), 32)) //nolint:gosec // at most 32
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "30s"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	return &Store{pool: pool, q: db.New(pool)}, nil
}

// Migrate applies pending migrations while holding an advisory lock, so
// processes starting together do not migrate concurrently.
func (s *Store) Migrate(ctx context.Context) error {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("postgres migration lock: %w", err)
	}
	sqlDB := stdlib.OpenDBFromPool(s.pool)
	defer func() { _ = sqlDB.Close() }()
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.Postgres, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("postgres migrations: %w", err)
	}
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("postgres migrate: %w", err)
	}
	return nil
}

// Close closes the pool.
func (s *Store) Close() {
	s.pool.Close()
}
