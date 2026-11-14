---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0006: Bring your own market data — Wickframe never provides or redistributes data

## Context and Problem Statement

A charting application is useless without market data. Data can be (a) provided by
the project from a central service, or (b) fetched by each installation directly
from sources the user chooses and is entitled to use. Market data is licensed
content: exchanges and vendors restrict redistribution, and real-time equity data
in particular carries exchange fees and agreements. Which model does Wickframe use?

Related requirements: FR-DATA-01…15, FR-OPS-06, NFR-SEC-02.

## Decision Drivers

* The project has no budget for data licensing or central infrastructure.
* The project must not expose maintainers or users to licensing violations.
* Users must be able to use data they already have access to (broker accounts,
  vendor subscriptions, free public crypto feeds).
* Privacy: watchlists and alerts should not flow through a central service.

## Considered Options

* **Bring your own data (BYOD):** each installation connects directly to sources
  the user configures, under the user's own credentials and terms.
* **Central data relay** operated by the project.
* **Bundled free datasets** shipped with releases.

## Decision Outcome

Chosen option: **Bring your own data**, because it is the only option that needs no
data licenses, no central infrastructure and no trust in the project, and it keeps
users' trading intent private.

Rules that follow from this decision:

1. Connectors call sources **directly from the user's installation**, using the
   user's own credentials where required. There is no Wickframe-operated proxy.
2. Wickframe **MUST NOT** expose stored market data to unauthenticated parties,
   and MUST NOT offer features designed for bulk redistribution (for example a
   public data API or public chart embeds with live data) (FR-DATA-14).
3. Every connector's documentation states the source's terms-of-use link, what
   authentication is needed, and whether the data is real-time or delayed. The UI
   displays freshness (FR-DATA-13).
4. Connectors **only** use official, documented APIs. Connectors that scrape
   websites or rely on undocumented endpoints MUST NOT be included in the default
   distribution. If they exist at all, they live in third-party plugins, outside
   this repository.
5. The default configuration enables only sources whose public market data is free
   and requires no account, so the first-run experience works (NFR-UX-01).
6. Sample datasets used in tests and demos MUST be either synthetic or from sources
   whose terms permit redistribution, with that permission documented in the
   dataset's README.

### Consequences

* Good, because there is no data cost and no data liability for the project.
* Good, because users keep full privacy and control.
* Good, because the connector interface becomes a first-class extension point,
  which invites community connectors.
* Bad, because real-time equity data usually requires the user to have a broker or
  vendor account, so the out-of-the-box experience is crypto-first.
* Bad, because each connector must track its source's API changes. This is
  mitigated by conformance tests and recorded fixtures
  ([quality](../architecture/14-quality.md)).

### Confirmation

* Code review checklist item: "Does this change serve stored market data to
  unauthenticated or third-party clients?"
* Connector conformance tests verify that every connector documents its source,
  authentication and freshness metadata.

## Pros and Cons of the Options

### Central data relay

* Good, because it gives the best out-of-the-box experience.
* Bad, because it requires redistribution licenses and funding, and is a single
  point of failure.
* Bad, because it centralises users' watchlists, which contradicts the privacy
  principle.

### Bundled datasets

* Good, because it works offline for demos.
* Bad, because the data goes stale immediately, and most sources forbid
  redistributing it.

## More Information

* Revisit if a data provider offers a license that explicitly permits open-source
  redistribution at no cost.
* Connector details: [market data architecture](../architecture/04-market-data.md).
