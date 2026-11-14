# Wickframe Documentation

## Suggested reading order

1. [Product vision](product/vision.md): what we build and why, and what we won't build.
2. [Requirements](product/requirements.md): the IDs referenced everywhere else.
3. [Architecture overview](architecture/00-overview.md): quality goals, context, strategy.
4. [ADR index](adr/README.md): the decisions behind the design.
5. The architecture chapters relevant to your work.
6. [Development workflow](engineering/development-workflow.md) before your first pull request.

## Product

| Document | Content |
|---|---|
| [vision.md](product/vision.md) | Problem, personas, goals, non-goals, principles, positioning |
| [requirements.md](product/requirements.md) | FR-* and NFR-* with priorities and milestones |
| [roadmap.md](product/roadmap.md) | Milestones M0–M4 and v1.0 with exit criteria |

## Architecture

| Document | Content |
|---|---|
| [00 Overview](architecture/00-overview.md) | Goals, constraints, context (C4 L1), solution strategy, modes, principles |
| [01 Containers](architecture/01-containers.md) | C4 L2, roles, deployment variants, ports |
| [02 Components](architecture/02-components.md) | Modules, dependency rules, key interfaces, supervision |
| [03 Domain model](architecture/03-domain-model.md) | Concepts, invariants, identifiers |
| [04 Market data](architecture/04-market-data.md) | Connectors, limits, pipeline, history, gaps |
| [05 Real-time](architecture/05-realtime.md) | Event hub, WebSocket protocol `wickframe.v1` |
| [06 Indicators](architecture/06-indicators.md) | Engine, conventions, verification |
| [07 Alerts](architecture/07-alerts.md) | Guarantees, evaluation, outbox, channels, webhook signing, SSRF |
| [08 Storage](architecture/08-storage.md) | Schema, candle storage, migrations, backup |
| [09 API](architecture/09-api.md) | REST conventions, errors, pagination, versioning |
| [10 Frontend](architecture/10-frontend.md) | SPA structure, state, drawings, performance, a11y |
| [11 Security](architecture/11-security.md) | Assets, trust boundaries, STRIDE, secrets, hardening |
| [12 Observability](architecture/12-observability.md) | Logs, metrics, SLOs, tracing, health |
| [13 Deployment](architecture/13-deployment.md) | Distribution, configuration, container, upgrades |
| [14 Quality](architecture/14-quality.md) | Quality scenarios, test pyramid, gates |
| [15 Risks](architecture/15-risks.md) | Risk register, technical debt, open questions |
| [Glossary](architecture/glossary.md) | Terms |

## Decisions

[ADR index](adr/README.md) · [template](adr/template.md)

## Engineering

| Document | Content |
|---|---|
| [development-workflow.md](engineering/development-workflow.md) | Issues, branching, commits, PRs, CI gates, Definition of Done, versioning |
| [repository-layout.md](engineering/repository-layout.md) | Directory structure, generated code, make targets |
| [coding-standards-go.md](engineering/coding-standards-go.md) | Go rules |
| [coding-standards-frontend.md](engineering/coding-standards-frontend.md) | TypeScript and React rules |
| [release-process.md](engineering/release-process.md) | Releases, signing, verification |

## Conventions for these documents

- Markdown, rendered on GitHub; diagrams in Mermaid (rendered natively by GitHub).
- Each document has a status table with **Last reviewed**. Update it when you
  change the document.
- Facts about third-party tools carry the date they were verified. Re-verify before
  relying on them for implementation.
