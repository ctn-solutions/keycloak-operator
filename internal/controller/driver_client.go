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

	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
	"github.com/ctn-solutions/keycloak-operator/internal/keycloak"
)

// ClientDriver manages clients.
type ClientDriver struct{}

// Spec exposes the concrete spec.
func (ClientDriver) Spec(obj ManagedObject) Spec {
	return obj.(*keycloakv1alpha1.Client).GetSpec()
}

// OperatorFields lists the spec fields that are operator bookkeeping rather
// than Keycloak representation fields. The realm name is URL-scoped for
// sub-resources, so it is operator-only here.
func (ClientDriver) OperatorFields() []string {
	return []string{"keycloakRef", "adoptionPolicy", "deletionPolicy", "realm", "secretRef", "secretOutput"}
}

// Get resolves the client by clientId.
func (ClientDriver) Get(ctx context.Context, kc *keycloak.Client, obj ManagedObject) (map[string]any, error) {
	spec := specOf[*keycloakv1alpha1.ClientSpec](obj)
	return kc.FindClient(ctx, spec.TargetRealm(), spec.ClientID)
}

// ID extracts the server-side client id.
func (ClientDriver) ID(remote map[string]any) string {
	id, _ := remote["id"].(string)
	return id
}

// Create creates the client.
func (ClientDriver) Create(ctx context.Context, kc *keycloak.Client, obj ManagedObject, payload map[string]any) error {
	return kc.CreateClient(ctx, specOf[*keycloakv1alpha1.ClientSpec](obj).TargetRealm(), payload)
}

// Update applies the payload to the client.
func (ClientDriver) Update(ctx context.Context, kc *keycloak.Client, obj ManagedObject, id string, payload map[string]any) error {
	return kc.UpdateClient(ctx, specOf[*keycloakv1alpha1.ClientSpec](obj).TargetRealm(), id, payload)
}

// Delete removes the client.
func (ClientDriver) Delete(ctx context.Context, kc *keycloak.Client, obj ManagedObject) error {
	spec := specOf[*keycloakv1alpha1.ClientSpec](obj)
	remote, err := kc.FindClient(ctx, spec.TargetRealm(), spec.ClientID)
	if err != nil {
		return err
	}
	return kc.DeleteClient(ctx, spec.TargetRealm(), remote["id"].(string))
}

// ManagedMarker stamps the managed marker into the client attributes.
func (ClientDriver) ManagedMarker(payload map[string]any) {
	stampStringAttribute(payload, keycloakv1alpha1.ManagedAnnotation, keycloakv1alpha1.ManagedValue)
}

// IsManaged reports whether the client carries the managed marker.
func (ClientDriver) IsManaged(remote map[string]any) bool {
	return managedByStringAttribute(remote)
}

// PreparePayload injects the client secret from a Secret when secretRef is
// configured.
func (ClientDriver) PreparePayload(ctx context.Context, _ *keycloak.Client, obj ManagedObject, r client.Client, payload map[string]any) error {
	spec := specOf[*keycloakv1alpha1.ClientSpec](obj)
	if spec.SecretRef == nil {
		return nil
	}
	value, err := readSecretValue(ctx, r, obj.GetNamespace(), spec.SecretRef, keycloakv1alpha1.ClientSecretKey)
	if err != nil {
		return err
	}
	payload["secret"] = value
	return nil
}

