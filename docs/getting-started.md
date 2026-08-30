# Getting started

From zero to a fully managed realm in about ten minutes. Everything happens
inside Kubernetes — no local Keycloak needed.

## 0. Install the operator

Follow the [installation guide](installation.md). Then verify:

```bash
kubectl -n keycloak-operator-system get pods
# NAME                                READY   STATUS
# keycloak-operator-<id>              1/1     Running
```

## 1. Deploy a Keycloak server (skip if you have one)

For trying things out, deploy a development Keycloak into the cluster:

```bash
kubectl create namespace keycloak
kubectl -n keycloak create deployment keycloak \
  --image=quay.io/keycloak/keycloak:26.3 \
  -- start-dev --hostname-strict=false
kubectl -n keycloak set env deployment/keycloak \
  KC_BOOTSTRAP_ADMIN_USERNAME=admin \
  KC_BOOTSTRAP_ADMIN_PASSWORD=admin
kubectl -n keycloak expose deployment keycloak --port 8080
kubectl -n keycloak rollout status deployment/keycloak
```

> This is a development server (dev mode, ephemeral database). For
> production, deploy Keycloak with a database and TLS — the operator does
> not manage the Keycloak server itself, only its configuration.

## 2. Connect the operator to the server

Create a Secret with the administration credentials and a
`KeycloakConnection` pointing at the server:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: keycloak-admin-credentials
  namespace: keycloak-system
stringData:
  username: admin
  password: admin
---
apiVersion: keycloak.ctn-solutions.io/v1alpha1
kind: KeycloakConnection
metadata:
  name: dev
  namespace: keycloak-system
spec:
  url: http://keycloak.keycloak.svc.cluster.local:8080
  credentialsSecretRef: keycloak-admin-credentials
```

```bash
kubectl create namespace keycloak-system
kubectl apply -f connection.yaml
```

Within seconds the operator validates the credentials and reports the
server version:

```bash
kubectl -n keycloak-system get keycloakconnection dev
# NAME   URL                                        VERSION   READY
# dev    http://keycloak.keycloak.svc...            26.3.5    True
```

If `READY` stays `False`, the condition message says exactly why — see the
[troubleshooting guide](troubleshooting.md).

## 3. Declare your first realm

```yaml
apiVersion: keycloak.ctn-solutions.io/v1alpha1
kind: Realm
metadata:
  name: getting-started
  namespace: keycloak-system
spec:
  keycloakRef:
    name: dev
  realm: getting-started
  displayName: Getting Started
  enabled: true
  registrationAllowed: false
  accessTokenLifespan: 900
```

```bash
kubectl apply -f realm.yaml
```

The operator creates the realm on the server. Watch it converge:

```bash
kubectl -n keycloak-system get realm getting-started -w
# Synced flips to True when the server state matches the spec
```

Prove it exists on the server — get a token and query the Admin API:

```bash
TOKEN=$(curl -s -X POST http://keycloak.keycloak.svc.cluster.local:8080/realms/master/protocol/openid-connect/token \
  -d "grant_type=password&client_id=admin-cli&username=admin&password=admin" \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['access_token'])")
curl -s -H "Authorization: Bearer $TOKEN" \
  http://keycloak.keycloak.svc.cluster.local:8080/admin/realms/getting-started \
  | python3 -m json.tool | grep displayName
# "displayName": "Getting Started"
```

## 4. See the GitOps loop work

**Change the spec** — set `displayName: Renamed` and re-apply. Within
seconds the server reflects it.

**Change the server out-of-band** — edit the realm in the Keycloak admin
console (port-forward first: `kubectl -n keycloak port-forward svc/keycloak
8080`). The operator reverts your change on the next reconciliation
(default resync: 1h; set `--set resyncPeriod=30s` while learning).

**Remove a field from the spec** — the server value resets. Strings,
booleans, lists and maps reset; numeric fields are left unchanged
([why](design.md#removal-semantics)).

## 5. Add a client with a secret

```yaml
apiVersion: keycloak.ctn-solutions.io/v1alpha1
kind: Client
metadata:
  name: getting-started-app
  namespace: keycloak-system
spec:
  keycloakRef:
    name: dev
  realm: getting-started
  clientId: demo-app
  enabled: true
  protocol: openid-connect
  publicClient: false
  standardFlowEnabled: true
  redirectUris: ["https://demo.example.com/*"]
  secretOutput:
    name: demo-app-credentials
```

The operator creates the client and **exports its secret** to a Kubernetes
Secret your application can mount:

```bash
kubectl -n keycloak-system get secret demo-app-credentials \
  -o jsonpath='{.data.clientSecret}' | base64 -d
```

## 6. Clean up

```bash
kubectl delete client getting-started-app -n keycloak-system  # client deleted from the server
kubectl delete realm getting-started -n keycloak-system       # realm ORPHANED by default
```

The realm stays on the server — that is the safe default. Set
`deletionPolicy: Delete` on the Realm resource before deleting it if you
want the realm removed too.

## Next steps

- The [example stack](https://github.com/ctn-solutions/keycloak-operator/tree/main/examples):
  client scopes, composite roles, groups with role mappings, an OIDC
  identity provider
- The [CRD reference](crd-reference.md) for every supported field
- [Adoption and deletion policies](../README.md#semantics) — how the
  operator treats pre-existing server state
- [Metrics](metrics.md) — reconciliation outcomes, drift corrections and
  connection health for your dashboards
