# Architecture Decision Records

Significant decisions are recorded here as [MADR 4.0](https://adr.github.io/madr/)
files. The process is defined in [ADR-0001](0001-record-architecture-decisions.md).
Start new records from the [template](template.md).

| ADR | Title | Status | Date |
|---|---|---|---|
| [0001](0001-record-architecture-decisions.md) | Record architecture decisions as MADR files in the repository | Accepted | 2026-11-13 |
| [0002](0002-license-agpl-3.0.md) | License under AGPL-3.0-only, with DCO sign-off | Accepted | 2026-11-13 |
| [0003](0003-go-modular-monolith.md) | Go modular monolith — one binary, selectable roles | Accepted | 2026-11-13 |
| [0004](0004-storage-sqlite-and-postgresql.md) | SQLite (personal) and PostgreSQL (server) behind one store interface | Accepted | 2026-11-13 |
| [0005](0005-chart-engine.md) | Lightweight Charts v5 behind a ChartEngine adapter, with our own drawing layer | Accepted | 2026-11-13 |
| [0006](0006-bring-your-own-market-data.md) | Bring your own market data — never provide or redistribute data | Accepted | 2026-11-13 |
| [0007](0007-http-api-openapi-huma.md) | Code-first OpenAPI 3.1 HTTP API with huma on `net/http` | Accepted | 2026-11-13 |
| [0008](0008-realtime-transport.md) | WebSocket transport with an in-process conflating event hub | Accepted | 2026-11-13 |
| [0009](0009-desktop-shell.md) | Desktop app with Wails v3 | Accepted | 2026-11-13 |
| [0010](0010-single-indicator-engine.md) | One Go indicator engine for charts and alerts | Accepted | 2026-11-13 |
| [0011](0011-scripting-engine.md) | expr-lang for alert conditions; script-engine spike in M4 | Accepted | 2026-11-13 |
| [0012](0012-frontend-stack.md) | Frontend stack (React 19, TS 7, Vite 8, TanStack, Zustand, Tailwind 4, Biome) | Accepted | 2026-11-13 |
| [0013](0013-observability.md) | slog, Prometheus, opt-in OpenTelemetry | Accepted | 2026-11-13 |
| [0014](0014-authentication.md) | Sessions, Argon2id, opaque API tokens, OIDC, passkeys | Accepted | 2026-11-13 |
| [0015](0015-supply-chain-and-release.md) | Supply chain: GoReleaser, cosign, SBOM, provenance, distroless | Accepted | 2026-11-13 |
| [0016](0016-connector-extension-model.md) | Built-in connectors in-process; third-party plugins out-of-process (M4) | Accepted | 2026-11-13 |
| [0017](0017-cross-process-event-bus.md) | Cross-process event bus: PostgreSQL NOTIFY doorbell first, embedded NATS at scale | Accepted | 2026-11-14 |
| [0018](0018-user-script-engine.md) | Starlark as the user-script engine (pending the M4 spike) | Proposed | 2026-11-14 |
| [0019](0019-typescript-api-client-generator.md) | Generate the web UI's API client with orval | Accepted | 2026-11-14 |
| [0020](0020-parquet-export-library.md) | Write Parquet exports with Apache arrow-go | Accepted | 2026-11-14 |

## Pending decisions

| Topic | Needed by | Notes |
|---|---|---|
| Confirm or replace the user-script engine | M4 | [ADR-0018](0018-user-script-engine.md) is *proposed*; the spike criteria are listed there |
| Backtesting design | After v1.0 | See the [roadmap](../product/roadmap.md) |
