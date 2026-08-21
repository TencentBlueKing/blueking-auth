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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"bkauth/pkg/cache/impls"
	"bkauth/pkg/config"
	"bkauth/pkg/logging"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

// personalTokenPolicy builds the policy the service enforces per request. Called
// once per handler construction: the values come from config and cannot change
// while the process runs.
func personalTokenPolicy(cfg *config.Config) types.PersonalTokenPolicy {
	return types.PersonalTokenPolicy{
		MaxTTL:           cfg.PersonalToken.MaxTTL,
		MaxActivePerUser: cfg.PersonalToken.MaxActivePerUser,
	}
}

// NewPersonalAccessTokenCreateHandler creates a handler for
// POST /realms/:realm_name/personal-tokens.
//
// The owner identity is taken from the login session and the realm from the
// path, never from the body: binding any of them from the payload would let a
// caller mint a token for someone else.
func NewPersonalAccessTokenCreateHandler(cfg *config.Config) gin.HandlerFunc {
	policy := personalTokenPolicy(cfg)

	return func(c *gin.Context) {
		var req createPersonalAccessTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			util.WebBindError(c, err)
			return
		}

		name, ok := normalizeNameOrAbort(c, req.Name)
		if !ok {
			return
		}

		audience, ok := normalizeAudienceOrAbort(c, req.Audience)
		if !ok {
			return
		}

		svc := service.NewPersonalAccessTokenService()
		created, err := svc.Create(c.Request.Context(), types.CreatePersonalAccessTokenInput{
			RealmName:   util.GetRealmName(c),
			TenantID:    util.GetTenantID(c),
			Sub:         util.GetSub(c),
			Username:    util.GetUsername(c),
			Name:        name,
			Description: req.Description,
			Audience:    audience,
			ExpiresAt:   req.ExpiresAt,
		}, policy)
		if err != nil {
			handlePersonalAccessTokenError(c, err, policy)
			return
		}

		// The plaintext is handed out here and nowhere else; only its hash is
		// stored. The audit and web loggers both record this response body, so
		// logger.web and logger.audit carry a desensitization rule for data.token.
		util.WebSuccess(c, createdPersonalAccessTokenResponse{
			ID:    created.ID,
			Token: created.Token,
		})
	}
}

// NewPersonalAccessTokenListHandler creates a handler for
// GET /realms/:realm_name/personal-tokens.
//
// Returns every token the owner holds in the realm, in all lifecycle states.
// Retention bounds the row count, so the whole list is cheap to return, and the
// client groups it by revoked / expires_at itself: a server-side "active" filter
// would fold expired and revoked into one bucket and lose the distinction the UI
// has to draw.
func NewPersonalAccessTokenListHandler(cfg *config.Config) gin.HandlerFunc {
	policy := personalTokenPolicy(cfg)

	return func(c *gin.Context) {
		svc := service.NewPersonalAccessTokenService()
		tokens, err := svc.ListByOwner(c.Request.Context(), util.GetRealmName(c), util.GetSub(c))
		if err != nil {
			handlePersonalAccessTokenError(c, err, policy)
			return
		}

		// Resolved for the whole page in one pass so the realm can collapse the
		// upstream lookups the tokens have in common.
		displays := resolveAudienceDisplays(c, tokens)

		// Allocated with a zero length rather than declared nil, so an owner with
		// no tokens is answered with [] instead of null.
		responses := make([]personalAccessTokenResponse, 0, len(tokens))
		for _, t := range tokens {
			response := toPersonalAccessTokenResponse(t)
			response.Resources = audienceEntries(t.Audience, displays)
			responses = append(responses, response)
		}

		util.WebSuccess(c, responses)
	}
}

