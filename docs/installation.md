# Installation

Four ways to install the operator. All of them install the CRDs first —
Helm from the chart's `crds/` directory, the manifest bundle inline.

## Prerequisites

- Kubernetes **1.25+**
- Helm **3.8+** (for the OCI chart), or `kubectl` alone, or Homebrew
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

## Option 2 — The installer script (no Helm)

The installer script downloads a release bundle, verifies its sha256
checksum and applies it with `kubectl` alone:

```bash
curl -fsSL https://raw.githubusercontent.com/ctn-solutions/keycloak-operator/main/install.sh | bash
```

The piped-to-bash form is convenient but blind; to review before running:

```bash
curl -fsSLO https://github.com/ctn-solutions/keycloak-operator/releases/latest/download/install.yaml
# inspect, then:
kubectl apply -f install.yaml
```

The script also handles upgrades, uninstalls and version pinning — see
[Installing with the script](#option-4-the-installer-script) below and run
`install.sh --help` for the full reference.

## Option 3 — Helm from a checkout

```bash
git clone https://github.com/ctn-solutions/keycloak-operator.git
cd keycloak-operator
helm install keycloak-operator charts/keycloak-operator \
  --namespace keycloak-operator-system --create-namespace
```

## Option 4 — Plain manifest (no Helm)

```bash
kubectl apply -f https://github.com/ctn-solutions/keycloak-operator/releases/latest/download/install.yaml
```

The bundle contains the CRDs, the `keycloak-operator-system` namespace, RBAC
and the deployment, pinned to the released image. Every release also ships a
version-pinned bundle (`install-vX.Y.Z.yaml`) and a `sha256sums.txt`
manifest on the [release page](https://github.com/ctn-solutions/keycloak-operator/releases).

## The installer script

`install.sh` wraps the plain-manifest method with version resolution,
checksum verification and safe uninstall. It refuses to touch Helm-managed
installs — pick one method and stay with it.

```bash
# Install the latest release (or re-run to converge)
curl -fsSL https://raw.githubusercontent.com/ctn-solutions/keycloak-operator/main/install.sh -o install.sh
bash install.sh install

# Pin a version
bash install.sh install --version v0.2.0

# Upgrade to the latest release
bash install.sh upgrade

# Uninstall (CRDs and managed resources are kept)
bash install.sh uninstall

# Full teardown including CRDs — deletes every managed resource
bash install.sh uninstall --purge-crds --yes

# What is installed, and what is available
bash install.sh version
```

The script honors `GITHUB_TOKEN`/`GH_TOKEN` (higher API rate limits, and
access when the repository is private) and `KUBECTL` (alternate kubectl
binary). It never evaluates downloaded content: bundles are written to a
temporary directory, checksum-verified when the release publishes
`sha256sums.txt`, and handed to `kubectl` as files.

## Homebrew (macOS and Linux)

The [Homebrew tap](https://github.com/ctn-solutions/homebrew-tap) installs
the installer script as a CLI:

```bash
brew install ctn-solutions/tap/keycloak-operator
keycloak-operator install          # install the latest release
keycloak-operator upgrade          # move to a newer release
brew upgrade keycloak-operator     # update the CLI itself
```

The CLI is a convenience wrapper around the same install bundle — it is
not the operator binary (the operator runs in your cluster as a container
image).

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

With the installer script (keeps CRDs and managed resources by default):

```bash
bash install.sh uninstall              # or: keycloak-operator uninstall
bash install.sh uninstall --purge-crds # full teardown, deletes managed resources
```

With Helm:

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
