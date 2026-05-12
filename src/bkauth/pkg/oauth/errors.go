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

package oauth

import "errors"

// OAuthError represents an OAuth 2.0 error response (RFC 6749 Section 5.2).
//
// Field names (Code, Description) differ from JSON tags ("error", "error_description")
// to avoid collision with the error interface's Error() method.
//
// By implementing the error interface, OAuthError serves a dual role:
// it can be returned as an error from validation/service methods,
// and serialized directly as an OAuth HTTP error response body.
type OAuthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return e.Code + ": " + e.Description
	}
	return e.Code
}

// Standard OAuth 2.0 error codes.
const (
	// RFC 6749 §4.1.2.1, §5.2 — Authorization & Token Endpoint
	ErrorCodeInvalidRequest = "invalid_request"
	// RFC 6749 §5.2 — Token Endpoint
	ErrorCodeInvalidClient = "invalid_client"
	// RFC 6749 §5.2 — Token Endpoint
	ErrorCodeInvalidGrant = "invalid_grant"
	// RFC 6749 §4.1.2.1, §5.2 — Authorization & Token Endpoint
	ErrorCodeUnauthorizedClient = "unauthorized_client"
	// RFC 6749 §4.1.2.1 — Authorization Endpoint
	ErrorCodeAccessDenied = "access_denied"
	// RFC 6749 §4.1.2.1 — Authorization Endpoint
	ErrorCodeUnsupportedResponseType = "unsupported_response_type"
	// RFC 6749 §5.2 — Token Endpoint
	ErrorCodeUnsupportedGrantType = "unsupported_grant_type"
	// RFC 7009 §2.2.1 — Token Revocation
	ErrorCodeUnsupportedTokenType = "unsupported_token_type"
	// RFC 6749 §4.1.2.1 — Authorization Endpoint
	ErrorCodeServerError = "server_error"
	// RFC 7591 §3.2.2 — Dynamic Client Registration
	ErrorCodeInvalidClientMetadata = "invalid_client_metadata"
	// RFC 7591 §3.2.2 — Dynamic Client Registration
	ErrorCodeInvalidRedirectURI = "invalid_redirect_uri"
	// RFC 8628 §3.5 — Device Authorization Grant
	ErrorCodeAuthorizationPending = "authorization_pending"
	// RFC 8628 §3.5 — Device Authorization Grant
	ErrorCodeSlowDown = "slow_down"
	// RFC 8628 §3.5 — Device Authorization Grant
	ErrorCodeExpiredToken = "expired_token"
)

func NewInvalidRequestError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeInvalidRequest, Description: description}
}

func NewInvalidClientError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeInvalidClient, Description: description}
}

func NewInvalidGrantError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeInvalidGrant, Description: description}
}

func NewUnauthorizedClientError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeUnauthorizedClient, Description: description}
}

func NewAccessDeniedError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeAccessDenied, Description: description}
}

func NewUnsupportedResponseTypeError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeUnsupportedResponseType, Description: description}
}

func NewUnsupportedGrantTypeError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeUnsupportedGrantType, Description: description}
}

func NewUnsupportedTokenTypeError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeUnsupportedTokenType, Description: description}
}

func NewServerError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeServerError, Description: description}
}

func NewInvalidClientMetadataError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeInvalidClientMetadata, Description: description}
}

func NewInvalidRedirectURIError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeInvalidRedirectURI, Description: description}
}

func NewAuthorizationPendingError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeAuthorizationPending, Description: description}
}

func NewSlowDownError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeSlowDown, Description: description}
}

func NewExpiredTokenError(description string) *OAuthError {
	return &OAuthError{Code: ErrorCodeExpiredToken, Description: description}
}

// AsOAuthError extracts an *OAuthError from err using errors.As.
func AsOAuthError(err error) (*OAuthError, bool) {
	var oauthErr *OAuthError
	if errors.As(err, &oauthErr) {
		return oauthErr, true
	}
	return nil, false
}

// Realm errors
var ErrRealmMismatch = errors.New("realm mismatch")

// Authorization code errors
var (
	ErrInvalidAuthorizationCode = errors.New("invalid authorization code")
	ErrAuthorizationCodeExpired = errors.New("authorization code expired")
	ErrAuthorizationCodeUsed    = errors.New("authorization code already used")
	ErrInvalidCodeVerifier      = errors.New("invalid code verifier")
	ErrClientMismatch           = errors.New("client id mismatch")
	ErrRedirectURIMismatch      = errors.New("redirect uri mismatch")
)

// Access token errors
var (
	ErrInvalidAccessToken = errors.New("invalid access token")
	ErrAccessTokenExpired = errors.New("access token expired")
	ErrAccessTokenRevoked = errors.New("access token revoked")
)

// Refresh token errors
var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
)

// Device code errors (RFC 8628)
var (
	ErrInvalidDeviceCode     = errors.New("invalid device code")
	ErrDeviceCodeExpired     = errors.New("device code expired")
	ErrDeviceCodeDenied      = errors.New("device code denied by user")
	ErrDeviceCodeConsumed    = errors.New("device code already consumed")
	ErrAuthorizationPending  = errors.New("authorization pending")
	ErrSlowDown              = errors.New("slow down")
	ErrInvalidUserCode       = errors.New("invalid user code")
	ErrUserCodeExpired       = errors.New("user code expired")
	ErrUserCodeAlreadyUsed   = errors.New("user code already used")
	ErrDeviceCodeClientMatch = errors.New("device code client mismatch")
)

// Client authentication errors
var (
	ErrMissingClientSecret = errors.New("client_secret is required for confidential clients")
	ErrInvalidClientSecret = errors.New("invalid client_secret")
)
