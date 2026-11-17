// SPDX-License-Identifier: AGPL-3.0-only

// Package migrations embeds the goose SQL migrations of both dialects. Every
// migration exists in both directories with the same number and name
// (08-storage §3); they are forward-only, so they have no Down sections.
package migrations

import (
	"embed"
	"io/fs"
)

//go:embed sqlite/*.sql postgres/*.sql
var files embed.FS

// SQLite and Postgres hold each dialect's migrations at their root.
var (
	SQLite   = sub("sqlite")
	Postgres = sub("postgres")
)

func sub(dir string) fs.FS {
	f, err := fs.Sub(files, dir)
	if err != nil {
		panic(err) // dir is a constant that the embed pattern above guarantees
	}
	return f
}
