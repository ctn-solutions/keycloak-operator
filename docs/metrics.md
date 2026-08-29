# Metrics reference

The operator exposes Prometheus metrics on `:8443` (HTTPS with
authentication and authorization by default, or plain HTTP with
`metrics.secure=false`). Custom metrics are served alongside the standard
controller-runtime metrics (Go runtime, workqueue, REST client latency).

## Custom metrics

### `keycloak_operator_reconciliations_total`

Counter — reconciliations by resource kind and outcome.

| Label | Values |
|---|---|
| `kind` | `Realm`, `Client`, `ClientScope`, `RealmRole`, `IdentityProvider`, `Group` |
| `outcome` | `success` (server state matches), `error` (transient failure, retried with backoff), `terminal` (condition `Ready=False`, needs attention) |

Useful queries:

```promql
# Terminal failure rate per kind — alert on this
sum(rate(keycloak_operator_reconciliations_total{outcome="terminal"}[10m])) by (kind)

# Reconciliation error ratio
sum(rate(keycloak_operator_reconciliations_total{outcome="error"}[5m]))
  / sum(rate(keycloak_operator_reconciliations_total[5m]))
```

### `keycloak_operator_reconcile_duration_seconds`

Histogram — reconciliation duration by resource kind. Useful for spotting
slow Keycloak servers (reconciliation includes Admin API round-trips).

### `keycloak_operator_drift_corrections_total`

Counter — server-side updates issued because the Keycloak server diverged
from the declared state, by resource kind. A sustained rate means something
is fighting the operator (another automation, manual console changes, or a
non-convergent field). A healthy GitOps setup shows near-zero drift outside
deployments.

### `keycloak_operator_connection_up`

Gauge — `1` when the operator can obtain an administration token for the
connection, `0` when it cannot.

| Label | Example |
|---|---|
| `namespace` | `keycloak-system` |
| `connection` | `production` |

Alert when this is `0` for more than a couple of minutes:

```promql
keycloak_operator_connection_up == 0
```

### `keycloak_operator_server_info`

Info-style gauge, always `1`, labelled with the server version:

```
keycloak_operator_server_info{namespace="keycloak-system",connection="production",version="26.3.5"} 1
```

Only populated when the connection's account may read `/admin/serverinfo`
(master-level accounts can; realm-scoped service accounts cannot — the
connection still works, see the authentication model below).

### `keycloak_operator_admin_requests_total`

Counter — Admin API requests by connection, HTTP method and status class
(`2xx`, `4xx`, `5xx`, `error`). A rising `4xx` rate with `connection_up = 1`
usually means an account whose roles are insufficient for the resources
being managed.

### `keycloak_operator_admin_request_duration_seconds`

Histogram — Admin API request latency by connection and method. Use it to
size the resync period: the periodic drift-correction pass issues one read
(and at most one write) per managed resource.

## Authentication model (relevant to metrics)

A connection is **ready** when its credentials can obtain an administration
token. Two enterprise patterns are supported:

1. **Master admin account** (`auth: password` against `master`) — full
   access, `server_info` populated.
2. **Realm-scoped service account** (`auth: client` with `adminRealm` set to
   the target realm) — least privilege. Create a confidential,
   service-account-enabled client inside the target realm and grant it the
   `realm-management` client roles it needs (`view-clients`,
   `manage-clients`, `view-identity-providers`,
   `manage-identity-providers`, …). `server_info` stays empty for these
   accounts; everything else works.

## Scraping

The metrics port is `8443`. With `metrics.secure=true` (default) the
endpoint requires a token whose service account may access the `/metrics`
non-resource URL:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus-metrics-reader
rules:
  - nonResourceURLs: ["/metrics"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus-metrics-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus-metrics-reader
subjects:
  - kind: ServiceAccount
    name: prometheus-k8s
    namespace: monitoring
```

Enable the chart's Service and ServiceMonitor:

```bash
helm upgrade keycloak-operator charts/keycloak-operator \
  --set metrics.service.enabled=true \
  --set metrics.serviceMonitor.enabled=true \
  --set metrics.serviceMonitor.bearerTokenSecret.name=prometheus-k8s-token \
  ...
```

For environments where the Prometheus scraper cannot authenticate (for
example when the network layer already enforces access control), set
`metrics.secure=false` to serve plain HTTP on the same port.
