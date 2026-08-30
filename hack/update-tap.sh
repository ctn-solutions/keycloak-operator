#!/usr/bin/env bash
#
# Update the Homebrew tap formula for a release.
#
# Usage: hack/update-tap.sh vX.Y.Z
#
# Renders packaging/homebrew/keycloak-operator.rb.in with the release
# version and the sha256 of the release tarball, then commits and pushes
# the result to ctn-solutions/homebrew-tap. Requires push access to the
# tap repository (uses the gh/git credentials of the caller).

set -euo pipefail

REPO="ctn-solutions/keycloak-operator"
TAP_REPO="ctn-solutions/homebrew-tap"
FORMULA_PATH="Formula/keycloak-operator.rb"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/packaging/homebrew/keycloak-operator.rb.in"

die() { printf 'error: %s\n' "$*" >&2; exit 1; }

[ $# -eq 1 ] || die "usage: $0 vX.Y.Z"
TAG="$1"
case "${TAG}" in v*) ;; *) TAG="v${TAG}" ;; esac
VERSION="${TAG#v}"

command -v gh >/dev/null 2>&1 || die "gh is required (https://cli.github.com/)."
command -v shasum >/dev/null 2>&1 || command -v sha256sum >/dev/null 2>&1 \
  || die "shasum or sha256sum is required."
[ -f "${TEMPLATE}" ] || die "formula template not found at ${TEMPLATE}"

WORK="$(mktemp -d)"
trap 'rm -rf "${WORK}"' EXIT

echo "==> Fetching the ${TAG} tarball checksum"
TARBALL_URL="https://github.com/${REPO}/archive/refs/tags/${TAG}.tar.gz"
if command -v shasum >/dev/null 2>&1; then
  SHA256="$(curl -fsSL "${TARBALL_URL}" | shasum -a 256 | awk '{print $1}')" \
    || die "could not download ${TARBALL_URL}"
else
  SHA256="$(curl -fsSL "${TARBALL_URL}" | sha256sum | awk '{print $1}')" \
    || die "could not query the latest release (network error)."
fi
[ -n "${SHA256}" ] || die "could not compute the tarball checksum"

echo "==> Rendering the formula for ${VERSION}"
sed -e "s|@VERSION@|${VERSION}|g" -e "s|@SHA256@|${SHA256}|g" "${TEMPLATE}" \
  > "${WORK}/keycloak-operator.rb"

echo "==> Cloning ${TAP_REPO}"
gh repo clone "${TAP_REPO}" "${WORK}/tap" -- --depth 1
mkdir -p "${WORK}/tap/Formula"
cp "${WORK}/keycloak-operator.rb" "${WORK}/tap/${FORMULA_PATH}"

git -C "${WORK}/tap" add "${FORMULA_PATH}"
if git -C "${WORK}/tap" diff --cached --quiet; then
  echo "Formula already at ${VERSION}; nothing to do."
  exit 0
fi

git -C "${WORK}/tap" commit -m "keycloak-operator ${VERSION}"
git -C "${WORK}/tap" push origin HEAD

echo "==> Tap updated for ${VERSION}"
echo "    Verify with: brew install ctn-solutions/tap/keycloak-operator"
