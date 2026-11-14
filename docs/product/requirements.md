# Requirements

| | |
|---|---|
| **Status** | Accepted (living document) |
| **Owner** | @amanuelmr |
| **Last reviewed** | 2026-11-13 |

This document is the single source of truth for **what** Wickframe must do. The
architecture documents describe **how**. Every requirement has a stable ID, which
issues, tests, pull requests and ADRs reference.

## 1. Conventions

- **IDs.** `FR-<area>-<nn>` for functional requirements and `NFR-<area>-<nn>` for
  non-functional ones. IDs are never reused; a dropped requirement keeps its ID and
  is marked *Withdrawn*.
- **Priority** ([MoSCoW](https://en.wikipedia.org/wiki/MoSCoW_method)): **M**ust,
  **S**hould, **C**ould, **W**on't (this time).
- **Milestone.** The roadmap milestone that delivers it (see [roadmap](roadmap.md)):
  `M1` … `M5`.
- **Keywords.** MUST, SHOULD and MAY follow [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).
- **Modes.** *Personal mode* means a single user with SQLite and no login (desktop
  app, or a local binary on localhost). *Server mode* means one or more users with
  PostgreSQL, authentication, and alerts running 24/7.

## 2. Functional requirements

### 2.1 Market data and connectors (`DATA`)

| ID | Requirement | Pri | MS |
|---|---|---|---|
| FR-DATA-01 | The system MUST define a connector interface for market-data sources covering instrument discovery, historical candles and real-time streams, with optional capabilities (for example trades, order book, extended hours) declared by each connector. | M | M1 |
| FR-DATA-02 | The system MUST ship connectors for at least three free public crypto sources at the first public release. | M | M1 |
| FR-DATA-03 | The system MUST ship at least one stock/ETF connector that works on a free plan. | M | M2 |
| FR-DATA-04 | Users MUST be able to add, configure, test, enable and disable data sources from the UI and the API. Credentials MUST be stored encrypted ([security](../architecture/11-security.md)). | M | M1 |
| FR-DATA-05 | The system MUST load historical candles on demand when a chart scrolls back, and MUST cache them locally so repeat views need no network. | M | M1 |
| FR-DATA-06 | The system MUST stream real-time updates to the forming (last) candle and roll to a new candle on interval boundaries. | M | M1 |
| FR-DATA-07 | The system MUST support the timeframes 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 12h, 1D, 1W and 1M. Timeframes a source does not provide natively MUST be derived by aggregating a finer timeframe. | M | M1 |
| FR-DATA-08 | The system MUST detect gaps in stored candle history and backfill them from the source when possible, and MUST mark unrecoverable gaps instead of interpolating. | M | M1 |
| FR-DATA-09 | The system MUST recover automatically from connector disconnects (exponential backoff with jitter), backfilling any candles missed while disconnected. | M | M1 |
| FR-DATA-10 | Users MUST be able to import candles from CSV files with a documented format. | S | M2 |
| FR-DATA-11 | The system SHOULD support exchange trading sessions and time zones for session-based markets (for example US equities regular and extended hours). | S | M2 |
| FR-DATA-12 | The system SHOULD support split and dividend adjustment where the source provides corporate actions. | S | M3 |
| FR-DATA-13 | The system MUST show each source's data freshness (real-time or delayed) and connection status in the UI. | M | M1 |
| FR-DATA-14 | The system MUST NOT expose an endpoint that serves bulk market data to unauthenticated third parties (non-redistribution, [ADR-0006](../adr/0006-bring-your-own-market-data.md)). | M | M1 |
| FR-DATA-15 | Users MUST be able to export stored candles to CSV and Parquet. | S | M3 |

### 2.2 Charting (`CHART`)

| ID | Requirement | Pri | MS |
|---|---|---|---|
| FR-CHART-01 | Users MUST be able to view candlestick, OHLC bar, line and area charts with volume. | M | M1 |
| FR-CHART-02 | Charts MUST support smooth pan and zoom, a crosshair with an OHLCV data readout, and price and time axes with logical formatting for the instrument's tick size. | M | M1 |
| FR-CHART-03 | Users MUST be able to search for and switch instruments, and switch timeframes, via keyboard and mouse. | M | M1 |
| FR-CHART-04 | Users MUST be able to add indicators as overlays (on price) or in separate panes, configure their parameters and styles, and remove them. | M | M1 |
| FR-CHART-05 | Users MUST be able to draw and edit: trend line, ray, horizontal line, vertical line, rectangle, Fibonacci retracement and text label. | M | M1 |
| FR-CHART-06 | Users SHOULD be able to use additional drawings: parallel channel, Fibonacci extension, long/short position tool, price range and date range measure. | S | M2 |
| FR-CHART-07 | Drawings MUST be anchored to time and price, so they stay correct across timeframes and after reloads. | M | M1 |
| FR-CHART-08 | Users MUST be able to save, rename, load and delete named chart layouts (instrument, timeframe, indicators, drawings, styles). | M | M1 |
| FR-CHART-09 | Users SHOULD be able to display multiple charts in one view (grid layouts) with optional crosshair, instrument and timeframe sync. | S | M3 |
| FR-CHART-10 | Users SHOULD be able to switch the chart scale between linear, logarithmic and percentage. | S | M2 |
| FR-CHART-11 | The UI MUST support light and dark themes and SHOULD follow the OS preference by default. | M | M1 |
| FR-CHART-12 | Users SHOULD be able to export a chart snapshot as PNG. | S | M2 |
| FR-CHART-13 | Users MUST be able to manage watchlists (create, reorder, group) showing last price and change. | M | M1 |
| FR-CHART-14 | The chart MUST indicate visually when data is delayed, stale or disconnected. | M | M1 |
| FR-CHART-15 | Users COULD use Heikin-Ashi, Renko and range charts. | C | M4 |

### 2.3 Indicators (`IND`)

| ID | Requirement | Pri | MS |
|---|---|---|---|
| FR-IND-01 | The system MUST provide at least these built-in indicators at the first public release: SMA, EMA, WMA, VWMA, VWAP (session), Bollinger Bands, RSI, MACD, Stochastic, ATR, ADX/DMI, OBV, Volume, Parabolic SAR, Ichimoku, Keltner Channels, Donchian Channels, CCI, MFI and Williams %R. | M | M1 |
| FR-IND-02 | Every built-in indicator MUST be verified in CI against an independent reference implementation over a fixed dataset, to a documented tolerance ([quality](../architecture/14-quality.md)). | M | M1 |
| FR-IND-03 | Indicators MUST update incrementally on live ticks, without recomputing the whole history. | M | M1 |
| FR-IND-04 | Indicator outputs used for display and for alerts MUST come from the same implementation ([ADR-0010](../adr/0010-single-indicator-engine.md)). | M | M1 |
| FR-IND-05 | Each indicator MUST document its formula, parameters, warm-up period and reference source. | M | M1 |
| FR-IND-06 | Users SHOULD be able to write custom indicators in a sandboxed scripting language with CPU-time and memory limits ([ADR-0011](../adr/0011-scripting-engine.md)). | S | M4 |
| FR-IND-07 | Users COULD share custom indicators as files (import/export). | C | M4 |

### 2.4 Alerts and notifications (`ALERT`)

| ID | Requirement | Pri | MS |
|---|---|---|---|
| FR-ALERT-01 | Users MUST be able to create alerts on price conditions: crossing up/down, greater/less than, entering/exiting a channel, and moving by a percentage within a time window. | M | M2 |
| FR-ALERT-02 | Users MUST be able to create alerts on indicator conditions (for example RSI(14) crossing above 70, or EMA(20) crossing EMA(50)). | M | M2 |
| FR-ALERT-03 | Users SHOULD be able to create alerts from drawings (for example a price crossing a trend line). | S | M3 |
| FR-ALERT-04 | Alerts MUST support trigger policies: once, once per bar, once per bar close, and every time (with a minimum re-arm interval). | M | M2 |
| FR-ALERT-05 | Alerts MUST support expiry and can be paused and resumed. | M | M2 |
| FR-ALERT-06 | The system MUST deliver notifications via webhook, email (SMTP), Telegram, Discord and ntfy, with a templated message body. | M | M2 |
| FR-ALERT-07 | Webhook deliveries MUST be signed (HMAC-SHA256 with a timestamp) so receivers can verify authenticity and reject replays. | M | M2 |
| FR-ALERT-08 | Notification delivery MUST retry with exponential backoff, and record every attempt and its outcome in an alert history the user can view. | M | M2 |
| FR-ALERT-09 | In server mode, alerts MUST be evaluated continuously whether or not any browser is open. | M | M2 |
| FR-ALERT-10 | After a restart or connector outage, the alert engine MUST resume from persisted state without duplicate firing. Bars missed during the outage MUST be evaluated once backfilled, and alerts fired late MUST be clearly marked as late. | M | M2 |
| FR-ALERT-11 | Users SHOULD see alert lines on the chart and be able to drag them to edit thresholds. | S | M3 |

### 2.5 Users, auth and settings (`USER`)

| ID | Requirement | Pri | MS |
|---|---|---|---|
| FR-USER-01 | Personal mode MUST work without accounts, but MUST only listen on the loopback interface unless authentication is enabled. | M | M1 |
| FR-USER-02 | Server mode MUST require authentication: local accounts with password (Argon2id) at minimum. | M | M2 |
| FR-USER-03 | Server mode SHOULD support OpenID Connect single sign-on. | S | M3 |
| FR-USER-04 | Server mode SHOULD support passkeys (WebAuthn) and TOTP second factor. | S | M3 |
| FR-USER-05 | Server mode MUST support the roles *admin* and *member*. Users' layouts, alerts and secrets are private unless explicitly shared. | M | M3 |
| FR-USER-06 | Users MUST be able to create scoped, revocable API tokens. | M | M2 |
| FR-USER-07 | The system MUST record an audit log of security-relevant events (sign-in, token creation, data source credential changes, role changes). | M | M2 |
| FR-USER-08 | Users MUST be able to export all of their data (layouts, drawings, watchlists, alerts, settings) as a versioned JSON archive, and import it again. | M | M2 |

### 2.6 API and integration (`API`)

| ID | Requirement | Pri | MS |
|---|---|---|---|
| FR-API-01 | All functionality available in the UI MUST be available through a versioned HTTP API described by an OpenAPI 3.1 document published with each release. | M | M1 |
| FR-API-02 | Real-time data MUST be available through a documented, versioned WebSocket protocol ([real-time](../architecture/05-realtime.md)). | M | M1 |
| FR-API-03 | API errors MUST use the RFC 9457 Problem Details format. | M | M1 |
| FR-API-04 | The system SHOULD offer outbound event webhooks (alert fired, data source down/up). | S | M2 |

### 2.7 Operations (`OPS`)

| ID | Requirement | Pri | MS |
|---|---|---|---|
| FR-OPS-01 | Wickframe MUST be distributed as a single static binary per platform, a multi-arch OCI image and a Docker Compose example. | M | M1 |
| FR-OPS-02 | Database migrations MUST run automatically at startup (with an opt-out) and MUST be forward-only, with tested upgrade paths between consecutive minor versions. | M | M1 |
| FR-OPS-03 | The system MUST expose liveness (`/livez`) and readiness (`/readyz`) endpoints and Prometheus metrics. | M | M1 |
| FR-OPS-04 | The system MUST provide documented, tested backup and restore commands for both database modes. | M | M2 |
| FR-OPS-05 | The system MUST be configurable through a config file, environment variables and flags, with that precedence (flags highest), and MUST validate configuration at startup with clear errors. | M | M1 |
| FR-OPS-06 | The system MUST send no telemetry by default. Any opt-in telemetry MUST be documented, anonymous and inspectable. | M | M1 |
| FR-OPS-07 | A desktop build (Windows, macOS, Linux) SHOULD be offered for personal mode ([ADR-0009](../adr/0009-desktop-shell.md)). | S | M3 |

## 3. Non-functional requirements

Targets are measured on the **reference hardware** unless stated otherwise:
**server** = 2 vCPU, 4 GB RAM, SSD, Linux; **client** = a mid-range 2022 laptop,
current Chrome or Firefox.

### 3.1 Performance (`PERF`)

| ID | Requirement |
|---|---|
| NFR-PERF-01 | Opening a chart with 5,000 cached bars and 3 indicators MUST render the first frame in ≤ 1.0 s (p95), measured from navigation to first paint of the candles. |
| NFR-PERF-02 | Pan and zoom MUST sustain ≥ 55 fps with 20,000 loaded bars and 5 indicators. |
| NFR-PERF-03 | Added latency from Wickframe receiving a source update to the browser rendering it MUST be ≤ 100 ms (p95) on a LAN. |
| NFR-PERF-04 | One server instance MUST support ≥ 200 concurrent WebSocket clients and ≥ 500 actively streamed instruments within the reference server's resources. |
| NFR-PERF-05 | The alert engine MUST evaluate ≥ 10,000 active alerts with p95 evaluation latency ≤ 1 s after the triggering update. |
| NFR-PERF-06 | Idle personal-mode memory MUST be ≤ 150 MB RSS; server-mode memory MUST be ≤ 1 GB RSS at the NFR-PERF-04 load. |
| NFR-PERF-07 | Candle history queries MUST return 5,000 bars in ≤ 50 ms (p95) from local storage. |

### 3.2 Reliability (`REL`)

| ID | Requirement |
|---|---|
| NFR-REL-01 | No data loss of user-created objects (layouts, drawings, alerts) on crash: every acknowledged write is durable. |
| NFR-REL-02 | Connector disconnects MUST be detected within 30 s (heartbeat) and recovery attempted automatically. |
| NFR-REL-03 | Alert notifications MUST be delivered at-least-once, with idempotency keys so receivers can deduplicate. |
| NFR-REL-04 | Graceful shutdown MUST drain in-flight requests and flush state within 30 s. |
| NFR-REL-05 | A slow WebSocket client MUST NOT degrade other clients or the alert engine (bounded queues; slow consumers are disconnected). |

### 3.3 Security and privacy (`SEC`)

| ID | Requirement |
|---|---|
| NFR-SEC-01 | Server mode MUST meet [OWASP ASVS 5.0](https://owasp.org/www-project-application-security-verification-standard/) Level 2 for the in-scope requirements, tracked in the [security architecture](../architecture/11-security.md). |
| NFR-SEC-02 | Secrets (data source API keys, notification tokens) MUST be encrypted at rest with an authenticated cipher, and MUST never be returned by the API or written to logs. |
| NFR-SEC-03 | Releases MUST be reproducible where feasible, signed, and published with an SBOM and build provenance. |
| NFR-SEC-04 | Dependencies MUST be scanned for known vulnerabilities in CI, and on every release. |
| NFR-SEC-05 | User scripts (M4) MUST run in a sandbox with no filesystem, network or process access, and with enforced CPU-time and memory limits. |
| NFR-SEC-06 | Vulnerabilities MUST be handled under the published [security policy](../../SECURITY.md). |

### 3.4 Usability and accessibility (`UX`)

| ID | Requirement |
|---|---|
| NFR-UX-01 | A new user MUST be able to go from install to a live chart in ≤ 5 minutes with no configuration file (a default crypto source is pre-configured). |
| NFR-UX-02 | Non-canvas UI MUST conform to [WCAG 2.2](https://www.w3.org/TR/WCAG22/) Level AA. The chart MUST offer keyboard navigation and an accessible data-table alternative. |
| NFR-UX-03 | All user-facing strings MUST be externalised for translation (i18n-ready from M1; English only at launch). |
| NFR-UX-04 | The UI MUST be usable on viewports from 360 px (phone, read-mostly) to 4K. |

### 3.5 Maintainability and portability (`MAINT`)

| ID | Requirement |
|---|---|
| NFR-MAINT-01 | Go code MUST pass the configured linters, `go vet`, and race-detector tests in CI. Frontend code MUST pass type-checking and linting. |
| NFR-MAINT-02 | Line coverage MUST be ≥ 80% for domain packages (indicators, aggregation, alert evaluation) and ≥ 60% overall, enforced in CI as a ratchet (coverage may not decrease). |
| NFR-MAINT-03 | Release targets: Linux amd64/arm64, macOS arm64/amd64, Windows amd64. |
| NFR-MAINT-04 | Supported browsers: the last two major versions of Chrome, Edge, Firefox and Safari. |
| NFR-MAINT-05 | Public APIs (HTTP, WebSocket, connector interface) follow SemVer. Breaking changes require a major version and a deprecation period of at least one minor release. |

## 4. Traceability

- Every issue labelled `type:feature` MUST reference at least one requirement ID.
- Tests that verify a requirement SHOULD include its ID in the test name or a
  comment (for example `TestAggregator_FRDATA07_DerivesWeeklyFromDaily`).
- The [roadmap](roadmap.md) lists which requirements each milestone delivers.
