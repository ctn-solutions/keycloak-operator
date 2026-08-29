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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
	"github.com/ctn-solutions/keycloak-operator/internal/keycloak"
)

// DefaultResync is the periodic drift-correction interval.
const DefaultResync = time.Hour

// connectionRetry is the requeue interval while the connection or its
// credentials are unavailable.
const connectionRetry = 30 * time.Second

// secretRetry is the requeue interval for recoverable terminal errors such
// as a missing Secret.
const secretRetry = 30 * time.Second

// Driver encapsulates the kind-specific behaviour of one managed resource
// type. The engine drives the shared lifecycle through it.
type Driver interface {
	// Get fetches the remote representation, returning keycloak.ErrNotFound
	// when the resource does not exist.
	Get(ctx context.Context, kc *keycloak.Client, obj ManagedObject) (map[string]any, error)
	// ID extracts the server-side identifier from a remote representation.
	ID(remote map[string]any) string
	// Create applies the payload as a new resource.
	Create(ctx context.Context, kc *keycloak.Client, obj ManagedObject, payload map[string]any) error
	// Update applies the payload to the resource with the given id.
	Update(ctx context.Context, kc *keycloak.Client, obj ManagedObject, id string, payload map[string]any) error
	// Delete removes the remote resource, resolving its id as needed.
	Delete(ctx context.Context, kc *keycloak.Client, obj ManagedObject) error
	// ManagedMarker stamps the managed annotation onto the payload where the
	// representation allows it.
	ManagedMarker(payload map[string]any)
	// IsManaged reports whether the remote representation carries the
	// managed marker.
	IsManaged(remote map[string]any) bool
	// PreparePayload injects kind-specific values (inbound secrets) into the
	// payload before create or update.
	PreparePayload(ctx context.Context, kc *keycloak.Client, obj ManagedObject, r client.Client, payload map[string]any) error
	// PostApply enforces kind-specific state that lives outside the main
	// representation (outbound secrets, scope assignments, role mappings).
	// It reports whether it changed server-side state.
	PostApply(ctx context.Context, kc *keycloak.Client, obj ManagedObject, r client.Client, remote map[string]any) (bool, error)
	// Protected reports whether the resource targets a protected server-side
	// resource, with an explanation.
	Protected(obj ManagedObject) (bool, string)
	// Spec exposes the concrete spec through the shared accessor interface.
	Spec(obj ManagedObject) Spec
	// OperatorFields lists the top-level spec fields that are operator
	// bookkeeping (references and policies) rather than Keycloak
	// representation fields. The engine strips them from the payload.
	OperatorFields() []string
}

// Engine implements the shared reconciliation lifecycle for all managed
// resource kinds.
type Engine struct {
	client   client.Client
	provider *keycloak.Provider
	recorder record.EventRecorder
	resync   time.Duration
}

// NewEngine builds an engine.
func NewEngine(c client.Client, provider *keycloak.Provider, recorder record.EventRecorder, resync time.Duration) *Engine {
	if resync <= 0 {
		resync = DefaultResync
	}
	return &Engine{client: c, provider: provider, recorder: recorder, resync: resync}
}

// Reconcile runs the shared lifecycle for one object.
func (e *Engine) Reconcile(ctx context.Context, obj ManagedObject, drv Driver) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	status := obj.GetResourceStatus()
	base := obj.DeepCopyObject().(client.Object)

	// Connection resolution.
	kc, err := e.provider.For(ctx, obj.GetNamespace(), drv.Spec(obj).ConnectionName())
	if err != nil {
		log.Error(err, "Keycloak connection unavailable", "connection", drv.Spec(obj).ConnectionName())
		e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionReady, metav1.ConditionFalse,
			keycloakv1alpha1.ReasonConnectionUnavailable, err.Error())
		e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionSynced, metav1.ConditionFalse,
			keycloakv1alpha1.ReasonConnectionUnavailable, "Connection unavailable")
		return ctrl.Result{RequeueAfter: connectionRetry}, nil
	}

	// Deletion.
	if !obj.GetDeletionTimestamp().IsZero() {
		return e.handleDeletion(ctx, obj, drv, kc)
	}

	// Finalizer.
	if !controllerutil.ContainsFinalizer(obj, keycloakv1alpha1.FinalizerName) {
		controllerutil.AddFinalizer(obj, keycloakv1alpha1.FinalizerName)
		if err := e.client.Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Protected resources.
	if protected, message := drv.Protected(obj); protected {
		e.fail(ctx, obj, base, keycloakv1alpha1.ReasonProtectedRealm, message)
		return ctrl.Result{}, nil
	}

	// Effective payload from spec + last-applied. Operator-only fields are
	// stripped so the payload contains representation fields only.
	specMap, err := toJSONMap(drv.Spec(obj))
	if err == nil {
		for _, field := range drv.OperatorFields() {
			delete(specMap, field)
		}
	}
	if err != nil {
		e.fail(ctx, obj, base, keycloakv1alpha1.ReasonFailed, fmt.Sprintf("encode spec: %v", err))
		return ctrl.Result{}, nil
	}
	lastApplied := obj.GetAnnotations()[keycloakv1alpha1.LastAppliedAnnotation]
	payload, err := EffectivePayload(specMap, lastApplied)
	if err != nil {
		e.fail(ctx, obj, base, keycloakv1alpha1.ReasonFailed, err.Error())
		return ctrl.Result{}, nil
	}

	remote, err := drv.Get(ctx, kc, obj)
	switch {
	case errors.Is(err, keycloak.ErrNotFound):
		return e.handleCreate(ctx, obj, drv, kc, base, specMap, payload)
	case err != nil:
		log.Error(err, "Failed to read remote resource")
		e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionReady, metav1.ConditionFalse,
			keycloakv1alpha1.ReasonRetrying, err.Error())
		e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionSynced, metav1.ConditionFalse,
			keycloakv1alpha1.ReasonRetrying, "Server unreachable")
		return ctrl.Result{}, err
	default:
		return e.handleExisting(ctx, obj, drv, kc, base, specMap, payload, remote)
	}
}

