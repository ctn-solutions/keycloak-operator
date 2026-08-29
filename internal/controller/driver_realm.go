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

// RealmDriver manages realms.
type RealmDriver struct{}

// Spec exposes the concrete spec.
func (RealmDriver) Spec(obj ManagedObject) Spec {
	return obj.(*keycloakv1alpha1.Realm).GetSpec()
}

// Get fetches the realm representation.
func (RealmDriver) Get(ctx context.Context, kc *keycloak.Client, obj ManagedObject) (map[string]any, error) {
	return kc.GetRealm(ctx, specOf[*keycloakv1alpha1.RealmSpec](obj).TargetRealm())
}

// ID returns the realm name; realms are identified by name.
func (RealmDriver) ID(map[string]any) string { return "" }

// Create creates the realm.
func (RealmDriver) Create(ctx context.Context, kc *keycloak.Client, obj ManagedObject, payload map[string]any) error {
	return kc.CreateRealm(ctx, payload)
}

// Update applies the payload to the realm.
func (RealmDriver) Update(ctx context.Context, kc *keycloak.Client, obj ManagedObject, _ string, payload map[string]any) error {
	return kc.UpdateRealm(ctx, specOf[*keycloakv1alpha1.RealmSpec](obj).TargetRealm(), payload)
}

// Delete removes the realm.
func (RealmDriver) Delete(ctx context.Context, kc *keycloak.Client, obj ManagedObject) error {
	return kc.DeleteRealm(ctx, specOf[*keycloakv1alpha1.RealmSpec](obj).TargetRealm())
}

// ManagedMarker stamps the managed marker into the realm attributes.
func (RealmDriver) ManagedMarker(payload map[string]any) {
	stampStringAttribute(payload, keycloakv1alpha1.ManagedAnnotation, keycloakv1alpha1.ManagedValue)
}

// IsManaged reports whether the realm carries the managed marker.
func (RealmDriver) IsManaged(remote map[string]any) bool {
	return managedByStringAttribute(remote)
}

// PreparePayload injects sensitive SMTP values from a Secret.
func (RealmDriver) PreparePayload(ctx context.Context, _ *keycloak.Client, obj ManagedObject, r client.Client, payload map[string]any) error {
	spec := specOf[*keycloakv1alpha1.RealmSpec](obj)
	if spec.SMTPServerSecretRef == nil {
		return nil
	}
	values, err := readSecretValues(ctx, r, obj.GetNamespace(), spec.SMTPServerSecretRef)
	if err != nil {
		return err
	}
	smtp, _ := payload["smtpServer"].(map[string]any)
	if smtp == nil {
		smtp = map[string]any{}
		payload["smtpServer"] = smtp
	}
	for key, value := range values {
		smtp[key] = value
	}
	return nil
}

// PostApply performs no extra work for realms.
func (RealmDriver) PostApply(context.Context, *keycloak.Client, ManagedObject, client.Client, map[string]any) (bool, error) {
	return false, nil
}

// Protected refuses protected realms unless the allow-protected annotation
// is set.
func (RealmDriver) Protected(obj ManagedObject) (bool, string) {
	realm := specOf[*keycloakv1alpha1.RealmSpec](obj).TargetRealm()
	if _, protected := keycloakv1alpha1.ProtectedRealmNames[realm]; !protected {
		return false, ""
	}
	if obj.GetAnnotations()[keycloakv1alpha1.AllowProtectedAnnotation] == "true" {
		return false, ""
	}
	return true, fmt.Sprintf("realm %q is protected; annotate the resource with %s: \"true\" to manage it deliberately", realm, keycloakv1alpha1.AllowProtectedAnnotation)
}
