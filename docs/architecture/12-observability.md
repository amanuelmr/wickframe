# 12 · Observability

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |
| **Requirements** | FR-OPS-03/06, NFR-REL-* |
| **Decisions** | [ADR-0013](../adr/0013-observability.md) |

## 1. Logging

* **API:** `log/slog` only. Loggers are passed by dependency injection (never a
  global logger in library code).
* **Format:** text on a TTY, JSON otherwise (`log.format: auto|text|json`).
  Levels: `debug`, `info` (default), `warn`, `error`.
* **Standard attributes:** `component`, `request_id`, `trace_id` (if tracing),
  `user_id` (if authenticated), `source` (for connector logs), `err`.
* **Rules:**
  1. Log an error **once**, at the boundary that handles it. Lower layers wrap
     errors with context (`fmt.Errorf("fetch klines %s: %w", sym, err)`).
  2. `info` is for state changes an operator cares about (startup, config summary,
     source connected or disconnected, migration applied). It is never per-tick or
     per-request. Per-request lines are access logs (sampled, configurable).
  3. Repeated identical warnings (for example a flapping source) are rate-limited,
     with a suppressed count.
  4. Secrets are redacted ([security §7](11-security.md#7-logging-and-privacy)).

## 2. Metrics

Prometheus exposition at `/metrics`; the namespace is `wickframe_`.

**Conventions** ([Prometheus naming](https://prometheus.io/docs/practices/naming/)):
base units (`_seconds`, `_bytes`), `_total` for counters, low-cardinality labels
only. Instrument symbols, user IDs, alert IDs and raw paths are **never** labels.

| Area | Metrics |
|---|---|
| Runtime | Go collector (goroutines, GC, memory), process collector, `wickframe_build_info{version,commit,go}` |
| HTTP | `wickframe_http_requests_total{route,method,code}`, `wickframe_http_request_duration_seconds{route,method}` |
| WebSocket | `wickframe_ws_clients`, `wickframe_ws_subscriptions`, `wickframe_ws_messages_sent_total{type}`, `wickframe_ws_disconnects_total{code}`, `wickframe_ws_lagging_clients` |
| Bus | `wickframe_bus_queue_depth{class,consumer}`, `wickframe_bus_stall_seconds_total{consumer}`, `wickframe_bus_conflated_total` |
| Market data | See [market data §10](04-market-data.md#10-metrics-subset) |
| Indicators | `wickframe_indicator_series_active`, `wickframe_indicator_update_duration_seconds`, `wickframe_indicator_cache_bytes` |
| Alerts and notify | See [alerts §8](07-alerts.md#8-metrics) |
| Storage | `wickframe_db_query_duration_seconds{op}`, `wickframe_db_pool_*`, `wickframe_persister_batch_size`, `wickframe_migrations_applied` |
| Supervision | `wickframe_component_restarts_total{component}`, `wickframe_component_up{component}` |

## 3. Service level objectives (server mode, reference deployment)

| SLO | Objective | Indicator |
|---|---|---|
| Live data freshness | 99% of minutes per enabled source have ingest lag p95 < 2 s | `wickframe_candle_ingest_lag_seconds` |
| Alert evaluation | 99.9% of evaluations within 1 s of the update | `wickframe_alert_eval_lag_seconds` |
| Notification delivery | 99% delivered within 60 s (excluding receiver outages) | `wickframe_notification_delivery_seconds` |
| API availability | 99.9% of non-4xx requests succeed | HTTP metrics |

These are **design targets**. Self-hosted availability also depends on the
operator's infrastructure. A Grafana dashboard JSON and Prometheus alerting rules
(`deploy/observability/`) ship with M2.

## 4. Tracing (opt-in)

Configuring `telemetry.otlp.endpoint` enables OpenTelemetry tracing:

* Spans: inbound HTTP (route template), outbound source requests, DB queries
  (sampled), alert evaluation (with firing), notification delivery.
* Sampling: parent-based, 10% by default; errors always recorded (tail sampling is
  left to the collector).
* The resource attributes include `service.name=wickframe`,
  `service.version`, and `deployment.environment` (configurable).

## 5. Health endpoints

| Endpoint | Semantics | Failure means |
|---|---|---|
| `GET /livez` | The process event loop responds; no deadlock detected in the supervisor | Restart the process |
| `GET /readyz` | DB reachable, migrations at head, config valid, required roles started | Do not route traffic yet |
| `GET /api/v1/system/status` (authenticated) | Detailed component status: sources, jobs, outbox backlog, versions | Human and UI diagnostics |

Neither `/livez` nor `/readyz` depends on external data sources. A data-source
outage is degraded functionality, not unreadiness.

## 6. `wickframe doctor`

A CLI diagnostic that checks the configuration, DB connectivity and schema version,
secret-key presence, clock skew (against the configured sources' server time where
available), outbound reachability of configured sources and notifiers, and file
permissions. It outputs a redacted report users can attach to bug reports.
