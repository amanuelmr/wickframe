---
status: accepted
date: 2026-11-14
decision-makers: "@amanuelmr"
---

# ADR-0017: Cross-process event bus — PostgreSQL NOTIFY as a doorbell first, embedded NATS when scale requires it

## Context and Problem Statement

[ADR-0003](0003-go-modular-monolith.md) lets one binary run as the roles `api`,
`ingest` and `alerts`, but today the roles talk only through the in-process event
hub ([05 Real-time](../architecture/05-realtime.md)). Running roles as separate
processes needs a network-capable bus. Which transport do we use, and when do we
adopt it?

## Decision Drivers

* Minimal operational burden for a solo maintainer and for self-hosters.
* No new infrastructure in personal (SQLite) mode, which never runs multiple
  processes.
* Alerts must stay at-least-once (NFR-REL-02). Ticks may be conflated (lossy is fine).
* The hub interface must not change; only its adapter.

## Considered Options

| Option | Status (verified 2026-11-14) | Delivery | Notes |
|---|---|---|---|
| PostgreSQL `LISTEN/NOTIFY` | PostgreSQL 18; pgx v5.13.0 | At-most-once; lost while a listener is disconnected | Payload ≈ 8,000 bytes max. Every notifying commit takes a database-wide lock, which serialises commits under load. `LISTEN` does not work through PgBouncer in transaction pooling mode. |
| Embedded NATS | nats-server v2.15.3, Apache-2.0 | Core: at-most-once. JetStream: at-least-once with acks and dedupe | Runs as a Go library (`server.NewServer`, `DontListen`, `InProcessConn`). Large dependency with a steady flow of security advisories, mostly in features we would not enable (leaf nodes, MQTT, auth callout). |
| Redis / Valkey | Valkey 9.1.3, BSD-3 | Pub/sub at-most-once; Streams at-least-once | A new server to operate, and no advantage over NATS here. |
| gRPC streams between roles | — | Whatever we build | No extra infrastructure, but we would rebuild discovery, reconnects, back-pressure, fan-out and replay: effectively a broker. |

## Decision Outcome

A staged approach, each stage behind the existing hub interface:

1. **Default: in-process hub.** `--target=all` remains the recommended topology.
   SQLite mode never uses anything else.
2. **First split-process deployment (PostgreSQL only): `LISTEN/NOTIFY` as a
   doorbell.** Payloads carry only a topic and an ID, never market data. Receivers
   re-read the source of truth (the alert outbox, the candle table). Alerts keep
   their at-least-once guarantee through the transactional outbox, not through the
   bus. Live ticks stay inside the `ingest` process; an `api` process in another
   process reads conflated snapshots. The listener uses a dedicated direct
   connection (not through PgBouncer).
3. **Adopt embedded NATS (core; JetStream only if needed)** when any of these
   triggers is met:
   * live ticks or bars must fan out across processes faster than about 1 update
     per second per symbol;
   * more than one `api` replica must push live data to clients;
   * waits on the NOTIFY commit lock appear in `pg_locks` / `pg_stat_activity`.

   Only core NATS publish/subscribe (and JetStream if required) are enabled; leaf
   nodes, MQTT, websockets and auth callout stay off. NATS can run embedded in one
   of the roles or as a separate server. The choice is documented when adopted.

Redis/Valkey and hand-written gRPC streaming are rejected.

### Consequences

* Good, because no new infrastructure is needed until a real scale signal appears.
* Good, because correctness never depends on bus delivery: the outbox and storage
  remain the source of truth.
* Good, because NATS can start embedded, so even stage 3 adds no new service to
  operate.
* Bad, because stage 2 limits what a separated `api` process can show live. This is
  accepted, since `--target=all` covers the target scale (NFR-PERF-04).
* Bad, because NATS is a large dependency to keep patched once adopted.

### Confirmation

* An integration test runs `ingest` and `alerts` as separate processes against
  PostgreSQL and checks that an alert fires exactly once per crossing (stage 2).
* Bus message types stay versioned, serialisable structs (ADR-0003 rule 3),
  checked by the architecture tests.
* The stage-3 triggers are exposed as metrics (`wickframe_bus_*`) so the decision
  is made on data.

## More Information

* PostgreSQL `NOTIFY`: <https://www.postgresql.org/docs/18/sql-notify.html>; the
  commit lock is in `src/backend/commands/async.c` (REL_18_STABLE).
* PgBouncer feature matrix: <https://www.pgbouncer.org/features.html>
* NATS server: <https://github.com/nats-io/nats-server>
* Related: [ADR-0003](0003-go-modular-monolith.md), [ADR-0008](0008-realtime-transport.md).
* Revisit if PostgreSQL removes the NOTIFY commit lock, or if a lighter embedded
  broker becomes available.
* Facts verified 2026-11-14.
