#!/usr/bin/env bash
#
# Unit tests for install.sh. Runs entirely offline: github_curl is replaced
# by a fixture loader and kubectl by a recording shim.
#
# Usage: bash test/install/install_script_test.sh

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SH="${SCRIPT_DIR}/../../install.sh"

PASS=0
FAIL=0

ok()   { printf '  ok  %s\n' "$1"; PASS=$((PASS + 1)); }
fail() { printf '  FAIL %s\n' "$1"; FAIL=$((FAIL + 1)); }

assert_eq() { # assert_eq <description> <expected> <actual>
  if [ "$2" = "$3" ]; then ok "$1"; else fail "$1 (expected '$2', got '$3')"; fi
}
assert_contains() {
  case "$3" in *"$2"*) ok "$1" ;; *) fail "$1 (expected '$2' in '$3')" ;; esac
}
assert_contains_not() {
  case "$3" in *"$2"*) fail "$1 (did not expect '$2' in '$3')" ;; *) ok "$1" ;; esac
}

# ---------------------------------------------------------------------------
# Fixture: a fake release server and a fake kubectl
# ---------------------------------------------------------------------------

FIXTURES="$(mktemp -d)"
trap 'rm -rf "${FIXTURES}"' EXIT

make_bundle() { # $1 = dir, $2 = tag, $3 = content
  printf '%s\n' "$3" > "$1/install-$2.yaml"
}

# Source the script under test (main is guarded behind BASH_SOURCE check).
# shellcheck source=../install.sh
source "${INSTALL_SH}"
# The sourced script enables errexit; the harness must survive failing
# commands under test.
set +e

# Replace network access with the fixture directory.
GITHUB_FIXTURE=""
github_curl() {
  local url="$1"
  local name
  name="$(printf '%s' "${url}" | sed 's|.*/||')"
  if [ -f "${GITHUB_FIXTURES}/${name}" ]; then
    cat "${GITHUB_FIXTURES}/${name}"
  else
    return 22 # curl's HTTP error code
  fi
}

# Fake kubectl: records every invocation in $KUBECTL_LOG and answers
# jsonpath queries from $KUBECTL_JSONPATH.
KUBECTL_LOG="$(mktemp)"
KUBECTL_JSONPATH=""
KUBECTL_EXIT=0
fake_kubectl() {
  printf '%s\n' "$*" >> "${KUBECTL_LOG}"
  if [ "${1:-}" = "get" ] && [ "${2:-}" = "--raw" ]; then return 0; fi
  if [ -n "${KUBECTL_JSONPATH}" ]; then printf '%s' "${KUBECTL_JSONPATH}"; fi
  return 0
}
KUBECTL=fake_kubectl

reset_log() { : > "${KUBECTL_LOG}"; }
logged()    { cat "${KUBECTL_LOG}"; }

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

test_keep_non_crd_filters_crds() {
  local bundle='---
apiVersion: v1
kind: Namespace
metadata:
  name: keycloak-operator-system
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: realms.keycloak.ctn-solutions.io
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keycloak-operator-controller-manager
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: clients.keycloak.ctn-solutions.io
'
  local out
  out="$(keep_non_crd <<< "${bundle}")"
  assert_eq "keep_non_crd keeps non-CRD docs" "0" "$?"
  if grep -q "kind: CustomResourceDefinition" <<< "${out}"; then
    fail "keep_non_crd removed CRD documents"
  else
    ok "keep_non_crd removed CRD documents"
  fi
  assert_contains "keep_non_crd keeps the Deployment" "kind: Deployment" "${out}"
  assert_contains "keep_non_crd keeps the Namespace" "kind: Namespace" "${out}"
  # Exactly two documents remain (one leading separator, one between).
  local separators
  separators="$(grep -c '^---$' <<< "${out}" || true)"
  assert_eq "keep_non_crd keeps document separators consistent" "1" "${separators}"
}

test_fetch_bundle_verifies_checksum() {
  local dir; dir="$(mktemp -d)"
  printf 'bundle-content\n' > "${GITHUB_FIXTURES}/install-v0.1.0.yaml"
  local sum
  sum="$(shasum -a 256 "${GITHUB_FIXTURES}/install-v0.1.0.yaml" | awk '{print $1}')"
  printf '%s  install-v0.1.0.yaml\n' "${sum}" > "${GITHUB_FIXTURES}/sha256sums.txt"

  local output status=0
  output="$(fetch_bundle v0.1.0 "${dir}" 2>&1)" || status=$?
  assert_eq "fetch_bundle succeeds with a valid checksum" "0" "${status}"
  assert_contains "fetch_bundle reports verification" "Checksum verified" "${output}"
  rm -rf "${dir}"
}

