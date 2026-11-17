// SPDX-License-Identifier: AGPL-3.0-only

package migrations_test

import (
	"io/fs"
	"slices"
	"testing"

	"github.com/amanuelmr/wickframe/migrations"
)

func TestBothDialectsHaveTheSameMigrations(t *testing.T) {
	t.Parallel()
	names := func(fsys fs.FS) []string {
		entries, err := fs.ReadDir(fsys, ".")
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, 0, len(entries))
		for _, e := range entries {
			out = append(out, e.Name())
		}
		return out
	}
	sqlite, postgres := names(migrations.SQLite), names(migrations.Postgres)
	if len(sqlite) == 0 || !slices.Equal(sqlite, postgres) {
		t.Fatalf("migration files differ:\nsqlite:   %v\npostgres: %v", sqlite, postgres)
	}
}
