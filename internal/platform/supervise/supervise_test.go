// SPDX-License-Identifier: AGPL-3.0-only

package supervise_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/amanuelmr/wickframe/internal/platform/clock"
	"github.com/amanuelmr/wickframe/internal/platform/logx"
	"github.com/amanuelmr/wickframe/internal/platform/supervise"
)

func newLogger(t *testing.T) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	log, err := logx.New(&buf, logx.FormatJSON, slog.LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	return log, &buf
}

// runFor runs s until d has passed in the bubble, then stops it and waits.
func runFor(t *testing.T, s *supervise.Supervisor, d time.Duration) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error)
	go func() { done <- s.Run(ctx) }()
	time.Sleep(d)
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run() = %v, want nil", err)
	}
}

func TestFailingChildIsRestartedWithBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, buf := newLogger(t)
		var starts []time.Time
		s := supervise.New(log, clock.System{}, supervise.Child{Name: "feed", Run: func(ctx context.Context) error {
			starts = append(starts, time.Now())
			if len(starts) < 3 {
				return errors.New("connection reset")
			}
			<-ctx.Done()
			return nil
		}})
		runFor(t, s, time.Hour)

		if len(starts) != 3 {
			t.Fatalf("child started %d times, want 3", len(starts))
		}
		for i, ceiling := range []time.Duration{time.Second, 2 * time.Second} {
			if gap := starts[i+1].Sub(starts[i]); gap > ceiling {
				t.Errorf("restart %d after %v, want at most %v", i+1, gap, ceiling)
			}
		}
		if n := strings.Count(buf.String(), `"msg":"component failed"`); n != 2 {
			t.Errorf("logged %d failures, want 2: %s", n, buf)
		}
		if !strings.Contains(buf.String(), `"component":"feed"`) {
			t.Errorf("log lacks the component name: %s", buf)
		}
	})
}

func TestPanicIsRecoveredLoggedWithStackAndRestarted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, buf := newLogger(t)
		runs := 0
		s := supervise.New(log, clock.System{}, supervise.Child{Name: "engine", Run: func(ctx context.Context) error {
			if runs++; runs == 1 {
				panic("index out of range")
			}
			<-ctx.Done()
			return nil
		}})
		runFor(t, s, time.Minute)

		if runs != 2 {
			t.Fatalf("child ran %d times, want 2", runs)
		}
		out := buf.String()
		for _, want := range []string{"panic: index out of range", `"stack":`, "supervise_test.TestPanicIsRecovered"} {
			if !strings.Contains(out, want) {
				t.Errorf("log lacks %q: %s", want, out)
			}
		}
	})
}

func TestFinishedChildIsNotRestarted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, buf := newLogger(t)
		runs := 0
		s := supervise.New(log, clock.System{}, supervise.Child{Name: "migrate", Run: func(context.Context) error {
			runs++
			return nil
		}})
		runFor(t, s, time.Hour)
		if runs != 1 || !strings.Contains(buf.String(), "component finished") {
			t.Fatalf("child ran %d times, want 1 and a finished log: %s", runs, buf)
		}
	})
}

func TestChildrenStopInReverseOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, _ := newLogger(t)
		var mu sync.Mutex
		var stopped []string
		child := func(name string) supervise.Child {
			return supervise.Child{Name: name, Run: func(ctx context.Context) error {
				<-ctx.Done()
				time.Sleep(time.Second) // a slow flush must finish before the next child stops
				mu.Lock()
				defer mu.Unlock()
				stopped = append(stopped, name)
				return nil
			}}
		}
		s := supervise.New(log, clock.System{}, child("store"), child("ingest"), child("api"))
		runFor(t, s, time.Minute)
		if want := []string{"api", "ingest", "store"}; !slices.Equal(stopped, want) {
			t.Fatalf("stopped %v, want %v", stopped, want)
		}
	})
}

func TestBackoffStartsOverAfterAStableRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, _ := newLogger(t)
		var starts []time.Time
		s := supervise.New(log, clock.System{}, supervise.Child{Name: "feed", Run: func(context.Context) error {
			starts = append(starts, time.Now())
			time.Sleep(2 * time.Minute)
			return errors.New("daily disconnect")
		}})
		runFor(t, s, 20*time.Minute)

		if len(starts) < 6 {
			t.Fatalf("child started %d times, want at least 6", len(starts))
		}
		for i := 1; i < len(starts); i++ {
			if pause := starts[i].Sub(starts[i-1]) - 2*time.Minute; pause > time.Second {
				t.Errorf("restart %d paused %v, want at most the first backoff step of 1s", i, pause)
			}
		}
	})
}

func TestSupervisorsNest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, _ := newLogger(t)
		ran := false
		inner := supervise.New(log, clock.System{}, supervise.Child{Name: "leaf", Run: func(ctx context.Context) error {
			ran = true
			<-ctx.Done()
			return nil
		}})
		runFor(t, supervise.New(log, clock.System{}, supervise.Child{Name: "role:api", Run: inner.Run}), time.Second)
		if !ran {
			t.Fatal("the nested child never ran")
		}
	})
}
