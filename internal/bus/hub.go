// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Queue sizes and time budgets of the delivery classes.
const (
	ConflatingQueue = 256
	LosslessQueue   = 4096
	LagTimeout      = 10 * time.Second
	StallTimeout    = 5 * time.Second
)

// ErrLagging ends a subscription that fell too far behind to keep every
// event. The subscriber must resync from storage or the source.
var ErrLagging = errors.New("subscriber fell too far behind")

var errClosed = errors.New("subscription closed")

// Hub delivers each topic's events to its subscribers in publish order.
// Events from Publish are never dropped. Events from PublishConflatable may be
// replaced by a newer one before a slow subscriber reads them.
type Hub[T any] struct {
	mu     sync.Mutex
	recent map[string]*recent[T]
	subs   map[string]map[*Sub[T]]struct{}
}

// recent holds a topic's newest kept and newest conflatable event.
type recent[T any] struct {
	kept, latest       T
	hasKept, hasLatest bool
	keptLast           bool
}

// New returns an empty Hub.
func New[T any]() *Hub[T] {
	return &Hub[T]{recent: make(map[string]*recent[T]), subs: make(map[string]map[*Sub[T]]struct{})}
}

// Publish delivers v to every subscriber of topic. It blocks while a lossless
// subscriber's queue is full, for at most StallTimeout per subscriber.
func (h *Hub[T]) Publish(topic string, v T) { h.publish(topic, v, true) }

// PublishConflatable delivers v unless a newer conflatable event replaces it
// first. It never blocks.
func (h *Hub[T]) PublishConflatable(topic string, v T) { h.publish(topic, v, false) }

func (h *Hub[T]) publish(topic string, v T, keep bool) {
	h.mu.Lock()
	r := h.recent[topic]
	if r == nil {
		r = &recent[T]{}
		h.recent[topic] = r
	}
	if keep {
		r.kept, r.hasKept = v, true
	} else {
		r.latest, r.hasLatest = v, true
	}
	r.keptLast = keep
	subs := make([]*Sub[T], 0, len(h.subs[topic]))
	for s := range h.subs[topic] {
		subs = append(subs, s)
	}
	h.mu.Unlock()
	for _, s := range subs {
		s.push(v, keep)
	}
}

// Subscribe returns a subscription for a UI client that starts with the
// topic's newest kept and newest conflatable event, in publish order. The kept
// one covers a closed bar that storage may not hold yet. Publish never waits
// for it: past
// ConflatingQueue kept events it is lagging, and after LagTimeout of lagging it
// ends with ErrLagging.
func (h *Hub[T]) Subscribe(topic string) *Sub[T] {
	return h.subscribe(topic, false, ConflatingQueue)
}

// SubscribeLossless returns a subscription that receives every kept event. A
// full queue makes Publish wait, and after StallTimeout the subscription ends
// with ErrLagging. It starts with the next event.
func (h *Hub[T]) SubscribeLossless(topic string) *Sub[T] {
	return h.subscribe(topic, true, LosslessQueue)
}

func (h *Hub[T]) subscribe(topic string, lossless bool, size int) *Sub[T] {
	s := &Sub[T]{
		hub: h, topic: topic, lossless: lossless, size: size,
		ready: make(chan struct{}, 1), space: make(chan struct{}, 1),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[topic] == nil {
		h.subs[topic] = make(map[*Sub[T]]struct{})
	}
	h.subs[topic][s] = struct{}{}
	if r := h.recent[topic]; r != nil && !lossless {
		if r.hasLatest && r.keptLast {
			s.push(r.latest, false)
		}
		if r.hasKept {
			s.push(r.kept, true)
		}
		if r.hasLatest && !r.keptLast {
			s.push(r.latest, false)
		}
	}
	return s
}

type entry[T any] struct {
	v   T
	seq uint64
}

// Sub is a subscription to one topic. One goroutine calls Next.
type Sub[T any] struct {
	hub      *Hub[T]
	topic    string
	lossless bool
	size     int
	ready    chan struct{} // an event or an error is waiting
	space    chan struct{} // the queue has room or the subscription ended

	mu        sync.Mutex
	lag       *time.Timer // runs while a conflating queue is over its size
	seq       uint64
	queue     []entry[T]
	latest    entry[T]
	hasLatest bool
	err       error
}

func (s *Sub[T]) push(v T, keep bool) {
	defer signal(s.ready)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return
	}
	s.seq++
	e := entry[T]{v, s.seq}
	if !keep {
		s.latest, s.hasLatest = e, true
		return
	}
	if len(s.queue) >= s.size && !s.waitForSpace() {
		return
	}
	s.queue = append(s.queue, e)
}

// waitForSpace is called with s.mu held and returns with it held. It reports
// whether the event may be queued; if not, the subscription has ended.
func (s *Sub[T]) waitForSpace() bool {
	if !s.lossless {
		if s.lag == nil {
			s.lag = time.AfterFunc(LagTimeout, func() { s.end(ErrLagging) })
		}
		return true
	}
	stall := time.NewTimer(StallTimeout)
	defer stall.Stop()
	for len(s.queue) >= s.size && s.err == nil {
		s.mu.Unlock()
		select {
		case <-s.space:
			s.mu.Lock()
		case <-stall.C:
			s.mu.Lock()
			if len(s.queue) >= s.size && s.err == nil {
				s.err = ErrLagging
			}
		}
	}
	return s.err == nil
}

// Next returns the subscription's next event, waiting for one if needed. After
// the queued events it returns ErrLagging if the subscription fell behind.
func (s *Sub[T]) Next(ctx context.Context) (T, error) {
	for {
		s.mu.Lock()
		if len(s.queue) > 0 && (!s.hasLatest || s.queue[0].seq < s.latest.seq) {
			v := s.queue[0].v
			s.queue = s.queue[1:]
			if s.lag != nil && len(s.queue) < s.size {
				s.lag.Stop()
				s.lag = nil
			}
			s.mu.Unlock()
			signal(s.space)
			return v, nil
		}
		if s.hasLatest {
			v := s.latest.v
			s.hasLatest = false
			s.mu.Unlock()
			return v, nil
		}
		err := s.err
		s.mu.Unlock()
		if err != nil {
			var zero T
			return zero, fmt.Errorf("next on %s: %w", s.topic, err)
		}
		select {
		case <-ctx.Done():
			var zero T
			return zero, fmt.Errorf("next on %s: %w", s.topic, ctx.Err())
		case <-s.ready:
		}
	}
}

// Close ends the subscription and releases a publisher waiting on it.
func (s *Sub[T]) Close() {
	s.hub.mu.Lock()
	delete(s.hub.subs[s.topic], s)
	if len(s.hub.subs[s.topic]) == 0 {
		delete(s.hub.subs, s.topic)
	}
	s.hub.mu.Unlock()
	s.mu.Lock()
	if s.lag != nil {
		s.lag.Stop()
	}
	s.mu.Unlock()
	s.end(errClosed)
}

func (s *Sub[T]) end(err error) {
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
	signal(s.ready)
	signal(s.space)
}

func signal(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}
