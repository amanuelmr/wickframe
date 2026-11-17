// SPDX-License-Identifier: AGPL-3.0-only

// Package sqlite is the SQLite store adapter for personal mode (ADR-0004,
// 08-storage §4). It implements the domain repository ports with sqlc-generated
// queries.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"github.com/amanuelmr/wickframe/internal/store/sqlite/db"
	"github.com/amanuelmr/wickframe/migrations"
)

// Write transactions start IMMEDIATE, so a transaction that reads before it
// writes cannot fail with SQLITE_BUSY.
const pragmas = "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)" +
	"&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_txlock=immediate"

// Store is a SQLite database with one writer connection and a pool of
// read-only connections.
type Store struct {
	write, read *sql.DB
	w, r        *db.Queries
}

// Open opens or creates the database file at path. Call Migrate before use.
func Open(ctx context.Context, path string) (*Store, error) {
	write, err := sql.Open("sqlite", path+"?"+pragmas)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	write.SetMaxOpenConns(1)
	read, err := sql.Open("sqlite", path+"?"+pragmas+"&_pragma=query_only(1)")
	if err != nil {
		_ = write.Close()
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	read.SetMaxOpenConns(max(4, runtime.GOMAXPROCS(0)))

	s := &Store{write: write, read: read, w: db.New(write), r: db.New(read)}
	// The first connection switches the file to WAL; the readers rely on it.
	if err := write.PingContext(ctx); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	return s, nil
}

// Migrate applies pending migrations.
func (s *Store) Migrate(ctx context.Context) error {
	p, err := goose.NewProvider(goose.DialectSQLite3, s.write, migrations.SQLite)
	if err != nil {
		return fmt.Errorf("sqlite migrations: %w", err)
	}
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("sqlite migrate: %w", err)
	}
	return nil
}

// Close closes both connection pools.
func (s *Store) Close() error {
	return errors.Join(s.read.Close(), s.write.Close())
}
