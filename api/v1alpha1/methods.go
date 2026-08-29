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

package v1alpha1

// Accessor methods shared by every managed resource. They let the
// reconciliation engine work with all resource kinds through small
// structural interfaces without the API package depending on it.

// ConnectionName returns the name of the referenced KeycloakConnection.
func (s *RealmSpec) ConnectionName() string { return s.KeycloakRef.Name }

// TargetRealm returns the realm this resource applies to.
func (s *RealmSpec) TargetRealm() string { return s.Realm }

// Adoption returns the adoption policy, defaulting to CreateOnly.
func (s *RealmSpec) Adoption() AdoptionPolicy {
	if s.AdoptionPolicy != nil && *s.AdoptionPolicy != "" {
		return *s.AdoptionPolicy
	}
	return AdoptionCreateOnly
}

// Deletion returns the deletion policy, defaulting to Orphan: realms are
// never deleted implicitly.
func (s *RealmSpec) Deletion() DeletionPolicy {
	if s.DeletionPolicy != nil && *s.DeletionPolicy != "" {
		return *s.DeletionPolicy
	}
	return DeletionOrphan
}

// GetResourceStatus exposes the shared status block.
func (r *Realm) GetResourceStatus() *ResourceStatus { return &r.Status }

// GetSpec exposes the spec through the shared accessor interface.
func (r *Realm) GetSpec() *RealmSpec { return &r.Spec }

// ConnectionName returns the name of the referenced KeycloakConnection.
func (s *ClientSpec) ConnectionName() string { return s.KeycloakRef.Name }

// TargetRealm returns the realm this resource applies to.
func (s *ClientSpec) TargetRealm() string { return s.Realm }

// Adoption returns the adoption policy, defaulting to CreateOnly.
func (s *ClientSpec) Adoption() AdoptionPolicy {
	if s.AdoptionPolicy != nil && *s.AdoptionPolicy != "" {
		return *s.AdoptionPolicy
	}
	return AdoptionCreateOnly
}

// Deletion returns the deletion policy, defaulting to Delete.
func (s *ClientSpec) Deletion() DeletionPolicy {
	if s.DeletionPolicy != nil && *s.DeletionPolicy != "" {
		return *s.DeletionPolicy
	}
	return DeletionDelete
}

// GetResourceStatus exposes the shared status block.
func (c *Client) GetResourceStatus() *ResourceStatus { return &c.Status.ResourceStatus }

// GetSpec exposes the spec through the shared accessor interface.
func (c *Client) GetSpec() *ClientSpec { return &c.Spec }

// ConnectionName returns the name of the referenced KeycloakConnection.
func (s *ClientScopeSpec) ConnectionName() string { return s.KeycloakRef.Name }

// TargetRealm returns the realm this resource applies to.
func (s *ClientScopeSpec) TargetRealm() string { return s.Realm }

// Adoption returns the adoption policy, defaulting to CreateOnly.
func (s *ClientScopeSpec) Adoption() AdoptionPolicy {
	if s.AdoptionPolicy != nil && *s.AdoptionPolicy != "" {
		return *s.AdoptionPolicy
	}
	return AdoptionCreateOnly
}

// Deletion returns the deletion policy, defaulting to Delete.
func (s *ClientScopeSpec) Deletion() DeletionPolicy {
	if s.DeletionPolicy != nil && *s.DeletionPolicy != "" {
		return *s.DeletionPolicy
	}
	return DeletionDelete
}

// GetResourceStatus exposes the shared status block.
func (s *ClientScope) GetResourceStatus() *ResourceStatus { return &s.Status }

// GetSpec exposes the spec through the shared accessor interface.
func (s *ClientScope) GetSpec() *ClientScopeSpec { return &s.Spec }

// ConnectionName returns the name of the referenced KeycloakConnection.
func (s *RealmRoleSpec) ConnectionName() string { return s.KeycloakRef.Name }

// TargetRealm returns the realm this resource applies to.
func (s *RealmRoleSpec) TargetRealm() string { return s.Realm }

// Adoption returns the adoption policy, defaulting to CreateOnly.
func (s *RealmRoleSpec) Adoption() AdoptionPolicy {
	if s.AdoptionPolicy != nil && *s.AdoptionPolicy != "" {
		return *s.AdoptionPolicy
	}
	return AdoptionCreateOnly
}

// Deletion returns the deletion policy, defaulting to Delete.
func (s *RealmRoleSpec) Deletion() DeletionPolicy {
	if s.DeletionPolicy != nil && *s.DeletionPolicy != "" {
		return *s.DeletionPolicy
	}
	return DeletionDelete
}

// GetResourceStatus exposes the shared status block.
func (r *RealmRole) GetResourceStatus() *ResourceStatus { return &r.Status }

// GetSpec exposes the spec through the shared accessor interface.
func (r *RealmRole) GetSpec() *RealmRoleSpec { return &r.Spec }

// ConnectionName returns the name of the referenced KeycloakConnection.
func (s *IdentityProviderSpec) ConnectionName() string { return s.KeycloakRef.Name }

// TargetRealm returns the realm this resource applies to.
func (s *IdentityProviderSpec) TargetRealm() string { return s.Realm }

// Adoption returns the adoption policy, defaulting to CreateOnly.
func (s *IdentityProviderSpec) Adoption() AdoptionPolicy {
	if s.AdoptionPolicy != nil && *s.AdoptionPolicy != "" {
		return *s.AdoptionPolicy
	}
	return AdoptionCreateOnly
}

// Deletion returns the deletion policy, defaulting to Delete.
func (s *IdentityProviderSpec) Deletion() DeletionPolicy {
	if s.DeletionPolicy != nil && *s.DeletionPolicy != "" {
		return *s.DeletionPolicy
	}
	return DeletionDelete
}

// GetResourceStatus exposes the shared status block.
func (i *IdentityProvider) GetResourceStatus() *ResourceStatus { return &i.Status }

// GetSpec exposes the spec through the shared accessor interface.
func (i *IdentityProvider) GetSpec() *IdentityProviderSpec { return &i.Spec }

// ConnectionName returns the name of the referenced KeycloakConnection.
func (s *GroupSpec) ConnectionName() string { return s.KeycloakRef.Name }

// TargetRealm returns the realm this resource applies to.
func (s *GroupSpec) TargetRealm() string { return s.Realm }

// Adoption returns the adoption policy, defaulting to CreateOnly.
func (s *GroupSpec) Adoption() AdoptionPolicy {
	if s.AdoptionPolicy != nil && *s.AdoptionPolicy != "" {
		return *s.AdoptionPolicy
	}
	return AdoptionCreateOnly
}

// Deletion returns the deletion policy, defaulting to Delete.
func (s *GroupSpec) Deletion() DeletionPolicy {
	if s.DeletionPolicy != nil && *s.DeletionPolicy != "" {
		return *s.DeletionPolicy
	}
	return DeletionDelete
}

// GetResourceStatus exposes the shared status block.
func (g *Group) GetResourceStatus() *ResourceStatus { return &g.Status }

// GetSpec exposes the spec through the shared accessor interface.
func (g *Group) GetSpec() *GroupSpec { return &g.Spec }
