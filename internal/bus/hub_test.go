// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
)

// ev is a test event: kept events model closed bars, the rest forming updates.
type ev struct {
	Bar  int
	Rev  int
	Kept bool
}

func publish(h *Hub[ev], events ...ev) {
	for _, e := range events {
		if e.Kept {
			h.Publish("a", e)
		} else {
			h.PublishConflatable("a", e)
		}
	}
}

// drain reads n events without waiting for more.
func drain(t *testing.T, s *Sub[ev], n int) []ev {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got := make([]ev, 0, n)
	for range n {
		e, err := s.Next(ctx)
		if err != nil {
			t.Fatalf("Next after %d events: %v", len(got), err)
		}
		got = append(got, e)
	}
	if e, err := s.Next(ctx); err == nil {
		t.Fatalf("got an extra event %+v", e)
	}
	return got
}

func TestSlowSubscriberGetsEveryClosedBarInOrder(t *testing.T) {
	for _, tt := range []struct {
		name      string
		subscribe func(*Hub[ev], string) *Sub[ev]
	}{
		{"conflating", (*Hub[ev]).Subscribe},
		{"lossless", (*Hub[ev]).SubscribeLossless},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := New[ev]()
			s := tt.subscribe(h, "a")
			defer s.Close()

			publish(h,
				ev{1, 0, false}, ev{1, 0, false}, ev{1, 0, true},
				ev{2, 0, false}, ev{2, 0, false}, ev{2, 0, true},
				ev{3, 0, false}, ev{2, 1, true}, // a correction while bar 3 forms
				ev{3, 0, false}, ev{3, 0, false},
			)

			want := []ev{{1, 0, true}, {2, 0, true}, {2, 1, true}, {3, 0, false}}
			if diff := cmp.Diff(want, drain(t, s, len(want))); diff != "" {
				t.Errorf("events (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormingUpdateKeepsItsPlaceBeforeALaterClose(t *testing.T) {
	h := New[ev]()
	s := h.Subscribe("a")
	defer s.Close()

	publish(h, ev{1, 0, false}, ev{0, 1, true})

	want := []ev{{1, 0, false}, {0, 1, true}}
	if diff := cmp.Diff(want, drain(t, s, len(want))); diff != "" {
		t.Errorf("events (-want +got):\n%s", diff)
	}
}

func TestConflatingSubscriberStartsWithLatestEvent(t *testing.T) {
	h := New[ev]()
	publish(h, ev{1, 0, true}, ev{2, 0, false})
	ui, lossless := h.Subscribe("a"), h.SubscribeLossless("a")
	defer ui.Close()
	defer lossless.Close()

	if diff := cmp.Diff([]ev{{2, 0, false}}, drain(t, ui, 1)); diff != "" {
		t.Errorf("conflating (-want +got):\n%s", diff)
	}
	drain(t, lossless, 0)
}

func TestConflatingSubscriberLaggingForLagTimeoutIsCutOff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := New[ev]()
		s := h.Subscribe("a")
		defer s.Close()
		start := time.Now()
		for i := range ConflatingQueue + 10 {
			h.Publish("a", ev{i, 0, true})
		}
		if waited := time.Since(start); waited != 0 {
			t.Errorf("publisher waited %v for a UI subscriber", waited)
		}
		time.Sleep(LagTimeout + time.Second)
		h.Publish("a", ev{-1, 0, true}) // too late, the subscription has ended

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		for i := range ConflatingQueue + 10 {
			if e, err := s.Next(ctx); err != nil || e.Bar != i {
				t.Fatalf("Next = %+v, %v, want bar %d", e, err, i)
			}
		}
		if _, err := s.Next(ctx); !errors.Is(err, ErrLagging) {
			t.Fatalf("Next = %v, want ErrLagging", err)
		}
	})
}

func TestConflatingSubscriberThatCatchesUpInTimeStays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := New[ev]()
		s := h.Subscribe("a")
		defer s.Close()
		for i := range ConflatingQueue + 10 {
			h.Publish("a", ev{i, 0, true})
		}
		time.Sleep(LagTimeout - time.Second)
		for range 11 {
			if _, err := s.Next(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
		time.Sleep(time.Hour)
		h.Publish("a", ev{-1, 0, true})

		drain(t, s, ConflatingQueue)
	})
}

func TestLosslessSubscriberHoldsThePublisherBack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := New[ev]()
		s := h.SubscribeLossless("a")
		defer s.Close()
		for i := range LosslessQueue {
			h.Publish("a", ev{i, 0, true})
		}

		published := make(chan struct{})
		go func() {
			h.Publish("a", ev{LosslessQueue, 0, true})
			close(published)
		}()
		time.Sleep(StallTimeout - time.Second)
		synctest.Wait()
		select {
		case <-published:
			t.Fatal("Publish returned while the lossless queue was full")
		default:
		}

		for i := range LosslessQueue + 1 {
			if e, err := s.Next(t.Context()); err != nil || e.Bar != i {
				t.Fatalf("Next = %+v, %v, want bar %d", e, err, i)
			}
		}
		<-published
	})
}

func TestStalledLosslessSubscriberIsCutOffAndThePublisherMovesOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := New[ev]()
		stalled, other := h.SubscribeLossless("a"), h.SubscribeLossless("a")
		defer stalled.Close()
		defer other.Close()
		go func() {
			for {
				if _, err := other.Next(t.Context()); err != nil {
					return
				}
			}
		}()

		start := time.Now()
		for i := range LosslessQueue + 2 {
			h.Publish("a", ev{i, 0, true})
		}
		if waited := time.Since(start); waited != StallTimeout {
			t.Errorf("publisher waited %v, want the stall budget %v", waited, StallTimeout)
		}

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		for range LosslessQueue {
			if _, err := stalled.Next(ctx); err != nil {
				t.Fatalf("Next = %v before the queued events ran out", err)
			}
		}
		if _, err := stalled.Next(ctx); !errors.Is(err, ErrLagging) {
			t.Fatalf("Next = %v, want ErrLagging", err)
		}
	})
}

func TestCloseReleasesABlockedPublisher(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := New[ev]()
		s := h.SubscribeLossless("a")
		for i := range LosslessQueue {
			h.Publish("a", ev{i, 0, true})
		}
		start := time.Now()
		go func() {
			time.Sleep(time.Second)
			s.Close()
		}()
		h.Publish("a", ev{LosslessQueue, 0, true})
		if waited := time.Since(start); waited != time.Second {
			t.Errorf("publisher waited %v, want 1s, until Close", waited)
		}
		if len(h.subs) != 0 {
			t.Errorf("hub still holds %d topics after the last Close", len(h.subs))
		}
	})
}

func TestNextWaitsForANewEvent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := New[ev]()
		s := h.Subscribe("a")
		defer s.Close()

		got := make(chan ev)
		go func() {
			e, _ := s.Next(t.Context())
			got <- e
		}()
		synctest.Wait()
		select {
		case e := <-got:
			t.Fatalf("Next returned %+v before any publish", e)
		default:
		}
		h.PublishConflatable("a", ev{1, 0, false})
		if e := <-got; e.Bar != 1 {
			t.Fatalf("Next = %+v, want bar 1", e)
		}
	})
}
