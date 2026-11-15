---
status: proposed
date: 2026-11-14
decision-makers: "@amanuelmr"
---

# ADR-0018: Starlark as the user-script engine for custom indicators (proposed, pending the M4 spike)

## Context and Problem Statement

Custom indicators (FR-IND-06, M4) need a scripting language that runs user code
per bar over up to 100,000 bars inside the server, under the constraints in
[ADR-0011](0011-scripting-engine.md) and NFR-SEC-05: no I/O, a step or time budget,
a memory budget, and deterministic output. ADR-0011 deferred the choice between
Starlark and goja to a spike. This ADR records the desk research and the
provisional choice, so the spike tests a concrete hypothesis.

## Decision Drivers

* Sandbox strength and crash safety (a script must never take down the process).
* Deterministic, reproducible CPU limits.
* Performance over long histories.
* Familiarity for traders (Pine Script is the reference point).
* Editor experience in the browser (Monaco).
* Maintenance health.

## Considered Options

| Engine | Status (verified 2026-11-14) | Execution limit | Memory limit | Notes |
|---|---|---|---|---|
| `go.starlark.net` | Untagged; commit 2026-10-21; BSD-3; Google-maintained | `SetMaxExecutionSteps` (deterministic) and `Cancel` (wall clock) | None. The README recommends a separate process for hard limits | Python-like. No clock, no randomness, ordered dicts: deterministic by design. No known advisories. |
| `dop251/goja` | Untagged; commit 2026-11-10; MIT; one main maintainer (Grafana keeps the `sobek` fork) | `Interrupt` from a timer (not deterministic), call-stack limit | None | JavaScript, with TypeScript completions in Monaco through a `.d.ts`. Open issue #848 (2026-11-03): fuzzing found inputs that crash the host, including stack overflows that `recover()` cannot catch. |
| `risor` | v2.2.0; v3 in alpha; Apache-2.0 | Exact step limit | Exact allocation limit | The only engine with built-in memory limits, but too young to depend on. Watch. |
| `wazero` + guest language | v1.12.0; Apache-2.0 | Context cancellation | Hard page limit | Strongest isolation, but needs a guest language and toolchain. Fallback for multi-tenant hosting. |
| `expr` | v1.17.8 | Always terminates | — | No per-bar state; stays for alert conditions only (ADR-0011). |
| `yaegi` | v0.16.3 | — | — | Rejected: exposes the Go standard library and `unsafe`, cannot be sandboxed. |

## Decision Outcome

**Proposed: Starlark**, because it has a deterministic step budget, the smaller
attack surface and no known host-crash bugs, and its Python-like syntax is close to
Pine Script. Its missing memory limit is handled by design:

* Price series stay in Go. Scripts see read-only `series` objects, not lists.
* Built-in functions cap the size of any collection they create.
* The evaluator runs in step chunks and checks heap growth between chunks.
* On multi-tenant servers, scripts run in a child process with OS memory limits
  (see [ADR-0016](0016-connector-extension-model.md)).

**goja** remains the alternative if the spike shows the editor experience or
performance gap outweighs the sandbox advantages. **wazero** remains the fallback
for hard isolation.

The status becomes *accepted* (or this ADR is rejected in favour of a new one) when
the M4 spike measures:

1. Time per bar and allocations for SMA, EMA, RSI and a three-plot script over
   100,000 bars, in both engines.
2. Whether steps per bar are stable enough to set a fixed budget.
3. Worst-case memory growth from adversarial scripts under the caps above.
4. Whether goja's known crash inputs kill the process.
5. The effort to provide completions in Monaco for Starlark versus a `.d.ts` for
   goja.

### Consequences

* Good, because the spike has a clear hypothesis and pass/fail criteria.
* Good, because Starlark's determinism suits the indicator reference tests.
* Bad, because Starlark is slower than JavaScript engines on numeric loops (to be
  measured), and browser editor support needs work.
* Bad, because there is no in-process hard memory limit; it is approximated.

### Confirmation

The spike's results and benchmarks are attached to this ADR before its status
changes. Fuzz tests on the script runtime run in CI from M4.

## More Information

* Starlark in Go: <https://github.com/google/starlark-go> (README section
  "Bounding time, memory, and other resources")
* goja: <https://github.com/dop251/goja> (issue #848); sobek: <https://github.com/grafana/sobek>
* risor: <https://github.com/risor-io/risor>
* wazero: <https://github.com/tetratelabs/wazero>
* Related: [ADR-0011](0011-scripting-engine.md), [ADR-0016](0016-connector-extension-model.md).
* Facts verified 2026-11-14.
