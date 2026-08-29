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
	"os"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
	"github.com/ctn-solutions/keycloak-operator/internal/keycloak"
	"github.com/ctn-solutions/keycloak-operator/internal/keycloak/fake"
)

// testResync is the drift-correction interval used in tests: short enough
// for fast assertions.
const testResync = 2 * time.Second

var (
	k8sClient client.Client
	testEnv   *envtest.Environment
	fakeKC    *fake.Server
	testNS    string
)

func TestMain(m *testing.M) {
	logf.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.UseDevMode(true)))

	os.Exit(runSuite(m))
}

func runSuite(m *testing.M) int {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(keycloakv1alpha1.AddToScheme(scheme))

	fakeKC = fake.New("admin", "test-admin-pass")

	crdDir := filepath.Join("..", "..", "config", "crd", "bases")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{crdDir},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start envtest: %v\n", err)
		return 1
	}
	defer func() { _ = testEnv.Stop() }()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create manager: %v\n", err)
		return 1
	}
	k8sClient = mgr.GetClient()

	provider := keycloak.NewProvider(k8sClient)
	engine := NewEngine(k8sClient, provider, mgr.GetEventRecorder("test"), testResync)

	for _, setup := range []func(ctrl.Manager) error{
		func(m ctrl.Manager) error {
			return (&KeycloakConnectionReconciler{Client: m.GetClient(), Scheme: m.GetScheme(), Provider: provider}).SetupWithManager(m)
		},
		func(m ctrl.Manager) error {
			return (&RealmReconciler{Client: m.GetClient(), Scheme: m.GetScheme(), Engine: engine}).SetupWithManager(m)
		},
		func(m ctrl.Manager) error {
			return (&ClientReconciler{Client: m.GetClient(), Scheme: m.GetScheme(), Engine: engine}).SetupWithManager(m)
		},
		func(m ctrl.Manager) error {
			return (&ClientScopeReconciler{Client: m.GetClient(), Scheme: m.GetScheme(), Engine: engine}).SetupWithManager(m)
		},
		func(m ctrl.Manager) error {
			return (&RealmRoleReconciler{Client: m.GetClient(), Scheme: m.GetScheme(), Engine: engine}).SetupWithManager(m)
		},
		func(m ctrl.Manager) error {
			return (&IdentityProviderReconciler{Client: m.GetClient(), Scheme: m.GetScheme(), Engine: engine}).SetupWithManager(m)
		},
		func(m ctrl.Manager) error {
			return (&GroupReconciler{Client: m.GetClient(), Scheme: m.GetScheme(), Engine: engine}).SetupWithManager(m)
		},
	} {
		if err := setup(mgr); err != nil {
			fmt.Fprintf(os.Stderr, "failed to set up controller: %v\n", err)
			return 1
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := mgr.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "manager stopped: %v\n", err)
		}
	}()

	// Dedicated namespace for the whole suite.
	testNS = "keycloak-operator-test"
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNS}}
	if err := k8sClient.Create(context.Background(), ns); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create test namespace: %v\n", err)
		return 1
	}

	return m.Run()
}

// eventuallyTimeout is the upper bound for asynchronous reconciliation.
const eventuallyTimeout = 15 * time.Second
const pollInterval = 200 * time.Millisecond
