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

// GroupSpec defines the desired state of a Group. Core fields mirror the
// Keycloak GroupRepresentation; role mappings are managed declaratively by
// name. Groups are flat in v1: nested sub-groups are not managed.
type GroupSpec struct {
	// KeycloakRef points to the KeycloakConnection managing this group.
	// +kubebuilder:validation:Required
	KeycloakRef KeycloakRef `json:"keycloakRef"`

	// Realm is the name of the realm the group lives in.
	// +kubebuilder:validation:MinLength=1
	Realm string `json:"realm"`

	// Name is the group name on the Keycloak server.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// AdoptionPolicy controls the behaviour when a group with the same name
	// already exists. Defaults to CreateOnly.
	// +kubebuilder:validation:Enum=CreateOnly;Adopt;FailIfExists
	// +optional
	AdoptionPolicy *AdoptionPolicy `json:"adoptionPolicy,omitempty"`

	// DeletionPolicy controls whether the group is deleted from the server
	// when this resource is deleted. Defaults to Delete.
	// +kubebuilder:validation:Enum=Orphan;Delete
	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`

	// Attributes holds group attributes.
	// +optional
	Attributes map[string][]string `json:"attributes,omitempty"`

	// RealmRoles lists realm role names granted to the group. The operator
	// enforces the exact set.
	// +optional
	RealmRoles []string `json:"realmRoles,omitempty"`

	// ClientRoles maps a client ID to the names of that client's roles
	// granted to the group. The operator enforces the exact set.
	// +optional
	ClientRoles map[string][]string `json:"clientRoles,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=keycloak,shortName=kcgroup
// +kubebuilder:printcolumn:name="Group",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Realm",type=string,JSONPath=`.spec.realm`
// +kubebuilder:printcolumn:name="Connection",type=string,JSONPath=`.spec.keycloakRef.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Group manages a group on a Keycloak server, including its realm and client
// role mappings.
type Group struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupSpec      `json:"spec,omitempty"`
	Status ResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupList contains a list of Group.
type GroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Group `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Group{}, &GroupList{})
}
