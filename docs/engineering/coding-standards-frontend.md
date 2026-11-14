# Frontend Coding Standards

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |

Stack per [ADR-0012](../adr/0012-frontend-stack.md); architecture per
[frontend](../architecture/10-frontend.md).

## 1. TypeScript

* TypeScript 7, `strict` (default), plus `noUncheckedIndexedAccess`,
  `exactOptionalPropertyTypes` and `noImplicitOverride`. `tsc --noEmit` must pass.
* No `any`. Use `unknown` and narrow it. `// @ts-expect-error` needs a comment
  explaining why.
* Prefer `type` aliases and discriminated unions for domain models. Exhaustive
  `switch` statements end with an `assertNever`.
* API types come **only** from the generated OpenAPI schema (`web/src/api`).
  Never hand-write duplicates.
* Every source file starts with `// SPDX-License-Identifier: AGPL-3.0-only`.

## 2. React

* Function components and hooks only. Components are named exports in PascalCase
  files.
* **Never put high-frequency data in React state** ([frontend §3.1](../architecture/10-frontend.md#31-real-time-data-path-hot-path)).
* Server data through TanStack Query hooks in `features/*/api.ts`; query keys
  from a typed factory.
* Side effects belong in event handlers or explicit hooks. `useEffect` is for
  synchronising with external systems, not for deriving state.
* Memoisation (`useMemo`, `useCallback`, `memo`) only where profiling shows it
  helps.
* Error boundaries per route and around the chart.

## 3. Styling and UI

* Tailwind utility classes plus design tokens (CSS variables). No hard-coded
  colours; use token classes.
* UI primitives from `src/ui` (shadcn/ui on Base UI). Don't introduce another
  component library.
* Every interactive element is reachable by keyboard and has an accessible name.
  axe checks must pass.

## 4. Text and i18n

* No user-visible string literals in components. Use `t('feature.key')` with keys
  in `src/i18n/locales/en/<feature>.json`.
* Numbers, dates and prices via the shared `Intl` formatters (`lib/format`).
  Prices use instrument tick size.

## 5. Module boundaries

* Enforced with Biome `noRestrictedImports` and the CI boundary check:
  * `chart/**`, `realtime/**`: no React, no `features/**`.
  * `features/x/**`: may import other features only through their `index.ts`.
  * `lightweight-charts`: only within `chart/engine/**`.

## 6. Testing

* Vitest for units, co-located as `*.test.ts(x)`. Testing Library queries by role
  and name, not test IDs (test IDs only for canvas-adjacent elements).
* MSW handlers in `src/test/msw` for API mocking.
* Playwright specs in `web/e2e/` per user journey. Use stable locators (roles,
  labels).
* Visual snapshots only for chart rendering, drawings and themes, with explicit
  thresholds.

## 7. Lint and format

* `biome ci` must pass. The Biome config is the source of truth for style.
* Imports are sorted by Biome. No default exports except where a tool requires
  them (route files, if needed).
