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
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
	"github.com/ctn-solutions/keycloak-operator/internal/keycloak"
)

// RealmRoleDriver manages realm-level roles, including composite membership.
type RealmRoleDriver struct{}

// Spec exposes the concrete spec.
func (RealmRoleDriver) Spec(obj ManagedObject) Spec {
	return obj.(*keycloakv1alpha1.RealmRole).GetSpec()
}

// Get resolves the role by name.
func (RealmRoleDriver) Get(ctx context.Context, kc *keycloak.Client, obj ManagedObject) (map[string]any, error) {
	spec := specOf[*keycloakv1alpha1.RealmRoleSpec](obj)
	return kc.GetRole(ctx, spec.TargetRealm(), spec.Name)
}

// ID extracts the server-side role id.
func (RealmRoleDriver) ID(remote map[string]any) string {
	id, _ := remote["id"].(string)
	return id
}

// Create creates the role.
func (RealmRoleDriver) Create(ctx context.Context, kc *keycloak.Client, obj ManagedObject, payload map[string]any) error {
	return kc.CreateRole(ctx, specOf[*keycloakv1alpha1.ClientSpec](obj).TargetRealm(), payload)
}

// Update applies the payload to the role.
func (RealmRoleDriver) Update(ctx context.Context, kc *keycloak.Client, obj ManagedObject, _ string, payload map[string]any) error {
	spec := specOf[*keycloakv1alpha1.RealmRoleSpec](obj)
	return kc.UpdateRole(ctx, spec.TargetRealm(), spec.Name, payload)
}

// Delete removes the role.
func (RealmRoleDriver) Delete(ctx context.Context, kc *keycloak.Client, obj ManagedObject) error {
	spec := specOf[*keycloakv1alpha1.RealmRoleSpec](obj)
	return kc.DeleteRole(ctx, spec.TargetRealm(), spec.Name)
}

// ManagedMarker stamps the managed marker into the role attributes.
func (RealmRoleDriver) ManagedMarker(payload map[string]any) {
	stampSliceAttribute(payload, keycloakv1alpha1.ManagedAnnotation, keycloakv1alpha1.ManagedValue)
}

// IsManaged reports whether the role carries the managed marker.
func (RealmRoleDriver) IsManaged(remote map[string]any) bool {
	return managedBySliceAttribute(remote)
}

// PreparePayload performs no injection for roles.
func (RealmRoleDriver) PreparePayload(context.Context, *keycloak.Client, ManagedObject, client.Client, map[string]any) error {
	return nil
}

// PostApply enforces the composite membership of the role.
func (RealmRoleDriver) PostApply(ctx context.Context, kc *keycloak.Client, obj ManagedObject, _ client.Client, _ map[string]any) (bool, error) {
	spec := specOf[*keycloakv1alpha1.RealmRoleSpec](obj)

	current, err := kc.GetRoleComposites(ctx, spec.TargetRealm(), spec.Name)
	if err != nil {
		return false, err
	}

	var desired []map[string]any
	if spec.Composite != nil && *spec.Composite && spec.Composites != nil {
		realmRoles, err := resolveRealmRoles(ctx, kc, spec.TargetRealm(), spec.Composites.RealmRoles)
		if err != nil {
			return false, err
		}
		desired = append(desired, realmRoles...)
		for clientID, roles := range spec.Composites.ClientRoles {
			clientRoles, err := resolveClientRoles(ctx, kc, spec.TargetRealm(), clientID, roles)
			if err != nil {
				return false, err
			}
			desired = append(desired, clientRoles...)
		}
	}

	desiredIDs := map[string]map[string]any{}
	for _, rep := range desired {
		if id, ok := rep["id"].(string); ok {
			desiredIDs[id] = rep
		}
	}
	currentIDs := map[string]map[string]any{}
	for _, rep := range current {
		if id, ok := rep["id"].(string); ok {
			currentIDs[id] = rep
		}
	}

	var toAdd, toRemove []map[string]any
	for id, rep := range desiredIDs {
		if _, ok := currentIDs[id]; !ok {
			toAdd = append(toAdd, rep)
		}
	}
	for id, rep := range currentIDs {
		if _, ok := desiredIDs[id]; !ok {
			toRemove = append(toRemove, rep)
		}
	}
	if len(toAdd) == 0 && len(toRemove) == 0 {
		return false, nil
	}

	if len(toAdd) > 0 {
		if err := kc.AddRoleComposites(ctx, spec.TargetRealm(), spec.Name, toAdd); err != nil {
			return false, err
		}
	}
	if len(toRemove) > 0 {
		if err := kc.RemoveRoleComposites(ctx, spec.TargetRealm(), spec.Name, toRemove); err != nil {
			return false, err
		}
	}
	return true, nil
}

