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

// ProtocolMapper mirrors the Keycloak ProtocolMapperRepresentation.
type ProtocolMapper struct {
	// +optional
	Name *string `json:"name,omitempty"`
	// +optional
	Protocol *string `json:"protocol,omitempty"`
	// +optional
	ProtocolMapper *string `json:"protocolMapper,omitempty"`
	// +optional
	ConsentRequired *bool `json:"consentRequired,omitempty"`
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

// IdentityProviderMapper mirrors the Keycloak
// IdentityProviderMapperRepresentation.
type IdentityProviderMapper struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	IdentityProviderMapper *string `json:"identityProviderMapper,omitempty"`
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

// ClientSpec defines the desired state of a Client. Fields mirror the
// Keycloak ClientRepresentation one-to-one. The client secret is never part
// of the spec: it is either injected from a Secret (secretRef) or exported to
// one (secretOutput).
type ClientSpec struct {
	// KeycloakRef points to the KeycloakConnection managing this client.
	// +kubebuilder:validation:Required
	KeycloakRef KeycloakRef `json:"keycloakRef"`

	// Realm is the name of the realm the client lives in.
	// +kubebuilder:validation:MinLength=1
	Realm string `json:"realm"`

	// ClientID is the client identifier used in OIDC/SAML requests.
	// +kubebuilder:validation:MinLength=1
	ClientID string `json:"clientId"`

	// AdoptionPolicy controls the behaviour when a client with the same
	// clientId already exists. Defaults to CreateOnly.
	// +kubebuilder:validation:Enum=CreateOnly;Adopt;FailIfExists
	// +optional
	AdoptionPolicy *AdoptionPolicy `json:"adoptionPolicy,omitempty"`

	// DeletionPolicy controls whether the client is deleted from the server
	// when this resource is deleted. Defaults to Delete.
	// +kubebuilder:validation:Enum=Orphan;Delete
	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`

	// SecretRef injects the client secret from a Secret in the same
	// namespace. When set, the operator keeps the server-side secret in sync
	// with the Secret value.
	// +optional
	SecretRef *SecretKeySelector `json:"secretRef,omitempty"`

	// SecretOutput exports the effective client secret to a Secret in the
	// same namespace so applications can mount it. The operator owns the
	// referenced Secret and garbage-collects it with the Client resource.
	// +optional
	SecretOutput *SecretKeySelector `json:"secretOutput,omitempty"`

	// --- ClientRepresentation fields ---

	// +optional
	Name *string `json:"name,omitempty"`
	// +optional
	Description *string `json:"description,omitempty"`
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// Protocol is "openid-connect" or "saml".
	// +kubebuilder:validation:Enum=openid-connect;saml
	// +optional
	Protocol *string `json:"protocol,omitempty"`
	// +optional
	PublicClient *bool `json:"publicClient,omitempty"`
	// +optional
	BearerOnly *bool `json:"bearerOnly,omitempty"`
	// +optional
	StandardFlowEnabled *bool `json:"standardFlowEnabled,omitempty"`
	// +optional
	ImplicitFlowEnabled *bool `json:"implicitFlowEnabled,omitempty"`
	// +optional
	DirectAccessGrantsEnabled *bool `json:"directAccessGrantsEnabled,omitempty"`
	// +optional
	ServiceAccountsEnabled *bool `json:"serviceAccountsEnabled,omitempty"`
	// +optional
	AuthorizationServicesEnabled *bool `json:"authorizationServicesEnabled,omitempty"`
	// +optional
	FrontchannelLogout *bool `json:"frontchannelLogout,omitempty"`
	// +optional
	FullScopeAllowed *bool `json:"fullScopeAllowed,omitempty"`
	// +optional
	ConsentRequired *bool `json:"consentRequired,omitempty"`
	// +optional
	DisplayOnConsentScreen *bool `json:"displayOnConsentScreen,omitempty"`
	// +optional
	ConsentScreenText *string `json:"consentScreenText,omitempty"`
	// +optional
	AlwaysDisplayInConsole *bool `json:"alwaysDisplayInConsole,omitempty"`
	// +optional
	SurrogateAuthRequired *bool `json:"surrogateAuthRequired,omitempty"`
	// +optional
	RootURL *string `json:"rootUrl,omitempty"`
	// +optional
	BaseURL *string `json:"baseUrl,omitempty"`
	// +optional
	AdminURL *string `json:"adminUrl,omitempty"`
	// +optional
	RedirectUris []string `json:"redirectUris,omitempty"`
	// +optional
	WebOrigins []string `json:"webOrigins,omitempty"`
	// +optional
	NodeReRegistrationTimeout *int `json:"nodeReRegistrationTimeout,omitempty"`
	// ClientAuthenticatorType is for example "client_secret" or
	// "client_jwt".
	// +optional
	ClientAuthenticatorType *string `json:"clientAuthenticatorType,omitempty"`
	// +optional
	ProtocolMappers []ProtocolMapper `json:"protocolMappers,omitempty"`
	// DefaultClientScopes lists client scope names granted by default. The
	// operator enforces the exact set on the server.
	// +optional
	DefaultClientScopes []string `json:"defaultClientScopes,omitempty"`
	// OptionalClientScopes lists client scope names the client may request.
	// The operator enforces the exact set on the server.
	// +optional
	OptionalClientScopes []string `json:"optionalClientScopes,omitempty"`
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
	// AuthenticationFlowBindingOverrides maps flow bindings such as "browser"
	// or "grant" to flow aliases.
	// +optional
	AuthenticationFlowBindingOverrides map[string]string `json:"authenticationFlowBindingOverrides,omitempty"`
}

// ClientStatus reports the observed state of a Client.
type ClientStatus struct {
	ResourceStatus `json:",inline"`

	// SecretName is the name of the Secret written by secretOutput, when
	// configured.
	// +optional
	SecretName string `json:"secretName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=keycloak,shortName=kcclient
// +kubebuilder:printcolumn:name="Client ID",type=string,JSONPath=`.spec.clientId`
// +kubebuilder:printcolumn:name="Realm",type=string,JSONPath=`.spec.realm`
// +kubebuilder:printcolumn:name="Connection",type=string,JSONPath=`.spec.keycloakRef.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Client manages a client on a Keycloak server. The spec mirrors the Keycloak
// ClientRepresentation; see the Keycloak server documentation for field
// semantics.
type Client struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClientSpec   `json:"spec,omitempty"`
	Status ClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClientList contains a list of Client.
type ClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Client `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Client{}, &ClientList{})
}
