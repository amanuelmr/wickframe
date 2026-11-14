---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0001: Record architecture decisions as MADR files in the repository

## Context and Problem Statement

Wickframe is a long-lived open-source project. Contributors arrive at any time and
need to understand *why* the system is built the way it is, not only *what* it is.
Decisions recorded only in chat, issues or someone's head get lost, and are then
re-argued or silently reversed.

## Decision Drivers

* Decisions must be discoverable by new contributors with no history.
* Records must be versioned and reviewed together with the code they affect.
* The format must be lightweight enough to actually be used.

## Considered Options

* Architecture Decision Records in Markdown, using the MADR 4.0 template, stored in `docs/adr/`.
* A wiki (GitHub Wiki or external).
* GitHub Discussions or issues only.

## Decision Outcome

Chosen option: **MADR 4.0 records in `docs/adr/`**, because they live next to the
code, go through the same pull-request review, and are the de-facto industry
standard format ([adr.github.io](https://adr.github.io/), [MADR](https://adr.github.io/madr/)).

Rules:

1. Any decision that is **costly to reverse** or **affects more than one component**
   needs an ADR. Examples: languages, frameworks, storage, public protocols,
   licensing, security model, and any change to a non-goal in the
   [vision](../product/vision.md#5-non-goals).
2. Files are named `NNNN-kebab-case-title.md` with a zero-padded number that is
   never reused.
3. A new ADR starts as `proposed` in a pull request. It becomes `accepted` when
   merged.
4. Accepted ADRs are **immutable**. To change a decision, write a new ADR that
   supersedes it, and update only the old one's `status` to
   `superseded by ADR-NNNN`.
5. Each ADR links the requirement IDs it serves, and names the conditions under
   which it should be revisited.
6. The [ADR index](README.md) is updated in the same pull request.

### Consequences

* Good, because the reasoning behind the architecture survives turnover.
* Good, because reviewers can challenge decisions before code is written.
* Bad, because it adds a small amount of process to significant changes.

### Confirmation

Pull requests that change an area governed by an ADR are reviewed against it. The
pull-request template asks "Does this change require or contradict an ADR?"

## More Information

* Michael Nygard, "Documenting Architecture Decisions" (2011): <https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions>
* MADR 4.0 template: <https://adr.github.io/madr/>