func (e *Engine) handleCreate(ctx context.Context, obj ManagedObject, drv Driver, kc *keycloak.Client,
	base client.Object, specMap, payload map[string]any) (ctrl.Result, error) {
	status := obj.GetResourceStatus()

	if err := drv.PreparePayload(ctx, kc, obj, e.client, payload); err != nil {
		e.fail(ctx, obj, base, keycloakv1alpha1.ReasonSecretMissing, err.Error())
		return ctrl.Result{RequeueAfter: secretRetry}, nil
	}
	drv.ManagedMarker(payload)

	if err := drv.Create(ctx, kc, obj, payload); err != nil {
		if errors.Is(err, keycloak.ErrConflict) {
			e.fail(ctx, obj, base, keycloakv1alpha1.ReasonAlreadyExists, "Resource appeared during reconciliation: "+err.Error())
			return ctrl.Result{}, nil
		}
		e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionReady, metav1.ConditionFalse,
			keycloakv1alpha1.ReasonRetrying, err.Error())
		e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionSynced, metav1.ConditionFalse,
			keycloakv1alpha1.ReasonRetrying, "Create failed")
		return ctrl.Result{}, err
	}

	e.record(obj, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created %s on Keycloak server", obj.GetObjectKind().GroupVersionKind().Kind))

	// Re-fetch the created resource so kind-specific post-processing can
	// resolve server-side identifiers.
	remote, err := drv.Get(ctx, kc, obj)
	if err != nil {
		e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionReady, metav1.ConditionFalse,
			keycloakv1alpha1.ReasonRetrying, err.Error())
		return ctrl.Result{}, err
	}
	postChanged, err := drv.PostApply(ctx, kc, obj, e.client, remote)
	if err != nil {
		e.fail(ctx, obj, base, keycloakv1alpha1.ReasonFailed, err.Error())
		return ctrl.Result{RequeueAfter: secretRetry}, nil
	}

	return e.finalize(ctx, obj, base, specMap, true || postChanged)
}

