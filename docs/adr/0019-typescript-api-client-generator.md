---
status: accepted
date: 2026-11-14
decision-makers: "@amanuelmr"
---

# ADR-0019: Generate the web UI's API client with orval

## Context and Problem Statement

The backend publishes an OpenAPI 3.1 document (`api/openapi.yaml`,
[ADR-0007](0007-http-api-openapi-huma.md)). The web UI must call the API through a
generated, type-safe client, so frontend and backend cannot drift
([ADR-0012](0012-frontend-stack.md)). Which generator do we use?

## Decision Drivers

* Correct OpenAPI 3.1 handling: `type` arrays with `"null"`, `const`, `oneOf`,
  `$ref` with sibling keywords.
* Works with **TypeScript 7** (the Go-based compiler) without a second compiler.
* Integrates with TanStack Query v5 and native `fetch`.
* Typed RFC 9457 Problem Details errors (`application/problem+json`).
* Maintenance health and more than one active maintainer.

## Considered Options

Facts verified 2026-11-14.

| Option | Version | Output | TypeScript 7 | Maintenance |
|---|---|---|---|---|
| `openapi-typescript` + `openapi-fetch` | 7.13.0 / 0.17.0 | Types plus a small fetch runtime | Broken (uses the TypeScript compiler API) | **Maintenance mode** since 2026-09-19, no further updates including security fixes (issue #2874) |
| `@hey-api/openapi-ts` | 0.99.0 (stable) | Types, SDK, query hooks | Broken on stable; fixed only on the `next` channel | One maintainer, 557 open issues, several open 3.1 bugs (#4477, #4387, #4412) |
| **`orval`** | 8.47.0 | Types, fetch functions, TanStack Query hooks, optional Zod and MSW | Works (does not use the compiler API) | Several active contributors, 17 open issues, 434 commits in 90 days |
| `kubb` | 5.20.0 | Types, client, TanStack Query hooks | Works | Strong 3.1 fixes, but effectively one maintainer and near-daily major-version churn |

## Decision Outcome

Chosen option: **orval 8.x**, configured with `client: 'react-query'` and
`httpClient: 'fetch'`, because it is the only well-maintained option that works
with TypeScript 7 today and has native TanStack Query support.

* Generated code lives in `web/src/api/generated/`, is committed, and is never
  edited by hand ([repository layout](../engineering/repository-layout.md)).
* `make generate` runs `wickframe openapi > api/openapi.yaml` and then orval.
* A custom fetch mutator adds the CSRF header, credentials and base URL, and
  converts `application/problem+json` responses into a typed `ProblemError`.
  orval treats any `+json` content type as JSON, so problem bodies are typed per
  operation.
* orval is pinned to an exact version, and updated through Renovate.

**Fallback:** kubb, if orval stops meeting the drivers.

### Consequences

* Good, because API changes become compile errors in the UI.
* Good, because no client runtime library is shipped: generated code uses native
  `fetch`.
* Bad, because generator bugs in 3.1 edge cases are possible (for example orval
  issue #4522, arrays without `items`). The backend avoids such schemas, and CI
  catches regressions.

### Confirmation

* CI regenerates the client and fails if the result differs from what is committed.
* CI type-checks the generated code with TypeScript 7.
* API contract tests in the frontend use MSW handlers generated from the same spec.

## More Information

* orval: <https://orval.dev/>, <https://github.com/orval-labs/orval>
* openapi-typescript maintenance notice: <https://github.com/openapi-ts/openapi-typescript/issues/2874>
* hey-api TypeScript 7 issue: <https://github.com/hey-api/hey-api/issues/4235>
* kubb: <https://github.com/kubb-labs/kubb>
* Related: [ADR-0007](0007-http-api-openapi-huma.md), [ADR-0012](0012-frontend-stack.md).
* Revisit if orval's maintenance declines or if a generator emerges from the
  TypeScript or OpenAPI projects themselves.
