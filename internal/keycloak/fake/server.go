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

// Package fake provides an in-memory Keycloak Admin API server for tests.
package fake

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
)

// Server is a minimal in-memory implementation of the Keycloak Admin API
// endpoints the operator uses. It behaves like the real server for the
// status codes and merge semantics the reconcilers rely on.
type Server struct {
	srv *httptest.Server

	mu      sync.Mutex
	creds   map[string]string // username -> password accepted by the token endpoint
	issued  int
	revoked bool

	realms      map[string]map[string]any
	clients     map[string]map[string]map[string]any // realm -> id -> representation
	scopes      map[string]map[string]map[string]any
	roles       map[string]map[string]map[string]any
	clientRoles map[string]map[string]map[string]any // realm -> clientUUID -> roleName -> rep
	idps        map[string]map[string]map[string]any
	groups      map[string]map[string]map[string]any
	groupRR     map[string]map[string][]map[string]any            // realm -> groupID -> realm roles
	groupCR     map[string]map[string]map[string][]map[string]any // realm -> groupID -> clientUUID -> roles
	defScope    map[string]map[string][]string                    // realm -> clientID -> scope ids
	optScope    map[string]map[string][]string
}

// New starts a fake server accepting the given credentials.
func New(username, password string) *Server {
	s := &Server{
		creds:       map[string]string{username: password},
		realms:      map[string]map[string]any{},
		clients:     map[string]map[string]map[string]any{},
		scopes:      map[string]map[string]map[string]any{},
		roles:       map[string]map[string]map[string]any{},
		clientRoles: map[string]map[string]map[string]any{},
		idps:        map[string]map[string]map[string]any{},
		groups:      map[string]map[string]map[string]any{},
		groupRR:     map[string]map[string][]map[string]any{},
		groupCR:     map[string]map[string]map[string][]map[string]any{},
		defScope:    map[string]map[string][]string{},
		optScope:    map[string]map[string][]string{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/realms/", s.handleToken)
	mux.HandleFunc("/admin/", s.handleAdmin)
	s.srv = httptest.NewServer(mux)
	return s
}

// URL returns the base URL of the fake server.
func (s *Server) URL() string { return s.srv.URL }

// Close shuts the server down.
func (s *Server) Close() { s.srv.Close() }

// RevokeTokens makes the server reject previously issued tokens once, which
// exercises the client retry path.
func (s *Server) RevokeTokens() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked = true
}

// Reset clears all state.
func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.realms = map[string]map[string]any{}
	s.clients = map[string]map[string]map[string]any{}
	s.scopes = map[string]map[string]map[string]any{}
	s.roles = map[string]map[string]map[string]any{}
	s.clientRoles = map[string]map[string]map[string]any{}
	s.idps = map[string]map[string]map[string]any{}
	s.groups = map[string]map[string]map[string]any{}
	s.groupRR = map[string]map[string][]map[string]any{}
	s.groupCR = map[string]map[string]map[string][]map[string]any{}
	s.defScope = map[string]map[string][]string{}
	s.optScope = map[string]map[string][]string{}
}

// SetRealmField mutates a realm field directly, simulating out-of-band
// drift from the Keycloak console.
func (s *Server) SetRealmField(realm, key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rep, ok := s.realms[realm]; ok {
		rep[key] = value
	}
}

// Realm returns a copy of a realm representation.
func (s *Server) Realm(name string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rep, ok := s.realms[name]
	return clone(rep), ok
}

// Client returns a copy of a client representation by clientId.
func (s *Server) Client(realm, clientID string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rep := range s.clients[realm] {
		if rep["clientId"] == clientID {
			return clone(rep), true
		}
	}
	return nil, false
}

// ClientScope returns a copy of a client scope representation by name.
func (s *Server) ClientScope(realm, name string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rep := range s.scopes[realm] {
		if rep["name"] == name {
			return clone(rep), true
		}
	}
	return nil, false
}

// Role returns a copy of a realm role representation by name.
func (s *Server) Role(realm, name string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rep, ok := s.roles[realm][name]
	return clone(rep), ok
}

