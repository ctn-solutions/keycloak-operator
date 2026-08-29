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

package keycloak

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func tokenServer(t *testing.T, issues *atomic.Int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/realms/master/protocol/openid-connect/token" {
			issues.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-1", "expires_in": 300})
			return
		}
		http.NotFound(w, r)
	}))
}

func TestTokenIsCached(t *testing.T) {
	var issues atomic.Int64
	srv := tokenServer(t, &issues)
	defer srv.Close()

	c := New(Config{URL: srv.URL, Username: "a", Password: "b"})
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := c.token(ctx); err != nil {
			t.Fatalf("token: %v", err)
		}
	}
	if got := issues.Load(); got != 1 {
		t.Fatalf("expected 1 token issue, got %d", got)
	}
}

func TestDoMapsErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/realms/master/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 300})
	})
	mux.HandleFunc("/admin/realms/r/gone", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	mux.HandleFunc("/admin/realms/r/clash", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "conflict", http.StatusConflict)
	})
	mux.HandleFunc("/admin/realms/r/boom", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(Config{URL: srv.URL, Username: "a", Password: "b"})
	ctx := context.Background()

	if err := c.Do(ctx, "GET", "/admin/realms/r/gone", nil, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := c.Do(ctx, "GET", "/admin/realms/r/clash", nil, nil); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	err, ok := c.Do(ctx, "GET", "/admin/realms/r/boom", nil, nil).(*APIError)
	if !ok || err.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected APIError 500, got %v", err)
	}
}

func TestDoRetriesOnceAfterAuthFailure(t *testing.T) {
	var tokenIssues, calls atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/realms/master/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		tokenIssues.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 300})
	})
	mux.HandleFunc("/admin/realms/r/thing", func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			// First call: the cached token is rejected server-side.
			http.Error(w, "expired", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(Config{URL: srv.URL, Username: "a", Password: "b"})
	ctx := context.Background()
	if _, err := c.token(ctx); err != nil {
		t.Fatalf("prime token: %v", err)
	}
	if err := c.Do(ctx, "GET", "/admin/realms/r/thing", nil, nil); err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("expected 2 calls (1 failure + 1 retry), got %d", got)
	}
	if got := tokenIssues.Load(); got != 2 {
		t.Fatalf("expected token refresh on retry, got %d issues", got)
	}
}

func TestClientCredentialsGrant(t *testing.T) {
	var grantType, clientID, clientSecret string
	mux := http.NewServeMux()
	mux.HandleFunc("/realms/master/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		grantType = r.PostFormValue("grant_type")
		clientID = r.PostFormValue("client_id")
		clientSecret = r.PostFormValue("client_secret")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 300})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(Config{URL: srv.URL, Auth: AuthClient, ClientID: "svc", ClientSecret: "s3cret"})
	if _, err := c.token(context.Background()); err != nil {
		t.Fatalf("token: %v", err)
	}
	if grantType != "client_credentials" || clientID != "svc" || clientSecret != "s3cret" {
		t.Fatalf("unexpected grant: %s %s %s", grantType, clientID, clientSecret)
	}
}
