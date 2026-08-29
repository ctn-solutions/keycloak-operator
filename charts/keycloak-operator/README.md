# keycloak-operator Helm chart

Kubernetes operator managing Keycloak realms, clients, roles, identity providers and groups declaratively.

## Installing

```bash
helm install keycloak-operator charts/keycloak-operator \
  --namespace keycloak-operator-system \
  --create-namespace
```

## Upgrading

```bash
helm upgrade keycloak-operator charts/keycloak-operator \
  --namespace keycloak-operator-system
```

### CRD upgrades

The CRDs in `crds/` are **installed by Helm on `helm install`, but they are
never upgraded or removed by Helm** — this is Helm's documented behaviour for
the `crds/` directory. When a chart upgrade ships CRD changes, apply them
manually before upgrading the release:

```bash
kubectl apply -f charts/keycloak-operator/crds/
```

### Pinning the image tag

By default the chart uses the `appVersion` tag (`0.1.0`). Pin a specific image
tag with:

```bash
helm install keycloak-operator charts/keycloak-operator \
  --namespace keycloak-operator-system --create-namespace \
  --set image.tag=0.2.0
```

## Uninstalling

```bash
helm uninstall keycloak-operator --namespace keycloak-operator-system
```

Note that the CRDs and the Keycloak custom resources they hold are not removed
by `helm uninstall`; remove them manually if desired.
