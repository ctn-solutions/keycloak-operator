# Public Release Uplift Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the keycloak-operator publicly releasable with a one-command installer, a Homebrew tap, version reporting, checksums, and a documented release cycle with curated release notes.

**Architecture:** A bash installer (`install.sh`) at the repo root downloads versioned install bundles from GitHub releases, verifies sha256 checksums, and applies them with kubectl (install/upgrade/uninstall). The operator binary learns to report its version via ldflags. The release workflow gains a stable-name bundle asset and a checksums file. A Homebrew tap distributes the installer as the `keycloak-operator` CLI. Release engineering is documented in `docs/release-process.md` with a Keep-a-Changelog `CHANGELOG.md` and categorized auto-notes via `.github/release.yml`.

**Tech Stack:** bash (installer), Go (version package), GitHub Actions, Homebrew tap, kustomize bundle.

## Global Constraints

- Repo: `github.com/ctn-solutions/keycloak-operator`, module `github.com/ctn-solutions/keycloak-operator`, Apache-2.0.
- Never edit generated files (`config/crd/bases/*`, `config/rbac/role.yaml`, `**/zz_generated.*.go`, `PROJECT`); regenerate with `make manifests generate`.
- Conventional Commits (`feat:`, `fix:`, `docs:`, `ci:`, `chore:`); no mention of the author in commits/docs.
- Deployment identity: `keycloak-operator-controller-manager` in namespace `keycloak-operator-system`; common label `app.kubernetes.io/name=keycloak-operator`.
- Release assets after this work: `install.yaml`, `install-vX.Y.Z.yaml`, `sha256sums.txt`.
- Next release version: **v0.2.0** (Chart.yaml `version`/`appVersion` must be bumped to `0.2.0` before tagging).
- The installer must never delete CRDs unless `--purge-crds` is passed (CRD deletion cascades to all custom resources).
- The installer must refuse to operate on Helm-managed installs (label `app.kubernetes.io/managed-by=Helm`).

---

### Task 1: Version reporting in the operator binary

**Files:**
- Create: `internal/version/version.go`
- Create: `internal/version/version_test.go`
- Modify: `cmd/main.go` (flags + startup log)
- Modify: `Makefile` (build target ldflags)
- Modify: `Dockerfile` (VERSION build-arg)
- Modify: `.github/workflows/release.yml` (build-args)

**Interfaces:**
- Produces: `package version` with `Version`, `Commit`, `Date` string vars; `--version` flag on the manager binary.

- [x] **Step 1: Write the version package**

```go
/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package version carries build metadata injected at link time.
package version

import "runtime"

// Build information, overridden at link time:
//
//	go build -ldflags "-X .../internal/version.Version=v0.2.0 ..."
var (
	// Version is the semantic version of the operator (e.g. "v0.2.0").
	Version = "dev"
	// Commit is the short git commit the binary was built from.
	Commit = "unknown"
	// Date is the UTC build date (RFC 3339).
	Date = "unknown"
)

// String renders the one-line version banner used by --version and the
// startup log.
func String() string {
	return "keycloak-operator " + Version +
		" (commit " + Commit + ", built " + Date +
		", " + runtime.Version() + ")"
}
```

- [x] **Step 2: Wire the `--version` flag and startup log in `cmd/main.go`**

Add import `"runtime"` and `keycloakversion "github.com/ctn-solutions/keycloak-operator/internal/version"`. In the flag block add:

```go
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Print the operator version and exit.")
```

After `ctrl.SetLogger(...)` insert:

```go
	if showVersion {
		setupLog.Info(keycloakversion.String())
		return
	}
```

And change the final startup line to:

```go
	setupLog.Info("Starting manager", "version", keycloakversion.Version, "commit", keycloakversion.Commit)
```

- [x] **Step 3: ldflags in Makefile and Dockerfile**

Makefile (top, after `IMG ?=`):

```make
# Version metadata baked into the binary. RELEASE_VERSION is set by CI;
# local builds fall back to git describe.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GO_LD_FLAGS ?= -s -w \
  -X github.com/ctn-solutions/keycloak-operator/internal/version.Version=$(VERSION) \
  -X github.com/ctn-solutions/keycloak-operator/internal/version.Commit=$(GIT_COMMIT) \
  -X github.com/ctn-solutions/keycloak-operator/internal/version.Date=$(BUILD_DATE)
```

