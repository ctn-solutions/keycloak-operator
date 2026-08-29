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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
	"github.com/ctn-solutions/keycloak-operator/internal/keycloak"
)

// stampStringAttribute records the managed marker in a map[string]string
// style attributes block (realms, clients, client scopes).
func stampStringAttribute(payload map[string]any, key, value string) {
	attrs, _ := payload["attributes"].(map[string]any)
	if attrs == nil {
		attrs = map[string]any{}
		payload["attributes"] = attrs
	}
	attrs[key] = value
}

// stampSliceAttribute records the managed marker in a map[string][]string
// style attributes block (roles, groups).
func stampSliceAttribute(payload map[string]any, key, value string) {
	attrs, _ := payload["attributes"].(map[string]any)
	if attrs == nil {
		attrs = map[string]any{}
		payload["attributes"] = attrs
	}
	list, _ := attrs[key].([]any)
	attrs[key] = append(list, value)
}

// managedByStringAttribute reports the marker in a string-valued attributes
// block.
func managedByStringAttribute(remote map[string]any) bool {
	attrs, _ := remote["attributes"].(map[string]any)
	if attrs == nil {
		return false
	}
	value, _ := attrs[keycloakv1alpha1.ManagedAnnotation].(string)
	return value == keycloakv1alpha1.ManagedValue
}

// managedBySliceAttribute reports the marker in a slice-valued attributes
// block.
func managedBySliceAttribute(remote map[string]any) bool {
	attrs, _ := remote["attributes"].(map[string]any)
	if attrs == nil {
		return false
	}
	list, _ := attrs[keycloakv1alpha1.ManagedAnnotation].([]any)
	for _, v := range list {
		if s, ok := v.(string); ok && s == keycloakv1alpha1.ManagedValue {
			return true
		}
	}
	return false
}

// readSecretValue reads one key from a Secret.
func readSecretValue(ctx context.Context, r client.Client, namespace string, selector *keycloakv1alpha1.SecretKeySelector, defaultKey string) (string, error) {
	key := selector.Key
	if key == "" {
		key = defaultKey
	}
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: selector.Name}, &secret); err != nil {
		return "", fmt.Errorf("secret %s/%s: %w", namespace, selector.Name, err)
	}
	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("secret %s/%s has no key %q", namespace, selector.Name, key)
	}
	return string(value), nil
}

// readSecretValues reads the mapped keys of a Secret.
func readSecretValues(ctx context.Context, r client.Client, namespace string, selector *keycloakv1alpha1.SecretKeysSelector) (map[string]string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: selector.Name}, &secret); err != nil {
		return nil, fmt.Errorf("secret %s/%s: %w", namespace, selector.Name, err)
	}
	out := make(map[string]string, len(selector.Keys))
	for target, key := range selector.Keys {
		value, ok := secret.Data[key]
		if !ok {
			return nil, fmt.Errorf("secret %s/%s has no key %q", namespace, selector.Name, key)
		}
		out[target] = string(value)
	}
	return out, nil
}

// ensureOutputSecret creates or updates a Secret holding an exported value
// and owns it through a controller reference. It reports whether the Secret
// changed.
func ensureOutputSecret(ctx context.Context, r client.Client, obj ManagedObject, name, key, value string) (bool, error) {
	var existing corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: name}, &existing)
	switch {
	case err == nil:
		// Never touch a Secret the operator does not own: a name collision
		// with a user-managed Secret must fail loudly instead of clobbering
		// foreign data.
		owned := false
		for _, ref := range existing.OwnerReferences {
			if ref.UID == obj.GetUID() && ref.Controller != nil && *ref.Controller {
				owned = true
				break
			}
		}
		if !owned {
			return false, fmt.Errorf("secret %s/%s already exists and is not owned by this resource; choose another secretOutput name", existing.Namespace, existing.Name)
		}
		if string(existing.Data[key]) == value {
			return false, nil
		}
		if existing.Data == nil {
			existing.Data = map[string][]byte{}
		}
		existing.Data[key] = []byte(value)
		return true, r.Update(ctx, &existing)
	case apierrors.IsNotFound(err):
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: obj.GetNamespace(),
			},
			Data: map[string][]byte{key: []byte(value)},
		}
		if err := controllerutil.SetControllerReference(obj, secret, r.Scheme()); err != nil {
			return false, err
		}
		return true, r.Create(ctx, secret)
	default:
		return false, err
	}
}

