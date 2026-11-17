// SPDX-License-Identifier: AGPL-3.0-only

// Package archtest enforces the dependency rules of 02-components §3 (ADR-0003).
// Test files are not checked: adapter tests import storetest, for example.
package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/amanuelmr/wickframe"

var domainModules = []string{"marketdata", "indicator", "workspace", "alert", "notify", "identity", "audit"}

// Domain modules may import these other domain modules.
var domainEdges = map[string][]string{
	"indicator": {"marketdata"},
	"alert":     {"indicator", "marketdata", "notify"},
}

// The only third-party code a domain module may use.
var domainThirdParty = map[string][]string{
	"alert": {"github.com/expr-lang/expr"},
}

// Domain code does no I/O of its own.
var domainForbiddenStd = []string{"database/sql", "net/http"}

// check returns why pkg may not import imp, or "" if it may. Both are paths
// relative to the module root for module packages, full paths otherwise.
func check(pkg, imp string) string {
	if root := adapterRoot(imp); root != "" && !within(pkg, root) && !within(pkg, "internal/app") {
		return "only internal/app may import the concrete adapter " + imp
	}
	m := domainModule(pkg)
	if m == "" {
		return ""
	}
	if !strings.Contains(strings.Split(imp, "/")[0], ".") {
		for _, f := range domainForbiddenStd {
			if within(imp, f) {
				return "domain module " + m + " may not do I/O through " + imp
			}
		}
		return ""
	}
	if after, ok := strings.CutPrefix(imp, module+"/"); ok {
		switch {
		case within(after, "internal/platform"), after == "internal/bus", domainModule(after) == m:
			return ""
		case slices.Contains(domainEdges[m], strings.TrimPrefix(after, "internal/")):
			return ""
		}
		return "domain module " + m + " may not import " + after
	}
	for _, allowed := range domainThirdParty[m] {
		if within(imp, allowed) {
			return ""
		}
	}
	return "domain module " + m + " may not import the third-party package " + imp
}

// domainModule returns the domain module that pkg belongs to, or "".
// Connectors live under marketdata but are adapters.
func domainModule(pkg string) string {
	if within(pkg, "internal/marketdata/connector") {
		return ""
	}
	for _, m := range domainModules {
		if within(pkg, "internal/"+m) {
			return m
		}
	}
	return ""
}

// adapterRoot returns the concrete adapter that imp belongs to, or "".
func adapterRoot(imp string) string {
	rel, ok := strings.CutPrefix(imp, module+"/")
	if !ok {
		return ""
	}
	parts := strings.Split(rel, "/")
	switch {
	case within(rel, "internal/api") && len(parts) >= 3,
		within(rel, "internal/store/sqlite"),
		within(rel, "internal/store/postgres"):
		return strings.Join(parts[:3], "/")
	case within(rel, "internal/marketdata/connector") && len(parts) >= 4:
		return strings.Join(parts[:4], "/")
	}
	return ""
}

// within reports whether path is dir or inside it.
func within(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, dir+"/")
}

func TestCheck(t *testing.T) {
	t.Parallel()
	const m = module + "/"
	tests := []struct {
		pkg, imp string
		ok       bool
	}{
		{"internal/marketdata", m + "internal/platform/clock", true},
		{"internal/marketdata", m + "internal/bus", true},
		{"internal/marketdata", "context", true},
		{"internal/marketdata", m + "internal/store/sqlite", false},
		{"internal/marketdata", m + "internal/api/httpapi", false},
		{"internal/marketdata", m + "internal/app", false},
		{"internal/marketdata", "database/sql", false},
		{"internal/marketdata", "net/http", false},
		{"internal/marketdata", "github.com/jackc/pgx/v5", false},
		{"internal/marketdata", m + "internal/alert", false},
		{"internal/indicator", m + "internal/marketdata", true},
		{"internal/indicator", m + "internal/marketdata/connector/fake", false},
		{"internal/alert", m + "internal/indicator", true},
		{"internal/alert", "github.com/expr-lang/expr/vm", true},
		{"internal/alert/internal/eval", m + "internal/alert", true},
		{"internal/notify", "github.com/expr-lang/expr", false},
		{"internal/store/sqlite", m + "internal/marketdata", true},
		{"internal/store/sqlite", m + "internal/store/sqlite/db", true},
		{"internal/store/sqlite", "database/sql", true},
		{"internal/store/sqlite", m + "internal/store/postgres", false},
		{"internal/api/httpapi", m + "internal/store/sqlite", false},
		{"internal/marketdata/connector/binance", "net/http", true},
		{"internal/marketdata/connector/binance", m + "internal/marketdata", true},
		{"internal/app", m + "internal/store/sqlite", true},
		{"internal/app", m + "internal/marketdata/connector/fake", true},
		{"cmd/wickframe", m + "internal/store/postgres", false},
	}
	for _, tt := range tests {
		if got := check(tt.pkg, tt.imp); (got == "") != tt.ok {
			t.Errorf("check(%s, %s) = %q, want allowed=%v", tt.pkg, tt.imp, got, tt.ok)
		}
	}
}

func TestDependencyRules(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	checked := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".claude", "node_modules", "testdata", "web", "tools":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		pkg := filepath.ToSlash(rel)
		for _, spec := range f.Imports {
			imp, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			checked++
			if why := check(pkg, imp); why != "" {
				t.Errorf("%s: %s", filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))), why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no imports were checked; is the module root right?")
	}
}