// Protected performs no extra checks for roles.
func (RealmRoleDriver) Protected(ManagedObject) (bool, string) { return false, "" }

// IdentityProviderDriver manages identity providers.
type IdentityProviderDriver struct{}

// Spec exposes the concrete spec.
func (IdentityProviderDriver) Spec(obj ManagedObject) Spec {
	return obj.(*keycloakv1alpha1.IdentityProvider).GetSpec()
}

// Get resolves the identity provider by alias.
func (IdentityProviderDriver) Get(ctx context.Context, kc *keycloak.Client, obj ManagedObject) (map[string]any, error) {
	spec := specOf[*keycloakv1alpha1.IdentityProviderSpec](obj)
	return kc.GetIdentityProvider(ctx, spec.TargetRealm(), spec.Alias)
}

// ID returns the alias; identity providers are addressed by alias.
func (IdentityProviderDriver) ID(remote map[string]any) string {
	alias, _ := remote["alias"].(string)
	return alias
}

// Create creates the identity provider.
func (IdentityProviderDriver) Create(ctx context.Context, kc *keycloak.Client, obj ManagedObject, payload map[string]any) error {
	return kc.CreateIdentityProvider(ctx, specOf[*keycloakv1alpha1.ClientSpec](obj).TargetRealm(), payload)
}

// Update applies the payload to the identity provider.
func (IdentityProviderDriver) Update(ctx context.Context, kc *keycloak.Client, obj ManagedObject, _ string, payload map[string]any) error {
	spec := specOf[*keycloakv1alpha1.IdentityProviderSpec](obj)
	return kc.UpdateIdentityProvider(ctx, spec.TargetRealm(), spec.Alias, payload)
}

// Delete removes the identity provider.
func (IdentityProviderDriver) Delete(ctx context.Context, kc *keycloak.Client, obj ManagedObject) error {
	spec := specOf[*keycloakv1alpha1.IdentityProviderSpec](obj)
	return kc.DeleteIdentityProvider(ctx, spec.TargetRealm(), spec.Alias)
}

// ManagedMarker is a no-op: the identity provider representation has no
// attributes block, so adoption is tracked through the last-applied
// annotation on the resource.
func (IdentityProviderDriver) ManagedMarker(map[string]any) {}

// IsManaged always reports false; the engine falls back to the last-applied
// annotation to recognise resources it manages.
func (IdentityProviderDriver) IsManaged(map[string]any) bool { return false }

// PreparePayload injects sensitive config values from a Secret.
func (IdentityProviderDriver) PreparePayload(ctx context.Context, _ *keycloak.Client, obj ManagedObject, r client.Client, payload map[string]any) error {
	spec := specOf[*keycloakv1alpha1.IdentityProviderSpec](obj)
	if spec.ConfigSecretRef == nil {
		return nil
	}
	values, err := readSecretValues(ctx, r, obj.GetNamespace(), spec.ConfigSecretRef)
	if err != nil {
		return err
	}
	config, _ := payload["config"].(map[string]any)
	if config == nil {
		config = map[string]any{}
		payload["config"] = config
	}
	for key, value := range values {
		config[key] = value
	}
	return nil
}

// PostApply performs no extra work for identity providers.
func (IdentityProviderDriver) PostApply(context.Context, *keycloak.Client, ManagedObject, client.Client, map[string]any) (bool, error) {
	return false, nil
}