// PostApply exports the client secret and enforces client scope
// assignments.
func (ClientDriver) PostApply(ctx context.Context, kc *keycloak.Client, obj ManagedObject, r client.Client, remote map[string]any) (bool, error) {
	spec := specOf[*keycloakv1alpha1.ClientSpec](obj)
	id := remote["id"].(string)
	changed := false

	if spec.SecretOutput != nil {
		key := spec.SecretOutput.Key
		if key == "" {
			key = keycloakv1alpha1.ClientSecretKey
		}
		value, err := kc.GetClientSecret(ctx, spec.TargetRealm(), id)
		if err != nil {
			return changed, err
		}
		wrote, err := ensureOutputSecret(ctx, r, obj, spec.SecretOutput.Name, key, value)
		if err != nil {
			return changed, err
		}
		changed = changed || wrote
	}

	if spec.DefaultClientScopes != nil {
		wrote, err := enforceScopeAssignments(ctx, kc, spec.TargetRealm(), id, *spec.DefaultClientScopes, false)
		if err != nil {
			return changed, err
		}
		changed = changed || wrote
	}
	if spec.OptionalClientScopes != nil {
		wrote, err := enforceScopeAssignments(ctx, kc, spec.TargetRealm(), id, *spec.OptionalClientScopes, true)
		if err != nil {
			return changed, err
		}
		changed = changed || wrote
	}
	return changed, nil
}

// Protected performs no extra checks for clients.
func (ClientDriver) Protected(ManagedObject) (bool, string) { return false, "" }

// ClientScopeDriver manages client scopes.
type ClientScopeDriver struct{}

// Spec exposes the concrete spec.
func (ClientScopeDriver) Spec(obj ManagedObject) Spec {
	return obj.(*keycloakv1alpha1.ClientScope).GetSpec()
}

// OperatorFields lists the spec fields that are operator bookkeeping.
func (ClientScopeDriver) OperatorFields() []string {
	return []string{"keycloakRef", "adoptionPolicy", "deletionPolicy", "realm"}
}

// Get resolves the client scope by name.
func (ClientScopeDriver) Get(ctx context.Context, kc *keycloak.Client, obj ManagedObject) (map[string]any, error) {
	spec := specOf[*keycloakv1alpha1.ClientScopeSpec](obj)
	return kc.FindClientScope(ctx, spec.TargetRealm(), spec.Name)
}

// ID extracts the server-side scope id.
func (ClientScopeDriver) ID(remote map[string]any) string {
	id, _ := remote["id"].(string)
	return id
}

// Create creates the client scope.
func (ClientScopeDriver) Create(ctx context.Context, kc *keycloak.Client, obj ManagedObject, payload map[string]any) error {
	return kc.CreateClientScope(ctx, specOf[*keycloakv1alpha1.ClientScopeSpec](obj).TargetRealm(), payload)
}

// Update applies the payload to the client scope.
func (ClientScopeDriver) Update(ctx context.Context, kc *keycloak.Client, obj ManagedObject, id string, payload map[string]any) error {
	return kc.UpdateClientScope(ctx, specOf[*keycloakv1alpha1.ClientScopeSpec](obj).TargetRealm(), id, payload)
}

// Delete removes the client scope.
func (ClientScopeDriver) Delete(ctx context.Context, kc *keycloak.Client, obj ManagedObject) error {
	spec := specOf[*keycloakv1alpha1.ClientScopeSpec](obj)
	remote, err := kc.FindClientScope(ctx, spec.TargetRealm(), spec.Name)
	if err != nil {
		return err
	}
	return kc.DeleteClientScope(ctx, spec.TargetRealm(), remote["id"].(string))
}

// ManagedMarker stamps the managed marker into the scope attributes.
func (ClientScopeDriver) ManagedMarker(payload map[string]any) {
	stampStringAttribute(payload, keycloakv1alpha1.ManagedAnnotation, keycloakv1alpha1.ManagedValue)
}

// IsManaged reports whether the scope carries the managed marker.
func (ClientScopeDriver) IsManaged(remote map[string]any) bool {
	return managedByStringAttribute(remote)
}

// PreparePayload performs no injection for client scopes.
func (ClientScopeDriver) PreparePayload(context.Context, *keycloak.Client, ManagedObject, client.Client, map[string]any) error {
	return nil
}

// PostApply performs no extra work for client scopes.
func (ClientScopeDriver) PostApply(context.Context, *keycloak.Client, ManagedObject, client.Client, map[string]any) (bool, error) {
	return false, nil
}

// Protected performs no extra checks for client scopes.
func (ClientScopeDriver) Protected(ManagedObject) (bool, string) { return false, "" }
