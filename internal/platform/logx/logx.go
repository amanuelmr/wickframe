// SPDX-License-Identifier: AGPL-3.0-only

// Package logx builds the slog logger: text on a terminal, JSON otherwise, with
// secret-bearing attributes redacted (12-observability §1, 11-security §7).
package logx

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Keys that every component uses for the same attributes, so logs query alike.
const (
	KeyComponent = "component"
	KeyRequestID = "request_id"
	KeyTraceID   = "trace_id"
	KeyUserID    = "user_id"
	KeySource    = "source"
	KeyErr       = "err"
)

// Format is the log.format config value.
type Format string

// FormatAuto picks text when the output is a terminal and JSON otherwise.
const (
	FormatAuto Format = "auto"
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Redacted replaces the value of a secret-bearing attribute.
const Redacted = "[REDACTED]"

// Matched against keys lower-cased with '_' and '-' removed, so access_token,
// apiKey and Set-Cookie are all caught. Over-redaction is the safe failure.
var secretWords = []string{"password", "token", "secret", "apikey", "authorization", "cookie"}

var keyNormalizer = strings.NewReplacer("_", "", "-", "")

// New returns a logger writing to w.
func New(w io.Writer, format Format, level slog.Leveler) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{Level: level, ReplaceAttr: redact}
	switch format {
	case FormatText:
		return slog.New(slog.NewTextHandler(w, opts)), nil
	case FormatJSON:
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	case FormatAuto:
		if isTerminal(w) {
			return slog.New(slog.NewTextHandler(w, opts)), nil
		}
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("unknown log format %q", format)
	}
}

func redact(_ []string, a slog.Attr) slog.Attr {
	key := strings.ToLower(keyNormalizer.Replace(a.Key))
	for _, w := range secretWords {
		if strings.Contains(key, w) {
			return slog.String(a.Key, Redacted)
		}
	}
	return a
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
