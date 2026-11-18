// SPDX-License-Identifier: AGPL-3.0-only

package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

// make build copies web/dist here. Without it only .gitkeep is embedded, and
// the placeholder is served instead.
//
//go:embed all:dist
var dist embed.FS

const placeholder = `<!doctype html>
<meta charset="utf-8">
<title>Wickframe</title>
<p>The web UI is not built into this binary. Run <code>make build</code>, or <code>make dev</code> while developing.</p>
`

// Handler serves the built web UI, or a placeholder page when it is not built.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // dist is a constant that the embed pattern above guarantees
	}
	return handler(sub)
}

func handler(fsys fs.FS) http.Handler {
	if _, err := fs.Stat(fsys, "index.html"); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(placeholder))
		})
	}
	return http.FileServerFS(fsys)
}