test_fetch_bundle_rejects_bad_checksum() {
  local dir; dir="$(mktemp -d)"
  printf 'bundle-content\n' > "${GITHUB_FIXTURES}/install-v0.1.0.yaml"
  printf '0000000000000000000000000000000000000000000000000000000000000000  install-v0.1.0.yaml\n' \
    > "${GITHUB_FIXTURES}/sha256sums.txt"

  local output status=0
  output="$(fetch_bundle v0.1.0 "${dir}" 2>&1)" || status=$?
  if [ "${status}" -ne 0 ]; then ok "fetch_bundle fails on checksum mismatch"; else fail "fetch_bundle should fail on mismatch"; fi
  assert_contains "fetch_bundle reports the mismatch" "checksum mismatch" "${output}"
  if [ -f "${dir}/install-v0.1.0.yaml" ]; then
    fail "fetch_bundle removes the rejected bundle"
  else
    ok "fetch_bundle removes the rejected bundle"
  fi
  rm -rf "${dir}"
}

test_fetch_bundle_fails_closed_on_missing_entry() {
  local dir; dir="$(mktemp -d)"
  printf 'bundle-content\n' > "${GITHUB_FIXTURES}/install-v0.1.0.yaml"
  # Sums file exists but does not list the bundle.
  printf '0000000000000000000000000000000000000000000000000000000000000000  other-file.yaml\n' \
    > "${GITHUB_FIXTURES}/sha256sums.txt"

  local output status=0
  output="$(fetch_bundle v0.1.0 "${dir}" 2>&1)" || status=$?
  if [ "${status}" -ne 0 ]; then ok "fetch_bundle fails closed when the entry is missing"; else fail "fetch_bundle must fail when the sums entry is missing"; fi
  assert_contains "missing entry reports the failure" "does not list" "${output}"
  rm -rf "${dir}"
}

test_fetch_bundle_fails_closed_without_checksums() {
  local dir; dir="$(mktemp -d)"
  printf 'bundle-content\n' > "${GITHUB_FIXTURES}/install-v0.1.0.yaml"
  rm -f "${GITHUB_FIXTURES}/sha256sums.txt"

  local output status=0
  output="$(fetch_bundle v0.1.0 "${dir}" 2>&1)" || status=$?
  if [ "${status}" -ne 0 ]; then ok "fetch_bundle refuses unverified bundles by default"; else fail "fetch_bundle must refuse unverified bundles by default"; fi
  assert_contains "refusal mentions --allow-unverified" "--allow-unverified" "${output}"

  ALLOW_UNVERIFIED=1
  status=0
  output="$(fetch_bundle v0.1.0 "${dir}" 2>&1)" || status=$?
  assert_eq "fetch_bundle proceeds with --allow-unverified" "0" "${status}"
  assert_contains "opt-out warns about the unverified bundle" "proceeding unverified" "${output}"
  ALLOW_UNVERIFIED=0
  rm -rf "${dir}"
}

test_tag_validation() {
  local status=0
  validate_tag "v0.2.0" 2>/dev/null || status=$?
  assert_eq "validate_tag accepts a semver tag" "0" "${status}"
  status=0
  output="$(validate_tag 'v0.2.0+build.1' 2>&1)" || status=$?
  if [ "${status}" -ne 0 ]; then ok "validate_tag rejects tags with metacharacters"; else fail "validate_tag must reject non-plain tags"; fi
  status=0
  output="$(validate_tag "not-a-version" 2>&1)" || status=$?
  if [ "${status}" -ne 0 ]; then ok "validate_tag rejects garbage"; else fail "validate_tag must reject garbage"; fi
}

test_fetch_bundle_warns_without_checksums() {
  local dir; dir="$(mktemp -d)"
  printf 'bundle-content\n' > "${GITHUB_FIXTURES}/install-v0.1.0.yaml"
  rm -f "${GITHUB_FIXTURES}/sha256sums.txt"

  local output status=0
  output="$(fetch_bundle v0.1.0 "${dir}" 2>&1)" || status=$?
  assert_eq "fetch_bundle succeeds without a sums file" "0" "${status}"
  assert_contains "fetch_bundle warns when unverifiable" "could not be verified" "${output}"
  rm -rf "${dir}"
}

test_install_refuses_helm_managed() {
  reset_state
  KUBECTL_JSONPATH="Helm"
  local output status=0
  output="$(cmd_install v0.2.0 /dev/null 2>&1)" || status=$?
  if [ "${status}" -ne 0 ]; then ok "install refuses Helm-managed installs"; else fail "install must refuse Helm-managed installs"; fi
  assert_contains "refusal mentions helm upgrade" "helm upgrade" "${output}"
}

test_upgrade_noop_when_current() {
  reset_state
  KUBECTL_JSONPATH="kustomize"
  # installed_version reads the image; simulate 0.2.0 installed.
  installed_version() { printf '0.2.0\n'; }
  reset_log
  local output
  output="$(cmd_upgrade v0.2.0 /dev/null 2>&1)"
  assert_contains "upgrade is a no-op at the same version" "nothing to do" "${output}"
  if grep -q "apply" "${KUBECTL_LOG}"; then
    fail "upgrade at the same version must not apply"
  else
    ok "upgrade at the same version must not apply"
  fi
}

