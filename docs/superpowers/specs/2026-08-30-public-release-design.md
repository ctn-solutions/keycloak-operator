# Design: Public release uplift — distribution, versioning, release engineering

Date: 2026-08-30
Status: Approved (autonomous session — decisions recorded here in lieu of interactive review)

## Context

The operator is feature-complete for a first public lifecycle: CI (lint, unit,
envtest), an e2e suite against real Keycloak 26.0–26.3, a tag-triggered release
pipeline (multi-arch image, cosign, SBOM/provenance, Helm chart on OCI, install
bundle), and a full documentation set. `v0.1.0` is already published.

What is missing for a *public* release story:

1. **No host-side install path.** The only install methods are Helm and a raw
   `kubectl apply` of a release asset. There is no `curl | sh` installer and no
   package-manager entry point.
2. **No version reporting.** The binary has no `--version`; users cannot tell
   what is running without inspecting the image tag.
3. **Broken/fragile release links.** `docs/installation.md` and
   `docs/upgrading.md` link to `releases/latest/download/install.yaml`, but the
   v0.1.0 release only carries `install-v0.1.0.yaml`. The README links to
   `install-v0.1.0.yaml` via `latest/download`, which 404s once a newer release
   exists. The README also duplicates the "Why trust it?" section.
4. **No checksums** on release assets, so the installer cannot verify downloads.
5. **No documented release cycle** (versioning policy, cadence, support
   policy) and no curated release notes (only GitHub's auto-generated list).

## Goals

- One-command install, upgrade and uninstall for the manifest-based install
  method, with checksum verification.
- A Homebrew tap so macOS/Linux users get `brew install` and `brew upgrade`.
- The operator reports its version (`--version`, startup log).
- A clean, documented release cycle with categorized release notes and a
  changelog.

## Non-goals (rejected, with reasons)

- **homebrew-core submission** — notability bar not met for a v0.x operator; a
  tap is the correct first step.
- **goreleaser / standalone binaries** — the shipped artifact is a container
  image deployed into Kubernetes; there is no user-facing binary to distribute.
  The only host-side tool is the install script itself.
- **In-cluster automatic self-upgrade** — unsafe by design: CRD upgrades are
  irreversible-ish (deleting CRDs cascades to user data) and operators manage
  stateful external systems. Updates stay explicit (`install.sh upgrade`,
  `brew upgrade`). The decision and rationale are documented for users.
- **Renewable automation for the tap** — cross-repo pushes from Actions need a
  PAT secret that cannot be minted in this session. The tap update is a
  documented, scripted manual step (one command) until a `TAP_TOKEN` secret is
  provisioned; the release workflow skips the tap job when the secret is absent.

## Decisions

### D1 — Distribution channels

| Channel | Artifact | Status |
|---|---|---|
| `install.sh` (repo root, also a release asset) | bash script → `kubectl apply` of the versioned bundle | **Primary** for non-Helm users |
| Homebrew tap `ctn-solutions/homebrew-tap` | formula installs `install.sh` as the `keycloak-operator` CLI | New |
| Helm OCI chart | existing | unchanged |
| Plain `kubectl apply -f releases/latest/download/install.yaml` | existing, fixed by uploading a stable-name asset | fixed |

The install script is the "simple bundle" entry point: it resolves the latest
(or pinned) release, downloads `install-vX.Y.Z.yaml` + `sha256sums.txt`,
verifies the checksum, applies it, and waits for rollout. Subcommands:
`install` (default), `upgrade`, `uninstall`, `version`, `help`. Safety rails:
refuses to touch Helm-managed installs, never deletes CRDs unless
`--purge-crds` is passed, honors `GITHUB_TOKEN` for rate limits.

### D2 — Version reporting

`internal/version` package (`Version`, `Commit`, `Date`), injected via
`-ldflags -X` in the Makefile and Dockerfile (`VERSION` build-arg supplied by
the release workflow from the tag). `--version` prints and exits; the startup
log line carries the version.

### D3 — Release bundle contents

Every release attaches: `install.yaml` (stable name — fixes all
`latest/download` links), `install-vX.Y.Z.yaml` (pinned), `sha256sums.txt`
(sha256 of every asset). The installer verifies checksums when present and
warns loudly when not (pre-v0.2.0 releases).

### D4 — Release notes and changelog

- `.github/release.yml`: categorized auto-notes driven by Conventional Commit
  prefixes already used in the repo (`feat:` → Features, `fix:` → Bug fixes,
  `docs:` → Documentation, `ci:`/`build:` → CI & build, `chore:`/`refactor:` →
  Maintenance, `security` scope → Security).
- `CHANGELOG.md` (Keep a Changelog format), seeded with 0.1.0 (retrospective
  from the git history) and maintained per release.
- Release bodies keep `generate_release_notes: true` (per-contributor credit)
  on top of the curated categories.

### D5 — Release cycle (documented in `docs/release-process.md`)

- **Semver.** `0.x` while the API is `v1alpha1`; breaking CRD changes bump the
  minor, the CRD version bumps only when the API group version changes.
  `v1` API graduation is the gate for `1.0`.
- **Cadence.** Feature minors roughly monthly; patches for regressions and
  security fixes as needed; security fixes backported to the latest minor only.
- **Mechanics.** Bump `Chart.yaml`, green CI, tag `vX.Y.Z`, pipeline does the
  rest (integration matrix gate → image + signature + SBOM → chart → bundle +
  checksums → GitHub release with categorized notes). Then: update the tap
  formula (one command), update `CHANGELOG.md` if not already done.
- **Support policy.** Latest minor only; upgrade notes in `docs/upgrading.md`.

### D6 — Common labels on the kustomize bundle

`config/default/kustomization.yaml` gains a `labels:` block
(`app.kubernetes.io/name=keycloak-operator`, `app.kubernetes.io/managed-by=kustomize`)
so uninstall/teardown by label works and the existing docs
(`kubectl delete crd -l app.kubernetes.io/name=keycloak-operator`) become true.

## Components

| Unit | Purpose | Files |
|---|---|---|
| Installer CLI | install/upgrade/uninstall/version against GitHub releases | `install.sh` |
| Version reporting | `--version`, startup log, ldflags | `internal/version/version.go`, `cmd/main.go`, `Makefile`, `Dockerfile`, `release.yml` |
| Release pipeline | stable-name bundle, checksums, tap job (opt-in) | `.github/workflows/release.yml` |
| Release notes config | categorized auto-notes | `.github/release.yml` |
| Changelog | human-readable history | `CHANGELOG.md` |
| Release process doc | cycle, versioning, checklist, tap publishing | `docs/release-process.md` |
| Homebrew formula | `brew install ctn-solutions/tap/keycloak-operator` | `packaging/homebrew/keycloak-operator.rb` + tap repo |
| Docs updates | quick install, fixed links, upgrade paths | `README.md`, `docs/installation.md`, `docs/upgrading.md`, `CONTRIBUTING.md` |

## Verification plan

1. `make lint && make test` (unit + envtest) — existing gates stay green.
2. `bash -n install.sh` + `shellcheck` (if available) + a stubbed-kubectl test
   harness exercising install/upgrade/uninstall/purge paths.
3. Live download+checksum verification against the published v0.1.0 release.
4. End-to-end on a throwaway kind cluster (if Docker is available): install
   v0.1.0 via the script, upgrade to v0.2.0 after the release, uninstall.
5. Cut `v0.2.0`: watch the release workflow, verify assets + checksums, verify
   `releases/latest/download/install.yaml` resolves.
6. Publish the tap, then `brew install ctn-solutions/tap/keycloak-operator` and
   exercise `keycloak-operator version`.

## Risks / residual items

- Tap automation requires a `TAP_TOKEN` secret (fine-grained PAT with
  `contents:write` on the tap repo) — documented; manual one-command update
  until then.
- The v0.1.0 release lacks checksums; the installer degrades to a warning for
  that version only.
- Making the repo public was already done previously; no visibility change is
  required in this work.
