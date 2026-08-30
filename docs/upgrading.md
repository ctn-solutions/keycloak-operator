# Upgrading

## Compatibility matrix

| Operator version | Keycloak servers | Kubernetes | Chart |
|---|---|---|---|
| `v0.1.x` | 26.0 – 26.3 (integration-tested) | 1.25+ | `0.1.0` |

The integration suite runs against every listed Keycloak version on every
push and before every release. When a new Keycloak minor version is
released, add it to the matrix in `.github/workflows/e2e.yml` and
`.github/workflows/release.yml` and verify before claiming support.

## Upgrading the operator

### Helm

```bash
# 1. Update the CRDs first — Helm never upgrades CRDs from crds/.
kubectl apply -f charts/keycloak-operator/crds/

# 2. Upgrade the release.
helm upgrade keycloak-operator oci://ghcr.io/ctn-solutions/charts/keycloak-operator \
  --version <new-version> \
  -n keycloak-operator-system
```

### Plain manifest

```bash
kubectl apply -f https://github.com/ctn-solutions/keycloak-operator/releases/latest/download/install.yaml
```

The bundle includes the current CRDs and the deployment pinned to the
released image.

### Verify after upgrading

```bash
kubectl -n keycloak-operator-system rollout status deployment/keycloak-operator
kubectl get keycloakconnections,realms,clients -A -o wide
# Every managed resource should report Synced=True within one resync period.
```

## CRD upgrades — read this

Helm **installs** CRDs from the chart's `crds/` directory but **never
upgrades** them. This is deliberate: CRD changes can affect every resource
in the cluster and must not happen implicitly during a release upgrade.

Always apply the CRDs manually as part of an operator upgrade, in this
order:

1. `kubectl apply -f` the new CRDs
2. Upgrade the operator release

The operator is backward-compatible with resources created by older
versions; no manual migration of resource manifests is required within a
major version.

## Rolling back

```bash
helm rollback keycloak-operator <revision> -n keycloak-operator-system
```

Rolling back the deployment does **not** roll back the CRDs or any state on
the Keycloak servers. Server-side resources keep their current state; the
previous operator version reconciles them against the same specs.

## Maintenance notes

- **Resync period.** The drift-correction pass issues one Admin API read per
  managed resource per interval. Size `resyncPeriod` for your server: with
  the default 1h and a few hundred resources this is negligible; with
  thousands of resources consider 2h+ or a scoped install.
- **Credential rotation.** Rotate the credentials Secret at any time — the
  operator picks up the new values on the next reconciliation (the
  connection cache invalidates on Secret changes). No operator restart
  needed.
- **Keycloak server upgrades.** The operator talks to the Admin API of the
  configured server; upgrading the server does not require an operator
  change as long as the version stays within the compatibility matrix.
- **Multiple replicas.** Safe with leader election (enabled by default):
  only the leader reconciles.
