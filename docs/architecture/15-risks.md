# 15 · Risks and Technical Debt

| | |
|---|---|
| **Status** | Living document — reviewed at every milestone |
| **Last reviewed** | 2026-11-13 |

Likelihood (L) and impact (I) are rated 1 (low) to 3 (high). The score is L × I.

## 1. Risk register

| ID | Risk | L | I | Score | Mitigation | Owner |
|---|---|---|---|---|---|---|
| R1 | **Data-source API or terms changes** break a connector or restrict use. | 3 | 2 | 6 | Recorded fixtures plus a nightly live smoke test; several sources per asset class; connector interface isolates changes; terms reviewed per release; BYOD model (ADR-0006). | Maintainer |
| R2 | **Drawing tools take much longer than estimated** (own layer on Lightweight Charts). | 3 | 2 | 6 | Scheduled first in M1; minimal tool set first (FR-CHART-05); state-machine design; reuse ideas from the official plugin examples; Vela 1.0 re-evaluation (ADR-0005). | Maintainer |
| R3 | **Solo-maintainer bus factor** (burnout, availability). | 2 | 3 | 6 | Thorough docs and ADRs; automated quality gates; small modules; welcoming contributor experience (`good first issue`); a governance doc when contributors appear. | Maintainer |
| R4 | **Indicator mismatch** reports erode trust. | 2 | 3 | 6 | Reference verification against TA-Lib; documented conventions; a "report a value mismatch" issue template asking for a dataset. | Maintainer |
| R5 | **Rate-limit bans** from sources (for example Binance 418 IP bans) on busy servers. | 2 | 2 | 4 | Header-driven limiters; global per-host pause on 429; caching and coverage index; connection-attempt budgets. | Maintainer |
| R6 | **Young dependencies**: Wails v3 (weeks old), TypeScript 7 (months old), go-webauthn pre-1.0. | 2 | 2 | 4 | Pin versions; desktop is optional (M3); Biome instead of compiler-API tools; passkeys are M3 and behind a flag. | Maintainer |
| R7 | **Dual-dialect maintenance cost** (SQLite + PostgreSQL). | 2 | 2 | 4 | `storetest` contract suite; schema-equivalence check; AI-assisted generation of both variants; minimal dialect-specific features. | Maintainer |
| R8 | **Security incident** in a self-hosted deployment (exposed instance, leaked keys). | 2 | 3 | 6 | Secure defaults (loopback-only, auth required in server mode); encrypted secrets; docs recommending read-only API keys; security policy and fast patch releases. | Maintainer |
| R9 | **Trademark or confusion claims** about the name or UI. | 1 | 3 | 3 | Original name (checked 2026-11-13; formal trademark search pending); no imitation of competitors' branding; attribution only where licenses require it. | Maintainer |
| R10 | **WebKitGTK rendering performance** on Linux desktop. | 2 | 1 | 2 | Early per-platform performance tests; the browser edition remains available. | Maintainer |
| R11 | **Scope creep** (trading, social, AI features requested). | 3 | 2 | 6 | Non-goals in the vision; ADR required to change them; a public roadmap. | Maintainer |
| R12 | **AI-assisted code quality drift** (plausible but subtly wrong code). | 2 | 3 | 6 | Mandatory tests for every change; reference-verified indicators; review checklists; the architecture test; no merge without CI green; human review of every diff. | Maintainer |

## 2. Known technical debt (intentional)

| ID | Debt | Reason | Paydown trigger |
|---|---|---|---|
| D1 | Single-process only (roles cannot yet run in separate processes). | Simplicity for M1–M4 (ADR-0003). | When a single node can't meet NFR-PERF-04/05. |
| D2 | Forming bars are not persisted; they are re-derived on restart. | Avoids write amplification. | If restart gaps prove user-visible. |
| D3 | No per-tenant resource quotas beyond rate limits. | Small multi-user deployments are the target. | Before public multi-tenant hosting. |
| D4 | Desktop builds unsigned until certificates are funded. | Cost (ADR-0015). | M3 release. |
| D5 | English only at launch (i18n-ready). | Focus. | When community translators appear. |

## 3. Open questions

| # | Question | Decide by |
|---|---|---|
| Q1 | User-script language: Starlark or goja? | M4 spike (ADR-0011) |
| Q2 | Exact TypeScript OpenAPI client generator? | M1 start |
| Q3 | Parquet writer library for exports? | M3 |
| Q4 | Formal trademark search and domain registration for "Wickframe". | Before the first public release |
| Q5 | Code-of-conduct enforcement contact (an email address). | Before the repository goes public |
