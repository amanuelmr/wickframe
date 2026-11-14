# Development Workflow

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |

This document defines how changes move from an idea to a release. It applies to
maintainers and contributors equally.

## 1. Work tracking

| Kind | Where | Rule |
|---|---|---|
| Bug | GitHub issue (`type:bug`) | Must include version, mode (personal/server), steps to reproduce, expected and actual behaviour. |
| Feature | GitHub issue (`type:feature`) | Must reference requirement IDs from [requirements](../product/requirements.md), or propose new ones. |
| Significant design change | ADR pull request ([ADR-0001](../adr/0001-record-architecture-decisions.md)) | Must be accepted before implementation is merged. |
| Question or idea | GitHub Discussions | Promoted to an issue once it is actionable. |
| Security vulnerability | Private advisory | **Never** in public issues; see [SECURITY.md](../../SECURITY.md). |

**Labels** are namespaced: `type:*` (bug, feature, docs, chore, security),
`area:*` (data, chart, indicators, alerts, api, frontend, ops, desktop),
`priority:*` (p0–p3), `status:*` (needs-triage, blocked, needs-design),
plus `good first issue` and `help wanted`.

**Milestones** in GitHub mirror the [roadmap](../product/roadmap.md) (`M1` … `M5`).

## 2. Branching: trunk-based development

- `main` is the only long-lived branch. It is **always releasable**: CI is green,
  and migrations and docs are consistent.
- Work happens on short-lived branches named `<type>/<short-description>`, for
  example `feat/kraken-connector` or `fix/ws-backpressure`. Branches should live
  for days, not weeks.
- Large features land incrementally behind a **feature flag** (config key
  `features.<name>`, default off) rather than on a long-running branch.
- Releases are cut from tags on `main`. Patch releases for an older minor version
  use a `release/vX.Y` branch, created only when needed.

## 3. Commits

- Commit messages follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/):

  ```text
  <type>(<optional scope>): <imperative summary, ≤ 72 chars>

  <body: what and why, wrapped at 72 columns>

  <footer: Refs: #123, BREAKING CHANGE: ..., Signed-off-by: ...>
  ```

  Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`,
  `chore`, `revert`. Scopes match `area:*` labels (`data`, `chart`, `api`, …).
- Every commit carries a **DCO sign-off** (`git commit -s`)
  ([ADR-0002](../adr/0002-license-agpl-3.0.md)).
- A breaking change to a public interface (HTTP API, WebSocket protocol, connector
  interface, config keys, CLI flags) MUST include a `BREAKING CHANGE:` footer.
- Commits SHOULD be signed (SSH or GPG). Maintainer commits and tags MUST be signed.

## 4. Pull requests

- One logical change per pull request. Keep diffs reviewable: aim for under 400
  changed lines, excluding generated code and fixtures.
- The pull-request title is a Conventional Commit message. PRs are
  **squash-merged**, so the title becomes the commit on `main`.
- The pull-request template must be completed: what and why, requirement IDs,
  testing evidence, screenshots for UI changes, ADR impact, docs updated.
- **Required checks** (branch protection on `main`):

  | Check | Tooling |
  |---|---|
  | Go build, vet and lint | `go vet`, golangci-lint |
  | Go tests with race detector and coverage ratchet | `go test -race` |
  | Frontend type-check, lint and unit tests | tsc, linter, Vitest |
  | End-to-end smoke tests | Playwright |
  | Indicator reference verification | see [quality](../architecture/14-quality.md) |
  | Migration tests (SQLite and PostgreSQL) | testcontainers |
  | Vulnerability scan | govulncheck, `pnpm audit` |
  | License and SPDX header check | — |
  | DCO sign-off | — |
  | OpenAPI document is up to date with the code (no diff) | — |

- **Review:** at least one approving review from a maintainer. Changes to security,
  auth, crypto, the connector interface or public protocols require a review
  focused on that concern. Authors do not approve their own pull requests; a sole
  maintainer may self-merge only after all checks pass and a 24-hour comment window.
- `CODEOWNERS` routes reviews by area.

## 5. Definition of Done

A change is done when:

1. It meets the acceptance criteria of its issue and requirement IDs.
2. Tests cover the change: unit tests always, integration and E2E tests when
   behaviour crosses a boundary.
3. Public behaviour changes are documented: user docs, API reference, and
   `CHANGELOG` (generated from commits).
4. New configuration keys have defaults, validation and documentation.
5. Observability is in place: new failure modes are logged with context and, where
   relevant, counted in metrics.
6. Security implications have been considered; the threat model is updated if a
   new trust boundary is introduced.
7. All required CI checks are green.

## 6. Versioning and releases

- Wickframe follows [Semantic Versioning 2.0.0](https://semver.org/). Before
  `1.0.0`, minor versions may contain breaking changes, which are always called out
  in the changelog.
- **Versioned public surfaces:** HTTP API (`/api/v1`), WebSocket protocol
  (`wickframe.v1`), connector interface, configuration schema, CLI, export format.
- **Cadence:** a minor release roughly every 4–6 weeks during `0.x`; patch releases
  as needed. Security fixes are released as soon as a fix is ready.
- The changelog is generated from Conventional Commits and edited by hand for the
  release notes.
- The release pipeline is described in [release process](release-process.md).

## 7. Local development

The authoritative setup steps live in [CONTRIBUTING.md](../../CONTRIBUTING.md). The
principles:

- **One command to start:** `make dev` runs the backend with live reload and the
  frontend dev server, with a SQLite database and the default crypto connector.
- **Pinned toolchain:** the Go version comes from `go.mod` (`toolchain`
  directive), Node from `.nvmrc`, pnpm from `packageManager` in `package.json`,
  and Go-based tools via the `tool` directive in `go.mod`.
- `make check` runs the same checks as CI, locally.
- **No secrets in the repo:** local credentials go in `.env.local` (git-ignored).
