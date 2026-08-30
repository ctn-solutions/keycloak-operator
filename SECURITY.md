# Security policy

## Supported versions

Only the latest released version receives security fixes.

## Reporting a vulnerability

**Do not open a public issue for security vulnerabilities.**

Report privately through
[GitHub security advisories](https://github.com/ctn-solutions/keycloak-operator/security/advisories/new)
or by email to **security@ctn-solutions.com**. Include a description, the
affected version, reproduction steps and any known mitigations. You will
receive an acknowledgement within 72 hours.

## Security model

- The operator holds cluster-wide Secret read access (connection credentials
  may live in any namespace) and writes exported client secrets next to the
  resources that own them. Grant `KeycloakConnection` and `Client` resource
  access to trusted users only.
- Sensitive values are never stored in custom resources: they are injected
  from Secrets at reconciliation time and never recorded in status, events or
  annotations.
- The metrics endpoint requires authentication and authorization by default.
- Container images are multi-arch, built from a distroless base, signed with
  [Sigstore cosign](https://github.com/sigstore/cosign) (keyless) and carry
  build provenance and SBOM attestations. Verify before deploying:

  ```bash
  cosign verify \
    --certificate-identity-regexp 'https://github.com/ctn-solutions/keycloak-operator/\.github/workflows/.*' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com \
    ghcr.io/ctn-solutions/keycloak-operator:<tag>
  ```
- Release bundles (`install.yaml`, `install-vX.Y.Z.yaml`) ship with a
  `sha256sums.txt` manifest that the installer verifies, and the manifest
  itself is signed keyless. Verify the checksums manifest before trusting
  a downloaded bundle:

  ```bash
  cosign verify-blob \
    --bundle sha256sums.txt.cosign.bundle \
    --certificate-identity-regexp 'https://github.com/ctn-solutions/keycloak-operator/\.github/workflows/release\.yml@refs/tags/v.*' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com \
    sha256sums.txt
  ```
