/*
Copyright 2026 CTN Solutions

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
)

func conditionOf(t *testing.T, obj client.Object, condType string) metav1.Condition {
	t.Helper()
	status := obj.(interface {
		GetResourceStatus() *keycloakv1alpha1.ResourceStatus
	}).GetResourceStatus()
	cond := meta.FindStatusCondition(status.Conditions, condType)
	if cond == nil {
		t.Fatalf("condition %s not found on %s", condType, obj.GetName())
	}
	return *cond
}

func eventually(t *testing.T, description string, probe func() bool) {
	t.Helper()
	deadline := time.Now().Add(eventuallyTimeout)
	for time.Now().Before(deadline) {
		if probe() {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func createConnection(t *testing.T, name string) {
	t.Helper()
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-creds", Namespace: testNS},
		Data:       map[string][]byte{"username": []byte("admin"), "password": []byte("test-admin-pass")},
	}
	if err := k8sClient.Create(ctx, secret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create credentials secret: %v", err)
	}

	conn := &keycloakv1alpha1.KeycloakConnection{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNS},
		Spec: keycloakv1alpha1.KeycloakConnectionSpec{
			URL:                  fakeKC.URL(),
			CredentialsSecretRef: secret.Name,
		},
	}
	if err := k8sClient.Create(ctx, conn); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create connection: %v", err)
	}

	eventually(t, "connection "+name+" ready", func() bool {
		var latest keycloakv1alpha1.KeycloakConnection
		if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: testNS, Name: name}, &latest); err != nil {
			return false
		}
		cond := meta.FindStatusCondition(latest.Status.Conditions, keycloakv1alpha1.ConditionReady)
		return cond != nil && cond.Status == metav1.ConditionTrue
	})
}

func TestConnectionValidation(t *testing.T) {
	ctx := context.Background()

	badSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "bad-creds", Namespace: testNS},
		Data:       map[string][]byte{"username": []byte("admin"), "password": []byte("wrong")},
	}
	if err := k8sClient.Create(ctx, badSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create bad secret: %v", err)
	}
	badConn := &keycloakv1alpha1.KeycloakConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "bad-conn", Namespace: testNS},
		Spec: keycloakv1alpha1.KeycloakConnectionSpec{
			URL:                  fakeKC.URL(),
			CredentialsSecretRef: "bad-creds",
		},
	}
	if err := k8sClient.Create(ctx, badConn); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create bad connection: %v", err)
	}

	eventually(t, "bad connection reported not ready", func() bool {
		var latest keycloakv1alpha1.KeycloakConnection
		if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: testNS, Name: "bad-conn"}, &latest); err != nil {
			return false
		}
		cond := meta.FindStatusCondition(latest.Status.Conditions, keycloakv1alpha1.ConditionReady)
		return cond != nil && cond.Status == metav1.ConditionFalse && cond.Reason == keycloakv1alpha1.ReasonConnectionUnavailable
	})

	createConnection(t, "main-conn")
}

func TestRealmLifecycle(t *testing.T) {
	ctx := context.Background()
	createConnection(t, "realm-conn")

	realm := &keycloakv1alpha1.Realm{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-realm", Namespace: testNS},
		Spec: keycloakv1alpha1.RealmSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "realm-conn"},
			Realm:       "demo",
			Enabled:     ptr.To(true),
			DisplayName: ptr.To("Demo Realm"),
		},
	}
	if err := k8sClient.Create(ctx, realm); err != nil {
		t.Fatalf("create realm: %v", err)
	}

	// Created on the server with spec values.
	eventually(t, "realm created on server", func() bool {
		rep, ok := fakeKC.Realm("demo")
		return ok && rep["displayName"] == "Demo Realm"
	})

	// Synced condition true.
	eventually(t, "realm synced", func() bool {
		var latest keycloakv1alpha1.Realm
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(realm), &latest); err != nil {
			return false
		}
		cond := meta.FindStatusCondition(latest.Status.Conditions, keycloakv1alpha1.ConditionSynced)
		return cond != nil && cond.Status == metav1.ConditionTrue
	})

	// Drift correction: out-of-band change is reverted.
	fakeKC.SetRealmField("demo", "displayName", "Hacked")
	eventually(t, "drift reverted", func() bool {
		rep, _ := fakeKC.Realm("demo")
		return rep["displayName"] == "Demo Realm"
	})

	// Spec update is applied.
	updateWithRetry(t, realm, func(r *keycloakv1alpha1.Realm) {
		r.Spec.DisplayName = ptr.To("Renamed Realm")
		r.Spec.RegistrationAllowed = ptr.To(true)
	})
	eventually(t, "spec update applied", func() bool {
		rep, _ := fakeKC.Realm("demo")
		return rep["displayName"] == "Renamed Realm" && rep["registrationAllowed"] == true
	})

	// Field removal resets the server value.
	updateWithRetry(t, realm, func(r *keycloakv1alpha1.Realm) {
		r.Spec.DisplayName = nil
	})
	eventually(t, "removed field reset on server", func() bool {
		rep, _ := fakeKC.Realm("demo")
		name, present := rep["displayName"]
		return !present || name == ""
	})

	// Deletion with default Orphan policy keeps the realm.
	if err := k8sClient.Delete(ctx, realm); err != nil {
		t.Fatalf("delete realm: %v", err)
	}
	eventually(t, "realm CR gone", func() bool {
		return apierrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(realm), &keycloakv1alpha1.Realm{}))
	})
	if _, ok := fakeKC.Realm("demo"); !ok {
		t.Fatal("expected orphaned realm to remain on server")
	}
}

func TestRealmDeletionPolicyDelete(t *testing.T) {
	ctx := context.Background()
	createConnection(t, "del-conn")

	realm := &keycloakv1alpha1.Realm{
		ObjectMeta: metav1.ObjectMeta{Name: "doomed-realm", Namespace: testNS},
		Spec: keycloakv1alpha1.RealmSpec{
			KeycloakRef:    keycloakv1alpha1.KeycloakRef{Name: "del-conn"},
			Realm:          "doomed",
			DeletionPolicy: keycloakv1alpha1.DeletionDelete.Ptr(),
		},
	}
	if err := k8sClient.Create(ctx, realm); err != nil {
		t.Fatalf("create realm: %v", err)
	}
	eventually(t, "doomed realm created", func() bool {
		_, ok := fakeKC.Realm("doomed")
		return ok
	})

	if err := k8sClient.Delete(ctx, realm); err != nil {
		t.Fatalf("delete realm: %v", err)
	}
	eventually(t, "realm deleted from server", func() bool {
		_, ok := fakeKC.Realm("doomed")
		return !ok
	})
}

func TestProtectedRealm(t *testing.T) {
	ctx := context.Background()
	createConnection(t, "prot-conn")

	realm := &keycloakv1alpha1.Realm{
		ObjectMeta: metav1.ObjectMeta{Name: "master-attempt", Namespace: testNS},
		Spec: keycloakv1alpha1.RealmSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "prot-conn"},
			Realm:       "master",
		},
	}
	if err := k8sClient.Create(ctx, realm); err != nil {
		t.Fatalf("create realm: %v", err)
	}

	eventually(t, "protected realm refused", func() bool {
		var latest keycloakv1alpha1.Realm
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(realm), &latest); err != nil {
			return false
		}
		cond := meta.FindStatusCondition(latest.Status.Conditions, keycloakv1alpha1.ConditionReady)
		return cond != nil && cond.Status == metav1.ConditionFalse && cond.Reason == keycloakv1alpha1.ReasonProtectedRealm
	})

	// The escape hatch allows management.
	updateWithRetry(t, realm, func(r *keycloakv1alpha1.Realm) {
		annotations := r.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[keycloakv1alpha1.AllowProtectedAnnotation] = "true"
		r.SetAnnotations(annotations)
	})
	eventually(t, "protected realm managed with annotation", func() bool {
		var updated keycloakv1alpha1.Realm
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(realm), &updated); err != nil {
			return false
		}
		cond := meta.FindStatusCondition(updated.Status.Conditions, keycloakv1alpha1.ConditionSynced)
		return cond != nil && cond.Status == metav1.ConditionTrue
	})
}

func TestAdoptionPolicies(t *testing.T) {
	ctx := context.Background()
	createConnection(t, "adopt-conn")

	// A foreign realm exists on the server.
	if err := fakeKCCreateRealm("foreign"); err != nil {
		t.Fatalf("seed foreign realm: %v", err)
	}

	// CreateOnly refuses to touch it.
	foreign := &keycloakv1alpha1.Realm{
		ObjectMeta: metav1.ObjectMeta{Name: "foreign-realm", Namespace: testNS},
		Spec: keycloakv1alpha1.RealmSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "adopt-conn"},
			Realm:       "foreign",
		},
	}
	if err := k8sClient.Create(ctx, foreign); err != nil {
		t.Fatalf("create foreign realm CR: %v", err)
	}
	eventually(t, "foreign realm conflict reported", func() bool {
		var latest keycloakv1alpha1.Realm
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(foreign), &latest); err != nil {
			return false
		}
		cond := meta.FindStatusCondition(latest.Status.Conditions, keycloakv1alpha1.ConditionReady)
		return cond != nil && cond.Status == metav1.ConditionFalse && cond.Reason == keycloakv1alpha1.ReasonAlreadyExists
	})
	if rep, ok := fakeKC.Realm("foreign"); ok {
		if _, managed := rep["attributes"]; managed {
			t.Fatal("foreign realm must not be modified by CreateOnly")
		}
	}

	// Adopt takes it over.
	updateWithRetry(t, foreign, func(r *keycloakv1alpha1.Realm) {
		r.Spec.AdoptionPolicy = keycloakv1alpha1.AdoptionAdopt.Ptr()
		r.Spec.DisplayName = ptr.To("Adopted")
	})
	eventually(t, "foreign realm adopted and enforced", func() bool {
		rep, ok := fakeKC.Realm("foreign")
		if !ok || rep["displayName"] != "Adopted" {
			return false
		}
		attrs, _ := rep["attributes"].(map[string]any)
		return attrs[keycloakv1alpha1.ManagedAnnotation] == keycloakv1alpha1.ManagedValue
	})
}

func TestClientSecretsAndScopes(t *testing.T) {
	ctx := context.Background()
	createConnection(t, "client-conn")
	if err := fakeKCCreateRealm("app"); err != nil {
		t.Fatalf("seed realm: %v", err)
	}

	// A client scope to assign.
	scope := &keycloakv1alpha1.ClientScope{
		ObjectMeta: metav1.ObjectMeta{Name: "app-scope", Namespace: testNS},
		Spec: keycloakv1alpha1.ClientScopeSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "client-conn"},
			Realm:       "app",
			Name:        "custom-scope",
		},
	}
	if err := k8sClient.Create(ctx, scope); err != nil {
		t.Fatalf("create scope: %v", err)
	}
	eventually(t, "scope created", func() bool {
		_, ok := fakeKC.ClientScope("app", "custom-scope")
		return ok
	})

	// Inbound secret.
	inSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app-client-secret", Namespace: testNS},
		Data:       map[string][]byte{"clientSecret": []byte("inbound-secret-value")},
	}
	if err := k8sClient.Create(ctx, inSecret); err != nil {
		t.Fatalf("create inbound secret: %v", err)
	}

	appClient := &keycloakv1alpha1.Client{
		ObjectMeta: metav1.ObjectMeta{Name: "app-client", Namespace: testNS},
		Spec: keycloakv1alpha1.ClientSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "client-conn"},
			Realm:       "app",
			ClientID:    "app",
			Enabled:     ptr.To(true),
			SecretRef:   &keycloakv1alpha1.SecretKeySelector{Name: "app-client-secret"},
			SecretOutput: &keycloakv1alpha1.SecretKeySelector{
				Name: "app-client-output",
			},
			DefaultClientScopes: ptr.To([]string{"custom-scope"}),
		},
	}
	if err := k8sClient.Create(ctx, appClient); err != nil {
		t.Fatalf("create client: %v", err)
	}

	// Client created with the inbound secret.
	eventually(t, "client created with inbound secret", func() bool {
		rep, ok := fakeKC.Client("app", "app")
		return ok && rep["secret"] == "inbound-secret-value"
	})

	// Outbound secret written.
	eventually(t, "outbound secret written", func() bool {
		out := &corev1.Secret{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: testNS, Name: "app-client-output"}, out); err != nil {
			return false
		}
		return string(out.Data["clientSecret"]) == "inbound-secret-value"
	})

	// Default scope assigned.
	eventually(t, "default scope assigned", func() bool {
		names := fakeKC.ClientDefaultScopes("app", "app")
		for _, n := range names {
			if n == "custom-scope" {
				return true
			}
		}
		return false
	})

	// Clearing the list removes the assignments; removing the field would
	// leave them untouched (unmanaged).
	updateWithRetry(t, appClient, func(c *keycloakv1alpha1.Client) {
		c.Spec.DefaultClientScopes = ptr.To([]string{})
	})
	eventually(t, "default scope removed", func() bool {
		for _, n := range fakeKC.ClientDefaultScopes("app", "app") {
			if n == "custom-scope" {
				return false
			}
		}
		return true
	})

	// The output secret is owned by the Client resource (garbage collection
	// requires a real cluster and is covered by the e2e suite).
	out := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: testNS, Name: "app-client-output"}, out); err != nil {
		t.Fatalf("get output secret: %v", err)
	}
	if len(out.OwnerReferences) == 0 || out.OwnerReferences[0].Kind != "Client" {
		t.Fatal("output secret must be owned by the Client resource")
	}

	// Deletion removes the client from the server.
	if err := k8sClient.Delete(ctx, appClient); err != nil {
		t.Fatalf("delete client: %v", err)
	}
	eventually(t, "client deleted from server", func() bool {
		_, ok := fakeKC.Client("app", "app")
		return !ok
	})
}

func TestRealmRoleComposites(t *testing.T) {
	ctx := context.Background()
	createConnection(t, "role-conn")
	if err := fakeKCCreateRealm("roles"); err != nil {
		t.Fatalf("seed realm: %v", err)
	}

	basic := &keycloakv1alpha1.RealmRole{
		ObjectMeta: metav1.ObjectMeta{Name: "basic-role", Namespace: testNS},
		Spec: keycloakv1alpha1.RealmRoleSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "role-conn"},
			Realm:       "roles",
			Name:        "basic",
		},
	}
	if err := k8sClient.Create(ctx, basic); err != nil {
		t.Fatalf("create basic role: %v", err)
	}
	eventually(t, "basic role created", func() bool {
		_, ok := fakeKC.Role("roles", "basic")
		return ok
	})

	composite := &keycloakv1alpha1.RealmRole{
		ObjectMeta: metav1.ObjectMeta{Name: "composite-role", Namespace: testNS},
		Spec: keycloakv1alpha1.RealmRoleSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "role-conn"},
			Realm:       "roles",
			Name:        "manager",
			Composite:   ptr.To(true),
			Composites: &keycloakv1alpha1.RoleComposites{
				RealmRoles: []string{"basic"},
			},
		},
	}
	if err := k8sClient.Create(ctx, composite); err != nil {
		t.Fatalf("create composite role: %v", err)
	}
	eventually(t, "composite membership applied", func() bool {
		names := fakeKC.RoleComposites("roles", "manager")
		return len(names) == 1 && names[0] == "basic"
	})
}

func TestIdentityProviderConfigSecret(t *testing.T) {
	ctx := context.Background()
	createConnection(t, "idp-conn")
	if err := fakeKCCreateRealm("app"); err != nil {
		t.Fatalf("seed realm: %v", err)
	}

	idpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "idp-secret", Namespace: testNS},
		Data:       map[string][]byte{"oidc-client-secret": []byte("idp-s3cret")},
	}
	if err := k8sClient.Create(ctx, idpSecret); err != nil {
		t.Fatalf("create idp secret: %v", err)
	}

	idp := &keycloakv1alpha1.IdentityProvider{
		ObjectMeta: metav1.ObjectMeta{Name: "corp-oidc", Namespace: testNS},
		Spec: keycloakv1alpha1.IdentityProviderSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "idp-conn"},
			Realm:       "app",
			Alias:       "corp",
			ProviderID:  "oidc",
			Enabled:     ptr.To(true),
			Config: map[string]string{
				"clientId":         "corp-client",
				"tokenUrl":         "https://corp.example.com/token",
				"authorizationUrl": "https://corp.example.com/auth",
			},
			ConfigSecretRef: &keycloakv1alpha1.SecretKeysSelector{
				Name: "idp-secret",
				Keys: map[string]string{"clientSecret": "oidc-client-secret"},
			},
		},
	}
	if err := k8sClient.Create(ctx, idp); err != nil {
		t.Fatalf("create idp: %v", err)
	}

	eventually(t, "idp created with secret-backed config", func() bool {
		rep, ok := fakeKC.IdentityProvider("app", "corp")
		if !ok {
			return false
		}
		config, _ := rep["config"].(map[string]any)
		return config["clientSecret"] == "idp-s3cret" && config["clientId"] == "corp-client"
	})
}

func TestGroupRoleMappings(t *testing.T) {
	ctx := context.Background()
	createConnection(t, "group-conn")
	if err := fakeKCCreateRealm("teams"); err != nil {
		t.Fatalf("seed realm: %v", err)
	}

	// Seed a role and a client with a role.
	basic := &keycloakv1alpha1.RealmRole{
		ObjectMeta: metav1.ObjectMeta{Name: "group-basic", Namespace: testNS},
		Spec: keycloakv1alpha1.RealmRoleSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "group-conn"},
			Realm:       "teams",
			Name:        "member",
		},
	}
	if err := k8sClient.Create(ctx, basic); err != nil {
		t.Fatalf("create role: %v", err)
	}
	eventually(t, "role seeded", func() bool {
		_, ok := fakeKC.Role("teams", "member")
		return ok
	})

	appClient := &keycloakv1alpha1.Client{
		ObjectMeta: metav1.ObjectMeta{Name: "team-client", Namespace: testNS},
		Spec: keycloakv1alpha1.ClientSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "group-conn"},
			Realm:       "teams",
			ClientID:    "team-app",
		},
	}
	if err := k8sClient.Create(ctx, appClient); err != nil {
		t.Fatalf("create client: %v", err)
	}
	eventually(t, "client seeded", func() bool {
		_, ok := fakeKC.Client("teams", "team-app")
		return ok
	})
	// Seed a client role directly on the fake server via the admin client.
	if err := fakeKCSeedClientRole("teams", "team-app", "app-admin"); err != nil {
		t.Fatalf("seed client role: %v", err)
	}

	group := &keycloakv1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{Name: "platform-team", Namespace: testNS},
		Spec: keycloakv1alpha1.GroupSpec{
			KeycloakRef: keycloakv1alpha1.KeycloakRef{Name: "group-conn"},
			Realm:       "teams",
			Name:        "platform",
			RealmRoles:  ptr.To([]string{"member"}),
			ClientRoles: ptr.To(map[string][]string{"team-app": {"app-admin"}}),
		},
	}
	if err := k8sClient.Create(ctx, group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	eventually(t, "group created with role mappings", func() bool {
		if _, ok := fakeKC.Group("teams", "platform"); !ok {
			return false
		}
		realmOK := false
		for _, n := range fakeKC.GroupRealmRoles("teams", "platform") {
			if n == "member" {
				realmOK = true
			}
		}
		clientOK := false
		for _, n := range fakeKC.GroupClientRoles("teams", "platform", "client-team-app") {
			if n == "app-admin" {
				clientOK = true
			}
		}
		return realmOK && clientOK
	})
}

func TestFailIfExistsPolicy(t *testing.T) {
	ctx := context.Background()
	createConnection(t, "fail-conn")

	if err := fakeKCCreateRealm("taken"); err != nil {
		t.Fatalf("seed realm: %v", err)
	}

	realm := &keycloakv1alpha1.Realm{
		ObjectMeta: metav1.ObjectMeta{Name: "taken-realm", Namespace: testNS},
		Spec: keycloakv1alpha1.RealmSpec{
			KeycloakRef:    keycloakv1alpha1.KeycloakRef{Name: "fail-conn"},
			Realm:          "taken",
			AdoptionPolicy: keycloakv1alpha1.AdoptionFailIfExists.Ptr(),
		},
	}
	if err := k8sClient.Create(ctx, realm); err != nil {
		t.Fatalf("create realm: %v", err)
	}
	eventually(t, "FailIfExists conflict reported", func() bool {
		var latest keycloakv1alpha1.Realm
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(realm), &latest); err != nil {
			return false
		}
		cond := meta.FindStatusCondition(latest.Status.Conditions, keycloakv1alpha1.ConditionReady)
		return cond != nil && cond.Status == metav1.ConditionFalse && cond.Reason == keycloakv1alpha1.ReasonAlreadyExists
	})
}
