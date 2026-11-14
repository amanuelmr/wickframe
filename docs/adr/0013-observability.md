---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0013: Observability with log/slog, Prometheus metrics, and opt-in OpenTelemetry

## Context and Problem Statement

Self-hosters need to know whether connectors are up, whether alerts are firing,
and why something failed (FR-OPS-03, quality goal Q2/Q4), without operating an
observability platform. Contributors need traces to debug latency. Which signals,
libraries and defaults do we use?

## Decision Drivers

* Zero required infrastructure; useful out of the box.
* Industry-standard formats that plug into what operators already run (Prometheus,
  Grafana, OpenTelemetry collectors).
* No telemetry leaves the machine unless the operator configures it (FR-OPS-06).

## Decision Outcome

| Signal | Library | Default | Notes |
|---|---|---|---|
| Logs | `log/slog` (stdlib) | On. Text format on a TTY, JSON otherwise. Level `info`. | Structured key/value fields; request, user and trace IDs propagated through `context`. |
| Metrics | `prometheus/client_golang` v1.25.x | On, at `/metrics` (on the main listener, or a separate admin listener if configured) | Pull model: the simplest for self-hosters. |
| Traces | OpenTelemetry Go v1.47.x (traces, metrics and logs signals all stable) | **Off.** Enabled by configuring an OTLP endpoint. | Uses the `otelslog` bridge so logs correlate with traces. |
| Health | Built-in | `/livez` (process alive), `/readyz` (DB reachable, migrations applied, at least the configured critical connectors initialised) | Kubernetes conventions. |
| Profiling | `net/http/pprof` | Off. Enabled on the admin listener only. | Never exposed on the public listener. |

Metric and log conventions (names, labels, cardinality rules) are defined in
[observability](../architecture/12-observability.md).

### Consequences

* Good, because it works with a stock Prometheus and Grafana setup, and needs no
  agent.
* Good, because OTel is available for advanced users without burdening the rest.
* Bad, because there are two metric paths (Prometheus native, plus OTel metrics if
  enabled). Accepted: Prometheus is the default, and OTel metrics export is
  optional.

### Confirmation

* A test asserts that `/metrics` exposes the documented core metric names.
* A lint rule forbids `fmt.Print*` and the `log` package outside `cmd/`.

## More Information

* OpenTelemetry Go: <https://github.com/open-telemetry/opentelemetry-go>
* Prometheus naming best practices: <https://prometheus.io/docs/practices/naming/>
