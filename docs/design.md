# Design: Keycloak Operator

**Status:** Approved · **Target:** Keycloak 26.x · **API group:** `keycloak.ctn-solutions.io/v1alpha1`

## Goal

Manage Keycloak configuration declaratively through Kubernetes custom resources, so a Git
repository is the single source of truth for realms, clients, client scopes, roles,
identity providers and groups. The operator maps the Keycloak Admin API 1:1: every field in
a CRD spec mirrors the corresponding field of the Keycloak representation it manages.

Non-goals for v1: managing the Keycloak *deployment* itself (the official Keycloak operator
does that), user management, client authorization services, client policies.

## Architecture

Go operator built with Kubebuilder v4 and controller-runtime. One operator deployment can
serve any number of Keycloak servers.

```
GitOps repo ──apply──▶ Kubernetes API
                          │
                          ▼
                 ┌──────────────────┐        ┌─────────────────────┐
                 │ keycloak-operator│──HTTPS──▶ Keycloak Admin API │
                 │  (this project)  │        │  (any version 26.x) │
                 └──────────────────┘        └─────────────────────┘
```

### Components

| Component | Responsibility |
|---|---|
| `KeycloakConnection` controller | Validates credentials against the server, reports server version, keeps `Ready` fresh |
| Connection provider | Caches one authenticated Admin API client per connection, handles token refresh |
| Generic reconcile engine | Shared create/adopt/update/delete/drift machinery driven by per-kind drivers |
| Per-kind drivers | Natural key lookup, endpoint mapping, kind-specific extras (client secrets, role mappings, default scopes) |
| Admin API client | Thin typed HTTP client over the endpoints the operator needs, with token lifecycle |

### Resource model

Seven CRDs. Every managed resource carries `spec.keycloakRef.name` pointing to a
`KeycloakConnection` in the same namespace (cross-namespace secret reads are not allowed).

| CRD | Keycloak representation | Natural key |
|---|---|---|
| `KeycloakConnection` | — (server + credentials) | — |
| `Realm` | `RealmRepresentation` | `spec.realm` |
| `Client` | `ClientRepresentation` | `spec.clientId` |
| `ClientScope` | `ClientScopeRepresentation` | `spec.name` |
| `RealmRole` | `RoleRepresentation` (realm-level) | `spec.name` |
| `IdentityProvider` | `IdentityProviderRepresentation` | `spec.alias` |
| `Group` | `GroupRepresentation` + role mappings | `spec.name` |

**1:1 mapping.** Spec field names and JSON tags match the Keycloak representation
field-for-field. Conversion between spec and API payload is a JSON round-trip; upgrading
Keycloak support means diffing its representation against our spec. Fields the spec does not
model are never touched on the server.

**Tri-state fields.** Scalar spec fields are pointers so that *unset*, *explicitly false/zero*
and *default* are distinguishable. This is required for correct drift correction: the operator
must be able to set `enabled: false` and to reset a field that was removed from the spec.

**Removal semantics.** The operator stores a `last-applied` annotation on each CR (JSON of the
fields it last applied). A field present in `last-applied` but absent from the current spec is
reset (removed from the server-side payload, so Keycloak falls back to its default). This gives
kubectl-style declarative removal without clobbering fields the spec never managed.

## Semantics

### Adoption policy (`spec.adoptionPolicy`, default `CreateOnly`)

| Policy | Resource exists, unmanaged | Resource exists, already managed by us | Resource does not exist |
|---|---|---|---|
| `CreateOnly` | Fail (`AlreadyExists`) — never touches foreign resources | Resume managing it | Create |
| `Adopt` | Take over: stamp managed annotation, then enforce spec | Resume managing it | Create |
| `FailIfExists` | Fail (`AlreadyExists`) | Fail (`AlreadyExists`) | Create |

The operator stamps `keycloak.ctn-solutions.io/managed: "true"` on resources it creates or
adopts (in realm/client/role/group attributes where the representation allows it).

**Protected realms.** Realms named `master` are refused unless the CR carries the annotation
`keycloak.ctn-solutions.io/allow-protected: "true"`. Safe default, explicit escape hatch.

### Deletion policy (`spec.deletionPolicy`)

| Kind | Default | Behaviour on CR deletion |
|---|---|---|
| `Realm` | `Orphan` | Realm left on the server; finalizer removed |
| All others | `Delete` | Server resource deleted, then finalizer removed |

`deletionPolicy: Delete` on a Realm opts in to realm deletion. Every CR carries a finalizer
until the server-side outcome is settled, so deletion never races.

### Drift correction

Every reconcile compares the effective desired state against the server and issues an update
when they differ — including changes made out-of-band in the Keycloak console. A periodic
resync (default 1h, configurable) catches drift between reconciles.

### Secrets

Sensitive values never live in CRDs.

- **Inbound** — `Client.spec.secretRef`, `IdentityProvider.spec.configSecretRef` (maps config
  keys to secret keys), `Realm.spec.smtpServerSecretRef`: the operator reads the referenced
  Secret and injects values into the server-side payload. Missing secret/key → `Failed`
  condition, no retry storm.
- **Outbound** — `Client.spec.secretOutput`: the operator fetches the client secret from
  Keycloak and writes it into a Kubernetes Secret (owner-referenced to the Client CR) for
  applications to mount. Written only when the value differs. `secretRef` and `secretOutput`
  may coexist: the ref is the source of truth, the output mirrors it.

### Status

Standard `meta/v1` conditions:

- `Ready` — reconcile outcome (`Succeeded`, `Retrying`, `Failed`, reasons such as
  `ConnectionUnavailable`, `AlreadyExists`, `SecretMissing`, `ProtectedRealm`)
- `Synced` — server state matches the spec
- `observedGeneration` on both; `Client.status.secretName` for the outbound secret

Events are emitted for create/update/adopt/delete/fail. Transient errors (network, 5xx, 429)
retry with exponential backoff; terminal errors set `Failed` and wait for spec change or the
periodic resync.

## Testing

- **Unit + controller tests:** envtest with an in-repo fake Keycloak Admin API server
  (`httptest`, in-memory state). Covers the full lifecycle: create, adopt, conflict, drift
  revert, secret flows, deletion policies, protected realms.
- **E2E:** real Keycloak 26 on a local cluster (docker-desktop), operator deployed via the
  Helm chart, example manifests applied, server state asserted through the Admin API.
- `make lint` (golangci-lint), `make generate manifests` must be clean.

## Packaging

- Helm chart at `charts/keycloak-operator`: CRDs under `crds/` (Helm installs them but never
  upgrades — upgrades are manual by design, documented), RBAC, deployment, values for image,
  resync period, leader election.
- Examples under `examples/` covering a complete realm stack.
- No CI pipelines in v1.
