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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
)

// GroupReconciler reconciles Group resources through the shared engine.
type GroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Engine *Engine
}

// +kubebuilder:rbac:groups=keycloak.ctn-solutions.io,resources=group,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keycloak.ctn-solutions.io,resources=group/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=keycloak.ctn-solutions.io,resources=group/finalizers,verbs=update

// Reconcile runs the shared lifecycle for a Group.
func (r *GroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj keycloakv1alpha1.Group
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return r.Engine.Reconcile(ctx, &obj, GroupDriver{})
}

// SetupWithManager registers the controller.
func (r *GroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keycloakv1alpha1.Group{}).
		Named("group").
		Complete(r)
}
