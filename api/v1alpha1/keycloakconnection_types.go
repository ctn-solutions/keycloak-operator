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

// AuthType selects how the operator authenticates against the Keycloak
// administration interface.
//
// +kubebuilder:validation:Enum=password;client
type AuthType string

const (
	// AuthPassword authenticates with a username/password pair against the
	// built-in admin-cli client (password grant).
	AuthPassword AuthType = "password"

	// AuthClient authenticates with a service-account client (client
	// credentials grant).
	AuthClient AuthType = "client"
)

// TLSConfig configures transport security for the connection.
type TLSConfig struct {
	// InsecureSkipVerify disables TLS certificate verification. Use only in
	// trusted environments such as local development.
	// +optional
	InsecureSkipVerify *bool `json:"insecureSkipVerify,omitempty"`
}

// KeycloakConnectionSpec defines the desired state of a KeycloakConnection.
type KeycloakConnectionSpec struct {
	// URL is the base URL of the Keycloak server, for example
	// https://keycloak.example.com.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^https?://`
	URL string `json:"url"`

	// CredentialsSecretRef references a Secret in the same namespace holding
	// the administration credentials. For auth "password" the Secret must
	// provide the keys "username" and "password"; for auth "client" the keys
	// "clientId" and "clientSecret".
	// +kubebuilder:validation:MinLength=1
	CredentialsSecretRef string `json:"credentialsSecretRef"`

	// Auth selects the authentication method. Defaults to "password".
	// +kubebuilder:validation:Enum=password;client
	// +optional
	Auth *AuthType `json:"auth,omitempty"`

	// AdminRealm is the realm the administration credentials live in.
	// Defaults to "master".
	// +optional
	AdminRealm *string `json:"adminRealm,omitempty"`

	// TLS configures transport security.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty"`
}

// KeycloakConnectionStatus reports the observed state of the connection.
type KeycloakConnectionStatus struct {
	// Conditions report the outcome of the latest validation.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration is the generation most recently processed.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ServerVersion is the Keycloak version reported by the server, when the
	// connection is healthy.
	// +optional
	ServerVersion string `json:"serverVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=keycloak
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.url`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.serverVersion`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// KeycloakConnection describes a Keycloak server and the credentials used to
// administer it. All managed resources reference a connection through
// spec.keycloakRef.
type KeycloakConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakConnectionSpec   `json:"spec,omitempty"`
	Status KeycloakConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeycloakConnectionList contains a list of KeycloakConnection.
type KeycloakConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakConnection{}, &KeycloakConnectionList{})
}
