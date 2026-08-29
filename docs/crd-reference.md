# API Reference

## Packages
- [keycloak.ctn-solutions.io/v1alpha1](#keycloakctn-solutionsiov1alpha1)


## keycloak.ctn-solutions.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the keycloak v1alpha1 API group.

### Resource Types
- [Client](#client)
- [ClientScope](#clientscope)
- [Group](#group)
- [IdentityProvider](#identityprovider)
- [KeycloakConnection](#keycloakconnection)
- [Realm](#realm)
- [RealmRole](#realmrole)



#### AdoptionPolicy

_Underlying type:_ _string_

AdoptionPolicy controls how the operator treats a resource that already
exists on the Keycloak server.

_Validation:_
- Enum: [CreateOnly Adopt FailIfExists]

_Appears in:_
- [ClientScopeSpec](#clientscopespec)
- [ClientSpec](#clientspec)
- [GroupSpec](#groupspec)
- [IdentityProviderSpec](#identityproviderspec)
- [RealmRoleSpec](#realmrolespec)
- [RealmSpec](#realmspec)

| Field | Description |
| --- | --- |
| `CreateOnly` | AdoptionCreateOnly creates the resource if absent and fails if a foreign<br />(unmanaged) resource with the same key exists. Resources previously<br />created or adopted by the operator are resumed.<br /> |
| `Adopt` | AdoptionAdopt takes over an existing resource: the managed marker is<br />stamped and the spec is enforced from then on.<br /> |
| `FailIfExists` | AdoptionFailIfExists fails whenever a resource with the same key exists,<br />even one previously managed by the operator.<br /> |


#### AuthType

_Underlying type:_ _string_

AuthType selects how the operator authenticates against the Keycloak
administration interface.

_Validation:_
- Enum: [password client]

_Appears in:_
- [KeycloakConnectionSpec](#keycloakconnectionspec)

| Field | Description |
| --- | --- |
| `password` | AuthPassword authenticates with a username/password pair against the<br />built-in admin-cli client (password grant).<br /> |
| `client` | AuthClient authenticates with a service-account client (client<br />credentials grant).<br /> |


#### Client



Client manages a client on a Keycloak server. The spec mirrors the Keycloak
ClientRepresentation; see the Keycloak server documentation for field
semantics.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `keycloak.ctn-solutions.io/v1alpha1` | | |
| `kind` _string_ | `Client` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ClientSpec](#clientspec)_ |  |  |  |


#### ClientScope



ClientScope manages a client scope on a Keycloak server. The spec mirrors
the Keycloak ClientScopeRepresentation; see the Keycloak server
documentation for field semantics.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `keycloak.ctn-solutions.io/v1alpha1` | | |
| `kind` _string_ | `ClientScope` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ClientScopeSpec](#clientscopespec)_ |  |  |  |


#### ClientScopeSpec



ClientScopeSpec defines the desired state of a ClientScope. Fields mirror
the Keycloak ClientScopeRepresentation one-to-one.



_Appears in:_
- [ClientScope](#clientscope)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `keycloakRef` _[KeycloakRef](#keycloakref)_ | KeycloakRef points to the KeycloakConnection managing this client<br />scope. |  | Required: \{\} <br /> |
| `realm` _string_ | Realm is the name of the realm the client scope lives in. |  | MinLength: 1 <br /> |
| `name` _string_ | Name is the client scope name on the Keycloak server. |  | MinLength: 1 <br /> |
| `adoptionPolicy` _[AdoptionPolicy](#adoptionpolicy)_ | AdoptionPolicy controls the behaviour when a client scope with the same<br />name already exists. Defaults to CreateOnly. |  | Enum: [CreateOnly Adopt FailIfExists] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the client scope is deleted from the<br />server when this resource is deleted. Defaults to Delete. |  | Enum: [Orphan Delete] <br />Optional: \{\} <br /> |
| `description` _string_ |  |  | Optional: \{\} <br /> |
| `protocol` _string_ | Protocol is "openid-connect" or "saml". |  | Enum: [openid-connect saml] <br />Optional: \{\} <br /> |
| `attributes` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `protocolMappers` _[ProtocolMapper](#protocolmapper) array_ |  |  | Optional: \{\} <br /> |
| `displayOnConsentScreen` _boolean_ |  |  | Optional: \{\} <br /> |
| `consentScreenText` _string_ |  |  | Optional: \{\} <br /> |


#### ClientSpec



ClientSpec defines the desired state of a Client. Fields mirror the
Keycloak ClientRepresentation one-to-one. The client secret is never part
of the spec: it is either injected from a Secret (secretRef) or exported to
one (secretOutput).



_Appears in:_
- [Client](#client)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `keycloakRef` _[KeycloakRef](#keycloakref)_ | KeycloakRef points to the KeycloakConnection managing this client. |  | Required: \{\} <br /> |
| `realm` _string_ | Realm is the name of the realm the client lives in. |  | MinLength: 1 <br /> |
| `clientId` _string_ | ClientID is the client identifier used in OIDC/SAML requests. |  | MinLength: 1 <br /> |
| `adoptionPolicy` _[AdoptionPolicy](#adoptionpolicy)_ | AdoptionPolicy controls the behaviour when a client with the same<br />clientId already exists. Defaults to CreateOnly. |  | Enum: [CreateOnly Adopt FailIfExists] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the client is deleted from the server<br />when this resource is deleted. Defaults to Delete. |  | Enum: [Orphan Delete] <br />Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef injects the client secret from a Secret in the same<br />namespace. When set, the operator keeps the server-side secret in sync<br />with the Secret value. |  | Optional: \{\} <br /> |
| `secretOutput` _[SecretKeySelector](#secretkeyselector)_ | SecretOutput exports the effective client secret to a Secret in the<br />same namespace so applications can mount it. The operator owns the<br />referenced Secret and garbage-collects it with the Client resource. |  | Optional: \{\} <br /> |
| `name` _string_ |  |  | Optional: \{\} <br /> |
| `description` _string_ |  |  | Optional: \{\} <br /> |
| `enabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `protocol` _string_ | Protocol is "openid-connect" or "saml". |  | Enum: [openid-connect saml] <br />Optional: \{\} <br /> |
| `publicClient` _boolean_ |  |  | Optional: \{\} <br /> |
| `bearerOnly` _boolean_ |  |  | Optional: \{\} <br /> |
| `standardFlowEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `implicitFlowEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `directAccessGrantsEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `serviceAccountsEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `authorizationServicesEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `frontchannelLogout` _boolean_ |  |  | Optional: \{\} <br /> |
| `fullScopeAllowed` _boolean_ |  |  | Optional: \{\} <br /> |
| `consentRequired` _boolean_ |  |  | Optional: \{\} <br /> |
| `displayOnConsentScreen` _boolean_ |  |  | Optional: \{\} <br /> |
| `consentScreenText` _string_ |  |  | Optional: \{\} <br /> |
| `alwaysDisplayInConsole` _boolean_ |  |  | Optional: \{\} <br /> |
| `surrogateAuthRequired` _boolean_ |  |  | Optional: \{\} <br /> |
| `rootUrl` _string_ |  |  | Optional: \{\} <br /> |
| `baseUrl` _string_ |  |  | Optional: \{\} <br /> |
| `adminUrl` _string_ |  |  | Optional: \{\} <br /> |
| `redirectUris` _string array_ |  |  | Optional: \{\} <br /> |
| `webOrigins` _string array_ |  |  | Optional: \{\} <br /> |
| `nodeReRegistrationTimeout` _integer_ |  |  | Optional: \{\} <br /> |
| `clientAuthenticatorType` _string_ | ClientAuthenticatorType is for example "client_secret" or<br />"client_jwt". |  | Optional: \{\} <br /> |
| `protocolMappers` _[ProtocolMapper](#protocolmapper) array_ |  |  | Optional: \{\} <br /> |
| `defaultClientScopes` _string_ | DefaultClientScopes lists client scope names granted by default. When<br />set, the operator enforces the exact list; an empty list removes all<br />assignments; when unset, existing assignments are left untouched. |  | Optional: \{\} <br /> |
| `optionalClientScopes` _string_ | OptionalClientScopes lists client scope names the client may request.<br />When set, the operator enforces the exact list; an empty list removes<br />all assignments; when unset, existing assignments are left untouched. |  | Optional: \{\} <br /> |
| `attributes` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `authenticationFlowBindingOverrides` _object (keys:string, values:string)_ | AuthenticationFlowBindingOverrides maps flow bindings such as "browser"<br />or "grant" to flow aliases. |  | Optional: \{\} <br /> |




#### DeletionPolicy

_Underlying type:_ _string_

DeletionPolicy controls what happens to the Keycloak server-side resource
when the custom resource is deleted.

_Validation:_
- Enum: [Orphan Delete]

_Appears in:_
- [ClientScopeSpec](#clientscopespec)
- [ClientSpec](#clientspec)
- [GroupSpec](#groupspec)
- [IdentityProviderSpec](#identityproviderspec)
- [RealmRoleSpec](#realmrolespec)
- [RealmSpec](#realmspec)

| Field | Description |
| --- | --- |
| `Orphan` | DeletionOrphan leaves the server-side resource in place.<br /> |
| `Delete` | DeletionDelete removes the server-side resource.<br /> |


#### Group



Group manages a group on a Keycloak server, including its realm and client
role mappings.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `keycloak.ctn-solutions.io/v1alpha1` | | |
| `kind` _string_ | `Group` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GroupSpec](#groupspec)_ |  |  |  |


#### GroupSpec



GroupSpec defines the desired state of a Group. Core fields mirror the
Keycloak GroupRepresentation; role mappings are managed declaratively by
name. Groups are flat in v1: nested sub-groups are not managed.



_Appears in:_
- [Group](#group)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `keycloakRef` _[KeycloakRef](#keycloakref)_ | KeycloakRef points to the KeycloakConnection managing this group. |  | Required: \{\} <br /> |
| `realm` _string_ | Realm is the name of the realm the group lives in. |  | MinLength: 1 <br /> |
| `name` _string_ | Name is the group name on the Keycloak server. |  | MinLength: 1 <br /> |
| `adoptionPolicy` _[AdoptionPolicy](#adoptionpolicy)_ | AdoptionPolicy controls the behaviour when a group with the same name<br />already exists. Defaults to CreateOnly. |  | Enum: [CreateOnly Adopt FailIfExists] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the group is deleted from the server<br />when this resource is deleted. Defaults to Delete. |  | Enum: [Orphan Delete] <br />Optional: \{\} <br /> |
| `attributes` _object (keys:string, values:string array)_ | Attributes holds group attributes. |  | Optional: \{\} <br /> |
| `realmRoles` _string_ | RealmRoles lists realm role names granted to the group. When set, the<br />operator enforces the exact list; an empty list removes all mappings;<br />when unset, existing mappings are left untouched. |  | Optional: \{\} <br /> |
| `clientRoles` _map[string][]string_ | ClientRoles maps a client ID to the names of that client's roles<br />granted to the group. When set, the operator enforces the exact list<br />per client; an empty list removes all mappings of that client; when<br />unset, existing mappings are left untouched. |  | Optional: \{\} <br /> |


#### IdentityProvider



IdentityProvider manages an identity provider (broker) on a Keycloak
server. The spec mirrors the Keycloak IdentityProviderRepresentation; see
the Keycloak server documentation for field semantics.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `keycloak.ctn-solutions.io/v1alpha1` | | |
| `kind` _string_ | `IdentityProvider` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[IdentityProviderSpec](#identityproviderspec)_ |  |  |  |


#### IdentityProviderMapper



IdentityProviderMapper mirrors the Keycloak
IdentityProviderMapperRepresentation.



_Appears in:_
- [IdentityProviderSpec](#identityproviderspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  | MinLength: 1 <br /> |
| `identityProviderMapper` _string_ |  |  | Optional: \{\} <br /> |
| `config` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |


#### IdentityProviderSpec



IdentityProviderSpec defines the desired state of an IdentityProvider.
Fields mirror the Keycloak IdentityProviderRepresentation one-to-one. The
provider-specific "config" map carries the broker configuration; sensitive
entries such as "clientSecret" should be injected through ConfigSecretRef
instead of being written inline.



_Appears in:_
- [IdentityProvider](#identityprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `keycloakRef` _[KeycloakRef](#keycloakref)_ | KeycloakRef points to the KeycloakConnection managing this identity<br />provider. |  | Required: \{\} <br /> |
| `realm` _string_ | Realm is the name of the realm the identity provider lives in. |  | MinLength: 1 <br /> |
| `alias` _string_ | Alias is the unique identifier of the identity provider within the<br />realm. |  | MinLength: 1 <br /> |
| `providerId` _string_ | ProviderID is the Keycloak provider type, for example "oidc", "saml",<br />"google" or "github". |  | MinLength: 1 <br /> |
| `adoptionPolicy` _[AdoptionPolicy](#adoptionpolicy)_ | AdoptionPolicy controls the behaviour when an identity provider with<br />the same alias already exists. Defaults to CreateOnly. |  | Enum: [CreateOnly Adopt FailIfExists] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the identity provider is deleted from<br />the server when this resource is deleted. Defaults to Delete. |  | Enum: [Orphan Delete] <br />Optional: \{\} <br /> |
| `displayName` _string_ |  |  | Optional: \{\} <br /> |
| `enabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `trustEmail` _boolean_ |  |  | Optional: \{\} <br /> |
| `storeToken` _boolean_ |  |  | Optional: \{\} <br /> |
| `addReadTokenRoleOnCreate` _boolean_ |  |  | Optional: \{\} <br /> |
| `linkOnly` _boolean_ |  |  | Optional: \{\} <br /> |
| `firstBrokerLoginFlowAlias` _string_ |  |  | Optional: \{\} <br /> |
| `postBrokerLoginFlowAlias` _string_ |  |  | Optional: \{\} <br /> |
| `config` _object (keys:string, values:string)_ | Config holds the provider-specific configuration, for example<br />\{"authorizationUrl": "...", "tokenUrl": "...", "clientId": "..."\}. |  | Optional: \{\} <br /> |
| `configSecretRef` _[SecretKeysSelector](#secretkeysselector)_ | ConfigSecretRef injects sensitive config values from a Secret. Keys<br />maps a config entry (for example "clientSecret") to the Secret key<br />holding its value. |  | Optional: \{\} <br /> |
| `mappers` _[IdentityProviderMapper](#identityprovidermapper) array_ |  |  | Optional: \{\} <br /> |


#### KeycloakConnection



KeycloakConnection describes a Keycloak server and the credentials used to
administer it. All managed resources reference a connection through
spec.keycloakRef.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `keycloak.ctn-solutions.io/v1alpha1` | | |
| `kind` _string_ | `KeycloakConnection` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[KeycloakConnectionSpec](#keycloakconnectionspec)_ |  |  |  |


#### KeycloakConnectionSpec



KeycloakConnectionSpec defines the desired state of a KeycloakConnection.



_Appears in:_
- [KeycloakConnection](#keycloakconnection)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL is the base URL of the Keycloak server, for example<br />https://keycloak.example.com. |  | MinLength: 1 <br />Pattern: `^https?://` <br /> |
| `credentialsSecretRef` _string_ | CredentialsSecretRef references a Secret in the same namespace holding<br />the administration credentials. For auth "password" the Secret must<br />provide the keys "username" and "password"; for auth "client" the keys<br />"clientId" and "clientSecret". |  | MinLength: 1 <br /> |
| `auth` _[AuthType](#authtype)_ | Auth selects the authentication method. Defaults to "password". |  | Enum: [password client] <br />Optional: \{\} <br /> |
| `adminRealm` _string_ | AdminRealm is the realm the administration credentials live in.<br />Defaults to "master". |  | Optional: \{\} <br /> |
| `tls` _[TLSConfig](#tlsconfig)_ | TLS configures transport security. |  | Optional: \{\} <br /> |




#### KeycloakRef



KeycloakRef points to the KeycloakConnection resource governing the
Keycloak server a resource belongs to. The connection must live in the same
namespace as the referencing resource.



_Appears in:_
- [ClientScopeSpec](#clientscopespec)
- [ClientSpec](#clientspec)
- [GroupSpec](#groupspec)
- [IdentityProviderSpec](#identityproviderspec)
- [RealmRoleSpec](#realmrolespec)
- [RealmSpec](#realmspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the KeycloakConnection resource. |  | MinLength: 1 <br /> |


#### ProtocolMapper



ProtocolMapper mirrors the Keycloak ProtocolMapperRepresentation.



_Appears in:_
- [ClientScopeSpec](#clientscopespec)
- [ClientSpec](#clientspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  | Optional: \{\} <br /> |
| `protocol` _string_ |  |  | Optional: \{\} <br /> |
| `protocolMapper` _string_ |  |  | Optional: \{\} <br /> |
| `consentRequired` _boolean_ |  |  | Optional: \{\} <br /> |
| `config` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |


#### Realm



Realm manages a realm on a Keycloak server. The spec mirrors the Keycloak
RealmRepresentation; see the Keycloak server documentation for field
semantics.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `keycloak.ctn-solutions.io/v1alpha1` | | |
| `kind` _string_ | `Realm` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RealmSpec](#realmspec)_ |  |  |  |


#### RealmRole



RealmRole manages a realm-level role on a Keycloak server. The spec mirrors
the Keycloak RoleRepresentation; see the Keycloak server documentation for
field semantics.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `keycloak.ctn-solutions.io/v1alpha1` | | |
| `kind` _string_ | `RealmRole` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RealmRoleSpec](#realmrolespec)_ |  |  |  |


#### RealmRoleSpec



RealmRoleSpec defines the desired state of a RealmRole. Fields mirror the
Keycloak RoleRepresentation (realm level) one-to-one.



_Appears in:_
- [RealmRole](#realmrole)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `keycloakRef` _[KeycloakRef](#keycloakref)_ | KeycloakRef points to the KeycloakConnection managing this role. |  | Required: \{\} <br /> |
| `realm` _string_ | Realm is the name of the realm the role lives in. |  | MinLength: 1 <br /> |
| `name` _string_ | Name is the role name on the Keycloak server. |  | MinLength: 1 <br /> |
| `adoptionPolicy` _[AdoptionPolicy](#adoptionpolicy)_ | AdoptionPolicy controls the behaviour when a role with the same name<br />already exists. Defaults to CreateOnly. |  | Enum: [CreateOnly Adopt FailIfExists] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the role is deleted from the server<br />when this resource is deleted. Defaults to Delete. |  | Enum: [Orphan Delete] <br />Optional: \{\} <br /> |
| `description` _string_ |  |  | Optional: \{\} <br /> |
| `composite` _boolean_ | Composite marks the role as composite. When true, Composites defines<br />the exact set of included roles. |  | Optional: \{\} <br /> |
| `composites` _[RoleComposites](#rolecomposites)_ |  |  | Optional: \{\} <br /> |
| `attributes` _object (keys:string, values:string array)_ |  |  | Optional: \{\} <br /> |


#### RealmSpec



RealmSpec defines the desired state of a Realm. Fields mirror the Keycloak
RealmRepresentation one-to-one; scalar fields are pointers so that unset,
explicitly zero and default values are distinguishable.



_Appears in:_
- [Realm](#realm)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `keycloakRef` _[KeycloakRef](#keycloakref)_ | KeycloakRef points to the KeycloakConnection managing this realm. |  | Required: \{\} <br /> |
| `realm` _string_ | Realm is the name of the realm on the Keycloak server. |  | MinLength: 1 <br /> |
| `adoptionPolicy` _[AdoptionPolicy](#adoptionpolicy)_ | AdoptionPolicy controls the behaviour when the realm already exists on<br />the server. Defaults to CreateOnly. |  | Enum: [CreateOnly Adopt FailIfExists] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the realm is deleted from the server<br />when this resource is deleted. Defaults to Orphan: realms are never<br />deleted implicitly. |  | Enum: [Orphan Delete] <br />Optional: \{\} <br /> |
| `enabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `displayName` _string_ |  |  | Optional: \{\} <br /> |
| `displayNameHtml` _string_ |  |  | Optional: \{\} <br /> |
| `sslRequired` _string_ | SSLRequired is one of "all", "external" or "none". |  | Enum: [all external none] <br />Optional: \{\} <br /> |
| `notBefore` _integer_ |  |  | Optional: \{\} <br /> |
| `registrationAllowed` _boolean_ |  |  | Optional: \{\} <br /> |
| `registrationEmailAsUsername` _boolean_ |  |  | Optional: \{\} <br /> |
| `loginWithEmailAllowed` _boolean_ |  |  | Optional: \{\} <br /> |
| `duplicateEmailsAllowed` _boolean_ |  |  | Optional: \{\} <br /> |
| `resetPasswordAllowed` _boolean_ |  |  | Optional: \{\} <br /> |
| `rememberMe` _boolean_ |  |  | Optional: \{\} <br /> |
| `verifyEmail` _boolean_ |  |  | Optional: \{\} <br /> |
| `userManagedAccessAllowed` _boolean_ |  |  | Optional: \{\} <br /> |
| `ssoSessionIdleTimeout` _integer_ |  |  | Optional: \{\} <br /> |
| `ssoSessionMaxLifespan` _integer_ |  |  | Optional: \{\} <br /> |
| `ssoSessionIdleTimeoutRememberMe` _integer_ |  |  | Optional: \{\} <br /> |
| `ssoSessionMaxLifespanRememberMe` _integer_ |  |  | Optional: \{\} <br /> |
| `clientSessionIdleTimeout` _integer_ |  |  | Optional: \{\} <br /> |
| `clientSessionMaxLifespan` _integer_ |  |  | Optional: \{\} <br /> |
| `clientOfflineSessionIdleTimeout` _integer_ |  |  | Optional: \{\} <br /> |
| `clientOfflineSessionMaxLifespan` _integer_ |  |  | Optional: \{\} <br /> |
| `accessTokenLifespan` _integer_ |  |  | Optional: \{\} <br /> |
| `accessTokenLifespanForImplicitFlow` _integer_ |  |  | Optional: \{\} <br /> |
| `offlineSessionIdleTimeout` _integer_ |  |  | Optional: \{\} <br /> |
| `offlineSessionMaxLifespanEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `offlineSessionMaxLifespan` _integer_ |  |  | Optional: \{\} <br /> |
| `accessCodeLifespan` _integer_ |  |  | Optional: \{\} <br /> |
| `accessCodeLifespanUserAction` _integer_ |  |  | Optional: \{\} <br /> |
| `accessCodeLifespanLogin` _integer_ |  |  | Optional: \{\} <br /> |
| `actionTokenGeneratedByAdminLifespan` _integer_ |  |  | Optional: \{\} <br /> |
| `actionTokenGeneratedByUserLifespan` _integer_ |  |  | Optional: \{\} <br /> |
| `oauth2DeviceCodeLifespan` _integer_ |  |  | Optional: \{\} <br /> |
| `oauth2DevicePollingInterval` _integer_ |  |  | Optional: \{\} <br /> |
| `revokeRefreshToken` _boolean_ |  |  | Optional: \{\} <br /> |
| `refreshTokenMaxReuse` _integer_ |  |  | Optional: \{\} <br /> |
| `loginTheme` _string_ |  |  | Optional: \{\} <br /> |
| `accountTheme` _string_ |  |  | Optional: \{\} <br /> |
| `adminTheme` _string_ |  |  | Optional: \{\} <br /> |
| `emailTheme` _string_ |  |  | Optional: \{\} <br /> |
| `smtpServer` _object (keys:string, values:string)_ | SMTPServer holds the SMTP configuration, for example \{"host": "...",<br />"from": "...", "auth": "true", "starttls": "true"\}. Sensitive values<br />such as "password" should be provided through SMTPServerSecretRef<br />instead of inline. |  | Optional: \{\} <br /> |
| `smtpServerSecretRef` _[SecretKeysSelector](#secretkeysselector)_ | SMTPServerSecretRef injects sensitive SMTP values from a Secret into<br />the smtpServer map. Keys maps an smtpServer entry (for example<br />"password") to the Secret key holding its value. |  | Optional: \{\} <br /> |
| `eventsEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `eventsExpiration` _integer_ |  |  | Optional: \{\} <br /> |
| `eventsListeners` _string array_ |  |  | Optional: \{\} <br /> |
| `enabledEventTypes` _string array_ |  |  | Optional: \{\} <br /> |
| `adminEventsEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `adminEventsDetailsEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `passwordPolicy` _string_ |  |  | Optional: \{\} <br /> |
| `bruteForceProtected` _boolean_ |  |  | Optional: \{\} <br /> |
| `permanentLockout` _boolean_ |  |  | Optional: \{\} <br /> |
| `maxFailureWaitSeconds` _integer_ |  |  | Optional: \{\} <br /> |
| `minimumQuickLoginWaitSeconds` _integer_ |  |  | Optional: \{\} <br /> |
| `waitIncrementSeconds` _integer_ |  |  | Optional: \{\} <br /> |
| `quickLoginCheckMilliSeconds` _integer_ |  |  | Optional: \{\} <br /> |
| `maxDeltaTimeSeconds` _integer_ |  |  | Optional: \{\} <br /> |
| `failureFactor` _integer_ |  |  | Optional: \{\} <br /> |
| `otpPolicyType` _string_ |  |  | Optional: \{\} <br /> |
| `otpPolicyAlgorithm` _string_ |  |  | Optional: \{\} <br /> |
| `otpPolicyInitialCounter` _integer_ |  |  | Optional: \{\} <br /> |
| `otpPolicyDigits` _integer_ |  |  | Optional: \{\} <br /> |
| `otpPolicyLookAheadWindow` _integer_ |  |  | Optional: \{\} <br /> |
| `otpPolicyPeriod` _integer_ |  |  | Optional: \{\} <br /> |
| `otpPolicyCodeReusable` _boolean_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyRpEntityName` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicySignatureAlgorithms` _string array_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyRpId` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyAttestationConveyancePreference` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyAuthenticatorAttachment` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyRequireResidentKey` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyUserVerificationRequirement` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyCreateTimeout` _integer_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyAvoidSameAuthenticatorRegister` _boolean_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyAcceptableAaguids` _string array_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessRpEntityName` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessSignatureAlgorithms` _string array_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessRpId` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessAttestationConveyancePreference` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessAuthenticatorAttachment` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessRequireResidentKey` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessUserVerificationRequirement` _string_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessCreateTimeout` _integer_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessAvoidSameAuthenticatorRegister` _boolean_ |  |  | Optional: \{\} <br /> |
| `webAuthnPolicyPasswordlessAcceptableAaguids` _string array_ |  |  | Optional: \{\} <br /> |
| `internationalizationEnabled` _boolean_ |  |  | Optional: \{\} <br /> |
| `supportedLocales` _string array_ |  |  | Optional: \{\} <br /> |
| `defaultLocale` _string_ |  |  | Optional: \{\} <br /> |
| `defaultDefaultClientScopes` _string array_ | DefaultDefaultClientScopes lists client scopes granted to every client<br />by default. |  | Optional: \{\} <br /> |
| `defaultOptionalClientScopes` _string array_ | DefaultOptionalClientScopes lists client scopes clients may request. |  | Optional: \{\} <br /> |
| `defaultSignatureAlgorithm` _string_ |  |  | Optional: \{\} <br /> |
| `attributes` _object (keys:string, values:string)_ | Attributes is a free-form map passed through to the realm<br />representation. It is also where the operator records its managed<br />marker. |  | Optional: \{\} <br /> |
| `browserSecurityHeaders` _object (keys:string, values:string)_ | BrowserSecurityHeaders configures the security headers injected into<br />realm pages, for example \{"contentSecurityPolicy": "..."\}. |  | Optional: \{\} <br /> |
| `organizationsEnabled` _boolean_ |  |  | Optional: \{\} <br /> |


#### ResourceStatus



ResourceStatus is the shared status block of all managed resources.



_Appears in:_
- [ClientStatus](#clientstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.3/#condition-v1-meta) array_ | Conditions report the outcome of the latest reconciliation. |  | Optional: \{\} <br /> |
| `observedGeneration` _integer_ | ObservedGeneration is the generation most recently processed. |  | Optional: \{\} <br /> |


#### RoleComposites



RoleComposites describes the roles a composite role is made of.



_Appears in:_
- [RealmRoleSpec](#realmrolespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `realmRoles` _string array_ | RealmRoles lists realm role names included in the composite. |  | Optional: \{\} <br /> |
| `clientRoles` _object (keys:string, values:string array)_ | ClientRoles maps a client ID to the names of that client's roles<br />included in the composite. |  | Optional: \{\} <br /> |


#### SecretKeySelector



SecretKeySelector references a key in a Secret in the same namespace.



_Appears in:_
- [ClientSpec](#clientspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Secret. |  | MinLength: 1 <br /> |
| `key` _string_ | Key within the Secret. Defaults to "clientSecret" for client secrets. |  | MinLength: 1 <br />Optional: \{\} <br /> |


#### SecretKeysSelector



SecretKeysSelector references multiple keys of a Secret and maps them onto
target configuration keys (for example identity provider config entries or
SMTP server settings).



_Appears in:_
- [IdentityProviderSpec](#identityproviderspec)
- [RealmSpec](#realmspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Secret. |  | MinLength: 1 <br /> |
| `keys` _object (keys:string, values:string)_ | Keys maps a target configuration key to the Secret key holding its<br />value. |  | MinProperties: 1 <br /> |


#### TLSConfig



TLSConfig configures transport security for the connection.



_Appears in:_
- [KeycloakConnectionSpec](#keycloakconnectionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `insecureSkipVerify` _boolean_ | InsecureSkipVerify disables TLS certificate verification. Use only in<br />trusted environments such as local development. |  | Optional: \{\} <br /> |


