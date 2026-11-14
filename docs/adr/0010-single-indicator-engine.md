---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0010: Compute indicators in one Go engine on the backend, for both charts and alerts

## Context and Problem Statement

Indicators are drawn on charts and evaluated by alerts. If the chart and the alert
engine compute RSI differently, even by rounding or warm-up handling, a user sees
RSI cross 70 on the chart while the alert never fires, or the reverse. That is a
correctness failure (quality goal Q1, FR-IND-04). Where do indicators run?

## Decision Drivers

* One implementation means identical numbers everywhere.
* Incremental (streaming) updates on live ticks (FR-IND-03).
* Verifiable against reference implementations (FR-IND-02).
* Interactive latency when adding or reconfiguring an indicator (NFR-PERF-01).

## Considered Options

* **Go engine on the backend**; values streamed to the browser
* Go engine compiled to WebAssembly and run in the browser, plus native Go for alerts
* Two implementations: TypeScript in the browser and Go for alerts

## Decision Outcome

Chosen option: **a single Go indicator engine on the backend**
(`internal/indicator`).

* Each indicator is a **streaming state machine** with two operations:
  `Update(bar)`, which revises the forming bar idempotently, and `Close(bar)`,
  which commits it. The same code computes history (by replaying bars) and live
  updates. There is no separate batch path that could diverge.
* **Charts** request an indicator series over HTTP for the visible range plus the
  warm-up period, then receive incremental values alongside each candle update
  over the WebSocket ([real-time](../architecture/05-realtime.md)).
* **Alerts** use the same engine instances, through the lossless bus subscription
  ([ADR-0008](0008-realtime-transport.md)).
* Results for a given (instrument, timeframe, indicator, parameters) are cached and
  shared between all subscribers.

### Why not WebAssembly (now)

A minimal Go-to-WASM build measured about 2 MB raw (about 570 KB gzip) for a single
EMA function. The Go wiki notes that 10 MB+ is common once libraries are imported.
TinyGo is much smaller but supports only a subset of Go. With a self-hosted
backend, the server round-trip is small, and the desktop build runs the Go engine
natively anyway. WASM adds a large download and interop cost for no correctness
gain. **Revisit** if offline-in-browser scripting becomes a requirement.

### Consequences

* Good, because chart and alert values are identical by construction.
* Good, because the reference-verification suite tests one implementation.
* Good, because the frontend stays thin and fast to load.
* Bad, because adding an indicator needs a backend round-trip. Go computes 5,000
  bars of typical indicators in well under a millisecond, so network latency
  dominates. That is acceptable on LAN and desktop, and is measured against
  NFR-PERF-01.
* Bad, because the browser cannot compute indicators while offline from the
  backend. Acceptable: the backend is the app.

### Confirmation

* There is no indicator math in `web/`; a lint rule forbids indicator libraries in
  frontend dependencies.
* The reference suite runs every indicator through both replay and incremental
  paths, and asserts they produce identical output.

## More Information

* Go WebAssembly wiki: <https://go.dev/wiki/WebAssembly>
* Indicator design: [indicators](../architecture/06-indicators.md)
