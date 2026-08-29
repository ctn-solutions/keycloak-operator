# Troubleshooting

Every managed resource reports its state through the `Ready` and `Synced`
conditions. Start here:

```bash
kubectl describe <resource> -n <namespace>     # conditions + events
kubectl get <resource> -n <namespace> -o yaml  # full status
```

## Ready=False reasons

### `ConnectionUnavailable`

The operator cannot reach the Keycloak server or authenticate against it.

| Cause | Fix |
|---|---|
| `KeycloakConnection` missing or misnamed | Check `spec.keycloakRef.name` and that the connection lives in the **same namespace** as the resource |
| Credentials Secret missing, or missing `username`/`password` (or `clientId`/`clientSecret`) keys | Create the Secret with the expected keys |
| Wrong credentials | Fix the Secret; the operator revalidates every 5 minutes and on the next reconciliation |
| Server unreachable (DNS, network policy, TLS) | Check `spec.url` reachability **from the operator pod**; verify `spec.tls.insecureSkipVerify` for self-signed certificates |
| Service account lacks roles | See the authentication model in [metrics.md](metrics.md) — a realm-scoped service account needs `realm-management` client roles of its realm |

The resource reconciles automatically once the connection works — no restart
needed.

### `AlreadyExists`

A resource with the same natural key already exists on the Keycloak server
and the adoption policy refuses to touch it.

- `CreateOnly` (default): a **foreign** resource exists. Either rename yours,
  delete the server-side resource, or set `adoptionPolicy: Adopt` to take it
  over.
- `FailIfExists`: fails even for resources the operator manages itself. This
  is the strictest policy — switch to `CreateOnly` or `Adopt`.

### `ProtectedRealm`

The resource targets a protected realm (`master`). Managing it deliberately
requires:

```yaml
metadata:
  annotations:
    keycloak.ctn-solutions.io/allow-protected: "true"
spec:
  adoptionPolicy: Adopt   # master always exists
```

### `SecretMissing`

A Secret referenced by `secretRef`, `configSecretRef` or
`smtpServerSecretRef` does not exist, or lacks the expected key. The
reconciliation retries every 30 seconds — create the Secret (or the missing
key) and it converges without intervention.

If `secretOutput` names a Secret that already exists **and is not owned by
the resource**, the operator refuses to write it. Choose another name or
delete the foreign Secret.

### `Failed`

Anything else: invalid payloads, roles missing on the administration account
(403 from the Admin API), or kind-specific enforcement errors (a referenced
role or client scope does not exist, for example). The message in the
condition and the warning event describe the exact problem.

## Common situations

### The operator keeps reverting my console changes

That is drift correction working as designed. The Keycloak server is
corrected on every reconciliation (and at least once per resync period).
Change the desired state in the resource, not the console.

### I removed a field from the spec but the server kept the old value

Field removal resets strings, booleans, lists and maps to their zero value.
**Numeric fields are left unchanged** because the Admin API cannot
distinguish an unset number from zero — set the field explicitly instead.

### Deleting a Realm resource did not delete the realm

Realms default to `deletionPolicy: Orphan`. Set `deletionPolicy: Delete` on
the resource *before* deleting it if you want the realm removed from the
server.

### The client secret changed in Keycloak but my application still has the old one

With `secretRef` configured, the operator enforces the Secret's value on the
server. If you changed the secret in Keycloak directly, the operator reverts
it. Rotate by changing the Kubernetes Secret.

### Metrics endpoint returns 401/403

The metrics endpoint requires authentication by default. Configure the
scraper's token as described in [metrics.md](metrics.md), or set
`metrics.secure=false` in the chart for trusted network layers.

## Diagnostics checklist

```bash
# Operator health and version
kubectl -n keycloak-operator-system get pods
kubectl -n keycloak-operator-system logs deployment/keycloak-operator --tail=100

# Connection state
kubectl get keycloakconnections -A
kubectl -n <ns> describe keycloakconnection <name>

# Everything the operator manages and their sync state
kubectl get realms,clients,clientscopes,realmroles,identityproviders,groups -A -o wide

# Recent events across the namespace
kubectl get events -n <namespace> --sort-by=.lastTimestamp | tail -20
```
