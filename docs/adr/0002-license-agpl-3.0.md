---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0002: License Wickframe under AGPL-3.0-only, with DCO sign-off for contributions

## Context and Problem Statement

Wickframe is a network application: most users interact with it over HTTP, often on
a server they do not operate themselves. We need a license that keeps the project
and its improvements open, that users and companies trust, and that is compatible
with our dependencies. We also need a rule for how contributions are licensed.

## Decision Drivers

* Prevent a third party from taking the code, modifying it, and running it as a
  closed hosted service without sharing improvements.
* Use an [OSI-approved](https://opensource.org/licenses) license, so Wickframe is
  unambiguously "open source" (this matters for trust, packaging and distribution).
* Be compatible with the licenses of our planned dependencies (mostly MIT, BSD and
  Apache-2.0).
* Keep contribution friction low.

## Considered Options

* AGPL-3.0-only
* GPL-3.0-only
* Apache-2.0
* MIT
* Source-available licenses (Business Source License, Functional Source License)

## Decision Outcome

Chosen option: **AGPL-3.0-only**, because it is the only OSI-approved option that
closes the "SaaS loophole": section 13 requires anyone who runs a *modified*
version as a network service to offer its users the corresponding source.
Self-hosters running unmodified Wickframe have no extra obligation.

Contributions are accepted under the same license with a
**[Developer Certificate of Origin](https://developercertificate.org/) sign-off**
(`git commit -s`) on every commit, enforced in CI. We do **not** use a Contributor
License Agreement.

Implementation rules:

1. The repository root contains the verbatim license text in `LICENSE`.
2. Every source file carries an SPDX header:
   `// SPDX-License-Identifier: AGPL-3.0-only` (or the comment syntax of that
   language), checked in CI.
3. The web UI shows a "Source code" link in the About dialog that points to the
   exact version's source. This satisfies AGPL section 13 for unmodified builds;
   modified builds must point it at their own source.
4. Third-party dependency licenses are checked in CI against an allow-list
   (MIT, BSD-2/3-Clause, Apache-2.0, ISC, MPL-2.0, Zlib, 0BSD, Unlicense,
   CC0-1.0). Anything else needs maintainer approval.
5. Attribution notices required by dependencies (for example a chart library's
   attribution clause) are shipped in `THIRD_PARTY_NOTICES` and shown in the UI
   where the dependency's license requires it.

### Consequences

* Good, because improvements made by anyone who hosts Wickframe for others must be
  shared back.
* Good, because it is OSI-approved and widely understood (used by Grafana,
  Mastodon, Nextcloud and Plausible, among others).
* Good, because the DCO is lightweight and keeps every contributor's copyright
  with them.
* Bad, because some companies have policies that forbid AGPL software. We accept
  this: our launch audience is individuals and self-hosters.
* Bad, because without a CLA the project cannot later be relicensed, or
  dual-licensed under a proprietary license, without every contributor's consent.
  This is intentional: it is a credible promise to contributors that the code
  stays open.

### Confirmation

* CI checks DCO sign-off on every commit in a pull request.
* CI checks SPDX headers, and dependency licenses against the allow-list.

## Pros and Cons of the Options

### AGPL-3.0-only

* Good, because it covers network use (section 13).
* Good, because it is compatible with MIT, BSD and Apache-2.0 dependencies.
  Apache-2.0 code may be combined into a GPLv3-family work
  ([FSF compatibility list](https://www.gnu.org/licenses/license-list.html#apache2)).
* Bad, because some corporate users avoid it.

### GPL-3.0-only

* Good, because it is strong copyleft for distributed software.
* Bad, because it does not cover software offered over a network, which is
  Wickframe's main deployment mode.

### Apache-2.0 / MIT

* Good, because they allow maximum adoption, including by corporations.
* Bad, because anyone may offer a closed, improved hosted fork and compete with the
  project using its own work.

### Source-available (BSL, FSL)

* Good, because they give the strongest commercial protection.
* Bad, because they are not open source by the OSI definition. That undermines the
  core promise of the project ("the open-source alternative") and community trust.

## More Information

* AGPL-3.0 text: <https://www.gnu.org/licenses/agpl-3.0.html>
* Why the AGPL: <https://www.gnu.org/licenses/why-affero-gpl.html>
* DCO: <https://developercertificate.org/>
* Revisit only if the project gains a foundation home that requires a different
  license. A change would require every contributor's consent.
