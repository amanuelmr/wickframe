---
status: accepted
date: 2026-11-14
decision-makers: "@amanuelmr"
---

# ADR-0020: Write Parquet exports with Apache arrow-go

## Context and Problem Statement

Users can export stored candles to CSV and Parquet (FR-DATA-15, M3). Exports can
hold millions of rows, so they must stream. The binary is built with
`CGO_ENABLED=0` ([ADR-0015](0015-supply-chain-and-release.md)), so the library must
be pure Go. Which Parquet library do we use?

## Decision Drivers

* Pure Go, no cgo.
* Files readable by common tools: DuckDB, pandas/pyarrow, polars, Spark.
* Streaming writes in row groups, with zstd or snappy compression.
* Maintenance health.

## Considered Options

Facts verified 2026-11-14.

| Option | Version | Licence | Maintenance | Notes |
|---|---|---|---|---|
| **`github.com/apache/arrow-go/v18`** (`pqarrow`) | v18.8.0 (2026-09-04) | Apache-2.0 | Apache Software Foundation, 326 commits in 90 days | Reference Go implementation, same family as pyarrow. Column-oriented record batches. Requires Go 1.26+. |
| `github.com/parquet-go/parquet-go` | v0.32.0 (2026-08-10) | Apache-2.0 | Active, smaller team | Simple generic row writer (`GenericWriter[T]`). Some open correctness bugs in optional and list columns (none affect flat required numeric columns). |
| `github.com/xitongsys/parquet-go` | v1.6.2 (2021) | Apache-2.0 | Abandoned | Rejected. |

## Decision Outcome

Chosen option: **Apache arrow-go v18 (`pqarrow`)**, because it is the reference
implementation with the strongest interoperability guarantees, and column-oriented
batches fit OHLCV data naturally. Go 1.27 meets its requirement.

Export format:

* Columns: `time` as `TIMESTAMP(MICROS, UTC)`, `open`, `high`, `low`, `close`,
  `volume` as `DOUBLE`, plus `source`, `symbol`, `timeframe` as dictionary-encoded
  strings. All required (non-null).
* zstd compression, row groups of about 128,000 rows, written as a stream from the
  database cursor.
* File metadata records the Wickframe version and the export parameters.
* The library is used only in the export module, behind an interface, so it can be
  replaced by `parquet-go/parquet-go` if needed.

### Consequences

* Good, because exports open directly in standard data tools.
* Bad, because arrow-go is a large dependency that increases binary size. To be
  measured when implemented; if the increase is unacceptable, `parquet-go` is the
  fallback.

### Confirmation

* A CI round-trip test writes an export and reads it back with DuckDB, checking
  the schema and values.
* The binary size change is recorded in the pull request that adds the export.

## More Information

* arrow-go: <https://github.com/apache/arrow-go>
* parquet-go: <https://github.com/parquet-go/parquet-go>
* Parquet format: <https://parquet.apache.org/docs/>
* Related: [08 Storage](../architecture/08-storage.md).
