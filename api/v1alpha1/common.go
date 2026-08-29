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

// Group and resource constants.
const (
	// GroupName is the API group of all custom resources in this package.
	GroupName = "keycloak.ctn-solutions.io"

	// ManagedAnnotation marks a Keycloak server-side resource as managed by
	// this operator. It is stamped on create and on adoption.
	ManagedAnnotation = "keycloak.ctn-solutions.io/managed"
	// ManagedValue is the value stored under ManagedAnnotation.
	ManagedValue = "true"

	// LastAppliedAnnotation stores the JSON payload the operator last applied
	// for a resource. It powers declarative field removal: fields present in
	// the annotation but absent from the current spec are reset on the server.
	LastAppliedAnnotation = "keycloak.ctn-solutions.io/last-applied"

	// AllowProtectedAnnotation opts a Realm resource into managing a protected
	// realm such as "master".
	AllowProtectedAnnotation = "keycloak.ctn-solutions.io/allow-protected"

	// FinalizerName is the finalizer every managed resource carries until the
	// server-side deletion (or orphaning) has been settled.
	FinalizerName = "keycloak.ctn-solutions.io/finalizer"

	// ClientSecretKey is the default key holding the client secret in
	// inbound and outbound client secrets.
	ClientSecretKey = "clientSecret"
)

// ProtectedRealmNames lists realms the operator refuses to manage unless the
// resource carries the AllowProtectedAnnotation.
var ProtectedRealmNames = map[string]struct{}{
	"master": {},
}

// AdoptionPolicy controls how the operator treats a resource that already
// exists on the Keycloak server.
//
// +kubebuilder:validation:Enum=CreateOnly;Adopt;FailIfExists
type AdoptionPolicy string

const (
	// AdoptionCreateOnly creates the resource if absent and fails if a foreign
	// (unmanaged) resource with the same key exists. Resources previously
	// created or adopted by the operator are resumed.
	AdoptionCreateOnly AdoptionPolicy = "CreateOnly"

	// AdoptionAdopt takes over an existing resource: the managed marker is
	// stamped and the spec is enforced from then on.
	AdoptionAdopt AdoptionPolicy = "Adopt"

	// AdoptionFailIfExists fails whenever a resource with the same key exists,
	// even one previously managed by the operator.
	AdoptionFailIfExists AdoptionPolicy = "FailIfExists"
)

// DeletionPolicy controls what happens to the Keycloak server-side resource
// when the custom resource is deleted.
//
// +kubebuilder:validation:Enum=Orphan;Delete
type DeletionPolicy string

const (
	// DeletionOrphan leaves the server-side resource in place.
	DeletionOrphan DeletionPolicy = "Orphan"

	// DeletionDelete removes the server-side resource.
	DeletionDelete DeletionPolicy = "Delete"
)

// KeycloakRef points to the KeycloakConnection resource governing the
// Keycloak server a resource belongs to. The connection must live in the same
// namespace as the referencing resource.
type KeycloakRef struct {
	// Name of the KeycloakConnection resource.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// SecretKeySelector references a key in a Secret in the same namespace.
type SecretKeySelector struct {
	// Name of the Secret.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Key within the Secret. Defaults to "clientSecret" for client secrets.
	// +kubebuilder:validation:MinLength=1
	// +optional
	Key string `json:"key,omitempty"`
}

// SecretKeysSelector references multiple keys of a Secret and maps them onto
// target configuration keys (for example identity provider config entries or
// SMTP server settings).
type SecretKeysSelector struct {
	// Name of the Secret.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Keys maps a target configuration key to the Secret key holding its
	// value.
	// +kubebuilder:validation:MinProperties=1
	Keys map[string]string `json:"keys"`
}

// ResourceStatus is the shared status block of all managed resources.
type ResourceStatus struct {
	// Conditions report the outcome of the latest reconciliation.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration is the generation most recently processed.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// Condition types and reasons used across the operator.
const (
	// ConditionReady reports whether the resource is reconciled and healthy.
	ConditionReady = "Ready"
	// ConditionSynced reports whether the Keycloak server state matches the
	// resource spec.
	ConditionSynced = "Synced"

	// ReasonSucceeded marks a successful reconciliation.
	ReasonSucceeded = "Succeeded"
	// ReasonRetrying marks a transient error; the operator retries with
	// backoff.
	ReasonRetrying = "Retrying"
	// ReasonFailed marks a terminal error that requires operator attention.
	ReasonFailed = "Failed"
	// ReasonConnectionUnavailable marks an unreachable or misconfigured
	// KeycloakConnection.
	ReasonConnectionUnavailable = "ConnectionUnavailable"
	// ReasonAlreadyExists marks a conflict with an existing server-side
	// resource under the current adoption policy.
	ReasonAlreadyExists = "AlreadyExists"
	// ReasonSecretMissing marks a referenced Secret or key that does not
	// exist.
	ReasonSecretMissing = "SecretMissing"
	// ReasonProtectedRealm marks an attempt to manage a protected realm
	// without the allow-protected annotation.
	ReasonProtectedRealm = "ProtectedRealm"
)

// Ptr returns a pointer to the policy value, for convenient spec literals.
func (p AdoptionPolicy) Ptr() *AdoptionPolicy { return &p }

// Ptr returns a pointer to the policy value, for convenient spec literals.
func (p DeletionPolicy) Ptr() *DeletionPolicy { return &p }