Change build/run recipes to `go build -ldflags "$(GO_LD_FLAGS)" -o bin/manager cmd/main.go` (run keeps `go run` but exports flags via `go run -ldflags ...`).

Dockerfile builder stage:

```dockerfile
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -a -ldflags "-s -w \
      -X github.com/ctn-solutions/keycloak-operator/internal/version.Version=${VERSION} \
      -X github.com/ctn-solutions/keycloak-operator/internal/version.Commit=${COMMIT} \
      -X github.com/ctn-solutions/keycloak-operator/internal/version.Date=${BUILD_DATE}" \
    -o manager cmd/main.go
```

- [x] **Step 4: Verify**

Run: `make build && ./bin/manager --version && make test`
Expected: version string prints; tests pass.

- [x] **Step 5: Commit** — `feat: report the operator version (--version, startup log, ldflags)`

### Task 2: The installer script `install.sh`

**Files:**
- Create: `install.sh` (executable)
- Create: `test/install/install_script_test.sh`

**Interfaces:**
- Consumes: GitHub release assets `install-<tag>.yaml`, `sha256sums.txt` (from Task 4).
- Produces: CLI `install.sh {install|upgrade|uninstall|version|help}` with flags `--version`, `--namespace`, `--dry-run`, `--purge-crds`, `--yes`; env overrides `KUBECTL`, `KEYCLOAK_OPERATOR_VERSION`, `GITHUB_TOKEN`/`GH_TOKEN` for tests/CI.

- [x] **Step 1: Write `install.sh`** — full script (see repo file; key behaviors):
  - `set -euo pipefail`; no `eval` of downloaded content; downloads to a temp dir.
  - `resolve_latest`: GitHub API `releases/latest` parsed with sed (no jq dependency); honors `GITHUB_TOKEN`/`GH_TOKEN`.
  - `fetch_bundle TAG`: downloads `install-TAG.yaml` + `sha256sums.txt`; verifies with `sha256sum -c` when the sums file exists, warns loudly when it does not (v0.1.0 only).
  - `helm_guard`: refuses when the live deployment carries `app.kubernetes.io/managed-by=Helm`.
  - `install`: `kubectl apply -f` + `kubectl rollout status`.
  - `upgrade`: prints current → target, applies, waits for rollout.
  - `uninstall`: deletes all bundle resources **except** CRDs (stream-split on `^---`, skipping `kind: CustomResourceDefinition` docs) unless `--purge-crds`.
  - `version`: installer version + installed operator version (image tag) + latest release.
- [x] **Step 2: `shellcheck install.sh` and `bash -n`** — clean.
- [x] **Step 3: Script test harness** `test/install/install_script_test.sh` — sources `install.sh` with a fake `kubectl` shim on PATH; asserts: helm-managed refusal, uninstall keeps CRDs / `--purge-crds` deletes them, version comparison logic, checksum verification failure aborts.
- [x] **Step 4: Live download test against v0.1.0** — resolve latest, fetch, verify (warning path), `--dry-run` apply.
- [x] **Step 5: Commit** — `feat: install.sh — one-command install, upgrade and uninstall`

### Task 3: Release workflow — stable bundle name, checksums, version build-arg

**Files:**
- Modify: `.github/workflows/release.yml`

- [x] **Step 1:** Build step gains `build-args: VERSION=${{ github.ref_name }}`; bundle step generates `sha256sums.txt`; release files list includes `dist/install.yaml`, `dist/install-*.yaml`, `dist/sha256sums.txt`.
- [x] **Step 2:** Validate with `actionlint` if available, else YAML parse + eyeball; commit — `ci: stable install bundle name, sha256 checksums and versioned image build`

### Task 4: Common labels on the kustomize bundle

**Files:**
- Modify: `config/default/kustomization.yaml` (add `labels:` block)

- [x] **Step 1:** Add `labels: [{includeSelectors: false, pairs: {app.kubernetes.io/name: keycloak-operator, app.kubernetes.io/managed-by: kustomize}}]`.
- [x] **Step 2:** `make build-installer IMG=ghcr.io/ctn-solutions/keycloak-operator:v0.2.0`; assert every doc in `dist/install.yaml` carries the label (CRDs included) — makes `kubectl delete crd -l app.kubernetes.io/name=keycloak-operator` true.
- [x] **Step 3:** Commit — `feat: label every bundled manifest for label-based lifecycle operations`

