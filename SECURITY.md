# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| Latest minor release (0.x) | ✅ Security fixes |
| Older 0.x releases | ❌ Please upgrade |

After 1.0, the latest two minor versions receive security fixes.

## Reporting a vulnerability

**Please do not report security vulnerabilities in public issues, discussions or
pull requests.**

Report privately through **GitHub Security Advisories**:
<https://github.com/amanuelmr/wickframe/security/advisories/new>
("Report a vulnerability" on the Security tab).

Please include:

- the affected version or commit, and the mode (personal or server);
- a description of the issue and its impact;
- steps to reproduce or a proof of concept;
- any suggested fix or mitigation.

## What to expect

| Step | Target |
|---|---|
| Acknowledgement of your report | within 3 business days |
| Initial assessment (validity, severity using CVSS) | within 7 days |
| Fix or mitigation for critical and high severity issues | as fast as possible, aiming for 30 days |
| Public disclosure | coordinated with you, after a fixed release is available (default embargo up to 90 days) |

We will credit you in the advisory unless you prefer to stay anonymous. CVE IDs
are requested through GitHub when applicable.

## Scope

In scope: the Wickframe server and CLI, the web UI, the desktop app, official
container images, release artifacts, and the release pipeline.

Out of scope: vulnerabilities in third-party data sources or notification
services; issues that require a compromised host or a malicious administrator;
self-inflicted misconfigurations explicitly warned against in the documentation
(for example disabling authentication on a public server).

## Hardening guidance for operators

- Run server mode behind TLS, with authentication enabled (the default in server
  mode).
- Use **read-only API keys** for data sources. Never give Wickframe keys with
  trading or withdrawal permissions.
- Keep the secret key file (`security.secretKeyFile`) backed up and private.
- Verify release signatures and provenance (see the
  [release process](docs/engineering/release-process.md#5-user-verification-documented-in-install-docs)).

The security architecture and threat model are documented in
[docs/architecture/11-security.md](docs/architecture/11-security.md).
