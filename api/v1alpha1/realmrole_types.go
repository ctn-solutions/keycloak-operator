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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleComposites describes the roles a composite role is made of.
type RoleComposites struct {
	// RealmRoles lists realm role names included in the composite.
	// +optional
	RealmRoles []string `json:"realmRoles,omitempty"`
	// ClientRoles maps a client ID to the names of that client's roles
	// included in the composite.
	// +optional
	ClientRoles map[string][]string `json:"clientRoles,omitempty"`
}

// RealmRoleSpec defines the desired state of a RealmRole. Fields mirror the
// Keycloak RoleRepresentation (realm level) one-to-one.
type RealmRoleSpec struct {
	// KeycloakRef points to the KeycloakConnection managing this role.
	// +kubebuilder:validation:Required
	KeycloakRef KeycloakRef `json:"keycloakRef"`

	// Realm is the name of the realm the role lives in.
	// +kubebuilder:validation:MinLength=1
	Realm string `json:"realm"`

	// Name is the role name on the Keycloak server.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// AdoptionPolicy controls the behaviour when a role with the same name
	// already exists. Defaults to CreateOnly.
	// +kubebuilder:validation:Enum=CreateOnly;Adopt;FailIfExists
	// +optional
	AdoptionPolicy *AdoptionPolicy `json:"adoptionPolicy,omitempty"`

	// DeletionPolicy controls whether the role is deleted from the server
	// when this resource is deleted. Defaults to Delete.
	// +kubebuilder:validation:Enum=Orphan;Delete
	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`

	// +optional
	Description *string `json:"description,omitempty"`
	// Composite marks the role as composite. When true, Composites defines
	// the exact set of included roles.
	// +optional
	Composite *bool `json:"composite,omitempty"`
	// +optional
	Composites *RoleComposites `json:"composites,omitempty"`
	// +optional
	Attributes map[string][]string `json:"attributes,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=keycloak,shortName=kcrole
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Realm",type=string,JSONPath=`.spec.realm`
// +kubebuilder:printcolumn:name="Connection",type=string,JSONPath=`.spec.keycloakRef.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RealmRole manages a realm-level role on a Keycloak server. The spec mirrors
// the Keycloak RoleRepresentation; see the Keycloak server documentation for
// field semantics.
type RealmRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RealmRoleSpec  `json:"spec,omitempty"`
	Status ResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RealmRoleList contains a list of RealmRole.
type RealmRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RealmRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RealmRole{}, &RealmRoleList{})
}
