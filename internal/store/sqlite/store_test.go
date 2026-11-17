// SPDX-License-Identifier: AGPL-3.0-only

package sqlite

import (
	"path/filepath"
	"testing"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/store/storetest"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.Context(), filepath.Join(t.TempDir(), "wickframe.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCandleContract(t *testing.T) {
	storetest.TestCandles(t, func(t *testing.T) marketdata.CandleRepository { return newStore(t) })
}

func TestConnectionSettings(t *testing.T) {
	s := newStore(t)
	for _, tt := range []struct{ pragma, want string }{
		{"journal_mode", "wal"},
		{"foreign_keys", "1"},
		{"busy_timeout", "5000"},
		{"synchronous", "1"}, // NORMAL
	} {
		var got string
		if err := s.read.QueryRowContext(t.Context(), "PRAGMA "+tt.pragma).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("PRAGMA %s = %s, want %s", tt.pragma, got, tt.want)
		}
	}
	if n := s.write.Stats().MaxOpenConnections; n != 1 {
		t.Errorf("writer pool allows %d connections, want 1", n)
	}
	if _, err := s.read.ExecContext(t.Context(), "DELETE FROM candles"); err == nil {
		t.Error("a write through the read pool succeeded, want query_only to reject it")
	}
}
