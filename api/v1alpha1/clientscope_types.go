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

// ClientScopeSpec defines the desired state of a ClientScope. Fields mirror
// the Keycloak ClientScopeRepresentation one-to-one.
type ClientScopeSpec struct {
	// KeycloakRef points to the KeycloakConnection managing this client
	// scope.
	// +kubebuilder:validation:Required
	KeycloakRef KeycloakRef `json:"keycloakRef"`

	// Realm is the name of the realm the client scope lives in.
	// +kubebuilder:validation:MinLength=1
	Realm string `json:"realm"`

	// Name is the client scope name on the Keycloak server.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// AdoptionPolicy controls the behaviour when a client scope with the same
	// name already exists. Defaults to CreateOnly.
	// +kubebuilder:validation:Enum=CreateOnly;Adopt;FailIfExists
	// +optional
	AdoptionPolicy *AdoptionPolicy `json:"adoptionPolicy,omitempty"`

	// DeletionPolicy controls whether the client scope is deleted from the
	// server when this resource is deleted. Defaults to Delete.
	// +kubebuilder:validation:Enum=Orphan;Delete
	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`

	// +optional
	Description *string `json:"description,omitempty"`
	// Protocol is "openid-connect" or "saml".
	// +kubebuilder:validation:Enum=openid-connect;saml
	// +optional
	Protocol *string `json:"protocol,omitempty"`
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
	// +optional
	ProtocolMappers []ProtocolMapper `json:"protocolMappers,omitempty"`
	// +optional
	DisplayOnConsentScreen *bool `json:"displayOnConsentScreen,omitempty"`
	// +optional
	ConsentScreenText *string `json:"consentScreenText,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=keycloak,shortName=kcscope
// +kubebuilder:printcolumn:name="Scope",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Realm",type=string,JSONPath=`.spec.realm`
// +kubebuilder:printcolumn:name="Connection",type=string,JSONPath=`.spec.keycloakRef.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ClientScope manages a client scope on a Keycloak server. The spec mirrors
// the Keycloak ClientScopeRepresentation; see the Keycloak server
// documentation for field semantics.
type ClientScope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClientScopeSpec `json:"spec,omitempty"`
	Status ResourceStatus  `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClientScopeList contains a list of ClientScope.
type ClientScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClientScope `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClientScope{}, &ClientScopeList{})
}
