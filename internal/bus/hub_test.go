// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestSlowSubscriberSkipsToLatest(t *testing.T) {
	h := New[int]()
	s := h.Subscribe("a")
	defer s.Close()
	for i := range 1000 {
		h.Publish("a", i) // never blocks, although nobody reads
	}
	h.Publish("b", -1)

	if v, err := s.Next(t.Context()); err != nil || v != 999 {
		t.Fatalf("Next = %v, %v, want 999", v, err)
	}
}

func TestSubscribeStartsWithLatestValue(t *testing.T) {
	h := New[int]()
	h.Publish("a", 1)
	h.Publish("a", 2)
	s := h.Subscribe("a")
	defer s.Close()

	if v, err := s.Next(t.Context()); err != nil || v != 2 {
		t.Fatalf("Next = %v, %v, want 2", v, err)
	}
}

func TestNextWaitsForANewValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := New[int]()
		s := h.Subscribe("a")
		defer s.Close()
		h.Publish("a", 1)
		if v, _ := s.Next(t.Context()); v != 1 {
			t.Fatalf("Next = %v, want 1", v)
		}

		got := make(chan int)
		go func() {
			v, _ := s.Next(t.Context())
			got <- v
		}()
		synctest.Wait()
		select {
		case v := <-got:
			t.Fatalf("Next returned %v again before a new publish", v)
		default:
		}
		h.Publish("a", 2)
		if v := <-got; v != 2 {
			t.Fatalf("Next = %v, want 2", v)
		}

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		if _, err := s.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Next = %v, want context.DeadlineExceeded", err)
		}
	})
}

func TestClosedSubscriptionStopsReceiving(t *testing.T) {
	h := New[int]()
	s := h.Subscribe("a")
	s.Close()
	h.Publish("a", 1)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if v, err := s.Next(ctx); err == nil {
		t.Fatalf("Next = %v after Close, want an error", v)
	}
	if len(h.subs) != 0 {
		t.Errorf("hub still holds %d topics after the last Close", len(h.subs))
	}
}
