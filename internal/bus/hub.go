// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"context"
	"fmt"
	"sync"
)

// Hub keeps the latest value of each topic and hands it to the topic's
// subscribers. Publish never blocks: a subscriber that reads slowly skips to
// the latest value.
type Hub[T any] struct {
	mu     sync.Mutex
	latest map[string]T
	subs   map[string]map[*Sub[T]]struct{}
}

// New returns an empty Hub.
func New[T any]() *Hub[T] {
	return &Hub[T]{latest: make(map[string]T), subs: make(map[string]map[*Sub[T]]struct{})}
}

// Publish makes v the latest value of topic.
func (h *Hub[T]) Publish(topic string, v T) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.latest[topic] = v
	for s := range h.subs[topic] {
		s.set(v)
	}
}

// Subscribe returns a subscription to topic that starts with the topic's
// latest value, if it has one. Close it when done.
func (h *Hub[T]) Subscribe(topic string) *Sub[T] {
	s := &Sub[T]{hub: h, topic: topic, ready: make(chan struct{}, 1)}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[topic] == nil {
		h.subs[topic] = make(map[*Sub[T]]struct{})
	}
	h.subs[topic][s] = struct{}{}
	if v, ok := h.latest[topic]; ok {
		s.set(v)
	}
	return s
}

// Sub is a subscription to one topic.
type Sub[T any] struct {
	hub   *Hub[T]
	topic string
	ready chan struct{}

	mu    sync.Mutex
	value T
	has   bool
}

func (s *Sub[T]) set(v T) {
	s.mu.Lock()
	s.value, s.has = v, true
	s.mu.Unlock()
	select {
	case s.ready <- struct{}{}:
	default:
	}
}

// Next waits for a value newer than the last one Next returned.
func (s *Sub[T]) Next(ctx context.Context) (T, error) {
	for {
		s.mu.Lock()
		v, has := s.value, s.has
		s.has = false
		s.mu.Unlock()
		if has {
			return v, nil
		}
		select {
		case <-ctx.Done():
			var zero T
			return zero, fmt.Errorf("next on %s: %w", s.topic, ctx.Err())
		case <-s.ready:
		}
	}
}

// Close ends the subscription.
func (s *Sub[T]) Close() {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	delete(s.hub.subs[s.topic], s)
	if len(s.hub.subs[s.topic]) == 0 {
		delete(s.hub.subs, s.topic)
	}
}
