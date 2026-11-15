---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0015: Secure software supply chain — GoReleaser, keyless signing, SBOMs, build provenance, distroless images

## Context and Problem Statement

Users install Wickframe binaries and images that will hold their API keys. A
compromised build, dependency or release would compromise them (NFR-SEC-03/04).
How do we build, sign and publish releases, and how do we manage dependencies?

## Decision Drivers

* Users can verify that an artifact was built from this repository by CI.
* Transparent dependencies (SBOM) and fast vulnerability response.
* A fully automated release pipeline: no manual uploads from laptops.

## Decision Outcome

Versions at decision (2026-11-13) are in parentheses.

| Concern | Choice |
|---|---|
| Build and release | **GoReleaser v2** (v2.20.x) in GitHub Actions, triggered by signed tags `vX.Y.Z` on `main`. Builds `CGO_ENABLED=0` binaries for linux/amd64, linux/arm64, darwin/arm64, darwin/amd64 and windows/amd64; archives; checksums; changelog. Reproducible-build flags: `-trimpath`, `-buildvcs`, and `mod_timestamp` set to the commit time. |
| Container image | Multi-arch OCI image `ghcr.io/amanuelmr/wickframe` on **`gcr.io/distroless/static-debian13:nonroot`** (no shell, no package manager, non-root UID). Tags `X.Y.Z`, `X.Y`, `latest`. Referenced by digest in the documentation. |
| Signing | **Sigstore cosign** (v3.x), keyless, with GitHub OIDC identity. Signs checksums and images; signatures are recorded in the Rekor transparency log. |
| SBOM | **Syft** (v1.57.x) via GoReleaser, in SPDX JSON for each archive and image. Attached to the release and as an image attestation. |
| Provenance | **GitHub artifact attestations** (`actions/attest-build-provenance` v4) for binaries and images. Users verify with `gh attestation verify`. |
| Dependency updates | **Renovate**: grouped weekly updates for Go modules, npm, GitHub Actions and the Go toolchain; security updates immediately. |
| Vulnerability scanning | `govulncheck` (reachability-aware) and `pnpm audit` on every pull request and nightly; image scanning with Trivy or Grype on release. |
| CI hardening | Actions pinned by full commit SHA; `permissions:` least-privilege per job; no `pull_request_target` with untrusted checkout; `step-security/harden-runner` or equivalent egress audit. |
| Repository posture | **OpenSSF Scorecard** action (v2.5.x) weekly, badge in the README; branch protection; signed tags; secret scanning and push protection enabled. |
| Desktop artifacts | Covered by the same cosign-signed checksums, SBOMs and attestations as other artifacts. The updater accepts only signed releases. |

### Consequences

* Good, because users and packagers can verify origin and integrity.
* Good, because it meets OpenSSF and SLSA-aligned practice expected of serious
  projects.
* Bad, because desktop builds are not signed with Windows or macOS certificates,
  so users see an operating-system warning on first launch.

### Confirmation

* The release workflow fails if signing, SBOM generation or attestation fails.
* The documentation includes copy-paste verification commands (`cosign verify`,
  `gh attestation verify`).

## More Information

* GoReleaser: <https://goreleaser.com/>
* Sigstore cosign: <https://github.com/sigstore/cosign>
* GitHub artifact attestations: <https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations>
* SLSA: <https://slsa.dev/>
* OpenSSF Scorecard: <https://github.com/ossf/scorecard>
* Distroless: <https://github.com/GoogleContainerTools/distroless>