// NewPersonalAccessTokenGetHandler creates a handler for
// GET /realms/:realm_name/personal-tokens/:id.
func NewPersonalAccessTokenGetHandler(cfg *config.Config) gin.HandlerFunc {
	policy := personalTokenPolicy(cfg)

	return func(c *gin.Context) {
		id, ok := parsePersonalAccessTokenID(c)
		if !ok {
			return
		}

		svc := service.NewPersonalAccessTokenService()
		token, err := svc.GetByIDAndSub(c.Request.Context(), util.GetRealmName(c), id, util.GetSub(c))
		if err != nil {
			handlePersonalAccessTokenError(c, err, policy)
			return
		}

		displays := resolveAudienceDisplays(c, []types.PersonalAccessToken{token})

		response := toPersonalAccessTokenResponse(token)
		response.Resources = audienceEntries(token.Audience, displays)

		util.WebSuccess(c, response)
	}
}

// resolveAudienceDisplays renders every audience granted anywhere on the page,
// keyed by the token each rendering is of.
//
// A failure yields no displays rather than an error: the raw tokens are already
// in the response, so a client that gets no rendering can show them as they are.
// Failing the request instead would take a page the user can otherwise act on —
// revoke, renew, rename — and replace it with an error over a cosmetic lookup.
func resolveAudienceDisplays(
	c *gin.Context,
	tokens []types.PersonalAccessToken,
) map[string]oauth.AudienceDisplay {
	audiences := make([]string, 0, len(tokens))
	for _, t := range tokens {
		audiences = append(audiences, t.Audience...)
	}
	// Deduplicated because a token renders the same way wherever it appears, and
	// a page whose tokens grant much the same things is the normal case.
	audiences = util.Deduplicate(audiences)
	if len(audiences) == 0 {
		return nil
	}

	realmName := util.GetRealmName(c)
	displays, err := oauth.GetRealm(realmName).ResolveAudienceDisplays(
		c.Request.Context(), util.GetTenantID(c), audiences)
	if err != nil {
		logging.GetWebLogger().Warn("failed to resolve personal access token audiences",
			zap.Error(err), zap.String("realm_name", realmName))
		return nil
	}
	return displays
}

// audienceEntries picks one token's renderings out of the page's, in the order
// that token stores them. The order is this handler's to keep: the realm was
// handed the page's audiences flat and answers keyed, knowing nothing of which
// token granted what.
//
// An audience the realm did not render is left out, and a token left with
// nothing gets no entries rather than an empty list, so the response field is
// null in exactly the cases its contract names.
func audienceEntries(
	audiences []string,
	displays map[string]oauth.AudienceDisplay,
) []oauth.AudienceDisplay {
	entries := make([]oauth.AudienceDisplay, 0, len(audiences))
	for _, aud := range audiences {
		if display, ok := displays[aud]; ok {
			entries = append(entries, display)
		}
	}
	if len(entries) == 0 {
		return nil
	}
	return entries
}

// NewPersonalAccessTokenUpdateHandler creates a handler for
// PUT /realms/:realm_name/personal-tokens/:id.
func NewPersonalAccessTokenUpdateHandler(cfg *config.Config) gin.HandlerFunc {
	policy := personalTokenPolicy(cfg)

	return func(c *gin.Context) {
		id, ok := parsePersonalAccessTokenID(c)
		if !ok {
			return
		}

		var req updatePersonalAccessTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			util.WebBindError(c, err)
			return
		}

		name, ok := normalizeNameOrAbort(c, req.Name)
		if !ok {
			return
		}

		audience, ok := normalizeAudienceOrAbort(c, req.Audience)
		if !ok {
			return
		}

		svc := service.NewPersonalAccessTokenService()
		tokenHash, err := svc.Update(c.Request.Context(), util.GetRealmName(c), id, util.GetSub(c),
			types.UpdatePersonalAccessTokenInput{
				Name:        name,
				Description: req.Description,
				Audience:    audience,
				ExpiresAt:   req.ExpiresAt,
			}, policy)
		if err != nil {
			handlePersonalAccessTokenError(c, err, policy)
			return
		}

		// Audience is part of the cached introspection result, so an edit that
		// skipped this would keep introspection answering with the old audience
		// for as long as the entry lives.
		invalidatePersonalAccessTokenCache(c.Request.Context(), id, tokenHash)

		util.WebSuccess(c, gin.H{})
	}
}

