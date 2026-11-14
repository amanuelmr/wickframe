# 09 · HTTP API Conventions

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |
| **Requirements** | FR-API-01…04, NFR-MAINT-05 |
| **Decisions** | [ADR-0007](../adr/0007-http-api-openapi-huma.md), [ADR-0014](../adr/0014-authentication.md) |

These conventions apply to every endpoint. They follow established public-API
practice (Google AIP, Microsoft REST API Guidelines, Zalando RESTful API
Guidelines) adapted to Wickframe.

## 1. Basics

* Base path: **`/api/v1`**. The UI is served from `/`; the OpenAPI document from
  `/api/v1/openapi.json`, plus an interactive reference at `/api/docs` (can be
  disabled).
* JSON only (`application/json; charset=utf-8`). Field names are **camelCase**.
  Enums are lower_snake_case strings.
* Timestamps: **integer Unix milliseconds** for market data (`t`), and **RFC 3339
  UTC strings** for resource metadata (`createdAt`). Both are documented per field.
* Numbers: prices and volumes are JSON numbers (IEEE-754 double). Identifiers are
  strings (UUIDv7, or instrument keys like `binance:BTCUSDT`).
* Absent and `null` differ: `null` means "explicitly no value" (for example a
  warm-up indicator value).

## 2. Resources (v1 overview)

| Area | Endpoints (summary) |
|---|---|
| System | `GET /system/info` (version, mode, features) · `GET /system/status` (sources, jobs) |
| Data sources | `GET/POST /sources` · `GET/PATCH/DELETE /sources/{id}` · `POST /sources/{id}:test` · `GET /connectors` (available connector types and settings schemas) |
| Instruments | `GET /instruments?query=&source=&kind=&pageToken=` · `GET /instruments/{key}` |
| Candles | `GET /candles?instrument=&tf=&from=&to=&limit=` (max 5,000 per request; returns `bars`, `nextTo` for paging backwards, `historyComplete`) |
| Indicators | `GET /indicators` · `GET /indicators/{id}/series?…` · `POST /indicators:batchSeries` |
| Layouts | `GET/POST /layouts` · `GET/PUT/PATCH/DELETE /layouts/{id}` |
| Drawings | `GET /layouts/{id}/drawings` · `POST /drawings` · `PATCH/DELETE /drawings/{id}` · `POST /drawings:batchUpdate` |
| Watchlists | `GET/POST /watchlists` · `GET/PATCH/DELETE /watchlists/{id}` · `PUT /watchlists/{id}/items` |
| Alerts | `GET/POST /alerts` · `GET/PUT/PATCH/DELETE /alerts/{id}` · `POST /alerts/{id}:enable` · `POST /alerts/{id}:disable` · `POST /alerts:validate` · `GET /alerts/{id}/events` |
| Channels | `GET/POST /notification-channels` · `GET/PATCH/DELETE /notification-channels/{id}` · `POST /notification-channels/{id}:test` · `GET /notification-deliveries?status=` · `POST /notification-deliveries/{id}:retry` |
| Identity | `POST /auth/login` · `POST /auth/logout` · `GET /me` · `GET/POST /me/tokens` · `DELETE /me/tokens/{id}` · `GET/POST /users` (admin) · … |
| Data | `POST /me/export` · `POST /me/import` |
| Real-time | `GET /ws` (WebSocket upgrade; [protocol](05-realtime.md)) |

Custom methods use the `:verb` suffix (Google AIP-136 style) for actions that are
not CRUD.

## 3. Errors — RFC 9457 Problem Details

All errors use `application/problem+json`:

```json
{
  "type": "urn:wickframe:problem:validation-error",
  "title": "Request validation failed",
  "status": 422,
  "detail": "Alert condition does not compile",
  "instance": "/api/v1/alerts",
  "requestId": "01JD…",
  "errors": [
    { "location": "body.condition.expression", "message": "unknown function 'rsii'", "value": "rsii(14) > 70" }
  ]
}
```

