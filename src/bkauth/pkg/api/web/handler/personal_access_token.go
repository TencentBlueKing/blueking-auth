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
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/cache/impls"
	"bkauth/pkg/config"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

// personalTokenOwner bundles the caller identity taken from the login session and
// the realm from the path. sub/username/tenant_id are NEVER read from the request
// body — a user cannot mint a token in someone else's name.
type personalTokenOwner struct {
	realmName string
	sub       string
	username  string
	tenantID  string
}

func getPersonalTokenOwner(c *gin.Context) personalTokenOwner {
	return personalTokenOwner{
		realmName: util.GetRealmName(c),
		sub:       util.GetSub(c),
		username:  util.GetUsername(c),
		tenantID:  util.GetTenantID(c),
	}
}

func personalTokenPolicy(cfg *config.Config) types.PersonalTokenPolicy {
	return types.PersonalTokenPolicy{
		MaxTTL:           cfg.PersonalToken.MaxTTL,
		MaxActivePerUser: cfg.PersonalToken.MaxActivePerUser,
	}
}

func parsePersonalTokenID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		webJSONError(c, http.StatusBadRequest, webErrCodeInvalidArgument, "invalid token id")
		return 0, false
	}
	return id, true
}

// resolveAudiences validates the resource selector against the realm and extracts
// the audiences, then applies the write constraints. Writes only check format; the
// realm is the per-realm validation boundary (design 13.6).
func resolveAudiences(ctx context.Context, c *gin.Context, realmName, resource string) ([]string, bool) {
	realm := oauth.GetRealm(realmName)
	if realm == nil {
		webJSONError(c, http.StatusNotFound, webErrCodeNotFound, "unknown realm")
		return nil, false
	}
	if err := realm.ValidateResource(ctx, resource); err != nil {
		webJSONError(c, http.StatusBadRequest, webErrCodeInvalidArgument, "invalid resource: "+err.Error())
		return nil, false
	}
	audiences, err := realm.ExtractAudiences(ctx, resource)
	if err != nil {
		webJSONError(c, http.StatusBadRequest, webErrCodeInvalidArgument, "invalid resource: "+err.Error())
		return nil, false
	}
	if err := validateAudiences(audiences); err != nil {
		webJSONError(c, http.StatusBadRequest, webErrCodeInvalidArgument, err.Error())
		return nil, false
	}
	return audiences, true
}

// mapPersonalTokenError maps a service domain error to the client-facing response.
// Returns true when it wrote an error (i.e. err was non-nil).
func mapPersonalTokenError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrPersonalTokenNotFound):
		webJSONError(c, http.StatusNotFound, webErrCodeNotFound, "personal access token not found")
	case errors.Is(err, service.ErrPersonalTokenInvalidTTL):
		webJSONError(c, http.StatusBadRequest, webErrCodeInvalidArgument,
			"expires_in is invalid or exceeds the maximum allowed lifetime")
	case errors.Is(err, service.ErrPersonalTokenQuotaExceeded):
		// The message must explain that expired tokens keep occupying no quota but
		// do not vanish on their own, or a user seeing "limit reached" next to a
		// mostly-empty active list is baffled (design 4.3).
		webJSONError(c, http.StatusConflict, webErrCodeConflict,
			"active token quota reached; revoked/expired tokens do not count toward the quota "+
				"but are not removed automatically — revoke or wait for cleanup, or reuse an existing token")
	default:
		util.SetError(c, err)
		webJSONError(c, http.StatusInternalServerError, webErrCodeInternal, "internal error")
	}
	return true
}

// NewListPersonalTokenHandler handles GET .../personal-tokens?state=active|inactive
func NewListPersonalTokenHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var query listPersonalTokenQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			webJSONError(c, http.StatusBadRequest, webErrCodeInvalidArgument,
				"invalid state filter, must be one of: active, inactive")
			return
		}

		ctx := c.Request.Context()
		owner := getPersonalTokenOwner(c)

		svc := service.NewPersonalAccessTokenService()
		tokens, err := svc.ListByOwner(ctx, owner.realmName, owner.sub, query.State)
		if mapPersonalTokenError(c, err) {
			return
		}

		results := make([]personalTokenResponse, 0, len(tokens))
		for _, t := range tokens {
			results = append(results, newPersonalTokenResponse(t))
		}
		webJSONSuccess(c, results)
	}
}

