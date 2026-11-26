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

package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	defaultAccessTokenTTL  int64 = 7200    // 2 hours
	defaultRefreshTokenTTL int64 = 2592000 // 30 days

	defaultExternalTokenMaxLength      = 256
	defaultExternalTokenAdapterTimeout = 2 // seconds
	// minExternalTokenPrefixLength keeps a prefix specific enough to name one
	// vendor's format: a one-letter prefix would forward any garbage starting
	// with that letter to the adapter as a probe.
	minExternalTokenPrefixLength = 3

	reservedNativeTokenPrefix = "bk"
)

// TokenTTLOverride allows overriding the default AccessToken/RefreshToken TTL
// for a specific (RealmName, ClientID) combination.
// ClientID can be "*" to match all clients within a realm.
type TokenTTLOverride struct {
	RealmName       string
	ClientID        string
	AccessTokenTTL  int64
	RefreshTokenTTL int64
}

// tokenTTLKey is the lookup key for pre-computed TTL override map.
type tokenTTLKey struct {
	RealmName string
	ClientID  string
}

// ConfidentialClientSecretExemption exempts a confidential client from
// client_secret verification on the specified realm (exact match only).
type ConfidentialClientSecretExemption struct {
	RealmName string
	ClientID  string
}

// IntrospectAllowedAppCode grants an AppCode access to the introspect
// endpoint for a specific realm (exact match only).
type IntrospectAllowedAppCode struct {
	RealmName string
	AppCode   string
}

// ExternalTokenAdapter is the service that resolves third-party tokens on
// bkauth's behalf. Everything vendor-specific lives behind it; bkauth only
// decides, by format, which tokens to forward, and speaks one protocol to it.
type ExternalTokenAdapter struct {
	Enabled bool
	// Endpoint is the adapter's introspection URL.
	Endpoint string
	// ClientID and ClientSecret authenticate bkauth with HTTP Basic. They are
	// dedicated to the adapter rather than bkauth's own app secret, so a leak on
	// either side stays between the two.
	ClientID     string
	ClientSecret string
	// Timeout is in seconds (default: 2).
	Timeout int

	// TokenPrefixes are the formats forwarded to the adapter.
	TokenPrefixes []string
	// MaxTokenLength bounds the tokens forwarded (default: 256).
	MaxTokenLength int
}

// OAuth holds OAuth 2.0 protocol-specific configuration.
type OAuth struct {
	// AccessTokenTTL is the lifetime of access token in seconds (default: 7200)
	AccessTokenTTL int64
	// RefreshTokenTTL is the lifetime of refresh token in seconds (default: 2592000)
	RefreshTokenTTL int64
	// DCREnabled indicates whether Dynamic Client Registration is enabled
	DCREnabled bool
	// DefaultRealmName is used for backward-compatible endpoints that don't specify a realm.
	DefaultRealmName string
	// IntrospectAllowedAppCodes controls which AppCodes may call the introspect
	// endpoint, on a per-realm basis (exact match only).
	// If empty, all requests are denied.
	IntrospectAllowedAppCodes []IntrospectAllowedAppCode
	// ConfidentialClientSecretExemptions exempts specific confidential clients
	// from client_secret verification on a per-(Realm, ClientID) basis (exact match only).
	// Default: empty (all confidential clients must provide client_secret).
	ConfidentialClientSecretExemptions []ConfidentialClientSecretExemption
	// TokenTTLOverrides allows per-(realm, clientID) TTL configuration.
	// Lookup priority: exact (realm, clientID) > realm wildcard (realm, "*") > global default.
	TokenTTLOverrides []TokenTTLOverride
	// ExternalTokenAdapter resolves third-party tokens for introspection.
	// Default: disabled (native tokens only).
	ExternalTokenAdapter ExternalTokenAdapter

	// tokenTTLMap is pre-computed in Load() for O(1) lookups.
	tokenTTLMap map[tokenTTLKey]*TokenTTLOverride
	// secretExemptMap is pre-computed in Load() for O(1) lookups.
	secretExemptMap map[ConfidentialClientSecretExemption]struct{}
	// introspectAllowedMap is pre-computed in Load() for O(1) lookups.
	introspectAllowedMap map[IntrospectAllowedAppCode]struct{}
}

// load fills the defaults, pre-computes the lookup maps and validates the
// external token adapter.
func (o *OAuth) load() error {
	if o.AccessTokenTTL == 0 {
		o.AccessTokenTTL = defaultAccessTokenTTL
	}
	if o.RefreshTokenTTL == 0 {
		o.RefreshTokenTTL = defaultRefreshTokenTTL
	}

	o.tokenTTLMap = make(map[tokenTTLKey]*TokenTTLOverride, len(o.TokenTTLOverrides))
	for i := range o.TokenTTLOverrides {
		ov := &o.TokenTTLOverrides[i]
		o.tokenTTLMap[tokenTTLKey{RealmName: ov.RealmName, ClientID: ov.ClientID}] = ov
	}

	o.secretExemptMap = make(map[ConfidentialClientSecretExemption]struct{}, len(o.ConfidentialClientSecretExemptions))
	for _, ex := range o.ConfidentialClientSecretExemptions {
		o.secretExemptMap[ex] = struct{}{}
	}

	o.introspectAllowedMap = make(map[IntrospectAllowedAppCode]struct{}, len(o.IntrospectAllowedAppCodes))
	for _, entry := range o.IntrospectAllowedAppCodes {
		o.introspectAllowedMap[entry] = struct{}{}
	}

	return o.ExternalTokenAdapter.load()
}

