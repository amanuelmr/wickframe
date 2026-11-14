# Release Process

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |
| **Decision** | [ADR-0015](../adr/0015-supply-chain-and-release.md) |

## 1. Versioning

[SemVer 2.0.0](https://semver.org/). Tags are `vMAJOR.MINOR.PATCH`; pre-releases
are `vX.Y.Z-rc.N`. Before 1.0, breaking changes may land in minor versions and are
always listed under "Breaking changes" in the notes.

## 2. Release steps

1. **Freeze:** confirm all milestone issues are closed or moved; `main` is green
   including the nightly suites.
2. **Release candidate:** a maintainer creates a **signed** tag `vX.Y.Z-rc.1`.
   The release workflow builds everything as a pre-release.
3. **Validation:** the upgrade test from the previous minor version (automatic),
   plus the manual exploratory checklist (new features, onboarding on each
   operating system via the binaries and the container).
4. **Final:** signed tag `vX.Y.Z` on the same commit. The release workflow:
   1. Builds the web UI (`pnpm build`) and embeds it.
   2. Runs `goreleaser release`: binaries for 5 targets, archives, `checksums.txt`,
      SBOMs (Syft, SPDX JSON), and a multi-arch image on distroless.
   3. Signs checksums and images with cosign (keyless, GitHub OIDC).
   4. Creates GitHub artifact attestations (build provenance) for binaries and
      images.
   5. Publishes the GitHub Release with notes generated from Conventional Commits,
      edited by hand into: Highlights, Breaking changes, Features, Fixes, Security,
      Upgrade notes.
   6. Publishes `api/openapi.yaml` as a release asset.
5. **Announce:** release notes, docs site version switch, and GitHub Discussions
   post.

## 3. Patch and security releases

* Bug-fix patches are cut from `main`. If `main` has moved on with unreleased
  breaking work, cut from a `release/vX.Y` branch, with fixes cherry-picked.
* Security fixes follow [SECURITY.md](../../SECURITY.md): fixed in a private
  security advisory fork, released as a patch, and published with an advisory
  (and a CVE when applicable) at the same time.

## 4. Supported versions

Before 1.0: only the **latest minor** receives fixes. After 1.0: the latest two
minor versions receive security fixes.

## 5. User verification (documented in install docs)

```bash
# checksums signature (keyless; identity = this repo's release workflow)
cosign verify-blob \
  --certificate-identity-regexp 'https://github.com/amanuelmr/wickframe/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --bundle checksums.txt.sigstore.json checksums.txt

# build provenance
gh attestation verify wickframe_X.Y.Z_linux_amd64.tar.gz --repo amanuelmr/wickframe

# container image
cosign verify ghcr.io/amanuelmr/wickframe:X.Y.Z \
  --certificate-identity-regexp 'https://github.com/amanuelmr/wickframe/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

The exact flags are finalised with the first release workflow and tested in CI.
