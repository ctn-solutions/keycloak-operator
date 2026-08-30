# keycloak-operator Helm chart

Kubernetes operator managing Keycloak realms, clients, client scopes, realm
roles, identity providers and groups declaratively through GitOps.

## Installing

```bash
helm install keycloak-operator oci://ghcr.io/ctn-solutions/charts/keycloak-operator \
  --namespace keycloak-operator-system --create-namespace
```

Or from a checkout of this repository:

```bash
helm install keycloak-operator charts/keycloak-operator \
  --namespace keycloak-operator-system --create-namespace
```

Ready-made configurations live in
[`examples/`](./examples/): namespace-scoped, metrics with a
ServiceMonitor, and a production baseline.

## CRDs: installed, never upgraded

The chart ships the CRDs in its `crds/` directory. Helm installs them on
`helm install` but **never upgrades them** — that is Helm's documented
behaviour for the `crds/` directory, and it is deliberate: CRD changes
affect every resource in the cluster and must not happen implicitly.

When upgrading the operator, apply the CRDs manually first:

```bash
kubectl apply -f charts/keycloak-operator/crds/
helm upgrade keycloak-operator oci://ghcr.io/ctn-solutions/charts/keycloak-operator \
  --version <new-version> -n keycloak-operator-system
```

See the [upgrading guide](../../docs/upgrading.md) for the full procedure.

## Values

### Image and deployment

| Value | Default | Description |
|---|---|---|
| `image.repository` | `ghcr.io/ctn-solutions/keycloak-operator` | Image repository |
| `image.tag` | `""` (chart `appVersion`) | Image tag; pin explicitly in production |
| `image.pullPolicy` | `IfNotPresent` | Pull policy |
| `image.pullSecrets` | `[]` | Image pull secrets for private registries |
| `replicaCount` | `1` | Replicas; safe above 1 with leader election |
| `nameOverride` / `fullnameOverride` | `""` | Resource name overrides |

### RBAC and scoping

| Value | Default | Description |
|---|---|---|
| `serviceAccount.create` | `true` | Create a dedicated ServiceAccount |
| `serviceAccount.name` | `""` | Override the ServiceAccount name |
| `serviceAccount.annotations` | `{}` | E.g. workload identity annotations |
| `rbac.create` | `true` | Install Role/ClusterRole and bindings |
| `rbac.clusterScoped` | `true` | `false` installs Role/RoleBinding in the release namespace and restricts the operator's cache to it (`--watch-namespace`) |

### Behaviour

| Value | Default | Description |
|---|---|---|
| `leaderElection.enabled` | `true` | Elect a single active operator |
| `resyncPeriod` | `1h` | Drift-correction interval (Go duration) |

### Metrics

| Value | Default | Description |
|---|---|---|
| `metrics.secure` | `true` | HTTPS with authn/authz on the metrics endpoint; `false` serves plain HTTP |
| `metrics.service.enabled` | `false` | Create the metrics Service |
| `metrics.service.port` | `8443` | Service port |
| `metrics.service.type` | `ClusterIP` | Service type |
| `metrics.serviceMonitor.enabled` | `false` | Create a ServiceMonitor (requires `metrics.service.enabled` and the Prometheus Operator CRDs) |
| `metrics.serviceMonitor.interval` | `30s` | Scrape interval |
| `metrics.serviceMonitor.scheme` | `https` | Scrape scheme |
| `metrics.serviceMonitor.tlsConfig` | `{}` | TLS config for scraping the secure endpoint |
| `metrics.serviceMonitor.bearerTokenSecret` | `{}` | Token secret for scraping the secure endpoint |
| `metrics.serviceMonitor.additionalLabels` | `{}` | Extra ServiceMonitor labels (e.g. your Prometheus release selector) |

The custom metrics are documented in
[docs/metrics.md](../../docs/metrics.md).

### Workload

| Value | Default | Description |
|---|---|---|
| `resources` | `50m/64Mi` requests, `128Mi` limit | Container resources |
| `podSecurityContext` | `runAsNonRoot`, RuntimeDefault seccomp | Pod security context |
| `containerSecurityContext` | non-root, read-only rootfs, all capabilities dropped | Container security context |
| `nodeSelector` / `tolerations` / `affinity` | `{}` | Scheduling |
| `priorityClassName` | `""` | Priority class |

## Uninstalling

```bash
helm uninstall keycloak-operator -n keycloak-operator-system
```

CRDs are not removed (by design). See the
[installation guide](../../docs/installation.md#uninstalling).
