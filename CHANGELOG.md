# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
(see [docs/release-process.md](docs/release-process.md) for the versioning
policy while the API is still `v1alpha1`).

## [0.2.0] - 2026-08-30

### Features

- `install.sh` — a one-command installer for the plain-manifest method:
  `install`, `upgrade`, `uninstall` and `version` subcommands, release
  pinning with `--version`, sha256 verification of the downloaded bundle,
  and a guard that refuses to touch Helm-managed installs. Uninstall keeps
  CRDs (and therefore every managed resource) unless `--purge-crds` is
  passed. ([#install.sh](install.sh))
- Homebrew distribution: `brew install ctn-solutions/tap/keycloak-operator`
  installs the installer as the `keycloak-operator` CLI; updates flow
  through `brew upgrade`.
- The operator now reports its version: `--version` flag, a version line in
  the startup log, and build metadata injected at link time (image builds
  stamp the git tag and commit).
- Release bundles now ship `install.yaml` (stable name, resolves through
  `releases/latest/download/`) alongside the versioned bundle, plus a
  `sha256sums.txt` manifest that the installer verifies — and that manifest
  is itself signed keyless with cosign and verified by the installer when
  cosign is available.
- Every manifest in the install bundle carries
  `app.kubernetes.io/name=keycloak-operator`, enabling label-based
  teardown (`kubectl delete crd -l app.kubernetes.io/name=keycloak-operator`).

### Documentation

- New [release process](docs/release-process.md): versioning policy, release
  cadence, support policy and the full release checklist.
- `CHANGELOG.md` (this file) and categorized auto-generated release notes.
- Installation guide covers the installer script and Homebrew; upgrading
  guide covers `install.sh upgrade`.

### CI

- The release image build stamps the version into the binary; release
  assets gain checksums.

## [0.1.0] - 2026-08-30

Initial public release.

### Features

- Kubernetes operator managing Keycloak configuration declaratively through
  seven custom resources: `KeycloakConnection`, `Realm`, `Client`,
  `ClientScope`, `RealmRole`, `IdentityProvider` and `Group` — a 1:1 mapping
  of the Keycloak Admin API representations.
- Declarative field removal (last-applied tracking), drift correction on a
  periodic resync, and per-resource adoption policies (`CreateOnly`,
  `Adopt`, `FailIfExists`).
- Safe-by-default lifecycle: finalizers on every resource, realms orphaned
  by default, `master` protected behind an explicit annotation.
- Secrets in both directions: injection from Kubernetes Secrets
  (`secretRef`, `configSecretRef`, `smtpServerSecretRef`) and export of
  Keycloak-generated client secrets (`secretOutput`); inline secrets are
  rejected by CEL validation rules on the CRDs.
- Multi-server management through `KeycloakConnection` resources.
- Custom Prometheus metrics (reconciliations, drift corrections, connection
  health, Admin API latency) behind an authenticated endpoint.
- Helm chart (OCI and in-tree) with namespace-scoped install support, plus a
  plain-manifest install bundle.

### Security & supply chain

- Multi-arch distroless images, signed with Sigstore cosign (keyless), with
  SBOM and SLSA build provenance attestations.
- CI: lint, unit tests (envtest), end-to-end suite against real Keycloak
  26.0–26.3 on kind, CodeQL, dependency review, OpenSSF Scorecard.

[0.2.0]: https://github.com/ctn-solutions/keycloak-operator/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/ctn-solutions/keycloak-operator/releases/tag/v0.1.0
