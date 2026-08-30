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
#   --version vX.Y.Z   Pin the release (default: latest published release)
#   --namespace NS     Target namespace (default: keycloak-operator-system)
#   --dry-run          Server-side dry run; change nothing
#   --purge-crds       With uninstall: also delete the CRDs. WARNING: this
#                      deletes every Realm, Client, ClientScope, RealmRole,
#                      IdentityProvider and Group resource in the cluster.
#   --yes              Skip the confirmation prompt
#   -h, --help         Show this help
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
# The script never evaluates downloaded content: bundles are written to a
# temporary directory, checksum-verified when the release publishes
# sha256sums.txt, and handed to kubectl as files.

set -euo pipefail

REPO="ctn-solutions/keycloak-operator"
DEFAULT_NAMESPACE="keycloak-operator-system"
DEPLOYMENT="keycloak-operator-controller-manager"
INSTALLER_VERSION="0.2.0"
KUBECTL="${KUBECTL:-kubectl}"

NAMESPACE="$DEFAULT_NAMESPACE"
TARGET_TAG=""
DRY_RUN=0
ASSUME_YES=0
PURGE_CRDS=0
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
  --version vX.Y.Z   Pin the release (default: latest published release)
  --namespace NS     Target namespace (default: keycloak-operator-system)
  --dry-run          Server-side dry run; change nothing
  --purge-crds       With uninstall: also delete the CRDs. WARNING: this
                     deletes every Realm, Client, ClientScope, RealmRole,
                     IdentityProvider and Group resource in the cluster.
  --yes              Skip the confirmation prompt
  -h, --help         Show this help

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

github_curl() {
  # curl with an optional GitHub token; the token only raises the API rate
  # limit (and grants access when the repository is private).
  local url="$1"
  local token="${GITHUB_TOKEN:-${GH_TOKEN:-}}"
  if [ -n "${token}" ]; then
    curl -fsSL --retry 3 -H "Authorization: token ${token}" "${url}"
  else
    curl -fsSL --retry 3 "${url}"
  fi
}

resolve_latest() {
  # Print the tag of the latest published release.
  local tag
  tag="$(github_curl "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
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
  # into destdir and verify its checksum when the release publishes one.
  local tag="$1" dest="$2"
  local base="https://github.com/${REPO}/releases/download/${tag}"
  local bundle="install-${tag}.yaml"

  info "Downloading ${REPO} ${tag}"
  if ! github_curl "${base}/${bundle}" > "${dest}/${bundle}"; then
    die "release asset ${bundle} not found for ${tag} (does the release exist?)."
  fi

  if github_curl "${base}/sha256sums.txt" > "${dest}/sha256sums.txt" 2>/dev/null; then
    local expected actual
    expected="$(grep -E "^[0-9a-f]{64}[[:space:]]+${bundle}\$" "${dest}/sha256sums.txt" | awk '{print $1}')"
    if [ -z "${expected}" ]; then
      warn "sha256sums.txt does not list ${bundle}; skipping verification."
    else
      actual="$(sha256_of "${dest}/${bundle}")"
      if [ "${expected}" != "${actual}" ]; then
        rm -f "${dest}/${bundle}"
        die "checksum mismatch for ${bundle} (expected ${expected}, got ${actual})."
      fi
      info "Checksum verified for ${bundle}"
    fi
  else
    warn "Release ${tag} publishes no sha256sums.txt; the bundle could not be verified."
    warn "This is only expected for v0.1.0. Consider pinning a newer release."
  fi
}

# ---------------------------------------------------------------------------
# Cluster state helpers
# ---------------------------------------------------------------------------

helm_managed_label() {
  # Print the managed-by label of the live deployment (empty when absent).
  "${KUBECTL}" get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" \
    -o jsonpath='{.metadata.labels.app\.kubernetes\.io/managed-by}' 2>/dev/null || true
}

installed_version() {
  # Print the image tag of the installed operator (e.g. "0.2.0").
  local image
  image="$("${KUBECTL}" get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" \
    -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true)"
  [ -n "${image}" ] && printf '%s\n' "${image##*:}"
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
    die "the existing install is managed by Helm; use 'helm upgrade' instead of this script."
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

  if [ "${ASSUME_YES}" != "1" ]; then
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

  info "Removing the operator deployment, RBAC and namespace (CRDs kept)."
  "${KUBECTL}" delete --ignore-not-found -f "${non_crd}"

  if [ "${PURGE_CRDS}" = "1" ]; then
    warn "Deleting CRDs — all custom resources are removed with them."
    "${KUBECTL}" delete crd -l app.kubernetes.io/name=keycloak-operator --ignore-not-found
  else
    info "CRDs and managed resources were kept."
    info "Remove them too with: $0 uninstall --purge-crds --yes"
  fi
  info "keycloak-operator removed."
}

cmd_version() {
  printf 'installer %s\n' "${INSTALLER_VERSION}"
  if "${KUBECTL}" get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" >/dev/null 2>&1; then
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
      --namespace) [ $# -ge 2 ] || die "--namespace requires a value"; NAMESPACE="$2"; shift 2 ;;
      --dry-run) DRY_RUN=1; shift ;;
      --yes) ASSUME_YES=1; shift ;;
      --purge-crds) PURGE_CRDS=1; shift ;;
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
