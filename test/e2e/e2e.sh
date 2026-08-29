#!/usr/bin/env bash
# End-to-end test: deploys Keycloak 26 and the operator on the current
# Kubernetes context (docker-desktop by default), applies the example
# resources and verifies the server state through the Admin API.
#
# Usage:
#   bash test/e2e/e2e.sh              # run the full suite, leave the stack running
#   bash test/e2e/e2e.sh --cleanup    # tear the stack down afterwards
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NAMESPACE="keycloak-e2e"
IMG="keycloak-operator:e2e-$(date +%s)"
KC_URL_INCLUSTER="http://keycloak.${NAMESPACE}.svc.cluster.local:8080"
KC_URL="http://localhost:18080"
KC_ADMIN="admin"
KC_PASS="admin"
CLEANUP=false
[[ "${1:-}" == "--cleanup" ]] && CLEANUP=true

log() { printf '\n\033[1;34m==> %s\033[0m\n' "$*"; }
fail() { printf '\n\033[1;31mFAIL: %s\033[0m\n' "$*" >&2; exit 1; }

kubectl_context() {
  local current
  current="$(kubectl config current-context)"
  if [[ "$current" != "docker-desktop" ]]; then
    echo "Current context is '$current'. This suite targets docker-desktop."
    read -r -p "Continue anyway? [y/N] " answer
    [[ "$answer" == "y" ]] || exit 1
  fi
}

wait_for() { # wait_for <description> <timeout-seconds> <command...>
  local description="$1" timeout="$2"; shift 2
  local deadline=$((SECONDS + timeout))
  while (( SECONDS < deadline )); do
    if "$@" >/dev/null 2>&1; then
      echo "ok: $description"
      return 0
    fi
    sleep 3
  done
  fail "timed out waiting for $description"
}

token() {
  curl -sf -X POST "${KC_URL}/realms/master/protocol/openid-connect/token" \
    -d "grant_type=password&client_id=admin-cli&username=${KC_ADMIN}&password=${KC_PASS}" \
    | python3 -c "import sys,json;print(json.load(sys.stdin)['access_token'])"
}

api() { # api <method> <path> [body]
  local method="$1" path="$2" body="${3:-}"
  local t
  t="$(token)"
  if [[ -n "$body" ]]; then
    curl -sf -X "$method" -H "Authorization: Bearer $t" -H "Content-Type: application/json" \
      -d "$body" "${KC_URL}${path}"
  else
    curl -sf -X "$method" -H "Authorization: Bearer $t" "${KC_URL}${path}"
  fi
}

client_field() { # client_field <clientId> <json-key>
  api GET "/admin/realms/acme/clients?clientId=$1" | python3 -c "import sys,json;print(json.load(sys.stdin)[0]['$2'])"
}

group_field() { # group_field <groupName> <json-key>
  api GET "/admin/realms/acme/groups?search=$1&exact=true" | python3 -c "import sys,json;print(json.load(sys.stdin)[0]['$2'])"
}

# Assertions run in `bash -c` subshells, which need the helpers and the
# server coordinates exported.
export KC_URL KC_ADMIN KC_PASS
export -f token api client_field group_field

assert() { # assert <description> <command...>
  local description="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "PASS: $description"
  else
    fail "$description"
  fi
}

log "Checking cluster context"
kubectl_context

log "Building the operator image"
docker build -q -t "$IMG" "$REPO_ROOT" >/dev/null || fail "docker build"

log "Deploying Keycloak 26"
kubectl apply -f "$REPO_ROOT/test/e2e/manifests/keycloak.yaml"
wait_for "Keycloak ready" 300 kubectl rollout status deployment/keycloak -n "$NAMESPACE"

# Host-side access to the in-cluster server for assertions.
pkill -f "port-forward svc/keycloak" 2>/dev/null || true
sleep 1
kubectl -n "$NAMESPACE" port-forward svc/keycloak 18080:8080 >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
trap 'kill $PORT_FORWARD_PID 2>/dev/null || true' EXIT
wait_for "port-forward up" 60 bash -c "curl -sf ${KC_URL}/realms/master -o /dev/null"
wait_for "Keycloak API reachable" 120 curl -sf "${KC_URL}/realms/master" -o /dev/null

log "Installing the operator via Helm"
helm upgrade --install keycloak-operator "$REPO_ROOT/charts/keycloak-operator" \
  --namespace keycloak-operator-system --create-namespace \
  --set image.repository="${IMG%%:*}" --set image.tag="${IMG##*:}" \
  --set image.pullPolicy=IfNotPresent \
  --set resyncPeriod=30s >/dev/null
