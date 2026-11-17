// SPDX-License-Identifier: AGPL-3.0-only

package logx_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/amanuelmr/wickframe/internal/platform/logx"
)

type credentials struct{ pass string }

func (c credentials) LogValue() slog.Value { return slog.StringValue(c.pass) }

func TestSecretAttributesAreRedacted(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		log    func(*slog.Logger)
		secret string
	}{
		{"password", func(l *slog.Logger) { l.Info("m", "password", "s3cr3t") }, "s3cr3t"},
		{"snake case", func(l *slog.Logger) { l.Info("m", "db_password", "s3cr3t") }, "s3cr3t"},
		{"camel case", func(l *slog.Logger) { l.Info("m", "apiKey", "k-123") }, "k-123"},
		{"api_key", func(l *slog.Logger) { l.Info("m", "api_key", "k-123") }, "k-123"},
		{"access token", func(l *slog.Logger) { l.Info("m", "access_token", "a7f3e9") }, "a7f3e9"},
		{"client secret", func(l *slog.Logger) { l.Info("m", "client_secret", "cs") }, "cs"},
		{"header", func(l *slog.Logger) { l.Info("m", "Authorization", "Bearer abc") }, "Bearer abc"},
		{"set-cookie", func(l *slog.Logger) { l.Info("m", "Set-Cookie", "sid=1") }, "sid=1"},
		{"in a group", func(l *slog.Logger) {
			l.Info("m", slog.Group("request", slog.String("cookie", "sid=1")))
		}, "sid=1"},
		{"added with With", func(l *slog.Logger) { l.With("token", "a7f3e9").Info("m") }, "a7f3e9"},
		{"a LogValuer", func(l *slog.Logger) { l.Info("m", "password", credentials{"s3cr3t"}) }, "s3cr3t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, format := range []logx.Format{logx.FormatJSON, logx.FormatText} {
				var buf bytes.Buffer
				log, err := logx.New(&buf, format, slog.LevelInfo)
				if err != nil {
					t.Fatal(err)
				}
				tt.log(log)
				if out := buf.String(); strings.Contains(out, tt.secret) || !strings.Contains(out, logx.Redacted) {
					t.Errorf("%s output leaks the secret or lacks %s: %s", format, logx.Redacted, out)
				}
			}
		})
	}
}

func TestOrdinaryAttributesAreKept(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log, err := logx.New(&buf, logx.FormatJSON, slog.LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("source connected", logx.KeyComponent, "ingest", logx.KeySource, "binance", logx.KeyUserID, "u1")
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not one JSON object: %v: %s", err, buf.String())
	}
	for k, want := range map[string]string{"component": "ingest", "source": "binance", "user_id": "u1"} {
		if got[k] != want {
			t.Errorf("%s = %v, want %q", k, got[k], want)
		}
	}
}

func TestAutoFormatIsJSONWhenNotATerminal(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log, err := logx.New(&buf, logx.FormatAuto, slog.LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("started")
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("auto format to a buffer is not JSON: %s", buf.String())
	}
}

func TestLevelFiltersLowerRecords(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log, err := logx.New(&buf, logx.FormatText, slog.LevelWarn)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("hidden")
	log.Warn("shown")
	if out := buf.String(); strings.Contains(out, "hidden") || !strings.Contains(out, "shown") {
		t.Fatalf("level filtering failed: %s", out)
	}
}

func TestUnknownFormatIsAnError(t *testing.T) {
	t.Parallel()
	if _, err := logx.New(&bytes.Buffer{}, "yaml", slog.LevelInfo); err == nil {
		t.Fatal("New with format yaml succeeded, want error")
	}
}
