---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0012: Frontend stack — React 19, TypeScript 7, Vite 8, TanStack Router and Query, Zustand, Tailwind 4 with shadcn/ui, Biome

## Context and Problem Statement

The web UI is a single-page application embedded in the Go binary and also used by
the desktop shell. It combines a high-frequency real-time chart with ordinary
application UI (forms, lists, dialogs, settings). Which frameworks and tools do we
use?

## Decision Drivers

* Real-time chart updates must not trigger React re-renders (NFR-PERF-02/03).
* Type safety from the API to the UI (generated from OpenAPI, ADR-0007).
* Accessibility (WCAG 2.2 AA, NFR-UX-02) and i18n readiness (NFR-UX-03).
* Large ecosystem and contributor familiarity; works well with AI-assisted
  development.
* A static SPA build that the Go binary can embed (no Node server at runtime).

## Decision Outcome

Versions verified on npm and GitHub, 2026-11-13:

| Concern | Choice | Version | Rationale |
|---|---|---|---|
| UI library | React | 19.3.x | Largest ecosystem; concurrent rendering. |
| Language | TypeScript | 7.0.x | The native (Go) compiler, 8–12× faster type-checking. Strict mode is on (the default in 7.0). |
| Build tool | Vite | 8.3.x | Fast dev server; static output for `embed.FS`. |
| Routing | TanStack Router | 1.170.x | Type-safe routes and **search params**, so chart state (symbol, timeframe, layout) lives in the URL. |
| Server state | TanStack Query | 5.107.x | Caching, retries and invalidation for REST data. |
| Client state | Zustand | 5.1.x | Small, works outside React (the vanilla store), so the WebSocket layer can update state without renders. |
| Styling | Tailwind CSS | 4.3.x | Design tokens via CSS variables; light and dark themes. |
| Components | shadcn/ui on Base UI | CLI 4.x / `@base-ui/react` 1.9.x | Accessible primitives; code is copied in, so we own it. Base UI has been shadcn's default since 2026-07. |
| Lint and format | Biome | 2.6.x | One fast tool. Avoids the typescript-eslint gap: TypeScript 7.0 has no programmatic API until 7.1. |
| Unit tests | Vitest | 5.1.x | Vite-native. |
| End-to-end tests | Playwright | 1.65.x | Cross-browser (Chromium, Firefox, WebKit) and visual comparisons. |
| Package manager | pnpm | 12.x | Strict dependency resolution; pinned with the `packageManager` field. |
| i18n | i18next + react-i18next | latest at M1 | Mature; message catalogs in JSON. |
| API client | Generated from `api/openapi.yaml` with **orval** | 8.47.x | See [ADR-0019](0019-typescript-api-client-generator.md). |

**Data-flow rule for real-time updates:** WebSocket messages do **not** go through
React state. The real-time client pushes candle and indicator updates straight
into the `ChartEngine` adapter ([ADR-0005](0005-chart-engine.md)). Only low-rate,
throttled summaries (last price, connection status) go to Zustand, batched with
`requestAnimationFrame`. TanStack Query is used only for request/response data.
See [frontend](../architecture/10-frontend.md).

### Consequences

* Good, because every choice is mainstream, current and actively maintained.
* Good, because chart updates bypass React, which protects frame rate.
* Bad, because TypeScript 7 is young, and some tools that need the compiler API may
  lag until 7.1. Mitigated by choosing Biome, and by using TypeScript only as the
  type-checker (`tsc --noEmit`) in CI.
* Bad, because there are many dependencies. Mitigated by Renovate grouping and
  `pnpm audit` in CI ([ADR-0015](0015-supply-chain-and-release.md)).

### Confirmation

* CI: `tsc --noEmit`, `biome ci`, Vitest, and the Playwright smoke tests.
* A performance test asserts no React commits occur per tick on the chart route
  (React Profiler API in E2E).

## Pros and Cons of Alternatives Considered

* **SvelteKit or Solid:** excellent performance, but smaller ecosystems and fewer
  accessible component kits. React's cost is mitigated by the out-of-React data
  path.
* **React Router 8:** a fine alternative. TanStack Router was chosen for typed
  search params, which are central to shareable chart URLs.
* **ESLint + Prettier:** the most configurable option, but typescript-eslint
  depends on the TypeScript API, which TypeScript 7.0 lacks. Revisit after 7.1.
* **Redux Toolkit:** heavier than we need.

## More Information

* TypeScript 7.0 announcement: <https://devblogs.microsoft.com/typescript/announcing-typescript-7-0/>
* shadcn/ui Base UI default: <https://github.com/shadcn-ui/ui/blob/main/apps/v4/content/docs/changelog/2026-07-base-ui-default.mdx>
