# Go Coding Standards

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |

Baseline: [Effective Go](https://go.dev/doc/effective_go),
[Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) and the
[Google Go Style Guide](https://google.github.io/styleguide/go/). The rules below
are Wickframe-specific additions or emphasis. Linters enforce what they can
([quality §3](../architecture/14-quality.md#3-coverage-and-quality-gates)).

## 1. Formatting and naming

* `gofumpt` + `goimports` formatting; no exceptions.
* Package names: short, lower case, no underscores, not plural, and never `util`,
  `common` or `helpers`.
* Exported identifiers have doc comments that start with the name. Every package
  has a `doc.go` describing its responsibility and its place in the
  [component model](../architecture/02-components.md).
* Every source file starts with `// SPDX-License-Identifier: AGPL-3.0-only`.

## 2. Errors

* Wrap with context using `%w`: `fmt.Errorf("load candles %s/%s: %w", key, tf, err)`.
* **Sentinel errors and types for domain conditions** (`ErrNotFound`,
  `ErrConflict`, `*ValidationError`). Callers branch with `errors.Is` and
  `errors.As`, never with string matching.
* The HTTP layer maps domain errors to Problem Details in **one** place.
* Log or return, never both (log once at the boundary).
* No `panic` for expected failures. Panics are recovered only at supervision and
  request boundaries.

## 3. Context, goroutines, concurrency

* `ctx context.Context` is the first parameter of any function that blocks, does
  I/O or may be cancelled. It is never stored in structs.
* **Every goroutine has an owner** that can stop it and waits for it
  (`errgroup`, `sync.WaitGroup.Go`, or the supervisor). No fire-and-forget
  `go func()`.
* Prefer channels for ownership transfer and mutexes for protecting state. Do not
  mix both on the same data. Document which goroutine owns which state.
* Bounded queues only. Every channel buffer size is a named constant with a
  justification comment.
* Use `testing/synctest` for time- and concurrency-dependent tests.

## 4. Time

* Never call `time.Now()` directly in domain code. Use the injected
  `clock.Clock` (enforced by `forbidigo`).
* Internal timestamps are `int64` Unix milliseconds UTC, or `time.Time` in UTC at
  API edges. Durations use `time.Duration`. Time zones only via
  `time.LoadLocation` with IANA names, at session and display edges.

## 5. Interfaces and dependencies

* **Define interfaces where they are consumed** (the domain), keep them small, and
  return concrete types from constructors.
* Constructors take explicit dependencies (`New(repo Repository, clk clock.Clock,
  log *slog.Logger) *Service`). No global state, no `init()` side effects (except
  registering built-in descriptors), and no service locators.
* Respect the import rules in [components §3](../architecture/02-components.md#3-dependency-rules);
  `internal/archtest` fails the build otherwise.

## 6. Data and numbers

* Prices are `float64` (ADR-0004). Never use `==` to compare computed floats in
  logic; compare against tolerances or use exact inputs. Display rounding happens
  at the edge, using tick size.
* Parse external numeric strings with `strconv.ParseFloat(s, 64)`, and reject
  NaN or Inf.
* JSON: `encoding/json/v2` (`json.Marshal`/`Unmarshal` from the v2 package) for
  new code on hot paths. Struct tags are explicit; unknown fields are rejected on
  inbound API bodies.

## 7. Database

* SQL lives in `.sql` files consumed by sqlc. String-built SQL is not allowed.
* Every query runs with a context deadline. Long scans are paginated.
* Repositories return domain types, not sqlc row types.

## 8. HTTP and outbound calls

* Outbound HTTP uses `internal/platform/httpx` clients only: timeouts set, the
  SSRF-safe dialer where URLs are user-supplied, and response bodies always closed
  and size-limited (`bodyclose`, `noctx` linters).
* Handlers are thin: decode → call a service → encode. No business logic in
  handlers.

## 9. Logging and metrics

* `*slog.Logger` injected. Use `logger.With("component", "...")` at construction.
* Attribute keys are snake_case constants from `logx`. Never log secrets (the
  redaction layer is a safety net, not a licence).
* Metric names follow [observability §2](../architecture/12-observability.md#2-metrics).
  New metrics are documented in the same pull request.

## 10. Testing

* Table-driven tests with named cases; `t.Parallel()` where safe.
* Use the standard `testing` package with `go-cmp` for diffs. `testify/require` is
  allowed for brevity; `testify/suite` is not used.
* Test names may include requirement IDs (`TestAlert_FRALERT10_NoDuplicateAfterRestart`).
* Tests needing PostgreSQL use testcontainers-go and are tagged so `go test ./...`
  stays fast. `make test-integration` runs them.
* Benchmarks for hot paths (`internal/indicator`, `internal/bus`,
  `internal/marketdata/aggregate`) are mandatory.

## 11. Dependencies

* Adding a dependency requires justification in the PR: maintenance health,
  license (allow-list, ADR-0002), size, and the alternatives considered.
* Prefer the standard library. Avoid frameworks that take over `main` or the
  request context.
