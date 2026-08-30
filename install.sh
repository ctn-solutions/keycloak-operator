#!/usr/bin/env bash
#
# keycloak-operator installer.
#
# Installs, upgrades or uninstalls the keycloak-operator on the current
# Kubernetes context from the published GitHub release bundle.
#
# Usage:
#   install.sh [command] [options]
#
# Commands:
#   install     Install the operator (default)
#   upgrade     Upgrade an existing manifest-based install
#   uninstall   Remove the operator (CRDs are kept unless --purge-crds)
#   version     Print installer, installed-operator and latest versions
#
# Options:
#   --version vX.Y.Z     Pin the release (default: latest published release)
#   --dry-run            Server-side dry run; change nothing
#   --purge-crds         With uninstall: also delete the CRDs. WARNING: this
#                        deletes every Realm, Client, ClientScope, RealmRole,
#                        IdentityProvider and Group resource in the cluster.
#   --allow-unverified   Proceed when the release publishes no checksums
#                        (only needed for v0.1.0, which predates checksums)
#   --yes                Skip the confirmation prompt
#   -h, --help           Show this help
#
# Environment:
#   GITHUB_TOKEN / GH_TOKEN   GitHub token (avoids API rate limits; required
#                             for releases of private repositories)
#   KUBECTL                   kubectl binary to use (default: kubectl)
#
# Examples:
#   install.sh                          # install the latest release
#   install.sh install --version v0.2.0
#   install.sh upgrade
#   install.sh uninstall --purge-crds --yes
#
# The manifest bundle installs into the keycloak-operator-system namespace;
# for other namespaces use the Helm chart. The script never evaluates
# downloaded content: bundles are written to a temporary directory,
# checksum-verified against the release's sha256sums.txt (and
# signature-verified with cosign when it is installed), and handed to
# kubectl as files.

set -euo pipefail

REPO="ctn-solutions/keycloak-operator"
NAMESPACE="keycloak-operator-system"
DEPLOYMENT_LABEL="app.kubernetes.io/name=keycloak-operator"
DEPLOYMENT="keycloak-operator-controller-manager"
CRD_NAMES="clients.keycloak.ctn-solutions.io clientscopes.keycloak.ctn-solutions.io groups.keycloak.ctn-solutions.io identityproviders.keycloak.ctn-solutions.io keycloakconnections.keycloak.ctn-solutions.io realmroles.keycloak.ctn-solutions.io realms.keycloak.ctn-solutions.io"
DEPLOYMENT="keycloak-operator-controller-manager"
INSTALLER_VERSION="0.2.0"
KUBECTL="${KUBECTL:-kubectl}"

TARGET_TAG=""
DRY_RUN=0
ASSUME_YES=0
PURGE_CRDS=0
ALLOW_UNVERIFIED=0
TMP_DIR=""

info() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
keycloak-operator installer

Usage: install.sh [command] [options]

Commands:
  install     Install the operator (default)
  upgrade     Upgrade an existing manifest-based install
  uninstall   Remove the operator (CRDs are kept unless --purge-crds)
  version     Print installer, installed-operator and latest versions

Options:
  --version vX.Y.Z     Pin the release (default: latest published release)
  --dry-run            Server-side dry run; change nothing
  --purge-crds         With uninstall: also delete the CRDs. WARNING: this
                       deletes every Realm, Client, ClientScope, RealmRole,
                       IdentityProvider and Group resource in the cluster.
  --allow-unverified   Proceed when the release publishes no checksums
                       (only needed for v0.1.0)
  --yes                Skip the confirmation prompt
  -h, --help           Show this help

Environment:
  GITHUB_TOKEN / GH_TOKEN   GitHub token (avoids API rate limits; required
                            for releases of private repositories)
  KUBECTL                   kubectl binary to use (default: kubectl)

Examples:
  install.sh                          # install the latest release
  install.sh install --version v0.2.0
  install.sh upgrade
  install.sh uninstall --purge-crds --yes
EOF
  exit 0
}

# ---------------------------------------------------------------------------
# Prerequisites and release resolution
# ---------------------------------------------------------------------------

