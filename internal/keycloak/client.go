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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ctn-solutions/keycloak-operator/internal/metrics"
)

// AuthType selects the grant used to obtain administration tokens.
type AuthType string

const (
	// AuthPassword uses the password grant against the admin-cli client.
	AuthPassword AuthType = "password"
	// AuthClient uses the client credentials grant with a service account.
	AuthClient AuthType = "client"
)

// DefaultAdminRealm is used when a connection does not specify one.
const DefaultAdminRealm = "master"

// DefaultAdminClientID is the client used for the password grant when the
// connection does not specify one.
const DefaultAdminClientID = "admin-cli"

// Config describes how to reach and authenticate against a Keycloak server.
type Config struct {
	// ConnectionName labels the metrics emitted for this client. It is the
	// KeycloakConnection resource name and namespace, "ns/name".
	ConnectionName string
	// URL is the server base URL, without a trailing slash.
	URL string
	// AdminRealm holds the administration credentials. Defaults to master.
	AdminRealm string
	// Auth selects the grant. Defaults to AuthPassword.
	Auth AuthType
	// Username and Password are used by AuthPassword.
	Username, Password string
	// ClientID and ClientSecret are used by AuthClient, and ClientID alone
	// overrides the admin-cli default for AuthPassword.
	ClientID, ClientSecret string
	// InsecureSkipVerify disables TLS verification.
	InsecureSkipVerify bool
}

// Client talks to one Keycloak server. It is safe for concurrent use and
// refreshes its access token transparently.
type Client struct {
	cfg Config
	hc  *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpri  time.Time
}

// New builds a client for the given configuration.
func New(cfg Config) *Client {
	if cfg.AdminRealm == "" {
		cfg.AdminRealm = DefaultAdminRealm
	}
	if cfg.Auth == "" {
		cfg.Auth = AuthPassword
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicit opt-in for self-signed servers
	}
	return &Client{
		cfg: cfg,
		hc:  &http.Client{Timeout: 30 * time.Second, Transport: transport},
	}
}

// Token returns a valid access token, refreshing it when it is about to
// expire.
func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accessToken != "" && time.Until(c.tokenExpri) > 30*time.Second {
		return c.accessToken, nil
	}

	form := url.Values{}
	switch c.cfg.Auth {
	case AuthClient:
		form.Set("grant_type", "client_credentials")
		form.Set("client_id", c.cfg.ClientID)
		form.Set("client_secret", c.cfg.ClientSecret)
	default:
		form.Set("grant_type", "password")
		form.Set("client_id", c.cfg.ClientID)
		if form.Get("client_id") == "" {
			form.Set("client_id", DefaultAdminClientID)
		}
		form.Set("username", c.cfg.Username)
		form.Set("password", c.cfg.Password)
	}

	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.cfg.URL, url.PathEscape(c.cfg.AdminRealm))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("%w: %s", ErrAuth, truncate(body))
	}
	if resp.StatusCode != http.StatusOK {
		return "", &APIError{StatusCode: resp.StatusCode, Method: http.MethodPost, Path: "/token", Body: string(body)}
	}

	var parsed struct {
		AccessToken      string `json:"access_token"`
		ExpiresIn        int64  `json:"expires_in"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if parsed.AccessToken == "" {
		if parsed.Error != "" {
			return "", fmt.Errorf("%w: %s: %s", ErrAuth, parsed.Error, parsed.ErrorDescription)
		}
		return "", fmt.Errorf("%w: empty access token", ErrAuth)
	}

	c.accessToken = parsed.AccessToken
	c.tokenExpri = time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	return c.accessToken, nil
}

// invalidate drops the cached token after an authentication failure.
func (c *Client) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = ""
	c.tokenExpri = time.Time{}
}

// Do performs an Admin API request. body may be nil; out may be nil. It maps
// well-known status codes onto the sentinel errors and retries once after a
// token refresh when the server rejects the credentials.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	start := time.Now()
	err := c.doOnce(ctx, method, path, body, out)
	if !errors.Is(err, ErrAuth) {
		c.observeRequest(method, start, err)
		return err
	}
	// The cached token may have been revoked server-side: refresh once.
	c.invalidate()
	err = c.doOnce(ctx, method, path, body, out)
	c.observeRequest(method, start, err)
	return err
}

// observeRequest records one Admin API request in the metrics. The status
// class is derived from the error mapping: sentinel errors carry the class,
// anything else counts as a client-side failure (5xx-class).
func (c *Client) observeRequest(method string, start time.Time, err error) {
	if c.cfg.ConnectionName == "" {
		return
	}
	code := "2xx"
	switch {
	case errors.Is(err, ErrNotFound) || errors.Is(err, ErrConflict):
		code = "4xx"
	case errors.Is(err, ErrAuth):
		code = "4xx"
	case err != nil:
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			code = "4xx"
			if apiErr.StatusCode >= 500 {
				code = "5xx"
			}
		} else {
			code = "error"
		}
	}
	metrics.AdminRequestsTotal.WithLabelValues(c.cfg.ConnectionName, method, code).Inc()
	metrics.AdminRequestDuration.WithLabelValues(c.cfg.ConnectionName, method).Observe(time.Since(start).Seconds())
}

func (c *Client) doOnce(ctx context.Context, method, path string, body, out any) error {
	token, err := c.Token(ctx)
	if err != nil {
		return err
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	endpoint := c.cfg.URL + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if out != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decode response of %s %s: %w", method, path, err)
			}
		}
		return nil
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: %s %s", ErrNotFound, method, path)
	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("%w: %s %s: %s", ErrConflict, method, path, truncate(respBody))
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: %s %s: %s", ErrAuth, method, path, truncate(respBody))
	default:
		return &APIError{StatusCode: resp.StatusCode, Method: method, Path: path, Body: string(respBody)}
	}
}

func truncate(b []byte) string {
	const max = 512
	if len(b) > max {
		return string(b[:max]) + "..."
	}
	return string(b)
}
