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
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakv1alpha1 "github.com/ctn-solutions/keycloak-operator/api/v1alpha1"
)

// Secret keys read from the credentials Secret of a KeycloakConnection.
const (
	SecretKeyUsername     = "username"
	SecretKeyPassword     = "password"
	SecretKeyClientID     = "clientId"
	SecretKeyClientSecret = "clientSecret"
)

// Provider resolves KeycloakConnection resources into authenticated Admin API
// clients and caches them until the connection or its credentials change.
type Provider struct {
	reader client.Reader

	mu     sync.Mutex
	byName map[string]*cachedClient
}

type cachedClient struct {
	kc          *Client
	connVersion string
	secretVer   string
}

// NewProvider builds a provider reading connections and secrets through the
// given reader.
func NewProvider(reader client.Reader) *Provider {
	return &Provider{reader: reader, byName: map[string]*cachedClient{}}
}

// For returns the Admin API client for the connection in the given namespace.
// The returned error wraps ErrNotFound-style Kubernetes errors when the
// connection or its credentials Secret is missing.
func (p *Provider) For(ctx context.Context, namespace, name string) (*Client, error) {
	var conn keycloakv1alpha1.KeycloakConnection
	if err := p.reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &conn); err != nil {
		if apierrors.IsNotFound(err) {
			// Drop the cached client so deleted connections do not keep
			// credentials in memory.
			p.mu.Lock()
			delete(p.byName, namespace+"/"+name)
			p.mu.Unlock()
		}
		return nil, fmt.Errorf("get keycloak connection %s/%s: %w", namespace, name, err)
	}

	var secret corev1.Secret
	if err := p.reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: conn.Spec.CredentialsSecretRef}, &secret); err != nil {
		return nil, fmt.Errorf("get credentials secret %s/%s: %w", namespace, conn.Spec.CredentialsSecretRef, err)
	}

	key := namespace + "/" + name
	connVersion := conn.ResourceVersion
	secretVersion := secret.ResourceVersion

	p.mu.Lock()
	cached, ok := p.byName[key]
	if ok && cached.connVersion == connVersion && cached.secretVer == secretVersion {
		kc := cached.kc
		p.mu.Unlock()
		return kc, nil
	}
	p.mu.Unlock()

	cfg, err := clientConfig(&conn, &secret)
	if err != nil {
		return nil, err
	}
	cfg.ConnectionName = namespace + "/" + name
	kc := New(cfg)

	p.mu.Lock()
	p.byName[key] = &cachedClient{
		kc:          kc,
		connVersion: connVersion,
		secretVer:   secretVersion,
	}
	p.mu.Unlock()
	return kc, nil
}

// clientConfig translates a connection and its credentials Secret into a
// client configuration.
func clientConfig(conn *keycloakv1alpha1.KeycloakConnection, secret *corev1.Secret) (Config, error) {
	cfg := Config{
		URL:        conn.Spec.URL,
		AdminRealm: DefaultAdminRealm,
	}
	if conn.Spec.AdminRealm != nil && *conn.Spec.AdminRealm != "" {
		cfg.AdminRealm = *conn.Spec.AdminRealm
	}
	if conn.Spec.TLS != nil && conn.Spec.TLS.InsecureSkipVerify != nil {
		cfg.InsecureSkipVerify = *conn.Spec.TLS.InsecureSkipVerify
	}

	auth := keycloakv1alpha1.AuthPassword
	if conn.Spec.Auth != nil && *conn.Spec.Auth != "" {
		auth = *conn.Spec.Auth
	}

	switch auth {
	case keycloakv1alpha1.AuthClient:
		cfg.Auth = AuthClient
		cfg.ClientID = string(secret.Data[SecretKeyClientID])
		cfg.ClientSecret = string(secret.Data[SecretKeyClientSecret])
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return Config{}, fmt.Errorf("credentials secret %q must provide %q and %q for client authentication", secret.Name, SecretKeyClientID, SecretKeyClientSecret)
		}
	default:
		cfg.Auth = AuthPassword
		cfg.Username = string(secret.Data[SecretKeyUsername])
		cfg.Password = string(secret.Data[SecretKeyPassword])
		if cfg.Username == "" || cfg.Password == "" {
			return Config{}, fmt.Errorf("credentials secret %q must provide %q and %q for password authentication", secret.Name, SecretKeyUsername, SecretKeyPassword)
		}
	}
	return cfg, nil
}
