---
name: Bug report
about: Something does not work as documented
labels: bug
---

**What happened?**

**What did you expect to happen?**

**Environment**
- Operator version:
- Keycloak server version:
- Kubernetes version:
- Installation method (Helm / install.yaml):

**Minimal reproduction**

Resource manifests (redact secrets) and the steps to reproduce.

**Logs and status**

```text
kubectl -n <namespace> describe <resource>
kubectl -n keycloak-operator-system logs deployment/keycloak-operator
```