// Protected performs no extra checks for identity providers.
func (IdentityProviderDriver) Protected(ManagedObject) (bool, string) { return false, "" }

// GroupDriver manages groups and their role mappings.
type GroupDriver struct{}

// Spec exposes the concrete spec.
func (GroupDriver) Spec(obj ManagedObject) Spec {
	return obj.(*keycloakv1alpha1.Group).GetSpec()
}

// Get resolves the group by name.
func (GroupDriver) Get(ctx context.Context, kc *keycloak.Client, obj ManagedObject) (map[string]any, error) {
	spec := specOf[*keycloakv1alpha1.GroupSpec](obj)
	return kc.FindGroup(ctx, spec.TargetRealm(), spec.Name)
}

// ID extracts the server-side group id.
func (GroupDriver) ID(remote map[string]any) string {
	id, _ := remote["id"].(string)
	return id
}

// Create creates the group.
func (GroupDriver) Create(ctx context.Context, kc *keycloak.Client, obj ManagedObject, payload map[string]any) error {
	return kc.CreateGroup(ctx, specOf[*keycloakv1alpha1.ClientSpec](obj).TargetRealm(), payload)
}

// Update applies the payload to the group.
func (GroupDriver) Update(ctx context.Context, kc *keycloak.Client, obj ManagedObject, id string, payload map[string]any) error {
	return kc.UpdateGroup(ctx, specOf[*keycloakv1alpha1.ClientSpec](obj).TargetRealm(), id, payload)
}

// Delete removes the group.
func (GroupDriver) Delete(ctx context.Context, kc *keycloak.Client, obj ManagedObject) error {
	spec := specOf[*keycloakv1alpha1.GroupSpec](obj)
	remote, err := kc.FindGroup(ctx, spec.TargetRealm(), spec.Name)
	if err != nil {
		return err
	}
	return kc.DeleteGroup(ctx, spec.TargetRealm(), remote["id"].(string))
}

// ManagedMarker stamps the managed marker into the group attributes.
func (GroupDriver) ManagedMarker(payload map[string]any) {
	stampSliceAttribute(payload, keycloakv1alpha1.ManagedAnnotation, keycloakv1alpha1.ManagedValue)
}

// IsManaged reports whether the group carries the managed marker.
func (GroupDriver) IsManaged(remote map[string]any) bool {
	return managedBySliceAttribute(remote)
}

// PreparePayload performs no injection for groups.
func (GroupDriver) PreparePayload(context.Context, *keycloak.Client, ManagedObject, client.Client, map[string]any) error {
	return nil
}