require_tools() {
  command -v curl >/dev/null 2>&1 || die "curl is required but was not found in PATH."
  command -v "${KUBECTL}" >/dev/null 2>&1 \
    || die "kubectl not found in PATH. Install kubectl first: https://kubernetes.io/docs/tasks/tools/"
}

require_cluster() {
  "${KUBECTL}" get --raw /healthz >/dev/null 2>&1 \
    || die "cannot reach the cluster of the current kubeconfig context."
}

validate_tag() {
  # Accept only plain semver tags: the value is embedded in download URLs
  # and matched against sha256sums.txt, so no shell/regex metacharacters.
  if [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
    return 0
  fi
  die "invalid version '$1': expected vX.Y.Z (semver, no build metadata)"
}

github_curl() {
  # curl with an optional GitHub token. The token is passed through a
  # config read from stdin so it never appears in the process list.
  local url="$1"
  local token="${GITHUB_TOKEN:-${GH_TOKEN:-}}"
  if [ -n "${token}" ]; then
    printf 'header = "Authorization: token %s"\n' "${token}" | curl -K - -fsSL --retry 3 "${url}"
  else
    curl -fsSL --retry 3 "${url}"
  fi
}

resolve_latest() {
  # Print the tag of the latest published release.
  local tag
  tag="$(github_curl "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)" \
    || die "could not query the latest release (network error or rate limit; set GITHUB_TOKEN)."
  [ -n "${tag}" ] || die "could not determine the latest release (rate limited? set GITHUB_TOKEN)."
  printf '%s\n' "${tag}"
}

sha256_of() {
  # Print the sha256 hex digest of a file (portable across GNU/macOS).
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

fetch_bundle() {
  # fetch_bundle <tag> <destdir>: download the install bundle for a release
  # into destdir and verify its integrity. Fails closed: without a usable
  # checksum the bundle is rejected unless --allow-unverified was passed.
  local tag="$1" dest="$2"
  local base="https://github.com/${REPO}/releases/download/${tag}"
  local bundle="install-${tag}.yaml"

  info "Downloading ${REPO} ${tag}"
  if ! github_curl "${base}/${bundle}" > "${dest}/${bundle}"; then
    die "release asset ${bundle} not found for ${tag} (does the release exist?)."
  fi

  if ! github_curl "${base}/sha256sums.txt" > "${dest}/sha256sums.txt" 2>/dev/null; then
    if [ "${ALLOW_UNVERIFIED}" = "1" ]; then
      warn "Release ${tag} publishes no sha256sums.txt; proceeding unverified (--allow-unverified)."
      return 0
    fi
    die "release ${tag} publishes no sha256sums.txt; refusing an unverified bundle. Re-run with --allow-unverified to accept the risk."
  fi

  # Exact filename match (second field of the sha256sum format); immune to
  # regex metacharacters that legal git tags may contain.
  local expected actual
  expected="$(awk -v f="${bundle}" '$2 == f {print $1}' "${dest}/sha256sums.txt")"
  if [ -z "${expected}" ]; then
    die "sha256sums.txt does not list ${bundle}; refusing an unverifiable bundle."
  fi
  local actual
  actual="$(sha256_of "${dest}/${bundle}")"
  if [ "${expected}" != "${actual}" ]; then
    rm -f "${dest}/${bundle}"
    die "checksum mismatch for ${bundle} (expected ${expected}, got ${actual})."
  fi
  info "Checksum verified for ${bundle}"

  # When cosign is installed, also verify the authenticity of the checksum
  # manifest itself (keyless signature published with the release). The
  # checksum alone only protects against corruption, not tampering.
  if command -v cosign >/dev/null 2>&1; then
    if github_curl "${base}/sha256sums.txt.cosign.bundle" > "${dest}/sha256sums.txt.cosign.bundle" 2>/dev/null; then
      if cosign verify-blob \
        --bundle "${dest}/sha256sums.txt.cosign.bundle" \
        --certificate-identity-regexp '^https://github.com/ctn-solutions/keycloak-operator/\.github/workflows/release\.yml@refs/tags/' \
        --certificate-oidc-issuer https://token.actions.githubusercontent.com \
        "${dest}/sha256sums.txt" >/dev/null 2>&1; then
        info "Signature verified for sha256sums.txt (keyless cosign)"
      else
        die "signature verification failed for sha256sums.txt; the release may be tampered with."
      fi
    else
      warn "Release publishes no cosign bundle; the checksums could not be signature-verified."
    fi
  else
    info "cosign not installed; skipping signature verification (checksums still verified)."
  fi
}

# ---------------------------------------------------------------------------
# Cluster state helpers
# ---------------------------------------------------------------------------

helm_managed_label() {
  # Print the managed-by label of any keycloak-operator deployment in the
  # namespace (the Helm chart and the kustomize bundle use different
  # deployment names but the same app label).
  "${KUBECTL}" get deployments -n "${NAMESPACE}" -l "${DEPLOYMENT_LABEL}" \
    -o jsonpath='{range .items[*]}{.metadata.labels.app\.kubernetes\.io/managed-by}{"\n"}{end}' 2>/dev/null \
    | head -n1 || true
}

installed_version() {
  # Print the version of the installed operator (e.g. "0.2.0"), with any
  # leading "v" stripped so it compares equal to the release tag.
  local image
  image="$("${KUBECTL}" get deployment -n "${NAMESPACE}" -l "${DEPLOYMENT_LABEL}" \
    -o jsonpath='{.items[0].spec.template.spec.containers[0].image}' 2>/dev/null || true)"
  [ -n "${image}" ] || return 0
  local tag="${image##*:}"
  case "${tag}" in v*) tag="${tag#v}" ;; esac
  printf '%s\n' "${tag}"
  return 0
}

