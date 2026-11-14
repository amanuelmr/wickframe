---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0003: Build the backend as a Go modular monolith — one binary, selectable roles

## Context and Problem Statement

Wickframe must run as a desktop app, as a one-container homelab install, and as a
multi-user server (FR-OPS-01, NFR-UX-01). It is built by a very small team. How
should the backend be structured and deployed, and in what language?

## Decision Drivers

* One artifact that is trivial to install, upgrade and back up.
* Low memory and CPU use for 24/7 self-hosting (NFR-PERF-06).
* Strong concurrency for many live streams and WebSocket clients (NFR-PERF-04).
* A latency-sensitive hot path: source update → candle → indicator → alert or
  browser (NFR-PERF-03/05).
* Static, cross-compiled binaries for five platform targets (NFR-MAINT-03).
* A credible path to scale out later, without paying for it now.
* Maintainer expertise.

## Considered Options

* **A.** Go modular monolith, single process only
* **B.** Go modular monolith, one binary with **selectable roles** (`--target`)
* **C.** Go microservices (connectors, API, alerts as separately deployed services plus a broker)
* **D.** Rust monolith
* **E.** Node.js/TypeScript monolith

## Decision Outcome

Chosen option: **B — a Go modular monolith shipped as one statically linked binary
(`CGO_ENABLED=0`) with the web UI embedded, whose modules can be started
selectively by role.**

### Language and toolchain

Go. The current stable release at decision time is Go 1.27 (released 2026-08-19;
patch 1.27.4). `go.mod` declares `go 1.27` and pins the exact toolchain with a
`toolchain` directive. Each Go release is supported until two newer major releases
exist ([release policy](https://go.dev/doc/devel/release)), so we adopt each new Go
release within one Wickframe minor release.

### Modular

The code is split into modules (`internal/marketdata`, `internal/indicator`,
`internal/alert`, …) with explicit Go interfaces and enforced import rules
([components](../architecture/02-components.md)):

* A module accesses only its own tables, through its own repository interface.
* Modules interact only through each other's public interfaces or bus events
  ([ADR-0008](0008-realtime-transport.md)), never by reaching into internals.
* Domain code never imports adapters (hexagonal architecture).

This is the same principle as Shopify's componentized modular monolith: one
codebase, with strict boundaries checked by tooling.

### Selectable roles

The binary groups its modules into **roles**. By default, `wickframe serve` runs
**all roles in one process**, which is the only supported deployment for M1–M4.
The `--target` flag (config `server.targets`) selects a subset, following the
model used by Grafana Mimir and Tempo (one binary; `-target=all` is the default
monolithic mode):

| Role | Modules |
|---|---|
| `api` | HTTP API, WebSocket gateway, web UI, auth |
| `ingest` | connectors, candle pipeline, persistence |
| `alerts` | alert engine, notification delivery |
| `all` (default) | everything above |

Running roles in separate processes requires a **network-capable bus** and shared
PostgreSQL. That is *not* built now. It will be specified in a future ADR when
NFR-PERF-04/05 can no longer be met by one node. What we do now is cheap and
protects that path:

1. Modules talk only through interfaces and the bus, so moving a module across a
   process boundary means swapping the adapter, not rewriting it.
2. Role wiring lives in one place (`internal/app`), and roles start and stop
   independently under a supervisor.
3. Bus messages are versioned, serialisable structs (no pointers to shared mutable
   state).

### Extensions

Built-in data connectors run **in-process** for performance. A future
out-of-process plugin mechanism for third-party connectors is defined separately
in [ADR-0016](0016-connector-extension-model.md), so a faulty community connector
cannot crash the core.

### Subcommands

One binary covers all operations: `serve`, `migrate`, `backup`, `restore`,
`user`, `openapi`, `version`, `doctor`.

### Consequences

* Good, because there is one artifact, one process, one log stream and one thing to
  monitor. This matches the self-hosting audience.
* Good, because the hot path stays in memory with no serialisation or network hops.
* Good, because Go's goroutines and channels suit stream fan-out; the binaries are
  small and static; and cross-compilation is built in.
* Good, because selectable roles give a scale-out path validated by mature
  projects, without distributed-systems cost today.
* Bad, because a bug in one module can crash the whole process. Mitigated by
  panic-recovery boundaries around connectors and jobs, supervised restarts, and
  out-of-process third-party plugins (ADR-0016).
* Bad, because boundaries depend on discipline. Mitigated by an automated
  architecture test, see Confirmation.

### Confirmation

* An **architecture test** (a Go test that parses package imports) fails CI if a
  domain package imports an adapter, or if a module imports another module's
  non-public packages.
* CI builds `CGO_ENABLED=0` binaries for all five targets.
* CI starts `wickframe serve --target=api,ingest,alerts` (explicit list) and
  `--target=all`, and verifies both behave identically in the E2E smoke test.

## Pros and Cons of the Options

### A. Single-process monolith without roles

* Good, because it is the simplest.
* Bad, because it offers no clean path to scaling out; retrofitting role boundaries
  later is expensive.

### C. Microservices

* Good, because modules could scale and deploy independently.
* Bad, because it means many artifacts, a broker, and distributed failure modes.
  It contradicts the single-install goal and overwhelms a small team.
* Bad, because service boundaries must be guessed before the product exists.
  Martin Fowler's *MonolithFirst* argues for discovering boundaries in a monolith
  first.
* Bad, because the hot path would pay network and serialisation costs. Google's
  *Towards Modern Development of Cloud Applications* argues that microservices
  conflate logical and physical boundaries, and Amazon Prime Video reported a large
  cost reduction after consolidating a distributed monitoring service into one
  process.

### D. Rust monolith

* Good, because it offers the best raw performance and memory safety.
* Bad, because it means slower iteration, a smaller contributor pool for this
  domain, and no maintainer expertise.

### E. Node.js monolith

* Good, because it would share one language with the frontend.
* Bad, because of higher memory use, weaker CPU-bound performance for indicators
  and alerts, and harder single-binary distribution.

## More Information

* Martin Fowler, *Monolith First*: <https://martinfowler.com/bliki/MonolithFirst.html>;
  counterpoint: <https://martinfowler.com/articles/dont-start-monolith.html>
* Shopify, *Deconstructing the Monolith*: <https://shopify.engineering/deconstructing-monolith-designing-software-maximizes-developer-productivity>
* Shopify, *Enforcing Modularity with Packwerk*: <https://shopify.engineering/enforcing-modularity-rails-apps-packwerk>
* Google, *Towards Modern Development of Cloud Applications* (HotOS 2023), summary: <https://golab.io/talks/towards-modern-development-of-cloud-applications>
* Prime Video case (InfoQ): <https://www.infoq.com/news/2023/05/prime-ec2-ecs-saves-costs>
* Grafana Mimir deployment modes: <https://grafana.com/docs/mimir/v2.17.x/references/architecture/deployment-modes/>
* Go module layout: <https://go.dev/doc/modules/layout>
* **Revisit when:** a single node can no longer meet NFR-PERF-04/05 on reasonable
  hardware, or high availability becomes a requirement.