| Status | When |
|---|---|
| 400 | Malformed request (bad JSON, wrong types) |
| 401 | Not authenticated |
| 403 | Authenticated but not allowed (scope or role) |
| 404 | Not found, **or not visible to the caller** (never reveal existence across tenants) |
| 409 | Version conflict or uniqueness conflict |
| 412 | `If-Match` precondition failed |
| 422 | Semantically invalid (validation) |
| 429 | Rate limited (with `Retry-After`) |
| 500 | Unexpected error (no internals leaked; correlated via `requestId`) |
| 502 / 503 / 504 | Upstream data-source failure, overload or timeout (`type` says which source failed, without exposing credentials) |

The problem `type` values are stable, documented identifiers. Clients branch on
`type`, never on `title` or `detail`. They are URNs (`urn:wickframe:problem:<name>`)
so they don't depend on owning a domain. If the project later owns a stable docs
domain, resolvable URLs can be introduced in a new API major version.

## 4. Pagination, filtering, sorting

* **Cursor pagination** for collections: `?pageSize=` (default 50, max 500) and
  `?pageToken=`. The response has `items` and `nextPageToken` (absent at the end).
  Tokens are opaque (base64url of an encrypted cursor) and expire after 24 h.
* Candles are **range-paged by time** instead (`from`, `to`, `limit`), because
  their natural order is time.
* Filtering uses explicit, documented query parameters per resource. There is no
  generic query language in v1.
* Sorting: `?orderBy=updatedAt desc`, only on documented fields.

## 5. Concurrency and idempotency

* Mutable documents carry `version` (integer) and an `ETag: "<version>"`.
  `PUT`/`PATCH` require `If-Match`; a mismatch → `412`. Clients refetch and merge.
* `POST` creating resources accepts an **`Idempotency-Key`** header (UUID). The
  server stores the key with the response for 24 h, per user, and replays the
  stored response on retry. A different body with the same key → `422`.
* `PATCH` uses **JSON Merge Patch** (RFC 7396) with
  `Content-Type: application/merge-patch+json`.

## 6. Authentication and authorisation

* Browser: session cookie (ADR-0014). State-changing requests are protected by
  `http.CrossOriginProtection` (Fetch metadata and `Origin` checks).
* Programs: `Authorization: Bearer wf_…` API tokens with scopes:

  | Scope | Grants |
  |---|---|
  | `read` | All GET endpoints for own resources and market data |
  | `write` | Create, update and delete own workspace resources |
  | `alerts` | Manage own alerts and channels |
  | `admin` | User management, sources, system settings (admin users only) |

* Authorisation is enforced in the **service layer** (owner checks), not only in
  handlers, so every entry point (HTTP, WS, CLI) is covered.

## 7. Rate limiting

* Per principal (user or token) and per IP for unauthenticated endpoints:
  token-bucket limits, with stricter limits on `/auth/*` (credential stuffing
  protection).
* Responses include the `RateLimit-Policy` and `RateLimit` headers (IETF HTTPAPI
  draft), plus `Retry-After` on `429`.
* Market-data endpoints are additionally protected by the source limiter. A
  request needing source data that would exceed the source budget returns `503`
  with `Retry-After`, rather than queueing forever.

## 8. Versioning and deprecation

* The major version is in the path (`/api/v1`). Additive changes (new endpoints,
  optional fields, new enum values) do not change it. **Clients must tolerate
  unknown fields and enum values.**
* Breaking changes → `/api/v2`, served in parallel with `v1` for at least one
  minor release.
* Deprecated operations send `Deprecation` (RFC 9745) and `Sunset` (RFC 8594)
  headers, and are marked `deprecated: true` in OpenAPI.
* CI diffs the OpenAPI document against the last release and fails on undeclared
  breaking changes ([ADR-0007](../adr/0007-http-api-openapi-huma.md)).

## 9. Observability hooks

* Every response includes `X-Request-Id` (generated if the client did not send
  one; validated format). If tracing is enabled, the W3C `traceparent` header is
  honoured.
* Access logs include method, route template (not the raw path, to bound
  cardinality), status, duration, bytes, principal ID and request ID.
