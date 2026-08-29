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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RealmSpec defines the desired state of a Realm. Fields mirror the Keycloak
// RealmRepresentation one-to-one; scalar fields are pointers so that unset,
// explicitly zero and default values are distinguishable.
type RealmSpec struct {
	// KeycloakRef points to the KeycloakConnection managing this realm.
	// +kubebuilder:validation:Required
	KeycloakRef KeycloakRef `json:"keycloakRef"`

	// Realm is the name of the realm on the Keycloak server.
	// +kubebuilder:validation:MinLength=1
	Realm string `json:"realm"`

	// AdoptionPolicy controls the behaviour when the realm already exists on
	// the server. Defaults to CreateOnly.
	// +kubebuilder:validation:Enum=CreateOnly;Adopt;FailIfExists
	// +optional
	AdoptionPolicy *AdoptionPolicy `json:"adoptionPolicy,omitempty"`

	// DeletionPolicy controls whether the realm is deleted from the server
	// when this resource is deleted. Defaults to Orphan: realms are never
	// deleted implicitly.
	// +kubebuilder:validation:Enum=Orphan;Delete
	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`

	// --- Core ---

	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// +optional
	DisplayName *string `json:"displayName,omitempty"`
	// +optional
	DisplayNameHTML *string `json:"displayNameHtml,omitempty"`
	// SSLRequired is one of "all", "external" or "none".
	// +kubebuilder:validation:Enum=all;external;none
	// +optional
	SSLRequired *string `json:"sslRequired,omitempty"`
	// +optional
	NotBefore *int `json:"notBefore,omitempty"`

	// --- Login ---

	// +optional
	RegistrationAllowed *bool `json:"registrationAllowed,omitempty"`
	// +optional
	RegistrationEmailAsUsername *bool `json:"registrationEmailAsUsername,omitempty"`
	// +optional
	LoginWithEmailAllowed *bool `json:"loginWithEmailAllowed,omitempty"`
	// +optional
	DuplicateEmailsAllowed *bool `json:"duplicateEmailsAllowed,omitempty"`
	// +optional
	ResetPasswordAllowed *bool `json:"resetPasswordAllowed,omitempty"`
	// +optional
	RememberMe *bool `json:"rememberMe,omitempty"`
	// +optional
	VerifyEmail *bool `json:"verifyEmail,omitempty"`
	// +optional
	UserManagedAccessAllowed *bool `json:"userManagedAccessAllowed,omitempty"`

	// --- Sessions and tokens (seconds unless noted) ---

	// +optional
	SSOSessionIdleTimeout *int `json:"ssoSessionIdleTimeout,omitempty"`
	// +optional
	SSOSessionMaxLifespan *int `json:"ssoSessionMaxLifespan,omitempty"`
	// +optional
	SSOSessionIdleTimeoutRememberMe *int `json:"ssoSessionIdleTimeoutRememberMe,omitempty"`
	// +optional
	SSOSessionMaxLifespanRememberMe *int `json:"ssoSessionMaxLifespanRememberMe,omitempty"`
	// +optional
	ClientSessionIdleTimeout *int `json:"clientSessionIdleTimeout,omitempty"`
	// +optional
	ClientSessionMaxLifespan *int `json:"clientSessionMaxLifespan,omitempty"`
	// +optional
	ClientOfflineSessionIdleTimeout *int `json:"clientOfflineSessionIdleTimeout,omitempty"`
	// +optional
	ClientOfflineSessionMaxLifespan *int `json:"clientOfflineSessionMaxLifespan,omitempty"`
	// +optional
	AccessTokenLifespan *int `json:"accessTokenLifespan,omitempty"`
	// +optional
	AccessTokenLifespanForImplicitFlow *int `json:"accessTokenLifespanForImplicitFlow,omitempty"`
	// +optional
	OfflineSessionIdleTimeout *int `json:"offlineSessionIdleTimeout,omitempty"`
	// +optional
	OfflineSessionMaxLifespanEnabled *bool `json:"offlineSessionMaxLifespanEnabled,omitempty"`
	// +optional
	OfflineSessionMaxLifespan *int `json:"offlineSessionMaxLifespan,omitempty"`
	// +optional
	AccessCodeLifespan *int `json:"accessCodeLifespan,omitempty"`
	// +optional
	AccessCodeLifespanUserAction *int `json:"accessCodeLifespanUserAction,omitempty"`
	// +optional
	AccessCodeLifespanLogin *int `json:"accessCodeLifespanLogin,omitempty"`
	// +optional
	ActionTokenGeneratedByAdminLifespan *int `json:"actionTokenGeneratedByAdminLifespan,omitempty"`
	// +optional
	ActionTokenGeneratedByUserLifespan *int `json:"actionTokenGeneratedByUserLifespan,omitempty"`
	// +optional
	OAuth2DeviceCodeLifespan *int `json:"oauth2DeviceCodeLifespan,omitempty"`
	// +optional
	OAuth2DevicePollingInterval *int `json:"oauth2DevicePollingInterval,omitempty"`
	// +optional
	RevokeRefreshToken *bool `json:"revokeRefreshToken,omitempty"`
	// +optional
	RefreshTokenMaxReuse *int `json:"refreshTokenMaxReuse,omitempty"`

	// --- Themes ---

	// +optional
	LoginTheme *string `json:"loginTheme,omitempty"`
	// +optional
	AccountTheme *string `json:"accountTheme,omitempty"`
	// +optional
	AdminTheme *string `json:"adminTheme,omitempty"`
	// +optional
	EmailTheme *string `json:"emailTheme,omitempty"`

	// --- Email ---

	// SMTPServer holds the SMTP configuration, for example {"host": "...",
	// "from": "...", "auth": "true", "starttls": "true"}. Sensitive values
	// such as "password" should be provided through SMTPServerSecretRef
	// instead of inline.
	// +optional
	SMTPServer map[string]string `json:"smtpServer,omitempty"`
	// SMTPServerSecretRef injects sensitive SMTP values from a Secret into
	// the smtpServer map. Keys maps an smtpServer entry (for example
	// "password") to the Secret key holding its value.
	// +optional
	SMTPServerSecretRef *SecretKeysSelector `json:"smtpServerSecretRef,omitempty"`

	// --- Events ---

	// +optional
	EventsEnabled *bool `json:"eventsEnabled,omitempty"`
	// +optional
	EventsExpiration *int64 `json:"eventsExpiration,omitempty"`
	// +optional
	EventsListeners []string `json:"eventsListeners,omitempty"`
	// +optional
	EnabledEventTypes []string `json:"enabledEventTypes,omitempty"`
	// +optional
	AdminEventsEnabled *bool `json:"adminEventsEnabled,omitempty"`
	// +optional
	AdminEventsDetailsEnabled *bool `json:"adminEventsDetailsEnabled,omitempty"`

	// --- Password and brute-force protection ---

	// +optional
	PasswordPolicy *string `json:"passwordPolicy,omitempty"`
	// +optional
	BruteForceProtected *bool `json:"bruteForceProtected,omitempty"`
	// +optional
	PermanentLockout *bool `json:"permanentLockout,omitempty"`
	// +optional
	MaxFailureWaitSeconds *int `json:"maxFailureWaitSeconds,omitempty"`
	// +optional
	MinimumQuickLoginWaitSeconds *int `json:"minimumQuickLoginWaitSeconds,omitempty"`
	// +optional
	WaitIncrementSeconds *int `json:"waitIncrementSeconds,omitempty"`
	// +optional
	QuickLoginCheckMilliSeconds *int64 `json:"quickLoginCheckMilliSeconds,omitempty"`
	// +optional
	MaxDeltaTimeSeconds *int `json:"maxDeltaTimeSeconds,omitempty"`
	// +optional
	FailureFactor *int `json:"failureFactor,omitempty"`

	// --- One-time password policy ---

	// +optional
	OTPPolicyType *string `json:"otpPolicyType,omitempty"`
	// +optional
	OTPPolicyAlgorithm *string `json:"otpPolicyAlgorithm,omitempty"`
	// +optional
	OTPPolicyInitialCounter *int `json:"otpPolicyInitialCounter,omitempty"`
	// +optional
	OTPPolicyDigits *int `json:"otpPolicyDigits,omitempty"`
	// +optional
	OTPPolicyLookAheadWindow *int `json:"otpPolicyLookAheadWindow,omitempty"`
	// +optional
	OTPPolicyPeriod *int `json:"otpPolicyPeriod,omitempty"`
	// +optional
	OTPPolicyCodeReusable *bool `json:"otpPolicyCodeReusable,omitempty"`

	// --- WebAuthn (passwordless variants mirror the non-passwordless ones) ---

	// +optional
	WebAuthnPolicyRpEntityName *string `json:"webAuthnPolicyRpEntityName,omitempty"`
	// +optional
	WebAuthnPolicySignatureAlgorithms []string `json:"webAuthnPolicySignatureAlgorithms,omitempty"`
	// +optional
	WebAuthnPolicyRpID *string `json:"webAuthnPolicyRpId,omitempty"`
	// +optional
	WebAuthnPolicyAttestationConveyancePreference *string `json:"webAuthnPolicyAttestationConveyancePreference,omitempty"`
	// +optional
	WebAuthnPolicyAuthenticatorAttachment *string `json:"webAuthnPolicyAuthenticatorAttachment,omitempty"`
	// +optional
	WebAuthnPolicyRequireResidentKey *string `json:"webAuthnPolicyRequireResidentKey,omitempty"`
	// +optional
	WebAuthnPolicyUserVerificationRequirement *string `json:"webAuthnPolicyUserVerificationRequirement,omitempty"`
	// +optional
	WebAuthnPolicyCreateTimeout *int `json:"webAuthnPolicyCreateTimeout,omitempty"`
	// +optional
	WebAuthnPolicyAvoidSameAuthenticatorRegister *bool `json:"webAuthnPolicyAvoidSameAuthenticatorRegister,omitempty"`
	// +optional
	WebAuthnPolicyAcceptableAaguids []string `json:"webAuthnPolicyAcceptableAaguids,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessRpEntityName *string `json:"webAuthnPolicyPasswordlessRpEntityName,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessSignatureAlgorithms []string `json:"webAuthnPolicyPasswordlessSignatureAlgorithms,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessRpID *string `json:"webAuthnPolicyPasswordlessRpId,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessAttestationConveyancePreference *string `json:"webAuthnPolicyPasswordlessAttestationConveyancePreference,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessAuthenticatorAttachment *string `json:"webAuthnPolicyPasswordlessAuthenticatorAttachment,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessRequireResidentKey *string `json:"webAuthnPolicyPasswordlessRequireResidentKey,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessUserVerificationRequirement *string `json:"webAuthnPolicyPasswordlessUserVerificationRequirement,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessCreateTimeout *int `json:"webAuthnPolicyPasswordlessCreateTimeout,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessAvoidSameAuthenticatorRegister *bool `json:"webAuthnPolicyPasswordlessAvoidSameAuthenticatorRegister,omitempty"`
	// +optional
	WebAuthnPolicyPasswordlessAcceptableAaguids []string `json:"webAuthnPolicyPasswordlessAcceptableAaguids,omitempty"`

	// --- Internationalization ---

	// +optional
	InternationalizationEnabled *bool `json:"internationalizationEnabled,omitempty"`
	// +optional
	SupportedLocales []string `json:"supportedLocales,omitempty"`
	// +optional
	DefaultLocale *string `json:"defaultLocale,omitempty"`

	// --- Defaults ---

	// DefaultDefaultClientScopes lists client scopes granted to every client
	// by default.
	// +optional
	DefaultDefaultClientScopes []string `json:"defaultDefaultClientScopes,omitempty"`
	// DefaultOptionalClientScopes lists client scopes clients may request.
	// +optional
	DefaultOptionalClientScopes []string `json:"defaultOptionalClientScopes,omitempty"`
	// +optional
	DefaultSignatureAlgorithm *string `json:"defaultSignatureAlgorithm,omitempty"`

	// --- Misc ---

	// Attributes is a free-form map passed through to the realm
	// representation. It is also where the operator records its managed
	// marker.
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
	// BrowserSecurityHeaders configures the security headers injected into
	// realm pages, for example {"contentSecurityPolicy": "..."}.
	// +optional
	BrowserSecurityHeaders map[string]string `json:"browserSecurityHeaders,omitempty"`
	// +optional
	OrganizationsEnabled *bool `json:"organizationsEnabled,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=keycloak,shortName=kcrealm
// +kubebuilder:printcolumn:name="Realm",type=string,JSONPath=`.spec.realm`
// +kubebuilder:printcolumn:name="Connection",type=string,JSONPath=`.spec.keycloakRef.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Realm manages a realm on a Keycloak server. The spec mirrors the Keycloak
// RealmRepresentation; see the Keycloak server documentation for field
// semantics.
type Realm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RealmSpec       `json:"spec,omitempty"`
	Status ResourceStatus  `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RealmList contains a list of Realm.
type RealmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Realm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Realm{}, &RealmList{})
}
