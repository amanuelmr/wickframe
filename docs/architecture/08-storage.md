# 08 · Storage

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |
| **Requirements** | FR-DATA-05/08/15, FR-OPS-02/04, NFR-PERF-07, NFR-REL-01 |
| **Decisions** | [ADR-0004](../adr/0004-storage-sqlite-and-postgresql.md) |

## 1. Logical schema

Types are logical: `uuid` = UUIDv7 (SQLite `BLOB(16)`, PostgreSQL `uuid`); `ms` =
`BIGINT` Unix milliseconds UTC; `json` = SQLite `TEXT`, PostgreSQL `jsonb`;
`enc` = encrypted blob (`BLOB` / `bytea`, see [security](11-security.md#5-secrets-at-rest)).

### 1.1 Market data (module: marketdata)

| Table | Columns | Keys and indexes |
|---|---|---|
| `data_sources` | `id uuid`, `connector text`, `name text`, `settings json`, `credentials enc NULL`, `enabled bool`, `created_at ms`, `updated_at ms` | PK `id` |
| `instruments` | `id int`, `source_id uuid`, `symbol text`, `display text`, `kind text`, `base text`, `quote text`, `tick_size double`, `lot_size double`, `session text`, `timezone text`, `status text`, `meta json`, `updated_at ms` | PK `id`; UNIQUE `(source_id, symbol)`; index on `display` for search |
| `candles` | `instrument_id int`, `tf smallint`, `t ms`, `o`, `h`, `l`, `c`, `v double`, `qv double NULL`, `n int NULL`, `rev smallint`, `derived bool` | **PK `(instrument_id, tf, t)`**; SQLite `WITHOUT ROWID` |
| `candle_coverage` | `instrument_id`, `tf`, `from_t ms`, `to_t ms` | PK `(instrument_id, tf, from_t)`; non-overlapping ranges known complete |
| `candle_gaps` | `id uuid`, `instrument_id`, `tf`, `from_t`, `to_t`, `reason text`, `status text`, `updated_at ms` | index `(instrument_id, tf, status)` |
| `corporate_actions` (M3) | `instrument_id`, `t ms`, `kind text`, `ratio double NULL`, `amount double NULL` | PK `(instrument_id, t, kind)` |

### 1.2 Workspace

| Table | Columns | Keys |
|---|---|---|
| `layouts` | `id uuid`, `owner_id uuid`, `name text`, `doc json`, `schema_version int`, `created_at`, `updated_at`, `version int` (optimistic concurrency) | PK `id`; index `(owner_id, updated_at)` |
| `drawings` | `id uuid`, `layout_id uuid`, `owner_id uuid`, `instrument_key text`, `tool text`, `anchors json`, `style json`, `z int`, `locked bool`, `hidden bool`, `updated_at`, `version int` | PK `id`; index `(layout_id)`, `(owner_id, instrument_key)` |
| `watchlists`, `watchlist_items` | ordered items with `position` (fractional index text) | PK; UNIQUE `(watchlist_id, section, instrument_key)` |
| `preferences` | `owner_id uuid`, `doc json`, `updated_at` | PK `owner_id` |

### 1.3 Alerts and notifications

| Table | Columns | Keys |
|---|---|---|
| `alerts` | `id`, `owner_id`, `name`, `instrument_id`, `tf`, `condition json`, `policy text`, `rearm_ms int`, `expires_at ms NULL`, `template text`, `enabled bool`, `version int`, timestamps | PK; index `(instrument_id, tf) WHERE enabled` |
| `alert_channels` | `alert_id`, `channel_id` | PK `(alert_id, channel_id)` |
| `alert_state` | `alert_id`, `last_eval_t`, `last_eval_rev`, `prev json`, `armed bool`, `last_fired_t`, `updated_at` | PK `alert_id` |
| `alert_events` | `id uuid`, `alert_id`, `bar_t ms`, `fired_at ms`, `values json`, `late bool` | PK; index `(alert_id, fired_at DESC)` |
| `notification_channels` | `id`, `owner_id`, `kind`, `name`, `config enc`, `enabled`, timestamps | PK |
| `notification_outbox` | `id uuid`, `event_id`, `channel_id`, `idempotency_key text`, `status text`, `attempts int`, `next_attempt_at ms`, `last_error text NULL`, timestamps | PK; UNIQUE `idempotency_key`; index `(status, next_attempt_at)` |
| `notification_attempts` | `id`, `outbox_id`, `at ms`, `duration_ms int`, `result text`, `http_status int NULL`, `error text NULL` | PK; index `(outbox_id)` |

### 1.4 Identity and audit

| Table | Columns | Keys |
|---|---|---|
| `users` | `id`, `username text`, `display_name`, `role text`, `status text`, timestamps | PK; UNIQUE lower(`username`) |
| `password_credentials` | `user_id`, `hash text` (PHC string with Argon2id params), `updated_at` | PK `user_id` |
| `webauthn_credentials`, `totp_secrets`, `oidc_links` | per [ADR-0014](../adr/0014-authentication.md) | |
| `sessions` | the scs store schema (`token`, `data`, `expiry`) | PK `token`; index `expiry` |
| `api_tokens` | `id`, `user_id`, `name`, `hash bytea`, `scopes text`, `expires_at`, `last_used_at`, `created_at`, `revoked_at` | PK; UNIQUE `hash` |
| `audit_log` | `id uuid`, `at ms`, `actor_id uuid NULL`, `action text`, `target text`, `ip text`, `user_agent text`, `outcome text`, `detail json` | PK; index `(at)`; **append-only** (no UPDATE or DELETE grants in PostgreSQL; enforced by repository API in SQLite) |

## 2. Candle storage

### 2.1 Sizing

A candle row is ~70 bytes of data plus index overhead. With the clustered primary
key (`WITHOUT ROWID` in SQLite, B-tree PK in PostgreSQL), budget about **110 bytes
per row**.

| Timeframe | Rows per instrument per year | Size per instrument per year |
|---|---|---|
| 1m | 525,600 | ~58 MB |
| 5m | 105,120 | ~12 MB |
| 1h | 8,760 | ~1 MB |
| 1D | 365 | negligible |

### 2.2 Retention (configurable)

| Timeframe | Default retention |
|---|---|
| `1m`, `3m` | 365 days |
| `5m`–`30m` | 5 years |
| `≥ 1h` | unlimited |

A daily maintenance job deletes expired rows in small batches. Persisted derived
timeframes remain, so long-range charts stay available after 1m data expires.
Coverage ranges are trimmed accordingly.

### 2.3 PostgreSQL scaling options

* **Baseline:** declarative **range partitioning by month** on `t`, for `tf` ≤ 30m
  (`candles_intraday`, partitioned), and a non-partitioned table for higher
  timeframes, behind a view or the repository. Retention drops whole partitions:
  instant, with no vacuum debt.
* **Optional TimescaleDB:** if the extension is detected at startup and
  `storage.timescale.enabled` is true, the intraday table becomes a hypertable.
  Columnstore compression and retention policies are used **only if the
  Timescale-License edition is present**. The Apache-2.0 edition lacks
  compression, continuous-aggregate refresh and retention policies, so Wickframe
  falls back to its own retention job. This is detected at runtime; it is never
  assumed.

## 3. Migrations

* Tool: `pressly/goose` v3, SQL migrations embedded in the binary:
  `migrations/sqlite/NNNNN_<name>.sql` and `migrations/postgres/NNNNN_<name>.sql`,
  with the **same numbers and names** in both directories (checked in CI).
* Startup: acquire a migration lock (PostgreSQL advisory lock; SQLite exclusive
  transaction) → apply pending migrations → release. If `--no-migrate` is set and
  migrations are pending, refuse to start with a clear message.
* **Forward-only** in production. Down migrations exist for development only and
  are not shipped.
* **Expand/contract** for breaking schema changes. Release N adds the new
  structure and writes both; release N+1 migrates reads; release N+2 removes the
  old structure. This keeps an upgrade-then-rollback-of-the-binary safe across one
  minor version.
* Large data migrations run as resumable background jobs, not inside the startup
  migration.
* CI tests: migrate an empty DB; migrate a DB seeded at the **previous release's**
  schema (fixture snapshot) to head; run `storetest` on both dialects.

## 4. Transactions and concurrency

* Each service method is one unit of work: `repo.WithTx(ctx, func(tx) error)`.
* Optimistic concurrency on user documents (`layouts`, `drawings`, `alerts`)
  through a `version` column. A conflicting update returns HTTP `409` with the
  current version (see [API](09-api.md#5-concurrency-and-idempotency)).
* SQLite: one writer connection (serialised writes, no `SQLITE_BUSY` storms), plus
  N reader connections in WAL mode; `busy_timeout = 5000`; `foreign_keys = ON`.
* PostgreSQL: `pgxpool` (default max connections = 4 × GOMAXPROCS, capped at 32);
  statement timeout 30 s by default; isolation `READ COMMITTED`, except where a
  repository documents otherwise.

## 5. Backup and restore (FR-OPS-04)

| Mode | `wickframe backup` | `wickframe restore` |
|---|---|---|
| SQLite | Online, consistent snapshot via `VACUUM INTO '<file>'`, then a checksum (SHA-256) and a manifest (version, schema version, time). | Stop server → verify checksum and schema version → swap the file atomically → start. |
| PostgreSQL | Wraps `pg_dump --format=custom` if available, and documents the native-tool path. | `pg_restore` into an empty database, then `wickframe migrate`. |

* Backups can exclude candle data (`--without-market-data`), since candles can be
  re-fetched. This makes user-data backups tiny.
* Restores into a **newer** Wickframe version are supported (migrations run).
  Restores into older versions are refused.
* The documentation includes a tested restore drill. CI runs backup → restore →
  `storetest` smoke on both dialects.

## 6. Data export (FR-DATA-15, FR-USER-08)

* User data export: a versioned JSON archive (`wickframe-export-v1.zip`) containing
  layouts, drawings, watchlists, alerts (without secrets) and preferences, with a
  JSON Schema for each file.
* Candles: CSV (documented columns) and Parquet (columnar; via a pure-Go Parquet
  writer chosen at M3).
