# Roadmap

| | |
|---|---|
| **Status** | Accepted (living document) |
| **Last reviewed** | 2026-11-13 |

Milestones are defined by **exit criteria**, not dates. Each milestone ends with a
tagged release. Requirement IDs refer to [requirements](requirements.md).

```text
M0 Foundations ──▶ M1 Charts (v0.1) ──▶ M2 Alerts & Stocks (v0.2–0.3) ──▶ M3 Desktop & Teams (v0.4–0.5) ──▶ M4 Extensibility (v0.6+) ──▶ v1.0
```

## M0 — Foundations (no release)

**Goal:** a repository where good code is the easy path.

| Deliverable | Notes |
|---|---|
| Architecture and product docs, ADRs 0001–0016 | This documentation set |
| Repository skeleton | [Repository layout](../engineering/repository-layout.md); `go.mod` (Go 1.27), `web/` (pnpm, Vite, React, TS 7) |
| CI pipeline | Lint, test (race), build matrix, govulncheck, DCO, SPDX/license checks, OpenAPI freshness |
| Platform packages | `clock`, `id`, `logx`, `supervise`, `retry`, `secret` |
| Store scaffolding | SQLite and PostgreSQL adapters, goose, sqlc configs, `storetest` harness, archtest |
| Fake connector and test harness | Deterministic market simulator ([quality §2.1](../architecture/14-quality.md#21-the-fake-connector)) |
| Walking skeleton | `wickframe serve` serves an embedded page showing one live candle from the fake connector over WebSocket, end to end |

**Exit criteria:** walking skeleton works on Linux, macOS and Windows; all CI gates
are active and green; a contributor can run `make dev` from a clean clone in
≤ 10 minutes.

## M1 — Charts (release v0.1.0, first public release)

**Goal:** a genuinely useful self-hosted charting app for crypto.

| Area | Requirements |
|---|---|
| Data | FR-DATA-01, 02 (Binance family, Kraken, Coinbase), 04–09, 13, 14 |
| Charting | FR-CHART-01–05, 07, 08, 11, 13, 14 |
| Indicators | FR-IND-01–05 (20+ verified built-ins) |
| API | FR-API-01–03 |
| Ops | FR-OPS-01–03, 05, 06 (binaries, image, Compose, health, metrics) |
| Users | FR-USER-01 (personal mode, loopback-only) |

**Exit criteria:** NFR-PERF-01/02/03/07 met on the reference hardware;
NFR-UX-01 onboarding E2E passes; indicator reference suite green; signed release
with SBOM and provenance; docs site with install guide.

**Launch:** announce on r/selfhosted, r/algotrading and Show HN, with a demo GIF
(fake connector).

## M2 — Alerts and stocks (v0.2–v0.3)

**Goal:** 24/7 alerts on a self-hosted server; first stock data.

| Area | Requirements |
|---|---|
| Alerts | FR-ALERT-01, 02, 04–10 |
| Server mode and auth | FR-USER-02, 06, 07, 08; ASVS L2 checklist started |
| Data | FR-DATA-03 (Alpaca), 10 (CSV import), 11 (US equity sessions) |
| Charting | FR-CHART-06, 10, 12 |
| API | FR-API-04 |
| Ops | FR-OPS-04 (backup/restore), observability dashboards and alert rules |

**Exit criteria:** QS3, QS4 and QS5 pass in nightly CI; NFR-PERF-04/05 and
NFR-REL-01…05 met; SSRF and webhook-signing tests green; upgrade test from v0.1.

## M3 — Desktop and teams (v0.4–v0.5)

| Area | Requirements |
|---|---|
| Desktop | FR-OPS-07 (Wails v3, signed if funded) |
| Teams | FR-USER-03 (OIDC), 04 (passkeys, TOTP), 05 (roles and sharing) |
| Charting | FR-CHART-09 (multi-chart grid and sync), FR-ALERT-03, 11 (drawing alerts, draggable alert lines) |
| Data | FR-DATA-12 (corporate actions), 15 (CSV/Parquet export); OANDA, Bybit, OKX connectors |
| Ops | Helm chart |

**Exit criteria:** desktop smoke tests pass on 3 operating systems; WCAG 2.2 AA
audit of all routes; multi-user authorisation matrix tests complete.

## M4 — Extensibility (v0.6+)

| Area | Requirements |
|---|---|
| Scripting | FR-IND-06, 07: the script-engine spike (ADR-0011), then implementation |
| Plugins | Out-of-process connector plugins (ADR-0016) |
| Charting | FR-CHART-15 (Heikin-Ashi, Renko, range) |
| Analysis | Backtesting design (new ADR) |

## v1.0 — Stability

**Criteria:** public APIs (HTTP v1, WS v1, connector contract, config, export
format) frozen under SemVer; security review completed; documented upgrade
guarantees; at least two active maintainers or a documented succession plan.

## Explicitly not planned

See [vision §5](vision.md#5-non-goals). Changing that list requires an ADR.
