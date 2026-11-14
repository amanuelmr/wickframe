# 14 · Quality Requirements and Test Strategy

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |
| **Requirements** | NFR-MAINT-01/02, FR-IND-02, all NFR-PERF/REL |

## 1. Quality scenarios (arc42 §10)

Each scenario names a stimulus, an expected response, and how it is verified.

| ID | Quality | Scenario | Expected response | Verified by |
|---|---|---|---|---|
| QS1 | Correctness | A user adds RSI(14) on BTCUSDT 1h and creates an alert "RSI crosses above 70". | The chart value and the alert-evaluated value are identical for every bar; the alert fires on exactly the bar where the chart shows the cross. | Indicator path-equivalence tests; E2E "chart vs alert" test with a fake connector |
| QS2 | Correctness | A source sends a corrected closed candle (`rev` + 1). | The stored candle updates; indicators replay from a snapshot; the chart refreshes that bar; alerts re-evaluate it without double-firing. | Integration test with a scripted fake connector |
| QS3 | Reliability | Network loss for 10 minutes while 50 alerts are active. | On recovery: the gap is backfilled, missed bars are evaluated in order, firings are marked `late`, notifications are sent once each. | Chaos test (toxiproxy-style fault injection) in nightly CI |
| QS4 | Reliability | The process is killed (SIGKILL) mid-firing. | After restart, no duplicate events; pending outbox rows are delivered. | Crash-recovery integration test |
| QS5 | Performance | 200 WS clients, 500 instruments, 10% of clients throttled to a slow consumption rate. | p95 added latency ≤ 100 ms for healthy clients; memory bounded; slow clients disconnected with 1013. | Nightly load test (k6 or a Go load generator) |
| QS6 | Performance | Opening a layout with 5k bars and 3 indicators. | First candle paint ≤ 1.0 s p95. | Playwright performance test |
| QS7 | Operability | Upgrade from the previous minor version with a populated database. | Migrations apply; `storetest` passes; no data loss. | Upgrade test in the release pipeline |
| QS8 | Security | An unauthenticated request to another user's layout ID. | `404`; an audit entry if authenticated. | Generated authorisation test suite |
| QS9 | Security | A webhook URL resolving to `169.254.169.254`. | Connection refused before any bytes are sent; the user sees a clear error. | Unit test on the SSRF dialer |
| QS10 | Usability | A first-time user runs the binary. | A live BTC chart within 5 minutes, without any config. | Scripted onboarding E2E test |

## 2. Test pyramid

| Level | Scope | Tools | Runs |
|---|---|---|---|
| **Unit** | Pure functions and domain logic: aggregation, indicators, condition compiler, state machines, protocol codec | Go `testing` (+ `testing/synctest` for time and concurrency), `go-cmp`; Vitest | Every PR |
| **Property-based** | Aggregation laws, candle invariants, pagination stitching | `pgregory.net/rapid` | Every PR |
| **Fuzz** | Source message parsers, WS protocol decoder, condition expressions, CSV import | Go native fuzzing | 30 s per target on PR; longer nightly |
| **Contract** | Repository implementations (`storetest`, both dialects); connector conformance (`connectortest`) | Go + testcontainers-go (PostgreSQL) | Every PR |
| **Integration** | Module wiring through the bus with a **fake connector** (scriptable deterministic market) | Go | Every PR |
| **API** | Every OpenAPI operation: authZ matrix, validation, Problem Details shape | Go tests using the generated client | Every PR |
| **E2E** | Critical user journeys in real browsers against a real binary | Playwright (Chromium, Firefox, WebKit) | Every PR (Chromium); nightly (all) |
| **Visual regression** | Chart rendering, drawings, themes | Playwright screenshots | Every PR (affected areas) |
| **Accessibility** | WCAG 2.2 AA automated checks | axe-core | Every PR |
| **Performance and load** | QS5, QS6, benchmarks | Go benchmarks + benchstat; load generator; Playwright traces | Nightly; release gate |
| **Chaos / recovery** | QS3, QS4 | Fault injection, process kill | Nightly |
| **Upgrade** | QS7 | Snapshot DBs from previous releases | Release pipeline |

### 2.1 The fake connector

`internal/marketdata/connector/fake` is a first-class connector driven by a script
(YAML or Go) describing instruments, bars, ticks, disconnects, corrections and rate
limit responses at given times. Running under `synctest` or a simulated clock, it
makes complex real-time scenarios **deterministic** and fast. It is also used for
demos and screenshots, and needs no network.

### 2.2 Recorded fixtures for real connectors

Real connectors are tested against **recorded** HTTP and WebSocket sessions
(`testdata/connectors/<name>/`), sanitised of any credentials. A nightly,
non-blocking **live smoke test** checks real public endpoints (crypto only, no
keys) to detect upstream API changes early.

## 3. Coverage and quality gates

| Gate | Threshold |
|---|---|
| Go line coverage, domain packages (`marketdata`, `indicator`, `alert`, `notify`) | ≥ 80%, ratcheted (never decreases) |
| Go line coverage, overall | ≥ 60%, ratcheted |
| Frontend `chart/` and `realtime/` coverage | ≥ 80% |
| golangci-lint v2 | Zero issues (configured linters below) |
| Biome | Zero errors |
| govulncheck | No reachable vulnerabilities (or a documented exception with expiry) |
| Benchmarks | No regression > 20% without approval |
| Bundle size | Within budget ([frontend §7](10-frontend.md#7-performance-budgets)) |

**golangci-lint v2 linters** (initial set): `govet`, `staticcheck`, `errcheck`,
`ineffassign`, `unused`, `gosec`, `bodyclose`, `contextcheck`, `errorlint`,
`exhaustive`, `gocritic`, `misspell`, `nilerr`, `noctx`, `prealloc`, `revive`,
`rowserrcheck`, `sqlclosecheck`, `unconvert`, `unparam`, `wrapcheck` (domain
boundaries only), `forbidigo` (forbids `fmt.Print*`, the `log` package, and
`time.Now` outside `platform/clock`), and `depguard` (layering rules complementing
the architecture test). Formatters: `gofumpt`, `goimports`.

## 4. Test data policy

* No real credentials, ever, in fixtures (secret scanning enforces this).
* Market data in the repo is synthetic, or explicitly redistributable (ADR-0006
  rule 6), with provenance documented in each dataset's README.
* Time-dependent tests use the injected `Clock`. Tests never sleep on wall-clock
  time (use `synctest`).

## 5. Definition of "release-ready"

A release candidate passes: all PR gates; the nightly suites (load, chaos, all
browsers, fuzz without new crashers); the upgrade test from the previous minor
version; a manual exploratory checklist for new features; and updated docs and
changelog.
