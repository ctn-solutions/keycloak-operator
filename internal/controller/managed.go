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
	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Spec is the structural view of a managed resource spec the engine works
// with. It is satisfied by every spec type in api/v1alpha1.
type Spec interface {
	// ConnectionName returns the name of the referenced KeycloakConnection.
	ConnectionName() string
	// TargetRealm returns the realm the resource applies to.
	TargetRealm() string
	// Adoption returns the effective adoption policy.
	Adoption() keycloakv1alpha1.AdoptionPolicy
	// Deletion returns the effective deletion policy.
	Deletion() keycloakv1alpha1.DeletionPolicy
}

// ManagedObject is the structural view of a managed resource the engine and
// drivers work with. It is satisfied by every resource kind in
// api/v1alpha1.
type ManagedObject interface {
	client.Object
	// GetResourceStatus exposes the shared status block.
	GetResourceStatus() *keycloakv1alpha1.ResourceStatus
}

// specOf extracts the concrete spec of a managed object.
func specOf[T any](obj ManagedObject) T {
	return obj.(interface{ GetSpec() T }).GetSpec()
}
