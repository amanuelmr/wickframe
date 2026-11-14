---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0016: Built-in connectors in-process; third-party connectors as out-of-process plugins (M4)

## Context and Problem Statement

Data connectors are Wickframe's main extension point (quality goal Q6). Built-in
connectors are maintained in this repository. The community will want connectors
for sources we don't maintain, and some sources may not suit the main distribution
(ADR-0006 rule 4). A buggy or malicious third-party connector must not crash or
compromise the core process. How are connectors loaded?

## Decision Drivers

* The hot-path performance of built-in connectors.
* Fault isolation for third-party code.
* A stable, versioned, language-neutral contract for external authors.
* Go's `plugin` package is not viable: it needs cgo, is not supported on Windows,
  and requires an exact toolchain and dependency match.

## Considered Options

* Everything in-process, compiled in (no third-party plugins)
* Go's `plugin` package (shared objects)
* **Out-of-process plugins over gRPC** with `hashicorp/go-plugin` (the model used
  by Terraform, Vault, Packer and Grafana backend plugins)
* WASM plugins (wazero)

## Decision Outcome

* **M1–M3:** all connectors are **built in and in-process**, implementing the Go
  `marketdata.Connector` interface ([market data](../architecture/04-market-data.md)).
  The interface is written so it can be mapped one-to-one onto a protobuf service.
* **M4:** third-party connectors run as **separate processes** launched and
  supervised by Wickframe through **`hashicorp/go-plugin`** (v1.8.x, MPL-2.0,
  which is compatible with AGPL-3.0), over gRPC with a versioned protobuf contract
  (`wickframe.connector.v1`). A plugin crash only marks that data source as down,
  and the supervisor restarts it with backoff.
* Plugins are discovered in a configured plugin directory. They must be explicitly
  enabled by the operator, and their checksum is pinned in config. Wickframe never
  downloads plugins automatically.
* A **connector conformance kit** (`pkg/connectortest`) runs the same tests
  against in-process and plugin connectors.

### Consequences

* Good, because built-ins stay fast, and external code is isolated.
* Good, because plugins can be written in any language that speaks gRPC.
* Bad, because plugin data crosses a process boundary with serialisation cost.
  Acceptable for market-data rates, and benchmarked in M4.
* Bad, because a public protobuf contract must be versioned with care (SemVer,
  [workflow](../engineering/development-workflow.md#6-versioning-and-releases)).

### Confirmation

* M4 exit criteria: the Binance connector also runs as a plugin, passes the
  conformance kit, and meets latency budgets within 2× of the in-process version.

## More Information

* hashicorp/go-plugin: <https://github.com/hashicorp/go-plugin>
* Go `plugin` package limitations: <https://pkg.go.dev/plugin#hdr-Warnings>
