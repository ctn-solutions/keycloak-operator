# Release process

This document defines how the operator is versioned, how often it is
released, which versions are supported, and the exact checklist for cutting
a release. The pipeline itself is automated — a maintainer cuts a tag, the
[release workflow](../.github/workflows/release.yml) does the rest.

## Versioning

Releases follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
with a `v` prefix (`v0.2.0`). While the CRD API group is still
`v1alpha1`:

- **Patch** (`0.2.x`) — bug fixes, documentation, CI changes. No API or
  behaviour change.
- **Minor** (`0.x.0`) — new features **and** any change that is breaking for
  users of the CRDs (field removals, semantic changes, default changes).
  Because the API is `v1alpha1`, breaking CRD changes are allowed in minor
  releases; they must be documented in `CHANGELOG.md` under *Breaking
  changes* and in `docs/upgrading.md`.
- **Major** (`1.0.0`) — gated on graduating the API group from `v1alpha1`
  to a stable version. From `1.0` onward, breaking changes require a major
  version bump.

The Helm chart `version` and `appVersion` always match the release tag
(without the `v`).

## Cadence

| Release type | Cadence | Contents |
|---|---|---|
| Feature minor | roughly monthly, or when enough features accumulate | features, fixes, docs |
| Patch | as needed | regressions, security fixes |
| Security fix | as fast as possible | see [SECURITY.md](../SECURITY.md) |

There is no fixed calendar — a release is cut when there is something to
release, and always after the integration matrix is green on `main`.

## Support policy

Only the **latest minor** is supported: bug fixes are fixed on `main` and
shipped in the next patch or minor release. There are no LTS branches; the
[upgrading guide](upgrading.md) documents how to move between versions.

## What a release produces

Tagging `vX.Y.Z` triggers the release workflow, which:

1. Runs the full integration suite against Keycloak 26.0, 26.1, 26.2 and
   26.3 on the tagged commit — a failure aborts the release.
2. Builds and pushes the multi-arch image `ghcr.io/ctn-solutions/keycloak-operator:X.Y.Z`
   (and `X.Y`), signed with cosign, with SBOM and provenance attestations.
3. Publishes the Helm chart to `oci://ghcr.io/ctn-solutions/charts/keycloak-operator`
   with version and appVersion `X.Y.Z`.
4. Attaches to the GitHub release:
   - `install.yaml` — stable name, always the latest release
   - `install-vX.Y.Z.yaml` — version-pinned bundle
   - `sha256sums.txt` — sha256 of both bundles
5. Generates categorized release notes (`.github/release.yml`).

## Release checklist

1. **Bump the chart**: `charts/keycloak-operator/Chart.yaml` — set `version`
   and `appVersion` to `X.Y.Z`.
2. **Update the changelog**: move the `[Unreleased]`-style entries to
   `## [X.Y.Z] - YYYY-MM-DD` in `CHANGELOG.md` and add the compare link.
3. **Update the compatibility matrix** in `docs/upgrading.md` if the
   supported Keycloak or Kubernetes range changed.
4. **Verify `main` is green**: CI (lint, unit, generated-code check) and the
   Keycloak integration matrix must be green on the commit you tag.
5. **Tag and push**:
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```
6. **Watch the release workflow** (`gh run watch` on the Release run). A
   failed integration job aborts the release; fix and re-tag.
7. **Verify the release assets**: `install.yaml`, `install-vX.Y.Z.yaml`,
   `sha256sums.txt`; check `helm show chart
   oci://ghcr.io/ctn-solutions/charts/keycloak-operator --version X.Y.Z`.
8. **Update the Homebrew tap**:
   ```bash
   hack/update-tap.sh vX.Y.Z
   ```
   (requires push access to `ctn-solutions/homebrew-tap`; the script
   computes the tarball checksum, renders the formula and pushes it).
9. **Curate the release notes**: edit the GitHub release body to add a short
   summary and the install/upgrade commands on top of the auto-generated
   notes.
10. **Announce**: GitHub Discussions post (or a short note in the release
    body) summarizing highlights and any upgrade action.

## Homebrew tap

The tap repository [`ctn-solutions/homebrew-tap`](https://github.com/ctn-solutions/homebrew-tap)
carries the `keycloak-operator` formula, which installs `install.sh` as the
`keycloak-operator` CLI. The formula is generated from
[`packaging/homebrew/keycloak-operator.rb.in`](../packaging/homebrew/keycloak-operator.rb.in)
by `hack/update-tap.sh` — never edit the tap formula by hand.

Automating the tap update from the release workflow requires a
fine-grained personal access token with `contents: write` on the tap
repository, stored as the `TAP_TOKEN` secret. Until that token exists, the
tap update is the manual step 8 above.

## Versioning the installer

`install.sh` carries its own `INSTALLER_VERSION`. Bump it when the script's
behaviour changes; it is independent of the operator version (the script
installs whatever release you point it at). The Homebrew formula stamps the
operator version into the installed CLI.
