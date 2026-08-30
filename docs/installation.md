# Installation

Three ways to install the operator. All of them install the CRDs first —
Helm from the chart's `crds/` directory, the manifest bundle inline.

## Prerequisites

- Kubernetes **1.25+**
- Helm **3.8+** (for the OCI chart) or `kubectl` alone
- A Keycloak **26.x** server reachable from the cluster
- Administration credentials for that server (a master admin user, or a
  realm-scoped service account — see the [authentication
  model](metrics.md#authentication-model-relevant-to-metrics))

## Option 1 — Helm from the OCI registry (recommended)

```bash
helm install keycloak-operator oci://ghcr.io/ctn-solutions/charts/keycloak-operator \
  --namespace keycloak-operator-system \
  --create-namespace
```

Pin a version and verify before installing:

```bash
helm show chart oci://ghcr.io/ctn-solutions/charts/keycloak-operator --version 0.1.0
helm install keycloak-operator oci://ghcr.io/ctn-solutions/charts/keycloak-operator \
  --version 0.1.0 \
  --namespace keycloak-operator-system --create-namespace
```

## Option 2 — Helm from a checkout

```bash
git clone https://github.com/ctn-solutions/keycloak-operator.git
cd keycloak-operator
helm install keycloak-operator charts/keycloak-operator \
  --namespace keycloak-operator-system --create-namespace
```

## Option 3 — Plain manifest (no Helm)

```bash
kubectl apply -f https://github.com/ctn-solutions/keycloak-operator/releases/latest/download/install.yaml
```

The bundle contains the CRDs, the `keycloak-operator-system` namespace, RBAC
and the deployment, pinned to the released image.

## Namespace-scoped install

By default the operator watches **all namespaces** and holds cluster-wide
RBAC. To confine it to its own namespace (Role/RoleBinding instead of
ClusterRole, cache restricted to the release namespace):

```bash
helm install keycloak-operator charts/keycloak-operator \
  --namespace keycloak-operator-system --create-namespace \
  --set rbac.clusterScoped=false
```

See [example values profiles](https://github.com/ctn-solutions/keycloak-operator/tree/main/charts/keycloak-operator/examples)
for ready-made configurations: namespace-scoped, metrics with a
ServiceMonitor, and tuned resources.

## Verifying the installation

```bash
kubectl -n keycloak-operator-system get pods
kubectl -n keycloak-operator-system logs deployment/keycloak-operator --tail=20
```

The operator is ready when the deployment reports `1/1 READY` and the log
ends with `Starting manager`. The CRDs are discoverable immediately:

```bash
kubectl explain realm.spec
kubectl get crd | grep ctn-solutions.io
```

## Configuration

All chart values are documented in
[`charts/keycloak-operator/values.yaml`](https://github.com/ctn-solutions/keycloak-operator/blob/main/charts/keycloak-operator/values.yaml).
The most common ones:

| Value | Default | Purpose |
|---|---|---|
| `rbac.clusterScoped` | `true` | `false` installs Role/RoleBinding and restricts the cache to the release namespace |
| `resyncPeriod` | `1h` | Drift-correction interval |
| `leaderElection.enabled` | `true` | Safe multi-replica operation |
| `metrics.secure` | `true` | Authenticated metrics endpoint; `false` serves plain HTTP |
| `metrics.service.enabled` | `false` | Expose the metrics Service for scraping |
| `metrics.serviceMonitor.enabled` | `false` | Create a Prometheus ServiceMonitor (requires `metrics.service.enabled`) |
| `resources` | modest requests/limits | Container resources |

## Uninstalling

```bash
helm uninstall keycloak-operator -n keycloak-operator-system
```

**CRDs are not removed** by Helm (by design — deleting them would delete
every resource). Remove them explicitly if you want a full teardown:

```bash
kubectl delete crd -l app.kubernetes.io/name=keycloak-operator
```

Managed Keycloak server-side resources follow the [deletion
policy](../README.md#deletion-policy-specdeletionpolicy) of each resource:
realms are orphaned unless `deletionPolicy: Delete` was set.
