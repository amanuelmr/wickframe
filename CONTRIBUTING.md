# Contributing to Wickframe

Thank you for your interest in Wickframe! This guide explains how to propose
changes. Please also read our [Code of Conduct](CODE_OF_CONDUCT.md).

> **Project status:** pre-alpha. The architecture is documented, and the code
> skeleton is being built (milestone M0, see the [roadmap](docs/product/roadmap.md)).
> The setup commands below describe the target developer experience, and will work
> once M0 lands.

## Ways to contribute

- **Report bugs**: use the bug issue form, with version, mode and reproduction steps.
- **Report an indicator value mismatch**: use the dedicated form and attach a
  small dataset. We take correctness very seriously.
- **Request or build a data connector**: see the
  [connector contract](docs/architecture/04-market-data.md#2-connector-contract).
- **Improve documentation**: often the most valuable contribution.
- **Pick up an issue** labelled `good first issue` or `help wanted`.

For anything larger than a small fix, **open an issue first** so we can agree on
the approach before you invest time. Significant design changes need an
[Architecture Decision Record](docs/adr/README.md).

## Development setup

Prerequisites:

| Tool | Version |
|---|---|
| Go | 1.27.x (pinned by the `toolchain` line in `go.mod`) |
| Node.js | version in `.nvmrc` |
| pnpm | version in `web/package.json` `packageManager` (`corepack enable`) |
| Docker | optional, for PostgreSQL integration tests |
| make | any recent |

```bash
git clone https://github.com/amanuelmr/wickframe.git
cd wickframe
make dev      # backend with live reload + frontend dev server + fake market data
make check    # everything CI checks, locally
```

## Making changes

1. Fork the repository and create a branch: `feat/<short-name>` or `fix/<short-name>`.
2. Follow the [Go](docs/engineering/coding-standards-go.md) and
   [frontend](docs/engineering/coding-standards-frontend.md) coding standards.
3. Add or update tests. Every behaviour change needs a test.
4. Update documentation when behaviour, configuration or APIs change.
5. Run `make check`.
6. Commit using [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
   **with a DCO sign-off** (see below).
7. Open a pull request and complete the template.

The full workflow (branching, reviews, CI gates, Definition of Done) is in
[docs/engineering/development-workflow.md](docs/engineering/development-workflow.md).

## Developer Certificate of Origin (DCO)

Wickframe is licensed under [AGPL-3.0-only](LICENSE). We do not use a CLA.
Instead, every commit must be signed off, certifying that you wrote the change or
otherwise have the right to submit it under the project's license, as described
in the [Developer Certificate of Origin 1.1](https://developercertificate.org/).

Sign off by adding `-s` to your commit command:

```bash
git commit -s -m "feat(chart): add price range measure tool"
```

This adds a line like `Signed-off-by: Your Name <you@example.com>`. The name and
email must match your commit author. Pull requests with unsigned commits cannot be
merged. (Fix with `git rebase --signoff main`.)

## Licensing of contributions

By contributing, you agree that your contributions are licensed under
AGPL-3.0-only. New source files must start with the SPDX header:

```go
// SPDX-License-Identifier: AGPL-3.0-only
```

## Reporting security issues

Please **do not** open public issues for vulnerabilities. Follow
[SECURITY.md](SECURITY.md).
