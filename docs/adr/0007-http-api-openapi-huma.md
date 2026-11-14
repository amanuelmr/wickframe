---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0007: Code-first OpenAPI 3.1 HTTP API with huma on the standard library router

## Context and Problem Statement

Everything in the UI must also be available through a versioned, documented HTTP
API (FR-API-01), with standard error responses (FR-API-03). The frontend needs a
typed client. How do we define the API, keep the documentation correct, and route
requests?

## Decision Drivers

* The OpenAPI document must never drift from the implementation.
* Typed request and response handling, with validation, in Go.
* A generated, typed TypeScript client for the frontend.
* Minimal framework lock-in; compatible with standard `net/http` middleware.

## Considered Options

* Code-first with **huma v2** on `net/http` `ServeMux` (generates OpenAPI 3.1)
* Spec-first with **oapi-codegen** (hand-written YAML, generated server stubs)
* Spec-first with **ogen**
* A plain router (chi or echo) with a hand-maintained OpenAPI document

## Decision Outcome

Chosen option: **huma v2 on the standard library `ServeMux`**, because the Go
types *are* the contract. huma generates an OpenAPI 3.1 document and JSON Schemas
from them, validates requests, and emits
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) Problem Details errors. Version
at decision: v2.40.0 (2026-10-27, MIT).

Workflow:

1. Handlers are written as typed huma operations in `internal/api/http`.
2. `wickframe openapi > api/openapi.yaml` exports the document. CI fails if the
   committed file differs from the generated one.
3. The frontend's TypeScript types and client are generated from `api/openapi.yaml`.
4. Breaking-change detection: CI compares the document with the last release using
   an OpenAPI diff tool and fails on unannounced breaking changes.

API conventions (paths, pagination, idempotency, versioning) are defined in the
[API architecture](../architecture/09-api.md).

### Consequences

* Good, because the documentation is always accurate and the clients are always in
  sync.
* Good, because handlers are plain `net/http`, so standard middleware works (OTel,
  auth, logging).
* Bad, because we depend on huma's continued maintenance. Mitigated: the handlers'
  business logic lives in services, so replacing the transport layer is bounded work.
* Bad, because code-first makes design-before-implementation reviews less natural.
  Mitigated: API changes are reviewed through the generated `openapi.yaml` diff in
  every pull request.

### Confirmation

* CI: `openapi.yaml` freshness check and breaking-change diff.
* Contract tests call the API through the generated client.

## Pros and Cons of the Options

### oapi-codegen / ogen (spec-first)

* Good, because the API design is written and reviewed before code.
* Bad, because the hand-written YAML is verbose. oapi-codegen's OpenAPI 3.1 support
  was limited at the time of evaluation (not re-verified). Two sources of truth
  need syncing on every change.

### Plain router with a hand-written spec

* Good, because it is simple.
* Bad, because the documentation drifts from the code: a classic failure mode.

## More Information

* huma: <https://github.com/danielgtaylor/huma>
* Go 1.22 routing enhancements: <https://go.dev/blog/routing-enhancements>
* Revisit if huma is abandoned, or if OpenAPI 3.2 tooling requires it.
