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

// IdentityProviderSpec defines the desired state of an IdentityProvider.
// Fields mirror the Keycloak IdentityProviderRepresentation one-to-one. The
// provider-specific "config" map carries the broker configuration; sensitive
// entries such as "clientSecret" should be injected through ConfigSecretRef
// instead of being written inline.
type IdentityProviderSpec struct {
	// KeycloakRef points to the KeycloakConnection managing this identity
	// provider.
	// +kubebuilder:validation:Required
	KeycloakRef KeycloakRef `json:"keycloakRef"`

	// Realm is the name of the realm the identity provider lives in.
	// +kubebuilder:validation:MinLength=1
	Realm string `json:"realm"`

	// Alias is the unique identifier of the identity provider within the
	// realm.
	// +kubebuilder:validation:MinLength=1
	Alias string `json:"alias"`

	// ProviderID is the Keycloak provider type, for example "oidc", "saml",
	// "google" or "github".
	// +kubebuilder:validation:MinLength=1
	ProviderID string `json:"providerId"`

	// AdoptionPolicy controls the behaviour when an identity provider with
	// the same alias already exists. Defaults to CreateOnly.
	// +kubebuilder:validation:Enum=CreateOnly;Adopt;FailIfExists
	// +optional
	AdoptionPolicy *AdoptionPolicy `json:"adoptionPolicy,omitempty"`

	// DeletionPolicy controls whether the identity provider is deleted from
	// the server when this resource is deleted. Defaults to Delete.
	// +kubebuilder:validation:Enum=Orphan;Delete
	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`

	// +optional
	DisplayName *string `json:"displayName,omitempty"`
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// +optional
	TrustEmail *bool `json:"trustEmail,omitempty"`
	// +optional
	StoreToken *bool `json:"storeToken,omitempty"`
	// +optional
	AddReadTokenRoleOnCreate *bool `json:"addReadTokenRoleOnCreate,omitempty"`
	// +optional
	LinkOnly *bool `json:"linkOnly,omitempty"`
	// +optional
	FirstBrokerLoginFlowAlias *string `json:"firstBrokerLoginFlowAlias,omitempty"`
	// +optional
	PostBrokerLoginFlowAlias *string `json:"postBrokerLoginFlowAlias,omitempty"`
	// Config holds the provider-specific configuration, for example
	// {"authorizationUrl": "...", "tokenUrl": "...", "clientId": "..."}.
	// +optional
	Config map[string]string `json:"config,omitempty"`
	// ConfigSecretRef injects sensitive config values from a Secret. Keys
	// maps a config entry (for example "clientSecret") to the Secret key
	// holding its value.
	// +optional
	ConfigSecretRef *SecretKeysSelector `json:"configSecretRef,omitempty"`
	// +optional
	Mappers []IdentityProviderMapper `json:"mappers,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=keycloak,shortName=kcidp
// +kubebuilder:printcolumn:name="Alias",type=string,JSONPath=`.spec.alias`
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.providerId`
// +kubebuilder:printcolumn:name="Realm",type=string,JSONPath=`.spec.realm`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// IdentityProvider manages an identity provider (broker) on a Keycloak
// server. The spec mirrors the Keycloak IdentityProviderRepresentation; see
// the Keycloak server documentation for field semantics.
type IdentityProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IdentityProviderSpec `json:"spec,omitempty"`
	Status ResourceStatus       `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IdentityProviderList contains a list of IdentityProvider.
type IdentityProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IdentityProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IdentityProvider{}, &IdentityProviderList{})
}
