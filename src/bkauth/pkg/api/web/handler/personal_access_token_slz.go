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

package handler

import (
	"strings"

	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

// createPersonalAccessTokenRequest is the body of the create endpoint. The
// length bounds mirror the column widths so an over-long value is rejected here
// rather than truncated by MySQL.
//
// ExpiresAt is an absolute Unix-second timestamp rather than a lifetime. The
// client picks a date; sending a duration would have it computed against the
// client's clock and re-anchored to the server's, storing an expiry a round trip
// off from the date the user chose.
//
// It deliberately carries no binding rule. Its bounds are policy — the window is
// (now, now+MaxTTL] on the server's clock — the service enforces them, and
// duplicating them here would mean an omitted field answers "expires_at is
// required" while an out-of-window one answers with the max_expires_at bound:
// two spellings of one rule. Letting a missing field arrive as 0 routes both
// through ErrPersonalTokenInvalidExpiresAt, whose message names the window.
type createPersonalAccessTokenRequest struct {
	Name        string   `json:"name" binding:"required,max=64"`
	Description string   `json:"description" binding:"omitempty,max=255"`
	Audience    []string `json:"audience" binding:"required,min=1,max=20,dive,max=128"`
	ExpiresAt   int64    `json:"expires_at"`
}

// updatePersonalAccessTokenRequest is the body of the update endpoint. Replace
// semantics rather than patch, and no field is exempt: a client that omits
// audience gets a validation error instead of silently clearing it, and one that
// omits expires_at is not silently handed the stored expiry back.
//
// ExpiresAt carries no binding rule for the same reason it carries none on
// create: its bounds are policy, so an omitted field arriving as 0 falls through
// to the window check instead of answering one rule with two messages. What
// keeps that strictness workable is a service-side exemption, not an optional
// field — resubmitting the token's own expiry unchanged is always accepted, even
// when it is in the past, so echoing back the object as given never fails.
//
// Spelled out separately from createPersonalAccessTokenRequest rather than
// shared through embedding. The two shapes now coincide, but they answer to
// different contracts, and the audience one is still open — if create ever takes
// a realm resource that the backend resolves into audiences, they move apart
// again.
type updatePersonalAccessTokenRequest struct {
	Name        string   `json:"name" binding:"required,max=64"`
	Description string   `json:"description" binding:"omitempty,max=255"`
	Audience    []string `json:"audience" binding:"required,min=1,max=20,dive,max=128"`
	ExpiresAt   int64    `json:"expires_at"`
}

// renewPersonalAccessTokenRequest is the body of the renew endpoint.
//
// ExpiresAt carries no binding rule, for the reason it carries none on create:
// its bounds are policy the service already enforces, and duplicating them here
// would answer a missing field and an out-of-window one with two different
// messages for one rule. Unlike on update, 0 is not "leave it alone" here —
// there is nothing else this endpoint could do — so it falls through to the
// window check and is rejected.
type renewPersonalAccessTokenRequest struct {
	ExpiresAt int64 `json:"expires_at"`
}

// personalAccessTokenResponse is the management view of a token on the wire.
//
// Two fields of types.PersonalAccessToken are deliberately not here. realm_name
// is already in the request path, and echoing it invites a client to read the
// realm from the body instead. scope is a column that stays empty until
// fine-grained scopes exist, and adding a field to a response later is
// backward-compatible while removing one is not.
//
// Times are Unix seconds, matching the service view; the frontend derives the
// lifecycle state from revoked and expires_at rather than being handed one, so
// the state it renders is the same one introspection enforces.
type personalAccessTokenResponse struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TokenMask   string   `json:"token_mask"`
	Audience    []string `json:"audience"`
	// Resources is the realm's rendering of Audience: one entry per token, in the
	// same order, each carrying in its own audience field the token it renders.
	// Audience stays alongside it because that is what a write takes, and a
	// client should submit back exactly what it was given rather than collect
	// the strings out of here.
	//
	// The entries are flat, one per token. Grouping them by type is the client's
	// to do, the type labels being one static response away, and the level below
	// a type -- the gateway an API belongs to -- is not represented: it groups
	// the catalog for picking rather than being a level of the grant itself.
	//
	// Null when the realm could not resolve it. The tokens are still in Audience,
	// so a client that gets no rendering can show them raw rather than nothing.
	Resources []oauth.AudienceDisplay `json:"resources"`
	ExpiresAt int64                   `json:"expires_at"`
	Revoked   bool                    `json:"revoked"`
	// RevokedAt is omitted rather than sent as 0 on a live token, so a client
	// cannot mistake the zero value for a real epoch timestamp.
	RevokedAt int64 `json:"revoked_at,omitempty"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// createdPersonalAccessTokenResponse is returned by the create endpoint alone.
// Token carries the plaintext, which the service hands out exactly once.
type createdPersonalAccessTokenResponse struct {
	ID    int64  `json:"id"`
	Token string `json:"token"`
}

// normalizeAudience trims every entry, drops the blank ones and de-duplicates,
// preserving the caller's order.
//
// The audience contract puts this whole rule in the serializer: the service does
// not inspect audience, and there is no authoritative catalog to check it
// against, so the only guarantees the stored value carries are the ones made
// here. The result may be empty even when binding passed, because an entry of
// pure whitespace satisfies min=1 but survives nothing past the trim.
func normalizeAudience(audience []string) []string {
	seen := make(map[string]bool, len(audience))
	normalized := make([]string, 0, len(audience))

	for _, a := range audience {
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		normalized = append(normalized, a)
	}

	return normalized
}

func toPersonalAccessTokenResponse(t types.PersonalAccessToken) personalAccessTokenResponse {
	// A nil slice marshals to null, which would force every client to handle two
	// spellings of "no audience".
	audience := t.Audience
	if audience == nil {
		audience = []string{}
	}

	return personalAccessTokenResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		TokenMask:   t.TokenMask,
		Audience:    audience,
		ExpiresAt:   t.ExpiresAt,
		Revoked:     t.Revoked,
		RevokedAt:   t.RevokedAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
