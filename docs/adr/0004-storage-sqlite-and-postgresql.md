---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0004: Support SQLite (personal mode) and PostgreSQL (server mode) behind one store interface

## Context and Problem Statement

Personal mode must work with zero setup (NFR-UX-01). Server mode must handle many
users, durable alert state, and larger candle volumes (NFR-PERF-04/05/07), using a
database operators already know how to back up. Which databases, drivers, query
tooling and migration tooling do we use, and how do we support two SQL dialects
without duplicating business logic?

## Decision Drivers

* Zero-configuration first run; static `CGO_ENABLED=0` binaries (ADR-0003).
* Production-grade concurrency and operations in server mode.
* Type-safe SQL, with no hand-written row scanning.
* One set of business logic and one set of behavioural tests across both dialects.

## Considered Options

* **SQLite + PostgreSQL**, each with its own sqlc-generated adapter behind a shared repository interface
* PostgreSQL only (embedded or required)
* SQLite only
* An ORM (GORM or ent) abstracting both dialects

## Decision Outcome

Chosen option: **SQLite + PostgreSQL behind repository interfaces**.

| Concern | Choice | Version at decision | Notes |
|---|---|---|---|
| SQLite driver | `modernc.org/sqlite` | v1.62.0 (2026-11-07) | Pure Go: no cgo, so cross-compiling is trivial. Revisit `ncruces/go-sqlite3` when it reaches 1.0. |
| PostgreSQL driver | `github.com/jackc/pgx/v5` | v5.11.0 (2026-09-07) | Native protocol, pooling via `pgxpool`. Minimum supported server: PostgreSQL 16. |
| Query generation | `sqlc` | v1.31.1 (2026-04-22) | One sqlc config per dialect, generating `internal/store/sqlite/db` and `internal/store/postgres/db`. sqlc cannot share one query set across engines. |
| Migrations | `pressly/goose` v3 | v3.30.1 (2026-11-12) | SQL migrations embedded with `embed.FS`, in separate directories `migrations/sqlite` and `migrations/postgres`. Forward-only in production (FR-OPS-02). |

**Structure:**

```text
internal/<module>/            domain: defines repository interfaces it needs
internal/store/sqlite/        implements every interface for SQLite (sqlc + adapters)
internal/store/postgres/      implements every interface for PostgreSQL (sqlc + adapters)
internal/store/storetest/     ONE shared contract-test suite, run against both
```

**Portability rules** (so both schemas stay equivalent):

1. Timestamps are stored as `BIGINT` milliseconds since the Unix epoch, in UTC.
2. Identifiers are 128-bit **UUIDv7** values (time-ordered, index-friendly), stored
   as `BLOB(16)` in SQLite and `uuid` in PostgreSQL. High-volume candle rows use
   compact integer keys instead (see [storage](../architecture/08-storage.md)).
3. Prices and volumes are `float64` (`REAL` / `DOUBLE PRECISION`). This is the
   standard for charting and technical analysis: indicators are floating-point
   computations, and IEEE-754 double precision (≈15–17 significant digits) exceeds
   the precision of any market's quotes. Prices are rounded to the instrument's
   tick size **only for display**. Wickframe performs no accounting, so exact
   decimal arithmetic is not required (vision: no order execution).
4. JSON documents (layouts, indicator settings) are stored as `TEXT` in SQLite and
   `jsonb` in PostgreSQL, validated against versioned JSON Schemas in the
   application.
5. No dialect-specific features in shared semantics. Dialect-specific *performance*
   features are allowed inside an adapter: SQLite `WITHOUT ROWID` and WAL mode;
   PostgreSQL range partitioning; optional TimescaleDB hypertables when the
   extension is detected (see [storage](../architecture/08-storage.md)).
6. Every new migration ships for both dialects in the same pull request, and CI
   verifies the two schemas are equivalent.

**SQLite configuration:** WAL journal mode, `synchronous=NORMAL`,
`foreign_keys=ON`, `busy_timeout=5000`. One writer connection plus a pool of
reader connections.

### Consequences

* Good, because personal mode needs no setup, while server mode gets a robust,
  well-understood database.
* Good, because sqlc gives compile-time-checked, reflection-free data access.
* Good, because the contract-test suite guarantees identical behaviour.
* Bad, because every query and migration is written twice. This is accepted: the
  data model is modest, and AI-assisted development keeps the duplication cheap.
  The contract tests catch drift.
* Bad, because pure-Go SQLite is somewhat slower than the cgo driver. This is
  acceptable against our latency budgets, and it is re-measured during M1
  performance testing.

### Confirmation

* `storetest` runs in CI against SQLite (in-process) and PostgreSQL
  (testcontainers-go).
* A schema-equivalence check compares the migrated SQLite and PostgreSQL schemas
  (table and column names, nullability).

## Pros and Cons of the Options

### PostgreSQL only

* Good, because there is one dialect.
* Bad, because personal and desktop mode would need a bundled database server,
  which breaks the zero-setup and single-binary goals.

### SQLite only

* Good, because it is simplest.
* Bad, because a single writer limits multi-user server mode, and operators expect
  PostgreSQL for 24/7 multi-user services.

### ORM (GORM, ent)

* Good, because it abstracts dialects.
* Bad, because of runtime reflection, hidden queries, weaker control over
  time-series access patterns, and painful migrations at scale.

## More Information

* sqlc language support: <https://docs.sqlc.dev/en/latest/reference/language-support.html>
* goose: <https://github.com/pressly/goose>
* TimescaleDB editions (Apache-2.0 vs Timescale License features): <https://www.tigerdata.com/docs/about/latest/timescaledb-editions>
* Revisit if a third storage backend is requested, or if sqlc gains multi-dialect support.
