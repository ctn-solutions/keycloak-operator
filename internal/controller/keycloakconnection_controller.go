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
	"errors"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
	"github.com/ctn-solutions/keycloak-operator/internal/keycloak"
)

// connectionRetryInterval revalidates connections periodically so credential
// rotations and server outages are noticed without a spec change.
const connectionRetryInterval = 5 * time.Minute

// KeycloakConnectionReconciler validates KeycloakConnection resources against
// their server and reports the server version.
type KeycloakConnectionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Provider *keycloakProvider
}

// +kubebuilder:rbac:groups=keycloak.ctn-solutions.io,resources=keycloakconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups=keycloak.ctn-solutions.io,resources=keycloakconnections/status,verbs=get;update;patch
// Secrets are read for connection credentials and written for exported
// client secrets (secretOutput), which live in the resources' namespaces.
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch

// Reconcile validates the connection and refreshes its status.
func (r *KeycloakConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var conn keycloakv1alpha1.KeycloakConnection
	if err := r.Get(ctx, req.NamespacedName, &conn); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	base := conn.DeepCopy()

	kc, err := r.Provider.For(ctx, req.Namespace, req.Name)
	if err != nil {
		log.Error(err, "Keycloak connection unavailable", "connection", req.NamespacedName)
		setConnectionCondition(&conn, keycloakv1alpha1.ConditionReady, metav1.ConditionFalse,
			keycloakv1alpha1.ReasonConnectionUnavailable, err.Error(), conn.Generation)
		_ = r.Status().Patch(ctx, &conn, client.MergeFrom(base))
		return ctrl.Result{RequeueAfter: connectionRetry}, nil
	}

	info, err := kc.ServerInfo(ctx)
	if err != nil {
		log.Error(err, "Failed to reach Keycloak server", "connection", req.NamespacedName)
		reason := keycloakv1alpha1.ReasonRetrying
		if errors.Is(err, keycloak.ErrAuth) {
			reason = keycloakv1alpha1.ReasonConnectionUnavailable
		}
		setConnectionCondition(&conn, keycloakv1alpha1.ConditionReady, metav1.ConditionFalse,
			reason, err.Error(), conn.Generation)
		_ = r.Status().Patch(ctx, &conn, client.MergeFrom(base))
		return ctrl.Result{RequeueAfter: connectionRetry}, nil
	}

	version := serverVersion(info)
	conn.Status.ServerVersion = version
	setConnectionCondition(&conn, keycloakv1alpha1.ConditionReady, metav1.ConditionTrue,
		keycloakv1alpha1.ReasonSucceeded, "Connected to Keycloak "+version, conn.Generation)
	if err := r.Status().Patch(ctx, &conn, client.MergeFrom(base)); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: connectionRetryInterval}, nil
}

// SetupWithManager registers the controller.
func (r *KeycloakConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakv1alpha1.KeycloakConnection{}).
		Named("keycloakconnection").
		Complete(r)
}

func setConnectionCondition(conn *keycloakv1alpha1.KeycloakConnection, condType string,
	status metav1.ConditionStatus, reason, message string, generation int64) {
	meta.SetStatusCondition(&conn.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
}

func serverVersion(info map[string]any) string {
	if info == nil {
		return ""
	}
	systemInfo, _ := info["systemInfo"].(map[string]any)
	if systemInfo == nil {
		return ""
	}
	version, _ := systemInfo["version"].(string)
	return version
}

// keycloakProvider aliases the provider type so the controller package keeps
// a single import surface.
type keycloakProvider = keycloak.Provider