# Emit stdin minus every CustomResourceDefinition document, keeping the
# remaining documents separated by '---'.
keep_non_crd() {
  awk '
    function flush_doc() {
      if (doc == "") return
      if (!is_crd) {
        if (emitted++) printf "---\n"
        printf "%s", doc
      }
      doc = ""
      is_crd = 0
    }
    /^---[ \t]*$/ { flush_doc(); next }
    /^kind:[ \t]*CustomResourceDefinition[ \t]*$/ { is_crd = 1 }
    { doc = doc $0 "\n" }
    END { flush_doc() }'
}

# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

cmd_install() {
  local tag="$1" bundle="$2"

  if [ "$(helm_managed_label)" = "Helm" ]; then
    die "an existing install is managed by Helm; use 'helm upgrade' instead of this script."
  fi

  info "Installing keycloak-operator ${tag} (namespace ${NAMESPACE})"
  if [ "${DRY_RUN}" = "1" ]; then
    "${KUBECTL}" apply --dry-run=server -f "${bundle}"
  else
    "${KUBECTL}" apply -f "${bundle}"
    info "Waiting for the operator to become ready…"
    "${KUBECTL}" rollout status "deployment/${DEPLOYMENT}" -n "${NAMESPACE}" --timeout=180s
    info "keycloak-operator ${tag} is installed."
    info "Next step: declare a KeycloakConnection — https://github.com/${REPO}#quickstart"
  fi
}

cmd_upgrade() {
  local tag="$1" bundle="$2"

  if [ "$(helm_managed_label)" = "Helm" ]; then
    die "the existing install is managed by Helm; use 'helm upgrade' instead of this script."
  fi

  local current
  current="$(installed_version)"
  if [ -n "${current}" ] && [ "${current}" = "${tag#v}" ]; then
    info "Already at ${tag}; nothing to do."
    return 0
  fi
  if [ -n "${current}" ]; then
    info "Upgrading keycloak-operator ${current} -> ${tag#v}"
  else
    info "No existing install detected; installing ${tag#v}"
  fi

  if [ "${DRY_RUN}" = "1" ]; then
    "${KUBECTL}" apply --dry-run=server -f "${bundle}"
    return 0
  fi

  "${KUBECTL}" apply -f "${bundle}"
  info "Waiting for the operator to become ready…"
  "${KUBECTL}" rollout status "deployment/${DEPLOYMENT}" -n "${NAMESPACE}" --timeout=180s
  info "keycloak-operator is now at ${tag#v}."
}