// NewPersonalAccessTokenRenewHandler creates a handler for
// POST /realms/:realm_name/personal-tokens/:id/renew.
//
// Renewing an already-expired token is allowed and is the point of the endpoint;
// the service runs the active quota check on that path so a user cannot exceed
// the quota by letting tokens lapse and reviving them.
func NewPersonalAccessTokenRenewHandler(cfg *config.Config) gin.HandlerFunc {
	policy := personalTokenPolicy(cfg)

	return func(c *gin.Context) {
		id, ok := parsePersonalAccessTokenID(c)
		if !ok {
			return
		}

		var req renewPersonalAccessTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			util.WebBindError(c, err)
			return
		}

		svc := service.NewPersonalAccessTokenService()
		tokenHash, err := svc.Renew(c.Request.Context(), util.GetRealmName(c), id, util.GetSub(c),
			req.ExpiresAt, policy)
		if err != nil {
			handlePersonalAccessTokenError(c, err, policy)
			return
		}

		// expires_at is what introspection enforces. Skipping this would leave a
		// renewed token rejected until the cached copy of the old expiry lapsed.
		invalidatePersonalAccessTokenCache(c.Request.Context(), id, tokenHash)

		util.WebSuccess(c, gin.H{})
	}
}

// NewPersonalAccessTokenRevokeHandler creates a handler for
// POST /realms/:realm_name/personal-tokens/:id/revoke.
//
// POST rather than DELETE: revocation is a soft delete. The row stays and keeps
// appearing in the list, because the UI has to tell a revoked token from an
// expired one, and DELETE would promise the resource is gone.
func NewPersonalAccessTokenRevokeHandler(cfg *config.Config) gin.HandlerFunc {
	policy := personalTokenPolicy(cfg)

	return func(c *gin.Context) {
		id, ok := parsePersonalAccessTokenID(c)
		if !ok {
			return
		}

		svc := service.NewPersonalAccessTokenService()
		tokenHash, err := svc.Revoke(c.Request.Context(), util.GetRealmName(c), id, util.GetSub(c))
		if err != nil {
			handlePersonalAccessTokenError(c, err, policy)
			return
		}

		// The one invalidation that is a security control rather than a freshness
		// fix: without it the revoked token keeps passing introspection until the
		// cached entry expires.
		invalidatePersonalAccessTokenCache(c.Request.Context(), id, tokenHash)

		util.WebSuccess(c, gin.H{})
	}
}