// PostApply enforces the realm and client role mappings of the group.
func (GroupDriver) PostApply(ctx context.Context, kc *keycloak.Client, obj ManagedObject, _ client.Client, remote map[string]any) (bool, error) {
	spec := specOf[*keycloakv1alpha1.GroupSpec](obj)
	id := remote["id"].(string)

	mappings, err := kc.GetGroupRoleMappings(ctx, spec.TargetRealm(), id)
	if err != nil {
		return false, err
	}

	changed := false

	// Realm role mappings.
	var currentRealm []map[string]any
	if realmMappings, ok := mappings["realmMappings"].([]any); ok {
		for _, rep := range realmMappings {
			if m, ok := rep.(map[string]any); ok {
				currentRealm = append(currentRealm, m)
			}
		}
	}
	desiredRealm, err := resolveRealmRoles(ctx, kc, spec.TargetRealm(), spec.RealmRoles)
	if err != nil {
		return changed, err
	}
	added, removed, err := applyRoleDiff(ctx, kc, spec.TargetRealm(), id, "", desiredRealm, currentRealm, kc.AddGroupRealmRoles, kc.RemoveGroupRealmRoles)
	if err != nil {
		return changed, err
	}
	changed = changed || added || removed

	// Client role mappings, including removals for clients no longer listed.
	type clientMapping struct {
		uuid     string
		mappings []map[string]any
	}
	currentByClient := map[string]clientMapping{} // keyed by client ID
	if clientMappings, ok := mappings["clientMappings"].(map[string]any); ok {
		for uuid, entry := range clientMappings {
			m, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			clientID, _ := m["client"].(string)
			var reps []map[string]any
			if list, ok := m["mappings"].([]any); ok {
				for _, rep := range list {
					if r, ok := rep.(map[string]any); ok {
						reps = append(reps, r)
					}
				}
			}
			currentByClient[clientID] = clientMapping{uuid: uuid, mappings: reps}
		}
	}

	desiredClients := map[string]struct{}{}
	for clientID := range spec.ClientRoles {
		desiredClients[clientID] = struct{}{}
	}
	for clientID, mapping := range currentByClient {
		if _, ok := desiredClients[clientID]; !ok {
			if len(mapping.mappings) > 0 {
				if err := kc.RemoveGroupClientRoles(ctx, spec.TargetRealm(), id, mapping.uuid, mapping.mappings); err != nil {
					return changed, err
				}
				changed = true
			}
			continue
		}
	}

	for clientID, roles := range spec.ClientRoles {
		desiredRoles, err := resolveClientRoles(ctx, kc, spec.TargetRealm(), clientID, roles)
		if err != nil {
			return changed, err
		}
		uuid, err := clientUUID(kc, ctx, spec.TargetRealm(), clientID)
		if err != nil {
			return changed, err
		}
		current := []map[string]any(nil)
		if mapping, ok := currentByClient[clientID]; ok {
			current = mapping.mappings
		}
		added, removed, err := applyRoleDiff(ctx, kc, spec.TargetRealm(), id, uuid, desiredRoles, current,
			func(ctx context.Context, realm, groupID string, reps []map[string]any) error {
				return kc.AddGroupClientRoles(ctx, realm, groupID, uuid, reps)
			},
			func(ctx context.Context, realm, groupID string, reps []map[string]any) error {
				return kc.RemoveGroupClientRoles(ctx, realm, groupID, uuid, reps)
			})
		if err != nil {
			return changed, err
		}
		changed = changed || added || removed
	}

	return changed, nil
}

// applyRoleDiff diffs desired against current role representations by id and
// applies additions and removals through the given functions.
func applyRoleDiff(ctx context.Context, kc *keycloak.Client, realm, groupID, _ string,
	desired []map[string]any, current []map[string]any,
	add func(context.Context, string, string, []map[string]any) error,
	remove func(context.Context, string, string, []map[string]any) error) (bool, bool, error) {

	desiredIDs := map[string]map[string]any{}
	for _, rep := range desired {
		if id, ok := rep["id"].(string); ok {
			desiredIDs[id] = rep
		}
	}
	currentIDs := map[string]map[string]any{}
	for _, rep := range current {
		if id, ok := rep["id"].(string); ok {
			currentIDs[id] = rep
		}
	}

	var toAdd, toRemove []map[string]any
	for id, rep := range desiredIDs {
		if _, ok := currentIDs[id]; !ok {
			toAdd = append(toAdd, rep)
		}
	}
	for id, rep := range currentIDs {
		if _, ok := desiredIDs[id]; !ok {
			toRemove = append(toRemove, rep)
		}
	}
	if len(toAdd) == 0 && len(toRemove) == 0 {
		return false, false, nil
	}
	if len(toAdd) > 0 {
		if err := add(ctx, realm, groupID, toAdd); err != nil {
			return false, false, err
		}
	}
	if len(toRemove) > 0 {
		if err := remove(ctx, realm, groupID, toRemove); err != nil {
			return true, false, err
		}
	}
	return true, true, nil
}

func clientUUID(kc *keycloak.Client, ctx context.Context, realm, clientID string) (string, error) {
	remote, err := kc.FindClient(ctx, realm, clientID)
	if err != nil {
		return "", fmt.Errorf("client %q: %w", clientID, err)
	}
	uuid, _ := remote["id"].(string)
	return uuid, nil
}

// Protected performs no extra checks for groups.
func (GroupDriver) Protected(ManagedObject) (bool, string) { return false, "" }
