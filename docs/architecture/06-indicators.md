# 06 · Indicator Engine

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |
| **Requirements** | FR-IND-01…07, FR-CHART-04, NFR-PERF-01/05 |
| **Decisions** | [ADR-0010](../adr/0010-single-indicator-engine.md), [ADR-0011](../adr/0011-scripting-engine.md) |

## 1. Design goals

1. **One implementation** for history, live updates, charts and alerts.
2. **Bit-for-bit determinism**: identical inputs produce identical outputs on
   every platform (no map iteration order, no parallel float reductions).
3. **Verified correctness** against an independent reference.
4. **O(1) amortised work per update** for all built-ins (O(window) only where
   mathematically unavoidable, for example the max/min in Donchian Channels,
   which use monotonic deques).

## 2. Model

### 2.1 Streaming state machine

```go
type Indicator interface {
    Update(b Candle) Output // revise the forming bar; idempotent for the same (t, rev)
    Close(b Candle) Output  // commit bar b; state advances
    WarmUp() int            // bars before outputs are valid
}
```

* `Update` computes from **committed state + the forming bar** without mutating
  committed state. Repeated `Update` calls for the same bar are therefore safe,
  and their cost does not accumulate.
* `Close` commits. History is computed by calling `Close` for each bar in order,
  which is the **same code path** as live data (ADR-0010). There is no separate
  vectorised batch implementation that could disagree.
* **Corrections** (`rev` > 0 on a closed bar): the cache restores the nearest
  snapshot at or before the corrected bar and replays from there (see §4).

### 2.2 Output

An `Output` is a fixed-size array of `float64` values, one per declared output (for
example MACD has `macd`, `signal`, `histogram`). **Invalid values are `NaN`
internally** and are serialised as JSON `null`. They occur during warm-up and when
an output is undefined (for example Parabolic SAR at a reversal).

### 2.3 Descriptor (registry metadata)

Every indicator registers a descriptor. The UI is generated from it, with no
per-indicator frontend code:

| Field | Example (RSI) |
|---|---|
| `id` | `rsi` |
| `name`, `shortName` | "Relative Strength Index", "RSI" |
| `category` | `momentum` |
| `params` | JSON Schema: `length` integer 2–500, default 14; `source` enum (close, open, high, low, hl2, hlc3, ohlc4), default close |
| `outputs` | `[{id: "rsi", kind: "line"}]` |
| `pane` | `separate` (default) or `overlay` |
| `levels` | `[30, 70]` (default horizontal guides) |
| `range` | `[0, 100]` |
| `warmUp(params)` | `length` |
| `reference` | TA-Lib `RSI`; formula document link |
| `version` | `1`. Bumped if output semantics change. Saved layouts record the version. |

## 3. Built-in indicator set (M1)

| Group | Indicators | Primary reference |
|---|---|---|
| Moving averages | SMA, EMA, WMA, VWMA, DEMA, TEMA, HMA | TA-Lib (SMA, EMA, WMA, DEMA, TEMA); formula fixtures (VWMA, HMA) |
| Volatility and bands | Bollinger Bands, ATR, Keltner Channels, Donchian Channels | TA-Lib (BBANDS, ATR); formula fixtures (Keltner, Donchian) |
| Momentum | RSI, MACD, Stochastic, Stochastic RSI, CCI, Williams %R, ROC, Momentum | TA-Lib |
| Trend | ADX / +DI / −DI, Parabolic SAR, Ichimoku Cloud, Supertrend | TA-Lib (ADX, SAR); formula fixtures (Ichimoku, Supertrend) |
| Volume | Volume, OBV, MFI, VWAP (session-anchored), Accumulation/Distribution | TA-Lib (OBV, MFI, AD); formula fixtures (VWAP) |

This covers FR-IND-01's list plus common extras.

### 3.1 Conventions (documented because tools differ)

