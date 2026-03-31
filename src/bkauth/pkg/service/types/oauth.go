/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth 服务 (BlueKing - Auth) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package types

import (
	"time"

	"bkauth/pkg/oauth"
)

// OAuthClientDynamicRegistrationInput carries only the caller-provided fields
// for Dynamic Client Registration (RFC 7591).
type OAuthClientDynamicRegistrationInput struct {
	Name         string
	RedirectURIs []string
	GrantTypes   []string
	LogoURI      string
}

// OAuthClient represents the full OAuth client entity, used only by DynamicRegister.
// token_endpoint_auth_method is derived from Type at runtime:
//
//	public -> "none", confidential -> "client_secret_basic"
type OAuthClient struct {
	ID           string   `json:"client_id"`
	Name         string   `json:"client_name"`
	Type         string   `json:"client_type"`
	RedirectURIs []string `json:"redirect_uris"`
	GrantTypes   []string `json:"grant_types"`
	LogoURI      string   `json:"logo_uri,omitempty"`
	CreatedAt    int64    `json:"client_id_issued_at,omitempty"`
}

// TokenEndpointAuthMethod returns the auth method derived from client type.
func (c OAuthClient) TokenEndpointAuthMethod() string {
	if c.Type == oauth.ClientTypeConfidential {
		return oauth.AuthMethodClientSecretBasic
	}
	return oauth.AuthMethodNone
}

// OAuthClientFlowSpec holds the OAuth protocol parameters that constrain authorization flows.
// Used by token, authorize and device_authorize handlers to validate grant types and redirect URIs.
type OAuthClientFlowSpec struct {
	ID           string
	GrantTypes   []string
	RedirectURIs []string
}

// SupportsGrantType reports whether the client was registered with the given grant type.
func (s OAuthClientFlowSpec) SupportsGrantType(grantType string) bool {
	for _, gt := range s.GrantTypes {
		if gt == grantType {
			return true
		}
	}
	return false
}

// OAuthClientProfile holds the display-oriented fields for consent / device pages.
type OAuthClientProfile struct {
	ID      string
	Name    string
	Type    string
	LogoURI string
}

// CreateAuthorizationCodeInput carries the caller-provided fields needed
// to persist a new authorization code.
type CreateAuthorizationCodeInput struct {
	Code                string
	ClientID            string
	TenantID            string
	RealmName           string
	Sub                 string
	Username            string
	RedirectURI         string
	Audience            []string
	CodeChallenge       string
	CodeChallengeMethod string
}

// ConsumedAuthorizationCode is the minimal data set returned after an
// authorization code has been validated and consumed, used to issue tokens.
type ConsumedAuthorizationCode struct {
	TenantID string
	Sub      string
	Username string
	Audience []string
}

// ResolvedAccessToken contains the fields resolved from an opaque access token string,
// suitable for introspection, validation, and caching.
type ResolvedAccessToken struct {
	// Standard OAuth 2.0 / RFC 7662 claims
	// JTI      string
	ClientID  string
	TenantID  string
	RealmName string
	Sub       string
	Username  string
	Audience  []string
	// Scope    string

	// Token lifecycle
	ExpiresAt int64
	// IssuedAt  int64
	// NotBefore int64
	Revoked bool
}

// IsActive reports whether the token is currently usable (not revoked and not expired).
func (t ResolvedAccessToken) IsActive() bool {
	return !t.Revoked && time.Now().Unix() < t.ExpiresAt
}

// TokenIssuancePolicy holds the realm-specific parameters that govern token generation.
type TokenIssuancePolicy struct {
	Prefix          string
	AccessTokenTTL  int64
	RefreshTokenTTL int64
}

// TokenPair represents an access token and refresh token pair.
// TokenType is intentionally omitted — it is a presentation-layer concern
// and should be set by the HTTP handler (e.g. oauth.TokenTypeBearer).
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// CreatedDeviceCode is returned by CreateDeviceCode with the fields needed
// to build the device authorization response (RFC 8628 §3.2).
type CreatedDeviceCode struct {
	DeviceCode   string
	UserCode     string
	PollInterval int64
}

// PendingDeviceCode is returned by GetByUserCode with the fields needed
// to render the user-facing consent page.
type PendingDeviceCode struct {
	ClientID  string
	RealmName string
	Resource  string
}

// ApprovedDeviceCode is returned by PollAndConsumeDeviceCode when the device
// code has been approved and consumed, carrying the identity claims needed to issue tokens.
type ApprovedDeviceCode struct {
	TenantID string
	Sub      string
	Username string
	Audience []string
}
