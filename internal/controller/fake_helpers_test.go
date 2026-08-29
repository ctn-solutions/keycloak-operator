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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ctn-solutions/keycloak-operator/internal/keycloak"
)

// fakeKCCreateRealm seeds a realm directly on the fake server, bypassing the
// operator, to simulate brownfield state.
func fakeKCCreateRealm(name string) error {
	kc := keycloak.New(keycloak.Config{URL: fakeKC.URL(), Username: "admin", Password: "test-admin-pass"})
	err := kc.CreateRealm(context.Background(), map[string]any{"realm": name, "enabled": true})
	if err != nil && strings.Contains(err.Error(), "realm exists") {
		return nil
	}
	return err
}

// fakeKCSeedClientRole seeds a client role directly on the fake server.
func fakeKCSeedClientRole(realm, clientID, roleName string) error {
	kc := keycloak.New(keycloak.Config{URL: fakeKC.URL(), Username: "admin", Password: "test-admin-pass"})
	ctx := context.Background()
	remote, err := kc.FindClient(ctx, realm, clientID)
	if err != nil {
		return err
	}
	uuid, _ := remote["id"].(string)
	body, _ := json.Marshal(map[string]any{"name": roleName})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/admin/realms/%s/clients/%s/roles", fakeKC.URL(), realm, uuid), bytes.NewReader(body))
	if err != nil {
		return err
	}
	token, err := kc.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("seed client role: status %d", resp.StatusCode)
	}
	return nil
}

// updateWithRetry updates an object, retrying on the conflicts the running
// controllers routinely produce by patching status.
func updateWithRetry[T client.Object](t *testing.T, obj T, mutate func(T)) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(eventuallyTimeout)
	for {
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			t.Fatalf("get for update: %v", err)
		}
		mutate(obj)
		err := k8sClient.Update(ctx, obj)
		if err == nil {
			return
		}
		if apierrors.IsConflict(err) && time.Now().Before(deadline) {
			time.Sleep(pollInterval)
			continue
		}
		t.Fatalf("update: %v", err)
	}
}