// enforceScopeAssignments makes the default or optional client scopes of a
// client match the desired names exactly. It reports whether it changed
// server-side state.
func enforceScopeAssignments(ctx context.Context, kc *keycloak.Client, realm, clientID string, desired []string, optional bool) (bool, error) {
	var current []map[string]any
	var err error
	assign := kc.AssignClientDefaultScope
	if optional {
		current, err = kc.ClientOptionalScopeAssignments(ctx, realm, clientID)
		assign = kc.AssignClientOptionalScope
	} else {
		current, err = kc.ClientDefaultScopeAssignments(ctx, realm, clientID)
	}
	if err != nil {
		return false, err
	}

	desiredSet := map[string]struct{}{}
	for _, name := range desired {
		desiredSet[name] = struct{}{}
	}
	currentNames := map[string]struct{}{}
	for _, rep := range current {
		if name, ok := rep["name"].(string); ok {
			currentNames[name] = struct{}{}
		}
	}

	var toAdd, toRemove []string
	for name := range desiredSet {
		if _, ok := currentNames[name]; !ok {
			toAdd = append(toAdd, name)
		}
	}
	for name := range currentNames {
		if _, ok := desiredSet[name]; !ok {
			toRemove = append(toRemove, name)
		}
	}
	if len(toAdd) == 0 && len(toRemove) == 0 {
		return false, nil
	}

	scopes, err := kc.ListClientScopes(ctx, realm)
	if err != nil {
		return false, err
	}
	idsByName := map[string]string{}
	for _, rep := range scopes {
		if name, ok := rep["name"].(string); ok {
			if id, ok := rep["id"].(string); ok {
				idsByName[name] = id
			}
		}
	}

	for _, name := range toAdd {
		id, ok := idsByName[name]
		if !ok {
			return false, fmt.Errorf("client scope %q does not exist in realm %q", name, realm)
		}
		if err := assign(ctx, realm, clientID, id, true); err != nil {
			return false, err
		}
	}
	for _, name := range toRemove {
		id, ok := idsByName[name]
		if !ok {
			continue
		}
		if err := assign(ctx, realm, clientID, id, false); err != nil {
			return false, err
		}
	}
	return true, nil
}

// diffByName splits desired and current name sets into additions and
// removals.
func diffByName(desired []string, current []map[string]any) (toAdd []string, toRemove []string) {
	desiredSet := map[string]struct{}{}
	for _, name := range desired {
		desiredSet[name] = struct{}{}
	}
	currentSet := map[string]struct{}{}
	for _, rep := range current {
		if name, ok := rep["name"].(string); ok {
			currentSet[name] = struct{}{}
		}
	}
	for name := range desiredSet {
		if _, ok := currentSet[name]; !ok {
			toAdd = append(toAdd, name)
		}
	}
	for name := range currentSet {
		if _, ok := desiredSet[name]; !ok {
			toRemove = append(toRemove, name)
		}
	}
	return toAdd, toRemove
}

// resolveRealmRoles fetches the representations of realm roles by name.
func resolveRealmRoles(ctx context.Context, kc *keycloak.Client, realm string, names []string) ([]map[string]any, error) {
	var out []map[string]any
	for _, name := range names {
		rep, err := kc.GetRole(ctx, realm, name)
		if err != nil {
			return nil, fmt.Errorf("realm role %q: %w", name, err)
		}
		out = append(out, rep)
	}
	return out, nil
}

// resolveClientRoles fetches the representations of client roles by client ID
// and role name.
func resolveClientRoles(ctx context.Context, kc *keycloak.Client, realm string, clientID string, names []string) ([]map[string]any, error) {
	client, err := kc.FindClient(ctx, realm, clientID)
	if err != nil {
		return nil, fmt.Errorf("client %q: %w", clientID, err)
	}
	uuid, _ := client["id"].(string)
	var out []map[string]any
	for _, name := range names {
		rep, err := kc.GetClientRole(ctx, realm, uuid, name)
		if err != nil {
			return nil, fmt.Errorf("role %q of client %q: %w", name, clientID, err)
		}
		out = append(out, rep)
	}
	return out, nil
}
