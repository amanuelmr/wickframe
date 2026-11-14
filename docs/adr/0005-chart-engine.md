---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0005: Use TradingView Lightweight Charts v5 behind a ChartEngine adapter, with our own drawing-tools layer

## Context and Problem Statement

The chart is the heart of the product (FR-CHART-01…15, NFR-PERF-01/02). Building a
financial chart renderer from scratch is a multi-year effort. Which open-source
chart engine do we build on, and how do we avoid being locked into it?

## Decision Drivers

* Rendering performance for 20k+ bars, with real-time updates.
* Maintenance health and bus factor: the engine must outlive our dependency on it.
* License compatible with AGPL-3.0 ([ADR-0002](0002-license-agpl-3.0.md)).
* Extensibility: multiple panes, custom series, and the drawing primitives we need.
* Bundle size.

## Considered Options

Facts were gathered on 2026-11-13 from each project's GitHub repository and npm.

| | Lightweight Charts | KLineChart | Vela (LuxAlgo) | Apache ECharts |
|---|---|---|---|---|
| Version | 5.3.0 (2026-11-10) | 10.1.0 (2026-10-14) | 0.10.7 (2026-11-12), first release 2026-07 | 6.1.0 (2026-05-19) |
| License | Apache-2.0 plus NOTICE attribution | Apache-2.0 | Apache-2.0 plus a watermark/attribution clause | Apache-2.0 |
| Built-in drawings | None (plugin examples only) | Lines, channels, Fibonacci | Extensive | None |
| Built-in indicators | None | About 26 | 70+ | None |
| Extension API | Series primitives, custom series, panes (v5) | Custom overlays and indicators | Plugin SDK | Custom series |
| Rendering | Canvas 2D | Canvas | WebGL2 (canvas fallback) | Canvas or SVG |
| Size (min+gzip) | ~62 KB | ~59 KB | ~358 KB | Large |
| Commits, last 3 months | 438 | 95 | 761 | 13 |
| Maintainers | TradingView staff, 4+ active | ~1 main | ~1–2 main, pre-1.0 | Apache org, low activity |

## Decision Outcome

Chosen option: **Lightweight Charts v5**, because it has the strongest maintenance
and backing, the smallest footprint, panes and a primitives API, and a stable 5.x
API. Indicators come from our own engine
([ADR-0010](0010-single-indicator-engine.md)), not from the chart library.

Rules:

1. **Adapter boundary.** All chart access goes through a `ChartEngine` TypeScript
   interface in `web/src/chart/engine/`. No other code imports
   `lightweight-charts`. A later switch (for example to Vela at 1.0, or our own
   WebGL renderer) is then bounded to one directory.
2. **Our own drawing-tools layer** (`web/src/chart/drawings/`), built on series
   primitives: tool state machines, hit-testing, handles, magnet/snapping, undo
   and redo, serialisation. Drawings are stored as time/price anchors, so they are
   engine-independent (FR-CHART-07). This is the largest frontend work item and is
   scheduled first in M1.
3. **Attribution.** The license requires showing TradingView's attribution notice
   and a link to <https://www.tradingview.com/> on a page available to users. We
   keep the library's `attributionLogo` option enabled (the project README states
   this satisfies the link requirement), and we list the NOTICE text
   (`TradingView Lightweight Charts™ Copyright (с) 2025 TradingView, Inc.`) in
   `THIRD_PARTY_NOTICES` and in the About dialog. AGPL-3.0 §7(b) permits requiring
   the preservation of such author attributions.
4. **Branding separation.** The attribution names the library's author only.
   Wickframe's name, branding and visual design must not imitate TradingView
   ([vision, non-goals](../product/vision.md#5-non-goals)).
5. **Upgrade policy.** Pin the minor version. Upgrades go through the visual
   regression suite ([quality](../architecture/14-quality.md)).

### Consequences

* Good, because the engine is fast, small and well maintained, with a large user
  base.
* Good, because the adapter limits lock-in, and the drawing model is ours.
* Bad, because we must build the drawing tools ourselves (Fibonacci, channels,
  editing handles). Accepted: it is core product value, and the engines with
  built-in drawings are currently either single-maintainer or pre-1.0.
* Bad, because a visible attribution logo is required. Accepted.

### Confirmation

* A Biome `noRestrictedImports` rule forbids importing `lightweight-charts` outside
  `web/src/chart/engine/`.
* A test asserts that the attribution is present on the chart and in the About
  dialog.

## Pros and Cons of the Options

### KLineChart

* Good, because drawings and indicators come built in, and it is small.
* Bad, because it effectively has one maintainer (bus-factor risk), and its
  built-in indicators would duplicate our engine.

### Vela (LuxAlgo)

* Good, because its feature set matches ours almost exactly, it uses WebGL2, and it
  is very active.
* Bad, because it is pre-1.0 and four months old, depends on one or two
  maintainers, is about 6× larger, and has a mandatory watermark.
* **Revisit at Vela 1.0** with a one-week spike behind the `ChartEngine` adapter.

### Apache ECharts

* Good, because it is a mature general-purpose library.
* Bad, because it is not specialised for financial interaction (drawings,
  price-scale behaviour), and has low recent activity.

## More Information

* <https://github.com/tradingview/lightweight-charts> (README "License and attribution", NOTICE file)
* <https://github.com/LuxAlgo/Vela>
* <https://github.com/klinecharts/KLineChart>
* FSF on Apache-2.0 compatibility: <https://www.gnu.org/licenses/license-list.html#apache2>
* Not legal advice. Re-check the attribution terms on every major upgrade.
