# Keycloak Operator

A Kubernetes operator that manages [Keycloak](https://www.keycloak.org/) configuration
declaratively. Realms, clients, client scopes, roles, identity providers and groups live in
your Git repository; the operator keeps a Keycloak server in sync with them.

It maps the [Keycloak Admin API](https://www.keycloak.org/docs-api/latest/rest-api/) 1:1:
every field in a custom resource spec mirrors the corresponding field of the Keycloak
representation it manages, so what you write is exactly what the server stores.

```yaml
apiVersion: keycloak.ctn-solutions.io/v1alpha1
kind: Realm
metadata:
  name: acme
spec:
  keycloakRef:
    name: production
  realm: acme
  displayName: ACME Platform
  registrationAllowed: false
  accessTokenLifespan: 900
```

## Features

- **1:1 API mapping** — spec fields match the Keycloak representations field-for-field.
  Scalar fields are pointers, so *unset*, *explicitly false/zero* and *default* are always
  distinguishable.
- **Declarative field removal** — the operator records what it last applied; removing a
  field from a spec resets it on the server.
- **Drift correction** — changes made out-of-band in the Keycloak console are reverted on
  the next reconciliation (periodic resync, default 1h).
- **Adoption policies** — per-resource control over pre-existing server state:
  `CreateOnly` (default, never touches foreign resources), `Adopt` (take over),
  `FailIfExists`.
- **Protected realms** — `master` is refused unless a resource explicitly carries the
  `keycloak.ctn-solutions.io/allow-protected: "true"` annotation.
- **Safe deletion** — every resource carries a finalizer; realms are orphaned by default
  and only deleted with an explicit `deletionPolicy: Delete`.
- **Secrets in both directions** — client secrets, identity-provider credentials and SMTP
  passwords are read from Kubernetes Secrets (never inline in CRDs), and Keycloak-generated
  client secrets are exported to Secrets for applications to mount.
- **Multi-server** — one operator deployment manages any number of Keycloak servers through
  `KeycloakConnection` resources.

## Resource types

| CRD | Manages | Natural key |
|---|---|---|
| `KeycloakConnection` | A Keycloak server and its admin credentials | — |
| `Realm` | A realm (`RealmRepresentation`) | `spec.realm` |
| `Client` | A client (`ClientRepresentation`) | `spec.clientId` |
| `ClientScope` | A client scope (`ClientScopeRepresentation`) | `spec.name` |
| `RealmRole` | A realm role incl. composites (`RoleRepresentation`) | `spec.name` |
| `IdentityProvider` | An identity broker (`IdentityProviderRepresentation`) | `spec.alias` |
| `Group` | A group incl. role mappings (`GroupRepresentation`) | `spec.name` |

All resources live in the API group `keycloak.ctn-solutions.io/v1alpha1` and reference
their server through `spec.keycloakRef.name`, which must point to a `KeycloakConnection`
in the same namespace.

## Quickstart

Install the operator with Helm:

```bash
helm install keycloak-operator oci://ghcr.io/ctn-solutions/charts/keycloak-operator \
  --namespace keycloak-operator-system --create-namespace
```

Or from a checkout of this repository:

```bash
helm install keycloak-operator charts/keycloak-operator \
  --namespace keycloak-operator-system --create-namespace
```

> Helm installs the CRDs from the chart's `crds/` directory on install but never upgrades
> them. When upgrading the operator, apply the CRDs manually first:
> `kubectl apply -f charts/keycloak-operator/crds/`.

Point the operator at your Keycloak server and declare a realm:

```bash
kubectl create namespace keycloak-system
kubectl apply -f examples/
```

Watch it converge:

```bash
kubectl get realms,clients,groups -n keycloak-system -o wide
kubectl describe realm acme -n keycloak-system   # conditions and events
```

## Semantics

### Adoption policy (`spec.adoptionPolicy`)

| Policy | Resource exists, unmanaged | Resource exists, managed by us | Resource absent |
|---|---|---|---|
| `CreateOnly` *(default)* | Fail (`AlreadyExists`) | Resume managing | Create |
| `Adopt` | Take over and enforce | Resume managing | Create |
| `FailIfExists` | Fail | Fail | Create |

The operator stamps a managed marker on resources it creates or adopts (in the
representation's `attributes` where the API allows it). Identity providers have no
attributes block; there, management is tracked through the resource's last-applied
annotation.

### Deletion policy (`spec.deletionPolicy`)

| Kind | Default | On resource deletion |
|---|---|---|
| `Realm` | `Orphan` | Realm stays on the server |
| All others | `Delete` | Server resource is deleted |

Set `deletionPolicy: Delete` on a Realm to opt in to realm deletion.

### Status

Resources report standard conditions:

- `Ready` — the reconciliation outcome (`Succeeded`, `Retrying`, `Failed`, or reasons such
  as `ConnectionUnavailable`, `AlreadyExists`, `SecretMissing`, `ProtectedRealm`)
- `Synced` — the server state matches the spec

Transient errors (network, 5xx) retry with backoff. Terminal errors (conflicts, missing
secrets, protected realms) surface in the conditions and wait for a spec change or the
periodic resync.

### Secrets

Sensitive values never live in custom resources:

- `Client.spec.secretRef` — injects the client secret from a Secret you own.
- `Client.spec.secretOutput` — exports the effective client secret to a Secret owned by
  the `Client` resource, for applications to mount.
- `IdentityProvider.spec.configSecretRef` — injects broker config values (e.g.
  `clientSecret`) from a Secret.
- `Realm.spec.smtpServerSecretRef` — injects SMTP credentials.

## Compatibility

- Keycloak **26.x** (the Admin API paths and semantics are verified against 26.3).
- Kubernetes 1.25+.

## Development

```bash
make test          # unit + controller tests (envtest, no cluster needed)
make lint          # golangci-lint
make run           # run the operator against the current kubeconfig context
```

End-to-end tests against a real Keycloak 26 server on a local cluster:

```bash
bash test/e2e/e2e.sh
```

See [docs/design.md](docs/design.md) for the architecture and design decisions, and
[docs/crd-reference.md](docs/crd-reference.md) for the full CRD reference.

## Roadmap

- User management (with write-once, SecretRef-only credentials)
- Client authorization services
- Realm-level default groups and client policies
- CI pipelines and released images

## License

[Apache-2.0](LICENSE)