### Task 5: Release notes config + CHANGELOG

**Files:**
- Create: `.github/release.yml`
- Create: `CHANGELOG.md`

- [x] **Step 1:** `.github/release.yml` with categories: Breaking changes, Features, Bug fixes, Security, Documentation, CI & build, Other changes (label-driven, `feat`/`fix`/`security`/`docs`/`ci`/`build`).
- [x] **Step 2:** `CHANGELOG.md` (Keep a Changelog): `## [0.2.0] - 2026-08-30` (installer, tap, version reporting, checksums, docs) and `## [0.1.0] - 2026-08-30` (retrospective from git history).
- [x] **Step 3:** Commit — `docs: changelog and categorized release notes configuration`

### Task 6: Release cycle documentation

**Files:**
- Create: `docs/release-process.md`
- Modify: `CONTRIBUTING.md` (release section → link + tap step + changelog step)
- Modify: `docs/upgrading.md` (compatibility row 0.2.x, script upgrade path)
- Modify: `docs/installation.md` (script + brew options, fix broken `install.yaml` link note)
- Modify: `README.md` (dedupe "Why trust it?", quick-install one-liners, fix time-bombed link)

- [x] **Step 1:** `docs/release-process.md`: versioning policy (0.x while v1alpha1; breaking CRD change ⇒ minor bump; 1.0 gates on API graduation), cadence (feature minors ~monthly, patches as needed, security fixes prioritized), support policy (latest minor), full release checklist (Chart bump → green main → tag → verify workflow → update tap via `hack/update-tap.sh` → CHANGELOG → announcements), tap token setup for future automation.
- [x] **Step 2:** Docs edits above; README install section gains `curl -fsSL .../install.sh | bash` and `brew install ctn-solutions/tap/keycloak-operator`.
- [x] **Step 3:** Commit — `docs: release cycle, installer and Homebrew tap documentation`

### Task 7: Homebrew tap assets

**Files:**
- Create: `packaging/homebrew/keycloak-operator.rb.in` (template with `@VERSION@`/`@SHA256@`)
- Create: `hack/update-tap.sh` (renders formula, clones/updates `ctn-solutions/homebrew-tap`, commits+pushes)

- [x] **Step 1:** Formula template (class `KeycloakOperator`, `url` = tag tarball, `sha256`, `bin.install "install.sh" => "keycloak-operator"`, caveats, `test do assert_match "install", shell_output("#{bin}/keycloak-operator help")`).
- [x] **Step 2:** `hack/update-tap.sh VERSION` — computes tarball sha256, renders formula, pushes to the tap repo (uses `gh` auth of the caller).
- [x] **Step 3:** Test locally: render for v0.2.0 after the release exists, `brew install --build-from-source` from a local tap dir, run `keycloak-operator version`.
- [x] **Step 4:** Commit — `feat: Homebrew tap packaging and tap update tooling`

### Task 8: Full verification

- [x] `make lint && make test`
- [x] `bash test/install/install_script_test.sh`
- [x] kind cluster (brew install kind): `./install.sh install` (v0.1.0), verify pods Ready, `--version` flag on the running image
- [x] Reviewer pass on the full diff (reviewer subagent); security-auditor pass (CI/CD + supply chain changes)

### Task 9: Cut v0.2.0 and publish

- [x] **Step 1:** Bump `charts/keycloak-operator/Chart.yaml` to `0.2.0`/`0.2.0`; finalize CHANGELOG 0.2.0 date; commit `chore: release v0.2.0`; push main; tag `v0.2.0` and push.
- [x] **Step 2:** Watch `gh run watch` on the Release workflow; verify release assets: `install.yaml`, `install-v0.2.0.yaml`, `sha256sums.txt`; verify `releases/latest/download/install.yaml` resolves; verify image `v0.2.0` + chart 0.2.0 on GHCR.
- [x] **Step 3:** Live upgrade test on kind: `./install.sh upgrade` v0.1.0 → v0.2.0; `kubectl rollout status`; uninstall test.
- [x] **Step 4:** Create `ctn-solutions/homebrew-tap` (public), push formula for 0.2.0; verify `brew install ctn-solutions/tap/keycloak-operator` and `keycloak-operator version`.
- [x] **Step 5:** Curate the v0.2.0 release notes body (summary + highlights + install/upgrade commands + checksums note) via `gh release edit`.