// NewCreatePersonalTokenHandler handles POST .../personal-tokens
func NewCreatePersonalTokenHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createPersonalTokenSerializer
		if err := c.ShouldBindJSON(&body); err != nil {
			webJSONError(c, http.StatusBadRequest, webErrCodeInvalidArgument, "invalid request body")
			return
		}

		ctx := c.Request.Context()
		owner := getPersonalTokenOwner(c)

		audiences, ok := resolveAudiences(ctx, c, owner.realmName, body.Resource)
		if !ok {
			return
		}

		expiresIn := body.ExpiresIn
		if expiresIn == 0 {
			expiresIn = cfg.PersonalToken.DefaultTTL
		}

		svc := service.NewPersonalAccessTokenService()
		created, err := svc.Create(ctx, types.CreatePersonalAccessTokenInput{
			RealmName:   owner.realmName,
			TenantID:    owner.tenantID,
			Sub:         owner.sub,
			Username:    owner.username,
			Name:        body.Name,
			Description: body.Description,
			Audience:    audiences,
			ExpiresIn:   expiresIn,
		}, personalTokenPolicy(cfg))
		if mapPersonalTokenError(c, err) {
			return
		}

		webJSONSuccess(c, createPersonalTokenResponse{
			personalTokenResponse: newPersonalTokenResponse(created.PersonalAccessToken),
			Token:                 created.Token,
		})
	}
}

// NewGetPersonalTokenHandler handles GET .../personal-tokens/:id
func NewGetPersonalTokenHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parsePersonalTokenID(c)
		if !ok {
			return
		}

		ctx := c.Request.Context()
		owner := getPersonalTokenOwner(c)

		svc := service.NewPersonalAccessTokenService()
		token, err := svc.GetByIDAndSub(ctx, owner.realmName, id, owner.sub)
		if mapPersonalTokenError(c, err) {
			return
		}
		webJSONSuccess(c, newPersonalTokenResponse(token))
	}
}

// NewUpdatePersonalTokenHandler handles PUT .../personal-tokens/:id
func NewUpdatePersonalTokenHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parsePersonalTokenID(c)
		if !ok {
			return
		}

		var body updatePersonalTokenSerializer
		if err := c.ShouldBindJSON(&body); err != nil {
			webJSONError(c, http.StatusBadRequest, webErrCodeInvalidArgument, "invalid request body")
			return
		}

		ctx := c.Request.Context()
		owner := getPersonalTokenOwner(c)

		audiences, ok := resolveAudiences(ctx, c, owner.realmName, body.Resource)
		if !ok {
			return
		}

		svc := service.NewPersonalAccessTokenService()
		tokenHash, err := svc.Update(ctx, types.UpdatePersonalAccessTokenInput{
			RealmName:   owner.realmName,
			Sub:         owner.sub,
			ID:          id,
			Name:        body.Name,
			Description: body.Description,
			Audience:    audiences,
		})
		if mapPersonalTokenError(c, err) {
			return
		}
		_ = impls.DeletePersonalAccessTokenCache(ctx, tokenHash)

		respondUpdatedPersonalToken(c, svc, owner, id)
	}
}

// NewRenewPersonalTokenHandler handles POST .../personal-tokens/:id/renew
func NewRenewPersonalTokenHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parsePersonalTokenID(c)
		if !ok {
			return
		}

		var body renewPersonalTokenSerializer
		if err := c.ShouldBindJSON(&body); err != nil {
			webJSONError(c, http.StatusBadRequest, webErrCodeInvalidArgument, "invalid request body")
			return
		}

		expiresIn := body.ExpiresIn
		if expiresIn == 0 {
			expiresIn = cfg.PersonalToken.DefaultTTL
		}

		ctx := c.Request.Context()
		owner := getPersonalTokenOwner(c)

		svc := service.NewPersonalAccessTokenService()
		tokenHash, err := svc.Renew(ctx, owner.realmName, id, owner.sub, expiresIn, personalTokenPolicy(cfg))
		if mapPersonalTokenError(c, err) {
			return
		}
		_ = impls.DeletePersonalAccessTokenCache(ctx, tokenHash)

		respondUpdatedPersonalToken(c, svc, owner, id)
	}
}

// NewRevokePersonalTokenHandler handles POST .../personal-tokens/:id/revoke
func NewRevokePersonalTokenHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parsePersonalTokenID(c)
		if !ok {
			return
		}

		ctx := c.Request.Context()
		owner := getPersonalTokenOwner(c)

		svc := service.NewPersonalAccessTokenService()
		tokenHash, err := svc.Revoke(ctx, owner.realmName, id, owner.sub)
		if mapPersonalTokenError(c, err) {
			return
		}
		_ = impls.DeletePersonalAccessTokenCache(ctx, tokenHash)

		respondUpdatedPersonalToken(c, svc, owner, id)
	}
}

// respondUpdatedPersonalToken re-reads the token after a successful mutation and
// returns its fresh management view. The extra read keeps the response honest
// (e.g. renewed expires_at, revoked status) at the cost of one small indexed query.
func respondUpdatedPersonalToken(
	c *gin.Context, svc service.PersonalAccessTokenService, owner personalTokenOwner, id int64,
) {
	token, err := svc.GetByIDAndSub(c.Request.Context(), owner.realmName, id, owner.sub)
	if mapPersonalTokenError(c, err) {
		return
	}
	webJSONSuccess(c, newPersonalTokenResponse(token))
}
