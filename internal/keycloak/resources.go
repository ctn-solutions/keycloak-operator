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
	"net/url"
)

// realmPath builds the base path of a realm-scoped resource collection.
func realmPath(realm string) string {
	return "/admin/realms/" + url.PathEscape(realm)
}

// --- Server ---

// ServerInfo returns the server information document, including its version.
func (c *Client) ServerInfo(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.Do(ctx, "GET", "/admin/serverinfo", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Realms ---

// GetRealm fetches a realm representation.
func (c *Client) GetRealm(ctx context.Context, realm string) (map[string]any, error) {
	var out map[string]any
	if err := c.Do(ctx, "GET", realmPath(realm), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateRealm creates a realm from a representation payload.
func (c *Client) CreateRealm(ctx context.Context, payload map[string]any) error {
	return c.Do(ctx, "POST", "/admin/realms", payload, nil)
}

// UpdateRealm applies a representation payload to an existing realm.
func (c *Client) UpdateRealm(ctx context.Context, realm string, payload map[string]any) error {
	return c.Do(ctx, "PUT", realmPath(realm), payload, nil)
}

// DeleteRealm removes a realm.
func (c *Client) DeleteRealm(ctx context.Context, realm string) error {
	return c.Do(ctx, "DELETE", realmPath(realm), nil, nil)
}

// --- Clients ---

// FindClient resolves a client by its clientId. Returns ErrNotFound when no
// client matches.
func (c *Client) FindClient(ctx context.Context, realm, clientID string) (map[string]any, error) {
	path := fmt.Sprintf("%s/clients?clientId=%s", realmPath(realm), url.QueryEscape(clientID))
	var out []map[string]any
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	for _, rep := range out {
		if rep["clientId"] == clientID {
			return rep, nil
		}
	}
	return nil, fmt.Errorf("%w: client %q", ErrNotFound, clientID)
}

// CreateClient creates a client from a representation payload.
func (c *Client) CreateClient(ctx context.Context, realm string, payload map[string]any) error {
	return c.Do(ctx, "POST", realmPath(realm)+"/clients", payload, nil)
}

// UpdateClient applies a representation payload to an existing client.
func (c *Client) UpdateClient(ctx context.Context, realm, id string, payload map[string]any) error {
	return c.Do(ctx, "PUT", fmt.Sprintf("%s/clients/%s", realmPath(realm), url.PathEscape(id)), payload, nil)
}

// DeleteClient removes a client.
func (c *Client) DeleteClient(ctx context.Context, realm, id string) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("%s/clients/%s", realmPath(realm), url.PathEscape(id)), nil, nil)
}

// GetClientSecret returns the current secret of a client.
func (c *Client) GetClientSecret(ctx context.Context, realm, id string) (string, error) {
	path := fmt.Sprintf("%s/clients/%s/client-secret", realmPath(realm), url.PathEscape(id))
	var out struct {
		Value string `json:"value"`
	}
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return "", err
	}
	return out.Value, nil
}

// --- Client scopes ---

// ListClientScopes returns all client scopes of a realm.
func (c *Client) ListClientScopes(ctx context.Context, realm string) ([]map[string]any, error) {
	var out []map[string]any
	if err := c.Do(ctx, "GET", realmPath(realm)+"/client-scopes", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FindClientScope resolves a client scope by name.
func (c *Client) FindClientScope(ctx context.Context, realm, name string) (map[string]any, error) {
	scopes, err := c.ListClientScopes(ctx, realm)
	if err != nil {
		return nil, err
	}
	for _, scope := range scopes {
		if scope["name"] == name {
			return scope, nil
		}
	}
	return nil, fmt.Errorf("%w: client scope %q", ErrNotFound, name)
}

// CreateClientScope creates a client scope from a representation payload.
func (c *Client) CreateClientScope(ctx context.Context, realm string, payload map[string]any) error {
	return c.Do(ctx, "POST", realmPath(realm)+"/client-scopes", payload, nil)
}

// UpdateClientScope applies a representation payload to an existing client
// scope.
func (c *Client) UpdateClientScope(ctx context.Context, realm, id string, payload map[string]any) error {
	return c.Do(ctx, "PUT", fmt.Sprintf("%s/client-scopes/%s", realmPath(realm), url.PathEscape(id)), payload, nil)
}

// DeleteClientScope removes a client scope.
func (c *Client) DeleteClientScope(ctx context.Context, realm, id string) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("%s/client-scopes/%s", realmPath(realm), url.PathEscape(id)), nil, nil)
}

// --- Client scope assignments ---

// listClientScopeAssignments returns the scopes currently assigned to a
// client under "default-client-scopes" or "optional-client-scopes".
func (c *Client) listClientScopeAssignments(ctx context.Context, realm, clientID, kind string) ([]map[string]any, error) {
	path := fmt.Sprintf("%s/clients/%s/%s", realmPath(realm), url.PathEscape(clientID), kind)
	var out []map[string]any
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) assignClientScope(ctx context.Context, realm, clientID, kind, scopeID string, add bool) error {
	method := "DELETE"
	if add {
		method = "PUT"
	}
	path := fmt.Sprintf("%s/clients/%s/%s/%s", realmPath(realm), url.PathEscape(clientID), kind, url.PathEscape(scopeID))
	return c.Do(ctx, method, path, nil, nil)
}

// ClientDefaultScopeAssignments manages the default client scopes of a
// client.
func (c *Client) ClientDefaultScopeAssignments(ctx context.Context, realm, clientID string) ([]map[string]any, error) {
	return c.listClientScopeAssignments(ctx, realm, clientID, "default-client-scopes")
}

// AssignClientDefaultScope adds or removes a default client scope.
func (c *Client) AssignClientDefaultScope(ctx context.Context, realm, clientID, scopeID string, add bool) error {
	return c.assignClientScope(ctx, realm, clientID, "default-client-scopes", scopeID, add)
}

// ClientOptionalScopeAssignments manages the optional client scopes of a
// client.
func (c *Client) ClientOptionalScopeAssignments(ctx context.Context, realm, clientID string) ([]map[string]any, error) {
	return c.listClientScopeAssignments(ctx, realm, clientID, "optional-client-scopes")
}

// AssignClientOptionalScope adds or removes an optional client scope.
func (c *Client) AssignClientOptionalScope(ctx context.Context, realm, clientID, scopeID string, add bool) error {
	return c.assignClientScope(ctx, realm, clientID, "optional-client-scopes", scopeID, add)
}

// --- Realm roles ---

// GetRole fetches a realm role by name.
func (c *Client) GetRole(ctx context.Context, realm, name string) (map[string]any, error) {
	var out map[string]any
	if err := c.Do(ctx, "GET", fmt.Sprintf("%s/roles/%s", realmPath(realm), url.PathEscape(name)), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateRole creates a realm role from a representation payload.
func (c *Client) CreateRole(ctx context.Context, realm string, payload map[string]any) error {
	return c.Do(ctx, "POST", realmPath(realm)+"/roles", payload, nil)
}

// UpdateRole applies a representation payload to an existing realm role.
func (c *Client) UpdateRole(ctx context.Context, realm, name string, payload map[string]any) error {
	return c.Do(ctx, "PUT", fmt.Sprintf("%s/roles/%s", realmPath(realm), url.PathEscape(name)), payload, nil)
}

// DeleteRole removes a realm role.
func (c *Client) DeleteRole(ctx context.Context, realm, name string) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("%s/roles/%s", realmPath(realm), url.PathEscape(name)), nil, nil)
}

// GetRoleComposites returns the roles included in a composite role.
func (c *Client) GetRoleComposites(ctx context.Context, realm, name string) ([]map[string]any, error) {
	var out []map[string]any
	if err := c.Do(ctx, "GET", fmt.Sprintf("%s/roles/%s/composites", realmPath(realm), url.PathEscape(name)), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddRoleComposites includes roles in a composite role.
func (c *Client) AddRoleComposites(ctx context.Context, realm, name string, roles []map[string]any) error {
	return c.Do(ctx, "POST", fmt.Sprintf("%s/roles/%s/composites", realmPath(realm), url.PathEscape(name)), roles, nil)
}

// RemoveRoleComposites excludes roles from a composite role.
func (c *Client) RemoveRoleComposites(ctx context.Context, realm, name string, roles []map[string]any) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("%s/roles/%s/composites", realmPath(realm), url.PathEscape(name)), roles, nil)
}

// GetClientRole fetches a role of a specific client by name.
func (c *Client) GetClientRole(ctx context.Context, realm, clientUUID, name string) (map[string]any, error) {
	path := fmt.Sprintf("%s/clients/%s/roles/%s", realmPath(realm), url.PathEscape(clientUUID), url.PathEscape(name))
	var out map[string]any
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Identity providers ---

// GetIdentityProvider fetches an identity provider by alias.
func (c *Client) GetIdentityProvider(ctx context.Context, realm, alias string) (map[string]any, error) {
	var out map[string]any
	if err := c.Do(ctx, "GET", fmt.Sprintf("%s/identity-provider/instances/%s", realmPath(realm), url.PathEscape(alias)), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateIdentityProvider creates an identity provider from a representation
// payload.
func (c *Client) CreateIdentityProvider(ctx context.Context, realm string, payload map[string]any) error {
	return c.Do(ctx, "POST", realmPath(realm)+"/identity-provider/instances", payload, nil)
}

// UpdateIdentityProvider applies a representation payload to an existing
// identity provider.
func (c *Client) UpdateIdentityProvider(ctx context.Context, realm, alias string, payload map[string]any) error {
	return c.Do(ctx, "PUT", fmt.Sprintf("%s/identity-provider/instances/%s", realmPath(realm), url.PathEscape(alias)), payload, nil)
}

// ListIdentityProviderMappers returns the mappers of an identity provider.
func (c *Client) ListIdentityProviderMappers(ctx context.Context, realm, alias string) ([]map[string]any, error) {
	path := fmt.Sprintf("%s/identity-provider/instances/%s/mappers", realmPath(realm), url.PathEscape(alias))
	var out []map[string]any
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateIdentityProviderMapper adds a mapper to an identity provider.
func (c *Client) CreateIdentityProviderMapper(ctx context.Context, realm, alias string, payload map[string]any) error {
	path := fmt.Sprintf("%s/identity-provider/instances/%s/mappers", realmPath(realm), url.PathEscape(alias))
	return c.Do(ctx, "POST", path, payload, nil)
}

// UpdateIdentityProviderMapper applies a mapper payload.
func (c *Client) UpdateIdentityProviderMapper(ctx context.Context, realm, alias, mapperID string, payload map[string]any) error {
	path := fmt.Sprintf("%s/identity-provider/instances/%s/mappers/%s", realmPath(realm), url.PathEscape(alias), url.PathEscape(mapperID))
	return c.Do(ctx, "PUT", path, payload, nil)
}

// DeleteIdentityProviderMapper removes a mapper.
func (c *Client) DeleteIdentityProviderMapper(ctx context.Context, realm, alias, mapperID string) error {
	path := fmt.Sprintf("%s/identity-provider/instances/%s/mappers/%s", realmPath(realm), url.PathEscape(alias), url.PathEscape(mapperID))
	return c.Do(ctx, "DELETE", path, nil, nil)
}

// DeleteIdentityProvider removes an identity provider.
func (c *Client) DeleteIdentityProvider(ctx context.Context, realm, alias string) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("%s/identity-provider/instances/%s", realmPath(realm), url.PathEscape(alias)), nil, nil)
}

// --- Groups ---

// FindGroup resolves a top-level group by name. Returns ErrNotFound when no
// group matches.
func (c *Client) FindGroup(ctx context.Context, realm, name string) (map[string]any, error) {
	path := fmt.Sprintf("%s/groups?search=%s&exact=true&max=2", realmPath(realm), url.QueryEscape(name))
	var out []map[string]any
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	for _, rep := range out {
		if rep["name"] == name {
			return rep, nil
		}
	}
	return nil, fmt.Errorf("%w: group %q", ErrNotFound, name)
}

// CreateGroup creates a group from a representation payload.
func (c *Client) CreateGroup(ctx context.Context, realm string, payload map[string]any) error {
	return c.Do(ctx, "POST", realmPath(realm)+"/groups", payload, nil)
}

// UpdateGroup applies a representation payload to an existing group.
func (c *Client) UpdateGroup(ctx context.Context, realm, id string, payload map[string]any) error {
	return c.Do(ctx, "PUT", fmt.Sprintf("%s/groups/%s", realmPath(realm), url.PathEscape(id)), payload, nil)
}

// DeleteGroup removes a group.
func (c *Client) DeleteGroup(ctx context.Context, realm, id string) error {
	return c.Do(ctx, "DELETE", fmt.Sprintf("%s/groups/%s", realmPath(realm), url.PathEscape(id)), nil, nil)
}

// --- Group role mappings ---

// GetGroupRoleMappings returns the complete role-mapping document of a
// group, including realmMappings and per-client clientMappings.
func (c *Client) GetGroupRoleMappings(ctx context.Context, realm, groupID string) (map[string]any, error) {
	path := fmt.Sprintf("%s/groups/%s/role-mappings", realmPath(realm), url.PathEscape(groupID))
	var out map[string]any
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetGroupRealmRoles returns the realm roles mapped to a group.
func (c *Client) GetGroupRealmRoles(ctx context.Context, realm, groupID string) ([]map[string]any, error) {
	path := fmt.Sprintf("%s/groups/%s/role-mappings/realm", realmPath(realm), url.PathEscape(groupID))
	var out []map[string]any
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddGroupRealmRoles maps realm roles onto a group.
func (c *Client) AddGroupRealmRoles(ctx context.Context, realm, groupID string, roles []map[string]any) error {
	path := fmt.Sprintf("%s/groups/%s/role-mappings/realm", realmPath(realm), url.PathEscape(groupID))
	return c.Do(ctx, "POST", path, roles, nil)
}

// RemoveGroupRealmRoles unmaps realm roles from a group.
func (c *Client) RemoveGroupRealmRoles(ctx context.Context, realm, groupID string, roles []map[string]any) error {
	path := fmt.Sprintf("%s/groups/%s/role-mappings/realm", realmPath(realm), url.PathEscape(groupID))
	return c.Do(ctx, "DELETE", path, roles, nil)
}

// GetGroupClientRoles returns the roles of one client mapped to a group.
func (c *Client) GetGroupClientRoles(ctx context.Context, realm, groupID, clientUUID string) ([]map[string]any, error) {
	path := fmt.Sprintf("%s/groups/%s/role-mappings/clients/%s", realmPath(realm), url.PathEscape(groupID), url.PathEscape(clientUUID))
	var out []map[string]any
	if err := c.Do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddGroupClientRoles maps roles of one client onto a group.
func (c *Client) AddGroupClientRoles(ctx context.Context, realm, groupID, clientUUID string, roles []map[string]any) error {
	path := fmt.Sprintf("%s/groups/%s/role-mappings/clients/%s", realmPath(realm), url.PathEscape(groupID), url.PathEscape(clientUUID))
	return c.Do(ctx, "POST", path, roles, nil)
}

// RemoveGroupClientRoles unmaps roles of one client from a group.
func (c *Client) RemoveGroupClientRoles(ctx context.Context, realm, groupID, clientUUID string, roles []map[string]any) error {
	path := fmt.Sprintf("%s/groups/%s/role-mappings/clients/%s", realmPath(realm), url.PathEscape(groupID), url.PathEscape(clientUUID))
	return c.Do(ctx, "DELETE", path, roles, nil)
}
