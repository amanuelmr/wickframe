# Product Vision

| | |
|---|---|
| **Status** | Accepted |
| **Owner** | @amanuelmr |
| **Last reviewed** | 2026-11-13 |

## 1. One-line vision

**Wickframe is a free, open-source, self-hostable charting and technical-analysis
platform: professional-grade charts, indicators, drawings and alerts that run on
your own machine or server, using market data you connect yourself.**

## 2. Problem

Charting and technical analysis is one of the most-used categories of retail
finance software, and it is dominated by closed, subscription-based platforms.

- **Rising cost.** The market leader raised the price of every paid plan by roughly
  17–20% on 2026-04-10, its first increase in over a decade
  ([Stock Alarm](https://pro.stockalarm.io/blog/tradingview-price-increase-2026),
  [ChartingLens](https://chartinglens.com/blog/tradingview-price-increase-2026)).
  Core features — multiple indicators per chart, server-side alerts, multiple
  layouts — are gated behind those plans.
- **No ownership.** Layouts, drawings, alert rules and custom scripts live in a
  vendor cloud, in a proprietary scripting language, with no meaningful export.
- **No privacy.** Watchlists and alert levels reveal trading intent, and are stored
  and analysed by a third party.
- **No real open-source option.** Open-source building blocks exist (chart
  libraries, data libraries, research terminals), but there is no complete,
  maintained, self-hostable *application* that ties charts, data connectors,
  indicators and 24/7 alerts together. Existing attempts are early single-maintainer
  projects (largest ~260 GitHub stars as of 2026-11-13). Users ask for one directly,
  for example: *"I really need an open-source alternative to TradingView, ideally
  something that can run locally."* (Hacker News, 2026).

## 3. Target users

| ID | Persona | Needs | Primary mode |
|---|---|---|---|
| **P1** | **Self-directed retail trader** (stocks, crypto, FX) | Fast, good-looking charts; standard indicators; drawings; reliable price/indicator alerts on their phone; no subscription. | Desktop or self-hosted server |
| **P2** | **Algorithmic trader / developer** | Everything P1 has, plus an API, webhooks into their own bots, custom indicators in a real language, data they control, scriptable deployment. | Self-hosted server |
| **P3** | **Privacy-conscious self-hoster** (homelab) | One-command install, low resource use, no telemetry, backups, works behind a reverse proxy. | Self-hosted server |
| **P4** | **Small team / trading community** (later) | Shared watchlists and layouts, multiple users, single sign-on. | Self-hosted server, multi-user |

P1–P3 are the launch audience. P4 shapes the architecture (multi-user server mode)
but is not a launch target.

## 4. Goals

| ID | Goal | How we will know |
|---|---|---|
| **G1** | Be the default open-source answer to "self-hosted TradingView alternative". | Recommended in community lists and threads; steady GitHub growth. |
| **G2** | Chart quality and responsiveness comparable to commercial tools for everyday analysis. | Meets the performance budgets in [requirements](requirements.md#3-non-functional-requirements). |
| **G3** | Indicator values that traders can trust. | Every built-in indicator is verified against an independent reference implementation in CI. |
| **G4** | Alerts that fire reliably 24/7 in server mode. | Alert delivery SLOs met; no silent failures (every failure is logged and surfaced). |
| **G5** | Trivial to install, upgrade and back up. | `docker compose up` or a single binary; zero-downtime-safe upgrades with automatic migrations; documented backup and restore. |
| **G6** | Users own their data. | Everything exportable in open formats; no telemetry by default. |

## 5. Non-goals

These are deliberate. Changing any of them requires an Architecture Decision Record.

| Non-goal | Why |
|---|---|
| **Placing orders or managing funds.** | Order execution brings regulatory, security and liability exposure far beyond a charting tool. Users can wire Wickframe webhooks to their own execution systems. |
| **Providing or redistributing market data.** | Market data is licensed; redistribution needs exchange agreements and costs money. Users connect their own sources ([ADR-0006](../adr/0006-bring-your-own-market-data.md)). |
| **A social network** (public ideas, followers, chat). | Not what makes the product valuable to P1–P3, and moderation is costly. |
| **AI or LLM features at runtime.** | Project constraint: no inference cost, no AI dependency. |
| **Native mobile apps in app stores.** | Project constraint. Mobile use is served by a responsive web UI and push notifications. |
| **Financial advice, signals or recommendations.** | Legal exposure; out of scope for a tool. |
| **Being compatible with a proprietary scripting language.** | Licensing and trademark risk; we design an open scripting interface instead. |
| **Copying any commercial product's brand, assets or look.** | We compete on features and openness, not imitation. |

## 6. Product principles

Use these to settle design arguments, in this order:

1. **Correct before fast before featureful.** A wrong indicator value or a missed
   alert is worse than a missing feature.
2. **Local-first and private by default.** Works fully offline against local data;
   no phone-home; no telemetry unless the user explicitly opts in.
3. **Bring your own data.** Wickframe connects to sources the user is entitled
   to use and never redistributes data.
4. **One binary, sensible defaults.** A new user gets a working chart in under
   five minutes with zero configuration.
5. **Open standards over invention.** OpenAPI, WebSocket, OpenTelemetry, OIDC,
   SQL, CSV/Parquet, semantic versioning.
6. **Secure by default.** Safe defaults, least privilege, signed releases,
   documented threat model.
7. **Boring technology, excellent execution.** Prefer mature, well-maintained
   dependencies, and few of them.

## 7. Product scope at a glance

```text
            ┌────────────────────────── Wickframe ──────────────────────────┐
 Market     │  Connectors ─▶ Candle store ─▶ Indicators ─▶ Charts (web/desktop)
 data  ───▶ │      │               │              │                       │
 sources    │      └──── Live stream hub ─────────┴──▶ Alert engine ─▶ Notifications
 (user's)   │                                                 (Telegram, Discord,
            │  Layouts · Drawings · Watchlists · API · Users      email, webhook, ntfy)
            └────────────────────────────────────────────────────────────────┘
```

Scope is delivered in phases; see the [roadmap](roadmap.md).

## 8. Positioning

| | Commercial charting platforms | Broker-bundled charting | Research terminals (OSS) | **Wickframe** |
|---|---|---|---|---|
| Price | Subscription | Free with an account | Free / open core | **Free, open source** |
| Self-hostable | No | No | Partly | **Yes** |
| Data | Vendor-provided | Broker's own | User-provided | **User-provided, many sources** |
| Interactive charting and drawings | Excellent | Varies | Limited | **Core focus** |
| 24/7 alerts | Paid tiers | Varies | No | **Yes (server mode)** |
| Open API, webhooks, scripting | Limited or proprietary | Varies | Yes | **Yes** |
| Data ownership | Vendor | Broker | User | **User** |

## 9. Sustainability

Wickframe is developed in the open under AGPL-3.0 ([ADR-0002](../adr/0002-license-agpl-3.0.md)).
Possible future funding (GitHub Sponsors, an optional hosted edition, support) must
never cripple the self-hosted edition. Any change to this policy requires an ADR.