func (e *Engine) handleExisting(ctx context.Context, obj ManagedObject, drv Driver, kc *keycloak.Client,
	base client.Object, specMap, payload map[string]any, remote map[string]any) (ctrl.Result, error) {
	status := obj.GetResourceStatus()
	kind := obj.GetObjectKind().GroupVersionKind().Kind

	// Managed-by-us detection: the remote marker, or a last-applied
	// annotation from a previous reconciliation of this resource.
	managed := drv.IsManaged(remote) || obj.GetAnnotations()[keycloakv1alpha1.LastAppliedAnnotation] != ""

	if !managed {
		switch drv.Spec(obj).Adoption() {
		case keycloakv1alpha1.AdoptionAdopt:
			drv.ManagedMarker(payload)
			e.record(obj, corev1.EventTypeNormal, "Adopted", fmt.Sprintf("Adopted existing %s", kind))
		default:
			e.fail(ctx, obj, base, keycloakv1alpha1.ReasonAlreadyExists,
				"A foreign resource with the same key exists; set adoptionPolicy: Adopt to take it over")
			return ctrl.Result{}, nil
		}
	}

	if err := drv.PreparePayload(ctx, kc, obj, e.client, payload); err != nil {
		e.fail(ctx, obj, base, keycloakv1alpha1.ReasonSecretMissing, err.Error())
		return ctrl.Result{RequeueAfter: secretRetry}, nil
	}

	changed := PayloadDiffers(remote, payload)
	if changed {
		merged := MergePayload(cloneMap(remote), payload)
		if err := drv.Update(ctx, kc, obj, drv.ID(remote), merged); err != nil {
			if errors.Is(err, keycloak.ErrConflict) {
				e.fail(ctx, obj, base, keycloakv1alpha1.ReasonAlreadyExists, err.Error())
				return ctrl.Result{}, nil
			}
			e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionReady, metav1.ConditionFalse,
				keycloakv1alpha1.ReasonRetrying, err.Error())
			e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionSynced, metav1.ConditionFalse,
				keycloakv1alpha1.ReasonRetrying, "Update failed")
			return ctrl.Result{}, err
		}
		e.record(obj, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated %s on Keycloak server", kind))
	}

	postChanged, err := drv.PostApply(ctx, kc, obj, e.client, remote)
	if err != nil {
		e.fail(ctx, obj, base, keycloakv1alpha1.ReasonFailed, err.Error())
		return ctrl.Result{RequeueAfter: secretRetry}, nil
	}

	return e.finalize(ctx, obj, base, specMap, changed || postChanged)
}

// finalize records the last-applied annotation, marks the resource ready and
// schedules the next drift-correction pass.
func (e *Engine) finalize(ctx context.Context, obj ManagedObject, base client.Object, specMap map[string]any, synced bool) (ctrl.Result, error) {
	status := obj.GetResourceStatus()

	if encoded, err := json.Marshal(specMap); err == nil {
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		if annotations[keycloakv1alpha1.LastAppliedAnnotation] != string(encoded) {
			annotations[keycloakv1alpha1.LastAppliedAnnotation] = string(encoded)
			obj.SetAnnotations(annotations)
			if err := e.client.Patch(ctx, obj, client.MergeFrom(base)); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	reason := keycloakv1alpha1.ReasonSucceeded
	message := "Reconciliation succeeded"
	if !synced {
		message = "Server state matches spec"
	}
	e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionReady, metav1.ConditionTrue, reason, message)
	e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionSynced, metav1.ConditionTrue, reason, message)

	return ctrl.Result{RequeueAfter: e.resync}, nil
}

func (e *Engine) handleDeletion(ctx context.Context, obj ManagedObject, drv Driver, kc *keycloak.Client) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	if drv.Spec(obj).Deletion() == keycloakv1alpha1.DeletionDelete {
		if err := drv.Delete(ctx, kc, obj); err != nil && !errors.Is(err, keycloak.ErrNotFound) {
			log.Error(err, "Failed to delete remote resource")
			return ctrl.Result{}, err
		}
		e.record(obj, corev1.EventTypeNormal, "Deleted", "Deleted resource on Keycloak server")
	}
	controllerutil.RemoveFinalizer(obj, keycloakv1alpha1.FinalizerName)
	if err := e.client.Update(ctx, obj); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// setCondition writes one condition and patches the object status.
func (e *Engine) setCondition(ctx context.Context, status *keycloakv1alpha1.ResourceStatus, obj ManagedObject, base client.Object,
	condType string, condStatus metav1.ConditionStatus, reason, message string) {
	cond := metav1.Condition{
		Type:               condType,
		Status:             condStatus,
		ObservedGeneration: obj.GetGeneration(),
		Reason:             reason,
		Message:            message,
	}
	meta.SetStatusCondition(&status.Conditions, cond)
	e.patchStatus(ctx, obj, base)
}

// fail marks the resource as terminally failed.
func (e *Engine) fail(ctx context.Context, obj ManagedObject, base client.Object, reason, message string) {
	status := obj.GetResourceStatus()
	e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionReady, metav1.ConditionFalse, reason, message)
	e.setCondition(ctx, status, obj, base, keycloakv1alpha1.ConditionSynced, metav1.ConditionFalse, reason, message)
	e.record(obj, corev1.EventTypeWarning, reason, message)
}

func (e *Engine) patchStatus(ctx context.Context, obj ManagedObject, base client.Object) {
	if err := e.client.Status().Patch(ctx, obj, client.MergeFrom(base)); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to patch status")
	}
}

func (e *Engine) record(obj ManagedObject, eventType, reason, message string) {
	if e.recorder != nil {
		e.recorder.Event(obj, eventType, reason, message)
	}
}

func toJSONMap(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