cmd_uninstall() {
  local tag="$1" bundle="$2"

  if [ "$(helm_managed_label)" = "Helm" ]; then
    die "the existing install is managed by Helm; use 'helm uninstall' instead of this script."
  fi

  if [ "${ASSUME_YES}" != "1" ] && [ "${DRY_RUN}" != "1" ]; then
    if [ "${PURGE_CRDS}" = "1" ]; then
      warn "--purge-crds deletes every Realm, Client, ClientScope, RealmRole,"
      warn "IdentityProvider and Group resource in the cluster."
    fi
    local reply
    read -r -p "Remove keycloak-operator from namespace ${NAMESPACE}? [y/N] " reply
    case "${reply}" in y|Y|yes|YES) ;; *) info "Aborted."; return 0 ;; esac
  fi

  # Delete everything in the bundle except the CRDs: deleting a CRD
  # cascade-deletes every custom resource of that kind in the cluster.
  local non_crd="${TMP_DIR}/uninstall.yaml"
  keep_non_crd < "${bundle}" > "${non_crd}"

  if [ "${DRY_RUN}" = "1" ]; then
    info "Dry run: the following resources would be removed (CRDs kept):"
    "${KUBECTL}" delete --dry-run=server --ignore-not-found -f "${non_crd}"
    if [ "${PURGE_CRDS}" = "1" ]; then
      info "Dry run: the following CRDs would also be removed:"
      # shellcheck disable=SC2086
      "${KUBECTL}" get crd ${CRD_NAMES} --ignore-not-found -o name
    fi
    return 0
  fi

  info "Removing the operator deployment, RBAC and namespace (CRDs kept)."
  "${KUBECTL}" delete --ignore-not-found -f "${non_crd}"

  if [ "${PURGE_CRDS}" = "1" ]; then
    warn "Deleting CRDs — all custom resources are removed with them."
    # Delete by explicit name: CRDs from releases before v0.2.0 carry no
    # common label, so a label selector would silently match nothing.
    # shellcheck disable=SC2086
    "${KUBECTL}" delete crd --ignore-not-found ${CRD_NAMES}
  else
    info "CRDs and managed resources were kept."
    info "Remove them too with: $0 uninstall --purge-crds --yes"
  fi
  info "keycloak-operator removed."
}

cmd_version() {
  printf 'installer %s\n' "${INSTALLER_VERSION}"
  if "${KUBECTL}" get deployment -n "${NAMESPACE}" -l "${DEPLOYMENT_LABEL}" >/dev/null 2>&1; then
    printf 'installed operator %s\n' "$(installed_version || printf unknown)"
  else
    printf 'installed operator (not installed)\n'
  fi
  local latest
  if latest="$(resolve_latest 2>/dev/null)"; then
    printf 'latest release %s\n' "${latest}"
  else
    printf 'latest release (unavailable)\n'
  fi
}

# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

main() {
  local command="install"
  while [ $# -gt 0 ]; do
    case "$1" in
      install|upgrade|uninstall|version) command="$1"; shift ;;
      -h|--help) usage ;;
      --version) [ $# -ge 2 ] || die "--version requires a value"; TARGET_TAG="$2"; shift 2 ;;
      --dry-run) DRY_RUN=1; shift ;;
      --yes) ASSUME_YES=1; shift ;;
      --purge-crds) PURGE_CRDS=1; shift ;;
      --allow-unverified) ALLOW_UNVERIFIED=1; shift ;;
      *) die "unknown argument: $1 (see --help)" ;;
    esac
  done

  if [ "${command}" = "version" ]; then
    cmd_version
    return 0
  fi

  require_tools
  require_cluster

  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT

  if [ -z "${TARGET_TAG}" ]; then
    info "Resolving the latest release"
    TARGET_TAG="$(resolve_latest)"
  fi
  case "${TARGET_TAG}" in v*) ;; *) TARGET_TAG="v${TARGET_TAG}" ;; esac
  validate_tag "${TARGET_TAG}"

  fetch_bundle "${TARGET_TAG}" "${TMP_DIR}"
  BUNDLE="${TMP_DIR}/install-${TARGET_TAG}.yaml"

  case "${command}" in
    install)   cmd_install   "${TARGET_TAG}" "${BUNDLE}" ;;
    upgrade)   cmd_upgrade   "${TARGET_TAG}" "${BUNDLE}" ;;
    uninstall) cmd_uninstall "${TARGET_TAG}" "${BUNDLE}" ;;
  esac
}

# Allow sourcing for testing; run main only when executed directly.
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  main "$@"
fi
