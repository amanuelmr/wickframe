---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0008: WebSocket transport with an in-process, conflating event hub

## Context and Problem Statement

Live candles must reach browsers with ≤ 100 ms added latency (NFR-PERF-03), for
≥ 200 clients and ≥ 500 instruments (NFR-PERF-04). The same updates feed the alert
engine, which must never miss a closed bar (FR-ALERT-10). A slow client must not
harm anyone else (NFR-REL-05). Which transport, library and internal fan-out design
do we use?

## Decision Drivers

* Bidirectional: clients subscribe and unsubscribe dynamically per chart.
* Works through common reverse proxies and in desktop webviews.
* Bounded memory under slow consumers.
* Lossless delivery for the alert engine; latest-value delivery for UI rendering.

## Considered Options

* **Transport:** WebSocket · Server-Sent Events (SSE) · gRPC-Web / Connect streaming
* **Library:** `coder/websocket` · `gorilla/websocket` · `gobwas/ws`
* **Fan-out:** custom channel-based hub · Watermill (in-memory backend) · embedded NATS

## Decision Outcome

* **Transport: WebSocket**, using a versioned JSON sub-protocol
  `wickframe.v1` (the message schema is in [real-time](../architecture/05-realtime.md)).
  SSE was rejected because it is one-directional (subscriptions would need extra
  HTTP calls) and limited per-origin connection counts make multi-chart layouts
  awkward on HTTP/1.1.
* **Library: `github.com/coder/websocket`** (v1.8.15, 2026-06-15, ISC): actively
  maintained, `context`-based, no dependencies. `gorilla/websocket`'s last release
  was v1.5.3 in 2024-06, with little activity since.
* **Fan-out: a custom in-process hub** (`internal/bus`), with two delivery classes:
  * **Conflating** (UI subscribers): each subscriber has a bounded queue keyed by
    `(instrument, timeframe, stream)`. A newer update to the *forming* bar replaces
    the queued one, so the queue never grows unbounded and clients always receive
    the latest state. *Closed* bars are never conflated away.
  * **Lossless** (alert engine, candle persister): bounded queues with blocking
    back-pressure toward the source connector. If a lossless consumer falls behind
    beyond its budget, it is the system's fault: an error metric fires and the
    consumer resynchronises from storage. Data is never silently dropped.
* **Slow clients:** writes use a per-message deadline. A client whose queue stays
  full past a threshold (default 10 s) is disconnected with close code 1013 (try
  again later) and resyncs on reconnect.

### Consequences

* Good, because it has predictable memory, isolates slow clients, and keeps
  delivery semantics simple.
* Good, because it has no external broker to operate (ADR-0003).
* Bad, because the hub only works within one process. Multi-replica deployments
  would need a broker-backed bus, a future ADR behind the same `bus` interface.
* Bad, because a custom hub needs careful testing. Mitigated with
  `testing/synctest`, race tests and load tests ([quality](../architecture/14-quality.md)).

### Confirmation

* A load test in CI's nightly job: 200 clients × 500 instruments, including 10%
  deliberately slow clients. It asserts p95 latency and bounded memory.
* Race-detector and `synctest`-based unit tests for the hub.

## More Information

* coder/websocket: <https://github.com/coder/websocket>
* WebSocket close codes: <https://www.rfc-editor.org/rfc/rfc6455#section-7.4> and the IANA registry
* Revisit when multi-replica server mode becomes a requirement.