// parsePersonalAccessTokenID reads the :id path parameter, writing the error
// response itself and reporting whether the caller should continue.
//
// Ids are AUTO_INCREMENT and therefore always positive. Rejecting zero and
// negatives here keeps a malformed path from reaching the service as a lookup
// that would answer "not found" and hide the fact that the request was
// nonsense.
func parsePersonalAccessTokenID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		util.WebInvalidArgumentError(c, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

// normalizeNameOrAbort trims the name and rejects one left empty, writing the
// error response itself and reporting whether the caller should continue.
//
// binding's `required` only refuses the empty string, so a name of pure
// whitespace arrives here intact. Trimming is not cosmetic now that the name
// carries a uniqueness constraint: " ci" would otherwise be a second spelling of
// a name the owner already holds, indistinguishable from the first in the list
// it is meant to identify a token in. The `max=64` tag still holds afterwards,
// since trimming can only shorten.
func normalizeNameOrAbort(c *gin.Context, raw string) (string, bool) {
	name := strings.TrimSpace(raw)
	if name == "" {
		util.WebInvalidArgumentError(c, "name must not be blank")
		return "", false
	}
	return name, true
}

// normalizeAudienceOrAbort normalizes the audience and checks it against the
// realm, writing the error response itself and reporting whether the caller
// should continue.
//
// A body can satisfy the binding rules and still normalize to nothing — an entry
// of pure whitespace passes min=1 and does not survive the trim — so this check
// cannot be folded into the tags.
//
// The realm check is what keeps the audience column canonical. Its values are
// compared byte for byte by whoever consumes the token, so a second spelling of
// the same grant is a grant that silently never matches; letting one through on
// write would surface much later as a token that mysteriously has no access.
func normalizeAudienceOrAbort(c *gin.Context, raw []string) ([]string, bool) {
	audience := normalizeAudience(raw)
	if len(audience) == 0 {
		util.WebInvalidArgumentError(c, "audience must hold at least one non-blank entry")
		return nil, false
	}

	realmName := util.GetRealmName(c)
	if err := oauth.GetRealm(realmName).ValidateAudiences(c.Request.Context(), audience); err != nil {
		// The realm's message names the offending entry and why it was refused,
		// which is exactly what the user needs to fix their selection, and it
		// reveals nothing beyond the token they just submitted.
		util.WebInvalidArgumentError(c, err.Error())
		return nil, false
	}

	return audience, true
}

// handlePersonalAccessTokenError maps the service's domain errors onto the web
// error envelope.
//
// Anything unrecognised becomes a 500 with a fixed message: the service wraps
// infrastructure failures with layer, query and argument context, none of which
// may reach the client.
func handlePersonalAccessTokenError(c *gin.Context, err error, policy types.PersonalTokenPolicy) {
	switch {
	case errors.Is(err, service.ErrPersonalTokenNotFound):
		// The service overloads this error on purpose: the row may not exist, may
		// belong to another user, or may be in a state the operation forbids. The
		// response must not separate those cases, or it turns into an oracle for
		// other people's token ids.
		util.WebNotFoundError(c, "personal access token not found")

	case errors.Is(err, service.ErrPersonalTokenInvalidExpiresAt):
		// The ceiling has to be computed here rather than left to the client to
		// derive from the TTL: the client would add it to its own clock, which is
		// the very thing that put the value out of the window.
		maxExpiresAt := time.Now().Add(time.Duration(policy.MaxTTL) * time.Second).UTC()
		util.WebInvalidArgumentError(c, fmt.Sprintf(
			"expires_at must be in Unix seconds, after now and no later than %d (%s)",
			maxExpiresAt.Unix(), maxExpiresAt.Format(time.RFC3339)))

	case errors.Is(err, service.ErrPersonalTokenNameConflict):
		// 409 ALREADY_EXISTS: the owner already has a token under this name, and
		// the way out is to pick another or free this one. That is what separates
		// it from the quota's 429, which no edit to this request can satisfy.
		//
		// The message names the states the rule covers, because the conflicting
		// token may well be one the user considers finished with; without that,
		// the rejection reads as pointing at a token that is not there.
		util.WebAlreadyExistsError(c,
			"a personal access token with this name already exists, expired and revoked ones included")

	case errors.Is(err, service.ErrPersonalTokenQuotaExceeded):
		// 429 RESOURCE_EXHAUSTED: the request is well formed, the account is at
		// its active-token quota. 403 would imply the caller is forbidden; 409
		// would imply a conflicting resource rather than a spent allowance.
		util.WebResourceExhaustedError(c, fmt.Sprintf(
			"too many active personal access tokens (at most %d), "+
				"revoke an existing token before creating another",
			policy.MaxActivePerUser))

	default:
		// Hands the error to the logging middleware, which puts it in the audit
		// record and forwards it to Sentry. Without this a 500 here is invisible
		// beyond a status code.
		util.SetError(c, err)
		util.WebInternalError(c, "internal server error")
	}
}

// invalidatePersonalAccessTokenCache drops the introspection cache entry after a
// write.
//
// Failures are logged and swallowed. The write has already committed and cannot
// be rolled back, so surfacing an error would tell the user their revoke did not
// take effect when it did; the cache's five-minute TTL bounds how long a stale
// entry can outlive the write. The token hash stays out of the log line — the id
// is enough to find the row.
func invalidatePersonalAccessTokenCache(ctx context.Context, id int64, tokenHash string) {
	if err := impls.DeletePersonalAccessTokenCache(ctx, tokenHash); err != nil {
		logging.GetSystemLogger().Warn("personal access token cache invalidation fail",
			zap.Error(err), zap.Int64("id", id))
	}
}
