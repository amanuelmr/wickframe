---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0011: Use expr-lang for alert conditions; defer the user-script engine to an M4 spike between Starlark and goja

## Context and Problem Statement

Two different needs involve user-provided logic:

1. **Alert conditions** (M2): small expressions over prices and indicator values,
   such as `rsi(14) > 70 && close > ema(200)`. These must always terminate quickly.
2. **Custom indicators** (M4, FR-IND-06): full programs with state, loops and
   multiple outputs, run in a sandbox with CPU and memory limits (NFR-SEC-05).

## Decision Drivers

* Safety: no filesystem, network or process access; guaranteed termination; bounded
  memory.
* Performance in the alert hot path (NFR-PERF-05).
* A familiar syntax for traders and developers.
* Maintenance health.

## Considered Options

| Engine | Version / status (2026-11) | Termination and limits | Notes |
|---|---|---|---|
| `expr-lang/expr` | v1.17.8, MIT | Not Turing-complete; always terminates; type-checked at compile time | Purpose-built for rules and conditions |
| `go.starlark.net` | Untagged, actively committed, BSD-3 | Step limit (`SetMaxExecutionSteps`); no I/O unless granted; deterministic | Python-like; used by Bazel |
| `dop251/goja` | Untagged, actively committed, MIT | `Interrupt()` for time limits; no built-in memory cap | JavaScript (ES5.1 plus much of ES6+) |
| `yuin/gopher-lua` | v1.1.2, MIT | Context cancellation | Lua 5.1, niche for this audience |
| `tetratelabs/wazero` | v1.12.0, Apache-2.0 | Hard memory limit, context cancellation | WASM: any language, strongest isolation, hardest developer experience |

## Decision Outcome

1. **Alert conditions: `expr-lang/expr` — accepted.** Expressions are compiled once
   when the alert is saved (type errors are rejected then), then evaluated per
   update against an environment that exposes only whitelisted, pure functions
   (indicator values, price fields, math). Compiled programs are cached.
2. **Custom indicators: decision deferred to an M4 spike** comparing **Starlark**
   and **goja** on safety (memory accounting, escape surface), performance (an
   EMA/RSI-style indicator over 100k bars), and authoring experience. **wazero** is
   the documented fallback for multi-tenant servers that need hard memory
   isolation. The spike's outcome will be recorded as a new ADR that supersedes
   point 2 of this one.

Constraints any script engine must satisfy (NFR-SEC-05):

* No ambient capabilities: no file, network, clock (time comes from bars), random
  (or a seeded, deterministic one), or environment access.
* An enforced step or time budget per evaluation and per backfill. Exceeding it
  disables the script with a visible error.
* A memory budget, enforced or approximated by allocation accounting.
* Deterministic output for identical inputs, so the reference tests can run.
* Scripts run in a separate goroutine pool from the alert engine's core loop. On
  multi-tenant servers, they run in a separate process (see ADR-0016).

### Consequences

* Good, because alerts get a safe, fast, well-understood expression language now.
* Good, because the bigger choice is made on evidence, not guesswork.
* Bad, because there will eventually be two languages (conditions and scripts).
  Acceptable: they serve different purposes. Scripts may later be callable from
  conditions.

### Confirmation

* Fuzz tests on condition compilation and evaluation.
* A benchmark in CI: evaluating 10,000 compiled conditions stays within the
  NFR-PERF-05 budget.

## More Information

* expr: <https://github.com/expr-lang/expr>
* Starlark in Go: <https://github.com/google/starlark-go>
* goja: <https://github.com/dop251/goja>
* wazero: <https://github.com/tetratelabs/wazero>
