// SPDX-License-Identifier: AGPL-3.0-only

package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name string
		fsys fstest.MapFS
		want string
	}{
		{"placeholder without a build", fstest.MapFS{".gitkeep": {}}, "not built"},
		{"the build when present", fstest.MapFS{"index.html": {Data: []byte("<div id=root>")}}, "<div id=root>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler(tt.fsys).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), "GET", "/", nil))
			if body := rec.Body.String(); rec.Code != 200 || !strings.Contains(body, tt.want) {
				t.Errorf("GET / = %d %q, want 200 containing %q", rec.Code, body, tt.want)
			}
		})
	}
}