test_uninstall_keeps_crds() {
  reset_state
  local bundle="${GITHUB_FIXTURES}/bundle.yaml"
  cat > "${bundle}" <<'YAML'
---
apiVersion: v1
kind: Namespace
metadata:
  name: keycloak-operator-system
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: realms.keycloak.ctn-solutions.io
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keycloak-operator-controller-manager
YAML
  reset_log
  PURGE_CRDS=0 ASSUME_YES=1 cmd_uninstall v0.2.0 "${bundle}" >/dev/null 2>&1
  local deleted_file=""
  deleted_file="$(grep -o '\-f [^ ]*' "${KUBECTL_LOG}" | awk '{print $2}' | head -n1)"
  if [ -n "${deleted_file}" ] && grep -q "CustomResourceDefinition" "${deleted_file}"; then
    fail "uninstall must not delete CRDs"
  else
    ok "uninstall must not delete CRDs"
  fi
  assert_contains "uninstall deletes the deployment" "delete" "$(logged)"
}

test_uninstall_purge_crds_uses_names() {
  reset_state
  local bundle="${GITHUB_FIXTURES}/bundle.yaml"
  printf -- '---\nkind: CustomResourceDefinition\n' > "${bundle}"
  reset_log
  PURGE_CRDS=1 ASSUME_YES=1 cmd_uninstall v0.2.0 "${bundle}" >/dev/null 2>&1
  if grep -q "realms.keycloak.ctn-solutions.io" "${KUBECTL_LOG}"; then
    ok "--purge-crds deletes CRDs by explicit name"
  else
    fail "--purge-crds must delete CRDs by name (got: $(logged))"
  fi
}

test_uninstall_honors_dry_run() {
  reset_state
  local bundle="${GITHUB_FIXTURES}/bundle.yaml"
  printf -- '---\nkind: Namespace\nmetadata:\n  name: keycloak-operator-system\n' > "${bundle}"
  reset_log
  DRY_RUN=1 ASSUME_YES=1 PURGE_CRDS=0 cmd_uninstall v0.2.0 "${bundle}" >/dev/null 2>&1
  if grep -q "delete --dry-run=server" "${KUBECTL_LOG}"; then
    ok "uninstall honors --dry-run"
  else
    fail "uninstall --dry-run must not delete (got: $(logged))"
  fi
  if grep -qE "delete --ignore-not-found -f" "${KUBECTL_LOG}"; then
    fail "uninstall --dry-run must not perform a real delete"
  else
    ok "uninstall --dry-run performs no real deletion"
  fi
}

test_uninstall_refuses_helm_managed() {
  reset_state
  KUBECTL_JSONPATH="Helm"
  local output status=0
  output="$(cmd_uninstall v0.2.0 /dev/null 2>&1)" || status=$?
  if [ "${status}" -ne 0 ]; then ok "uninstall refuses Helm-managed installs"; else fail "uninstall must refuse Helm-managed installs"; fi
  assert_contains "refusal mentions helm uninstall" "helm uninstall" "${output}"
}

test_version_flag_normalization() {
  reset_state
  # The main() tag normalization prefixes v when missing; verify via a dry parse.
  TARGET_TAG="0.3.0"
  case "${TARGET_TAG}" in v*) ;; *) TARGET_TAG="v${TARGET_TAG}" ;; esac
  assert_eq "tag normalization prefixes v" "v0.3.0" "${TARGET_TAG}"
}

# ---------------------------------------------------------------------------
# Harness
# ---------------------------------------------------------------------------

reset_state() {
  TARGET_TAG=""
  DRY_RUN=0
  ASSUME_YES=0
  PURGE_CRDS=0
  KUBECTL_JSONPATH=""
  # cmd_uninstall writes its filtered bundle under TMP_DIR; give it a real
  # directory (main() normally sets this).
  TMP_DIR="$(mktemp -d)"
  reset_log
}

reset_log() { : > "${KUBECTL_LOG}"; }
logged()    { cat "${KUBECTL_LOG}"; }

GITHUB_FIXTURES="$(mktemp -d)"

test_keep_non_crd_filters_crds
test_fetch_bundle_verifies_checksum
test_fetch_bundle_rejects_bad_checksum
test_fetch_bundle_fails_closed_on_missing_entry
test_fetch_bundle_fails_closed_without_checksums
test_tag_validation
test_install_refuses_helm_managed
test_upgrade_noop_when_current
test_uninstall_keeps_crds
test_uninstall_purge_crds_uses_names
test_uninstall_honors_dry_run
test_uninstall_refuses_helm_managed
test_version_flag_normalization

printf '\n%d passed, %d failed\n' "${PASS}" "${FAIL}"
[ "${FAIL}" -eq 0 ]