wait_for "operator ready" 180 kubectl rollout status deployment/keycloak-operator -n keycloak-operator-system

log "Applying example resources"
kubectl create namespace keycloak-system --dry-run=client -o yaml | kubectl apply -f -
# Point the examples at the local Keycloak server.
python3 - "$REPO_ROOT" "$KC_URL_INCLUSTER" <<'EOF'
import re, sys, pathlib
root, url = sys.argv[1], sys.argv[2]
src = pathlib.Path(root) / "examples"
dst = pathlib.Path(root) / "test/e2e/manifests/generated"
dst.mkdir(parents=True, exist_ok=True)
for f in sorted(src.glob("*.yaml")):
    text = f.read_text()
    text = text.replace("https://keycloak.example.com", url)
    # Order matters: the longer placeholders must be replaced first.
    text = text.replace("change-me-too", "portal-secret-e2e")
    text = text.replace("change-me-as-well", "corp-secret-e2e")
    text = text.replace("change-me", "admin")
    (dst / f.name).write_text(text)
EOF
kubectl apply -f "$REPO_ROOT/test/e2e/manifests/generated/"

synced() {
  [[ "$(kubectl get "$1" -n keycloak-system -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null)" == "True" ]]
}

log "Waiting for resources to converge"
for kind in realm/acme client/acme-portal clientscope/acme-roles-scope realmrole/acme-reader realmrole/acme-manager group/acme-platform-team identityprovider/corp-oidc; do
  wait_for "$kind synced" 120 synced "$kind"
done

log "Verifying server state through the Admin API"
assert "realm acme exists with the configured display name" \
  bash -c "api GET /admin/realms/acme | grep -q 'ACME Platform'"
assert "client acme-portal exists" \
  bash -c "api GET '/admin/realms/acme/clients?clientId=acme-portal' | grep -q acme-portal"
# These retry: the operator converges on its resync period, not instantly.
wait_for "client secret injected from the Secret" 120 \
  bash -c "api GET /admin/realms/acme/clients/\$(client_field acme-portal id)/client-secret | grep -q portal-secret-e2e"
wait_for "outbound secret written for applications" 120 \
  bash -c "kubectl get secret acme-portal-credentials -n keycloak-system -o jsonpath='{.data.clientSecret}' | base64 -d | grep -q portal-secret-e2e"
# Keycloak masks the client secret in GET responses; a masked value proves
# the Secret-backed injection happened.
wait_for "identity provider corp has Secret-backed config" 120 \
  bash -c "api GET /admin/realms/acme/identity-provider/instances/corp | grep -q 'clientSecret.*\\*\\*\\*'"
wait_for "group platform-team has the reader role" 120 \
  bash -c "api GET /admin/realms/acme/groups/\$(group_field platform-team id)/role-mappings/realm | grep -q reader"
wait_for "composite role manager includes reader" 120 \
  bash -c "api GET /admin/realms/acme/roles/manager/composites | grep -q reader"

log "Drift test: out-of-band change must be reverted"
api PUT /admin/realms/acme '{"displayName":"Hacked"}' >/dev/null
wait_for "drift reverted" 120 bash -c "api GET /admin/realms/acme | grep -q 'ACME Platform'"
echo "PASS: drift reverted"

log "Deletion test: removing the client CR deletes the server-side client"
kubectl delete client acme-portal -n keycloak-system
wait_for "client deleted from server" 120 bash -c "! api GET '/admin/realms/acme/clients?clientId=acme-portal' | grep -q acme-portal"
echo "PASS: client deleted"

log "Realm orphan test: deleting the realm CR keeps the realm"
kubectl delete realm acme -n keycloak-system
wait_for "realm CR gone" 60 bash -c "! kubectl get realm acme -n keycloak-system -o name"
assert "realm still exists on the server (orphaned)" \
  bash -c "api GET /admin/realms/acme | grep -q 'ACME Platform'"

if $CLEANUP; then
  log "Cleaning up"
  kubectl delete namespace keycloak-system keycloak-e2e keycloak-operator-system --ignore-not-found
  echo "Stack removed."
else
  log "Suite passed. The stack is left running for exploration:"
  echo "  kubectl get keycloakconnections,realms,clients -A"
  echo "  Keycloak admin console: kubectl -n $NAMESPACE port-forward svc/keycloak 8080:8080"
  echo "  (admin / admin) — tear down with: bash test/e2e/e2e.sh --cleanup"
fi
