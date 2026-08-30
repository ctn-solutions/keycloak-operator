# Contributing

Thank you for considering a contribution.

## Development setup

- Go 1.26+
- Docker (for end-to-end tests)
- kubectl and Helm 3

```bash
make generate manifests   # regenerate deepcopy code and CRDs after API changes
make test                 # unit and controller tests (envtest, no cluster needed)
make lint                 # golangci-lint
```

## Guidelines

- Keep the 1:1 mapping promise: a spec field must match the corresponding
  Keycloak representation field in name, JSON tag and semantics. Scalar spec
  fields are pointers so unset, explicit zero and default stay distinguishable.
- Every behaviour change needs a test. Controller tests run against an
  in-memory Keycloak Admin API server (`internal/keycloak/fake`); behaviours
  that depend on real server semantics belong in the end-to-end suite.
- Regenerate manifests and deepcopy code whenever API markers change, and
  include them in the same commit.
- Commit messages follow the Conventional Commits style (`feat:`, `fix:`,
  `docs:`, `test:`, `chore:`).

## Release process

Releases are fully automated — a maintainer only cuts a tag:

1. Update `Chart.yaml` (`version` and `appVersion`) to match the new tag.
2. Verify `main` is green (CI + the Keycloak integration matrix).
3. Tag and push: `git tag -a vX.Y.Z -m "..." && git push origin vX.Y.Z`.

The release workflow runs the integration suite against every supported
Keycloak version on the tagged commit, builds and signs the multi-arch
image, publishes the Helm chart to `oci://ghcr.io/ctn-solutions/charts`,
attaches the install bundle to the GitHub release and generates the release
notes. A failed integration job aborts the release.

## Reporting issues

Open a GitHub issue with the operator logs, the resource manifests involved
(redact secrets), and the Keycloak server version.