| Topic | Wickframe convention |
|---|---|
| EMA seeding | The first EMA value is the SMA of the first `length` values (matches TA-Lib). |
| Wilder smoothing (RSI, ATR, ADX) | Smoothing factor `1/length`, seeded with the simple average of the first `length` values (matches TA-Lib). |
| Standard deviation (Bollinger) | **Population** standard deviation (matches TA-Lib and most charting platforms). |
| VWAP anchor | Session start (UTC midnight for 24x7 markets; regular-session open for equities). Other anchors (week, month) are configurable. |
| Source price | `close` by default; `hl2`, `hlc3`, `ohlc4` are available. |
| Gaps | Indicators run over the bars that exist. They do not insert synthetic bars across gaps. |

Each indicator's documentation page (generated from the descriptor plus a
Markdown formula file in `internal/indicator/docs/`) states the formula,
parameters, warm-up, outputs and reference (FR-IND-05).

## 4. Series cache and sharing

```text
SeriesKey = (instrument_id, tf, indicator_id, version, canonical_params)
```

* The **indicator cache** (`internal/indicator/cache`) holds one live instance
  per active `SeriesKey`, reference-counted by subscribers (chart sessions and
  alerts). Instances are fed by the lossless candle subscription.
* **Committed outputs** are kept in an in-memory ring per series (default: the
  last 5,000 bars). Requests for older ranges are computed by **replay** from
  stored candles: start at `from − warmUp` bars, discard warm-up outputs.
* **Snapshots:** every 1,000 closed bars the cache stores an in-memory snapshot of
  the indicator state, used for corrections and fast replays.
* Idle series (no subscribers for 10 min) are evicted. Memory is bounded by a
  global cap (default 256 MB) with LRU eviction.

## 5. API surface

* `GET /api/v1/indicators`: the descriptor catalogue.
* `GET /api/v1/indicators/{id}/series?instrument&tf&from&to&params…`: values
  aligned to bar times; `null` during warm-up. A batch form returns several
  indicators in one request (used when opening a layout).
* WebSocket: `indicators` listed on a `subscribe` message produce `indicator`
  messages after each `candle` message ([real-time](05-realtime.md#24-server--client)).

## 6. Verification (FR-IND-02)

1. **Reference fixtures** live in `testdata/indicators/`:
   * Input datasets: synthetic random walks with fixed seeds (including flat
     stretches, gaps, extreme values and very small prices), plus a real-data
     sample where the source's terms permit redistribution (ADR-0006 rule 6).
     Each dataset has ≥ 5,000 bars.
   * Expected outputs are generated **offline** with **TA-Lib** (C library v0.8.x,
     BSD-3-Clause, via the BSD-2 Python wrapper `ta-lib-python`) by a script in
     `tools/refgen/`. The fixtures are committed, so CI does not need Python or
     TA-Lib.
   * Indicators without a TA-Lib equivalent get fixtures from an independent
     implementation of the documented formula, written separately (different
     author or different language) and reviewed against a cited published source.
2. **Tolerance:** after warm-up, `|actual − expected| ≤ 1e-9 × max(1, |expected|)`.
   Warm-up outputs must be `null` exactly where the reference has no value.
3. **Path equivalence:** every test runs (a) full replay, (b) live simulation
   (each bar sent through multiple `Update` calls with partial OHLC, then `Close`),
   and (c) restart from a snapshot. All three outputs must be identical, bit for bit.
4. **Fuzzing:** Go native fuzz tests feed random but valid candles and assert no
   panics, no infinities, and that `Update` is idempotent.
5. **Benchmarks:** `BenchmarkIndicator/<id>` tracks ns per update. CI fails on a
   regression greater than 20% versus `main` (benchstat comparison in the nightly
   job).

## 7. Custom indicators (M4)

User scripts implement the same `Indicator` contract through the script runtime
([ADR-0011](../adr/0011-scripting-engine.md)). They are versioned, owned by a
user, and sandboxed. They are cached and shared exactly like built-ins, but are
isolated per owner unless explicitly shared.