// RoleComposites returns the names of the roles included in a composite.
func (s *Server) RoleComposites(realm, name string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	role, ok := s.roles[realm][name]
	if !ok {
		return nil
	}
	var out []string
	switch composites := role["composites"].(type) {
	case []map[string]any:
		for _, rep := range composites {
			if n, ok := rep["name"].(string); ok {
				out = append(out, n)
			}
		}
	case []any:
		for _, entry := range composites {
			if rep, ok := entry.(map[string]any); ok {
				if n, ok := rep["name"].(string); ok {
					out = append(out, n)
				}
			}
		}
	}
	return out
}

// IdentityProvider returns a copy of an identity provider by alias.
func (s *Server) IdentityProvider(realm, alias string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rep, ok := s.idps[realm][alias]
	return clone(rep), ok
}

// Group returns a copy of a group representation by name.
func (s *Server) Group(realm, name string) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rep := range s.groups[realm] {
		if rep["name"] == name {
			return clone(rep), true
		}
	}
	return nil, false
}

// GroupRealmRoles returns the names of the realm roles mapped to a group.
func (s *Server) GroupRealmRoles(realm, name string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.groupIDLocked(realm, name)
	var out []string
	for _, rep := range s.groupRR[realm][id] {
		if n, ok := rep["name"].(string); ok {
			out = append(out, n)
		}
	}
	return out
}

// GroupClientRoles returns the names of the roles of one client mapped to a
// group.
func (s *Server) GroupClientRoles(realm, name, clientUUID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.groupIDLocked(realm, name)
	var out []string
	for _, rep := range s.groupCR[realm][id][clientUUID] {
		if n, ok := rep["name"].(string); ok {
			out = append(out, n)
		}
	}
	return out
}

// ClientDefaultScopes returns the names of the default scopes of a client.
func (s *Server) ClientDefaultScopes(realm, clientID string) []string {
	return s.scopeNames(s.defScope[realm][s.clientIDLocked(realm, clientID)], s.scopes[realm])
}

// ClientOptionalScopes returns the names of the optional scopes of a client.
func (s *Server) ClientOptionalScopes(realm, clientID string) []string {
	return s.scopeNames(s.optScope[realm][s.clientIDLocked(realm, clientID)], s.scopes[realm])
}

func (s *Server) scopeNames(ids []string, scopes map[string]map[string]any) []string {
	var out []string
	for _, id := range ids {
		if rep, ok := scopes[id]; ok {
			out = append(out, rep["name"].(string))
		}
	}
	return out
}

func (s *Server) groupIDLocked(realm, name string) string {
	for id, rep := range s.groups[realm] {
		if rep["name"] == name {
			return id
		}
	}
	return ""
}

func (s *Server) clientIDLocked(realm, clientID string) string {
	for id, rep := range s.clients[realm] {
		if rep["clientId"] == clientID {
			return id
		}
	}
	return ""
}

func clone(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	raw, _ := json.Marshal(in)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}

