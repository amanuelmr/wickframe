---
status: "{proposed | accepted | rejected | deprecated | superseded by ADR-NNNN}"
date: {YYYY-MM-DD when the decision was last updated}
decision-makers: {list everyone involved in the decision}
consulted: {people whose opinions were sought; two-way communication}
informed: {people kept up to date; one-way communication}
---

<!-- Template based on MADR 4.0 (https://adr.github.io/madr/). Keep the headings. -->

# ADR-NNNN: {Short title, representative of the solved problem and found solution}

## Context and Problem Statement

{Describe the context and the problem in two or three sentences, or as a question.
Link to requirement IDs (FR-/NFR-) and issues.}

## Decision Drivers

* {driver 1, e.g. a force or a facing concern}
* {driver 2}

## Considered Options

* {title of option 1}
* {title of option 2}
* {title of option 3}

## Decision Outcome

Chosen option: "{title of option 1}", because {justification, e.g. the only option
that meets a k.o. criterion or decision driver, or the option that comes out best}.

### Consequences

* Good, because {positive consequence, e.g. improvement of one or more desired qualities}
* Bad, because {negative consequence, e.g. compromising one or more desired qualities}

### Confirmation

{How the implementation of, or compliance with, this ADR is confirmed: a review, an
automated test, a lint rule, a CI check.}

## Pros and Cons of the Options

### {title of option 1}

* Good, because {argument a}
* Neutral, because {argument b}
* Bad, because {argument c}

### {title of option 2}

* Good, because {argument a}
* Bad, because {argument b}

## More Information

{Evidence, links to sources (with access date), the conditions under which this
decision should be revisited, and related ADRs.}
