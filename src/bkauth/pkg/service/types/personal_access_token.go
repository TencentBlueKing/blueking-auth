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

// PersonalAccessToken is the management view of a personal access token, returned
// by the list and detail endpoints. It never carries the plaintext or the hash.
//
// Lifecycle state is not a field here: it is fully determined by Revoked and
// ExpiresAt, and storing or precomputing it would only create a second source of
// truth to keep in sync with the one introspection actually enforces.
//
// Times are Unix seconds, as everywhere else the service hands a row outward
// (AccessKeyWithCreatedAt, OAuthClient, ResolvedAccessToken). An instant with no
// location and no serialisation choice cannot be misread by a client, and it
// keeps the wire format independent of the DSN's loc setting. The sub-second
// precision the DATETIME(6) columns carry is dropped here on purpose: it matters
// for the expires_at guard in SQL, not for a view meant to be displayed.
//
// Fields follow the DAO row order so the two can be read side by side when
// checking that the mapping covers everything.
type PersonalAccessToken struct {
	ID          int64
	Name        string
	Description string
	TokenMask   string
	RealmName   string
	Audience    []string
	Scope       string
	ExpiresAt   int64
	Revoked     bool
	// RevokedAt is 0 when the token was never revoked; Revoked is the flag to
	// branch on, which is why this is a plain int64 rather than a pointer.
	RevokedAt int64
	CreatedAt int64
	UpdatedAt int64
}

// CreatedPersonalAccessToken is returned by Create. Token holds the plaintext,
// handed back exactly once at creation; only its hash is persisted, so it cannot
// be recovered afterwards.
//
// These are the only two things the caller cannot know on its own: the ID comes
// from AUTO_INCREMENT and the token from the CSPRNG. Everything else in the
// management view is either an echo of the request or a column the database
// defaulted and the service never read back. ExpiresAt is left out because the
// caller supplied it verbatim and it is stored verbatim. Add it back the moment
// that stops being true, e.g. if an over-long expiry is ever clamped instead of
// rejected, since a caller echoing its own input would then advertise an expiry
// the token does not have.
type CreatedPersonalAccessToken struct {
	ID    int64
	Token string
}

// CreatePersonalAccessTokenInput carries the fields needed to mint a personal
// access token. The two groups below have different provenance, and the split is
// load-bearing rather than cosmetic.
type CreatePersonalAccessTokenInput struct {
	// Identity of the token's owner, resolved from the login session. Binding any
	// of these from the request body would let a caller mint a token for someone
	// else.
	RealmName string
	TenantID  string
	Sub       string
	Username  string

	// Profile and lifetime, supplied by the request body and validated by the
	// handler's serializer and the service's policy checks.
	Name        string
	Description string
	Audience    []string
	// ExpiresAt is the absolute expiry in Unix seconds, stored verbatim rather
	// than derived from a duration. The client picks a date, so a duration would
	// be computed from the client's clock and then re-anchored to the server's,
	// landing the row a round trip away from the date the user actually chose.
	//
	// The window it must fall in is the server's (now, now+MaxTTL], so a client
	// whose clock is off gets a rejection naming the ceiling rather than a token
	// that silently expires at the wrong time.
	ExpiresAt int64
}

// PersonalTokenPolicy carries the per-request policy knobs the service enforces.
// It is passed in by the handler (built from config.PersonalToken) so the service
// stays free of config, mirroring how TokenIssuancePolicy is threaded through the
// OAuth token service.
type PersonalTokenPolicy struct {
	// MaxTTL is how far past now an expiry may be set, in seconds. It bounds
	// creation, renewal and the update path that carries an expiry.
	MaxTTL int64
	// MaxActivePerUser bounds active tokens per (realm, sub).
	MaxActivePerUser int
}

// UpdatePersonalAccessTokenInput carries the new values for the mutable profile
// fields, all of them from the request body. Which row they are written to is not
// part of this struct: the service takes (realmName, id, sub) as arguments, the
// same shape as Get / Renew / Revoke, so the row selector can never be confused
// with the payload.
type UpdatePersonalAccessTokenInput struct {
	Name        string
	Description string
	Audience    []string
	// ExpiresAt is mandatory like the other three -- PUT replaces the whole
	// object. Exactly one value escapes the (now, now+MaxTTL] window: the expiry
	// the row already holds, echoed back unchanged. Without that exemption a
	// client resubmitting the object it was handed would be rejected whenever the
	// token had already expired, which is when renaming it matters most.
	ExpiresAt int64
}