// --- HTTP plumbing ---

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/protocol/openid-connect/token") {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	clientID := r.PostFormValue("client_id")
	clientSecret := r.PostFormValue("client_secret")

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revoked {
		s.revoked = false
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusUnauthorized)
		return
	}
	ok := false
	if username != "" {
		ok = s.creds[username] == password
	} else if clientID != "" {
		ok = s.creds["client:"+clientID] == clientSecret
	}
	if !ok {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusUnauthorized)
		return
	}
	s.issued++
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": fmt.Sprintf("fake-token-%d", s.issued),
		"expires_in":   300,
		"token_type":   "Bearer",
	})
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/admin")
	switch {
	case rest == "/serverinfo":
		s.writeJSON(w, map[string]any{"systemInfo": map[string]any{"version": "26.3.5"}})
		return
	case rest == "/realms" && r.Method == http.MethodPost:
		s.createRealm(w, r)
		return
	case strings.HasPrefix(rest, "/realms/"):
		s.handleRealmScoped(w, r, strings.TrimPrefix(rest, "/realms/"))
		return
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) createRealm(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if !s.decode(w, r, &payload) {
		return
	}
	name, _ := payload["realm"].(string)
	if name == "" {
		http.Error(w, "realm name required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.realms[name]; exists {
		http.Error(w, "realm exists", http.StatusConflict)
		return
	}
	payload["id"] = name
	s.realms[name] = payload
	w.Header().Set("Location", "/admin/realms/"+url.PathEscape(name))
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleRealmScoped(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.Split(rest, "/")
	realm, err := url.PathUnescape(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sub := parts[1:]

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(sub) == 0 {
		s.handleRealm(w, r, realm)
		return
	}
	switch sub[0] {
	case "clients":
		s.handleClients(w, r, realm, sub[1:])
	case "client-scopes":
		s.handleClientScopes(w, r, realm, sub[1:])
	case "roles":
		s.handleRoles(w, r, realm, sub[1:])
	case "identity-provider":
		if len(sub) >= 2 && sub[1] == "instances" {
			s.handleIdentityProviders(w, r, realm, sub[2:])
		} else {
			http.NotFound(w, r)
		}
	case "groups":
		s.handleGroups(w, r, realm, sub[1:])
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleRealm(w http.ResponseWriter, r *http.Request, realm string) {
	switch r.Method {
	case http.MethodGet:
		rep, ok := s.realms[realm]
		if !ok {
			http.NotFound(w, r)
			return
		}
		s.writeJSON(w, rep)
	case http.MethodPut:
		var payload map[string]any
		if !s.decode(w, r, &payload) {
			return
		}
		rep, ok := s.realms[realm]
		if !ok {
			http.NotFound(w, r)
			return
		}
		merge(rep, payload)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if _, ok := s.realms[realm]; !ok {
			http.NotFound(w, r)
			return
		}
		delete(s.realms, realm)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleClients(w http.ResponseWriter, r *http.Request, realm string, sub []string) {
	if _, ok := s.realms[realm]; !ok {
		http.NotFound(w, r)
		return
	}
	if len(sub) == 0 {
		switch r.Method {
		case http.MethodGet:
			clientID := r.URL.Query().Get("clientId")
			var out []map[string]any
			for id, rep := range s.clients[realm] {
				if clientID == "" || rep["clientId"] == clientID {
					rep["id"] = id
					out = append(out, rep)
				}
			}
			s.writeJSON(w, out)
		case http.MethodPost:
			var payload map[string]any
			if !s.decode(w, r, &payload) {
				return
			}
			clientID, _ := payload["clientId"].(string)
			if clientID == "" {
				http.Error(w, "clientId required", http.StatusBadRequest)
				return
			}
			for _, rep := range s.clients[realm] {
				if rep["clientId"] == clientID {
					http.Error(w, "client exists", http.StatusConflict)
					return
				}
			}
			id := "client-" + clientID
			payload["id"] = id
			if s.clients[realm] == nil {
				s.clients[realm] = map[string]map[string]any{}
			}
			s.clients[realm][id] = payload
			w.Header().Set("Location", "/admin/realms/"+realm+"/clients/"+id)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	id := sub[0]
	if len(sub) == 3 && sub[1] == "roles" {
		// A single client role.
		roleName, err := url.PathUnescape(sub[2])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if rep, ok := s.clientRoles[realm][id][roleName]; ok {
			s.writeJSON(w, rep)
			return
		}
		http.NotFound(w, r)
		return
	}
	if len(sub) == 2 && sub[1] == "roles" {
		// Client-level role collection (used by tests to seed client roles).
		switch r.Method {
		case http.MethodPost:
			var payload map[string]any
			if !s.decode(w, r, &payload) {
				return
			}
			name, _ := payload["name"].(string)
			if s.clientRoles[realm] == nil {
				s.clientRoles[realm] = map[string]map[string]any{}
			}
			if s.clientRoles[realm][id] == nil {
				s.clientRoles[realm][id] = map[string]any{}
			}
			payload["id"] = "clientrole-" + id + "-" + name
			s.clientRoles[realm][id][name] = payload
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(sub) == 2 && sub[1] == "client-secret" {
		rep, ok := s.clients[realm][id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			secret, _ := rep["secret"].(string)
			s.writeJSON(w, map[string]any{"type": "secret", "value": secret})
		case http.MethodPost:
			secret := "regenerated-" + id
			rep["secret"] = secret
			s.writeJSON(w, map[string]any{"type": "secret", "value": secret})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(sub) >= 2 && (sub[1] == "default-client-scopes" || sub[1] == "optional-client-scopes") {
		kind := sub[1]
		store := s.defScope
		if kind == "optional-client-scopes" {
			store = s.optScope
		}
		switch r.Method {
		case http.MethodGet:
			var out []map[string]any
			for _, scopeID := range store[realm][id] {
				if rep, ok := s.scopes[realm][scopeID]; ok {
					out = append(out, rep)
				}
			}
			s.writeJSON(w, out)
		case http.MethodPut:
			scopeID := sub[2]
			if _, ok := s.scopes[realm][scopeID]; !ok {
				http.NotFound(w, r)
				return
			}
			if store[realm] == nil {
				store[realm] = map[string][]string{}
			}
			store[realm][id] = appendUnique(store[realm][id], scopeID)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			if len(sub) < 3 {
				http.NotFound(w, r)
				return
			}
			store[realm][id] = removeValue(store[realm][id], sub[2])
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	rep, ok := s.clients[realm][id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, rep)
	case http.MethodPut:
		var payload map[string]any
		if !s.decode(w, r, &payload) {
			return
		}
		merge(rep, payload)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		delete(s.clients[realm], id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleClientScopes(w http.ResponseWriter, r *http.Request, realm string, sub []string) {
	if _, ok := s.realms[realm]; !ok {
		http.NotFound(w, r)
		return
	}
	if len(sub) == 0 {
		switch r.Method {
		case http.MethodGet:
			var out []map[string]any
			for id, rep := range s.scopes[realm] {
				rep["id"] = id
				out = append(out, rep)
			}
			s.writeJSON(w, out)
		case http.MethodPost:
			var payload map[string]any
			if !s.decode(w, r, &payload) {
				return
			}
			name, _ := payload["name"].(string)
			for _, rep := range s.scopes[realm] {
				if rep["name"] == name {
					http.Error(w, "scope exists", http.StatusConflict)
					return
				}
			}
			id := "scope-" + name
			payload["id"] = id
			if s.scopes[realm] == nil {
				s.scopes[realm] = map[string]map[string]any{}
			}
			s.scopes[realm][id] = payload
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	id := sub[0]
	rep, ok := s.scopes[realm][id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, rep)
	case http.MethodPut:
		var payload map[string]any
		if !s.decode(w, r, &payload) {
			return
		}
		merge(rep, payload)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		delete(s.scopes[realm], id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request, realm string, sub []string) {
	if _, ok := s.realms[realm]; !ok {
		http.NotFound(w, r)
		return
	}
	if len(sub) == 0 {
		if r.Method == http.MethodPost {
			var payload map[string]any
			if !s.decode(w, r, &payload) {
				return
			}
			name, _ := payload["name"].(string)
			if _, exists := s.roles[realm][name]; exists {
				http.Error(w, "role exists", http.StatusConflict)
				return
			}
			if s.roles[realm] == nil {
				s.roles[realm] = map[string]map[string]any{}
			}
			payload["id"] = "role-" + name
			s.roles[realm][name] = payload
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name := sub[0]
	if len(sub) == 2 && sub[1] == "composites" {
		role, ok := s.roles[realm][name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			composites, _ := role["composites"].([]map[string]any)
			s.writeJSON(w, composites)
		case http.MethodPost, http.MethodDelete:
			var payload []map[string]any
			if !s.decode(w, r, &payload) {
				return
			}
			current, _ := role["composites"].([]map[string]any)
			if r.Method == http.MethodPost {
				role["composite"] = true
				role["composites"] = appendUniqueReps(current, payload)
			} else {
				role["composites"] = removeReps(current, payload)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	role, ok := s.roles[realm][name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, role)
	case http.MethodPut:
		var payload map[string]any
		if !s.decode(w, r, &payload) {
			return
		}
		merge(role, payload)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		delete(s.roles[realm], name)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleIdentityProviders(w http.ResponseWriter, r *http.Request, realm string, sub []string) {
	if _, ok := s.realms[realm]; !ok {
		http.NotFound(w, r)
		return
	}
	if len(sub) == 0 {
		if r.Method == http.MethodPost {
			var payload map[string]any
			if !s.decode(w, r, &payload) {
				return
			}
			alias, _ := payload["alias"].(string)
			if _, exists := s.idps[realm][alias]; exists {
				http.Error(w, "idp exists", http.StatusConflict)
				return
			}
			if s.idps[realm] == nil {
				s.idps[realm] = map[string]map[string]any{}
			}
			s.idps[realm][alias] = payload
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	alias := sub[0]
	rep, ok := s.idps[realm][alias]
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, rep)
	case http.MethodPut:
		var payload map[string]any
		if !s.decode(w, r, &payload) {
			return
		}
		merge(rep, payload)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		delete(s.idps[realm], alias)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request, realm string, sub []string) {
	if _, ok := s.realms[realm]; !ok {
		http.NotFound(w, r)
		return
	}
	if len(sub) == 0 {
		switch r.Method {
		case http.MethodGet:
			search := r.URL.Query().Get("search")
			var out []map[string]any
			for id, rep := range s.groups[realm] {
				if search == "" || rep["name"] == search {
					rep["id"] = id
					out = append(out, rep)
				}
			}
			s.writeJSON(w, out)
		case http.MethodPost:
			var payload map[string]any
			if !s.decode(w, r, &payload) {
				return
			}
			name, _ := payload["name"].(string)
			for _, rep := range s.groups[realm] {
				if rep["name"] == name {
					http.Error(w, "group exists", http.StatusConflict)
					return
				}
			}
			id := "group-" + name
			payload["id"] = id
			if s.groups[realm] == nil {
				s.groups[realm] = map[string]map[string]any{}
			}
			s.groups[realm][id] = payload
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	id := sub[0]
	if len(sub) >= 3 && sub[1] == "role-mappings" && (sub[2] == "realm" || sub[2] == "clients") {
		group, ok := s.groups[realm][id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_ = group
		var store map[string][]map[string]any
		key := ""
		if sub[2] == "realm" {
			if s.groupRR[realm] == nil {
				s.groupRR[realm] = map[string][]map[string]any{}
			}
			store = s.groupRR[realm]
		} else {
			if s.groupCR[realm] == nil {
				s.groupCR[realm] = map[string]map[string][]map[string]any{}
			}
			if s.groupCR[realm][id] == nil {
				s.groupCR[realm][id] = map[string][]map[string]any{}
			}
			key = sub[3]
		}
		switch r.Method {
		case http.MethodGet:
			var current []map[string]any
			if sub[2] == "realm" {
				current = store[id]
			} else {
				current = s.groupCR[realm][id][key]
			}
			s.writeJSON(w, current)
		case http.MethodPost, http.MethodDelete:
			var payload []map[string]any
			if !s.decode(w, r, &payload) {
				return
			}
			if sub[2] == "realm" {
				if r.Method == http.MethodPost {
					store[id] = appendUniqueReps(store[id], payload)
				} else {
					store[id] = removeReps(store[id], payload)
				}
			} else {
				current := s.groupCR[realm][id][key]
				if r.Method == http.MethodPost {
					s.groupCR[realm][id][key] = appendUniqueReps(current, payload)
				} else {
					s.groupCR[realm][id][key] = removeReps(current, payload)
				}
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	group, ok := s.groups[realm][id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, group)
	case http.MethodPut:
		var payload map[string]any
		if !s.decode(w, r, &payload) {
			return
		}
		merge(group, payload)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		delete(s.groups[realm], id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) decode(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close() //nolint:errcheck // read-only body
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func (s *Server) writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	if body == nil {
		_, _ = w.Write([]byte("null"))
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func merge(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

func appendUnique(list []string, value string) []string {
	for _, v := range list {
		if v == value {
			return list
		}
	}
	return append(list, value)
}

func removeValue(list []string, value string) []string {
	var out []string
	for _, v := range list {
		if v != value {
			out = append(out, v)
		}
	}
	return out
}

func appendUniqueReps(current, add []map[string]any) []map[string]any {
	out := current
	for _, rep := range add {
		found := false
		for _, existing := range current {
			if existing["id"] == rep["id"] {
				found = true
				break
			}
		}
		if !found {
			out = append(out, rep)
		}
	}
	return out
}

func removeReps(current, remove []map[string]any) []map[string]any {
	var out []map[string]any
	for _, existing := range current {
		drop := false
		for _, rep := range remove {
			if existing["id"] == rep["id"] {
				drop = true
				break
			}
		}
		if !drop {
			out = append(out, existing)
		}
	}
	return out
}
