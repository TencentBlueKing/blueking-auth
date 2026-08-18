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
	"errors"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/api/common"
	"bkauth/pkg/oauth"
	"bkauth/pkg/util"
)

// NewGrantableResourceTypeListHandler creates a handler for
// GET /realms/:realm_name/personal-tokens/grantable-resource-types.
//
// The types are static metadata and are split from the entries themselves
// because the frontend needs them before it knows how many panels to render, and
// because each type pages independently: one response carrying two cursors would
// have them interfere.
//
// It also supplies the label for the type code on a stored token's resources, so
// the token list needs this response too, not just the creation form.
func NewGrantableResourceTypeListHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		realm := oauth.GetRealm(util.GetRealmName(c))

		util.WebSuccess(c, realm.GrantableResourceTypes())
	}
}

// NewGrantableResourceListHandler creates a handler for
// GET /realms/:realm_name/personal-tokens/grantable-resources.
func NewGrantableResourceListHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		realm := oauth.GetRealm(util.GetRealmName(c))

		var req grantableResourceQueryRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			util.WebBindError(c, err)
			return
		}

		keywords, err := req.keywords()
		if err != nil {
			util.WebInvalidArgumentError(c, err.Error())
			return
		}

		page, err := realm.ListGrantableResource(
			c.Request.Context(), util.GetTenantID(c), oauth.GrantableResourceQuery{
				Type:     req.Type,
				Keywords: keywords,
				Limit:    req.limit(),
				Offset:   req.offset(),
			})
		if err != nil {
			// The realm rejects a type or a keyword level it does not have, and
			// the caller cannot tell which exist without asking; that is a bad
			// request, not a server fault. Everything else is an upstream failure
			// the client can do nothing about.
			if errors.Is(err, oauth.ErrUnknownGrantableResourceType) {
				util.WebInvalidArgumentError(c,
					"type must be one of those returned by grantable-resource-types")
				return
			}

			if errors.Is(err, oauth.ErrUnknownGrantableResourceLevel) {
				util.WebInvalidArgumentError(c, "each keyword must name one of this type's levels")
				return
			}

			util.SetError(c, err)
			util.WebInternalError(c, "failed to list grantable resources")
			return
		}

		// Allocated with a zero length so an empty page serializes as [] and not
		// null.
		results := page.Results
		if results == nil {
			results = []oauth.GrantableResource{}
		}

		util.WebSuccess(c, common.PaginatedResponse{Count: page.Count, Results: results})
	}
}
