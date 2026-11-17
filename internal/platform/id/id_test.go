// SPDX-License-Identifier: AGPL-3.0-only

package id

import (
	"bytes"
	"testing"
	"time"
)

func TestNew_VersionVariantAndTimestamp(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 11, 14, 12, 0, 0, 123_000_000, time.UTC)
	u := New(at)
	if v := u[6] >> 4; v != 7 {
		t.Fatalf("version = %d, want 7", v)
	}
	if u[8]&0xc0 != 0x80 {
		t.Fatalf("variant bits = %08b, want 10xxxxxx", u[8])
	}
	if !u.Time().Equal(at) {
		t.Fatalf("Time() = %v, want %v", u.Time(), at)
	}
}

func TestNew_BytesSortByTime(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	prev := New(base)
	for i := 1; i < 1000; i++ {
		next := New(base.Add(time.Duration(i) * time.Millisecond))
		if bytes.Compare(prev[:], next[:]) >= 0 {
			t.Fatalf("id %d does not sort after its predecessor: %s >= %s", i, prev, next)
		}
		prev = next
	}
}

func TestParse(t *testing.T) {
	t.Parallel()
	u := New(time.Unix(1_800_000_000, 0))
	got, err := Parse(u.String())
	if err != nil || got != u {
		t.Fatalf("round trip: got %s, %v; want %s", got, err, u)
	}
	for _, bad := range []string{"", "not-a-uuid", u.String()[:35], "0192f0c8-7b2a-7xyz-8000-000000000000", "0192f0c87b2a-7000-8000-0000-000000000000"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", bad)
		}
	}
}

func FuzzParse(f *testing.F) {
	f.Add("0192f0c8-7b2a-7000-8000-000000000000")
	f.Fuzz(func(t *testing.T, s string) {
		u, err := Parse(s)
		if err != nil {
			return
		}
		again, err := Parse(u.String())
		if err != nil || again != u {
			t.Fatalf("String() of a parsed id does not round-trip: %q", s)
		}
	})
}