// ResolveTokenTTL returns the effective (accessTokenTTL, refreshTokenTTL) for the
// given realm and clientID. Lookup priority:
//  1. Exact match: (realmName, clientID)
//  2. Realm wildcard: (realmName, "*")
//  3. Global defaults: OAuth.AccessTokenTTL / OAuth.RefreshTokenTTL
//
// Within each level, only non-zero override values replace the inherited value.
func (o *OAuth) ResolveTokenTTL(realmName, clientID string) (accessTTL, refreshTTL int64) {
	accessTTL = o.AccessTokenTTL
	refreshTTL = o.RefreshTokenTTL

	if o.tokenTTLMap == nil {
		return accessTTL, refreshTTL
	}

	if ov, ok := o.tokenTTLMap[tokenTTLKey{RealmName: realmName, ClientID: "*"}]; ok {
		if ov.AccessTokenTTL > 0 {
			accessTTL = ov.AccessTokenTTL
		}
		if ov.RefreshTokenTTL > 0 {
			refreshTTL = ov.RefreshTokenTTL
		}
	}

	if ov, ok := o.tokenTTLMap[tokenTTLKey{RealmName: realmName, ClientID: clientID}]; ok {
		if ov.AccessTokenTTL > 0 {
			accessTTL = ov.AccessTokenTTL
		}
		if ov.RefreshTokenTTL > 0 {
			refreshTTL = ov.RefreshTokenTTL
		}
	}

	return accessTTL, refreshTTL
}

// IsIntrospectAllowed reports whether the given appCode is allowed to call
// the introspect endpoint for the specified realm.
// Returns false when no entries are configured (deny by default).
func (o *OAuth) IsIntrospectAllowed(realmName, appCode string) bool {
	_, ok := o.introspectAllowedMap[IntrospectAllowedAppCode{RealmName: realmName, AppCode: appCode}]
	return ok
}

// IsClientSecretExempt reports whether the given (realmName, clientID) is exempt
// from client_secret verification (exact match only).
func (o *OAuth) IsClientSecretExempt(realmName, clientID string) bool {
	_, ok := o.secretExemptMap[ConfidentialClientSecretExemption{RealmName: realmName, ClientID: clientID}]
	return ok
}

// IsAcceptedTokenFormat reports whether raw has a format the adapter is
// configured for. Only such tokens may leave bkauth; anything else is never
// sent as a probe. Like oauth.IsAcceptedTokenFormat it is a pure string check:
// a token that passes may still be unknown, expired or revoked.
func (a *ExternalTokenAdapter) IsAcceptedTokenFormat(raw string) bool {
	if !a.Enabled || len(raw) > a.MaxTokenLength {
		return false
	}
	for _, prefix := range a.TokenPrefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}
	return false
}

// load fills the defaults and validates an enabled adapter.
func (a *ExternalTokenAdapter) load() error {
	if !a.Enabled {
		return nil
	}
	if a.MaxTokenLength == 0 {
		a.MaxTokenLength = defaultExternalTokenMaxLength
	}
	if a.Timeout == 0 {
		a.Timeout = defaultExternalTokenAdapterTimeout
	}

	if err := a.validate(); err != nil {
		return fmt.Errorf("oauth.externalTokenAdapter: %w", err)
	}
	return nil
}

func (a *ExternalTokenAdapter) validate() error {
	u, err := url.Parse(a.Endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("endpoint %q must be an absolute http(s) URL", a.Endpoint)
	}
	// The endpoint is quoted in logs and errors; credentials embedded in it would
	// travel along. They belong in clientID / clientSecret.
	if u.User != nil {
		return errors.New("endpoint must not carry userinfo")
	}
	if a.ClientID == "" || a.ClientSecret == "" {
		return errors.New("clientID and clientSecret are required")
	}
	// HTTP Basic splits user-id from password at the first colon (RFC 7617), so
	// the adapter would read a different client id than the one configured.
	if strings.Contains(a.ClientID, ":") {
		return fmt.Errorf("clientID %q must not contain ':'", a.ClientID)
	}
	if a.Timeout < 0 {
		return fmt.Errorf("timeout %d must not be negative", a.Timeout)
	}

	if len(a.TokenPrefixes) == 0 {
		return errors.New("tokenPrefixes is required")
	}
	for i, prefix := range a.TokenPrefixes {
		if prefix == "" {
			return fmt.Errorf("tokenPrefixes[%d] must not be empty", i)
		}
		// Native tokens are bk[orps]_...; reserving the whole bk namespace keeps
		// a native token, even a malformed one, from ever being forwarded,
		// whatever formats come later. A prefix of "bk" itself (e.g. "b") would
		// claim the namespace just as one starting with it would.
		lower := strings.ToLower(prefix)
		if strings.HasPrefix(lower, reservedNativeTokenPrefix) || strings.HasPrefix(reservedNativeTokenPrefix, lower) {
			return fmt.Errorf("tokenPrefixes[%d] %q overlaps the %q namespace reserved for native tokens",
				i, prefix, reservedNativeTokenPrefix)
		}
		if len(prefix) < minExternalTokenPrefixLength {
			return fmt.Errorf("tokenPrefixes[%d] %q must be at least %d characters",
				i, prefix, minExternalTokenPrefixLength)
		}
		if a.MaxTokenLength <= len(prefix) {
			return fmt.Errorf("maxTokenLength %d must exceed the length of prefix %q", a.MaxTokenLength, prefix)
		}
	}
	return nil
}
