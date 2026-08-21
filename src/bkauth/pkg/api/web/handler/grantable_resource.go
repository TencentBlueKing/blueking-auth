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

// NewGrantableResourceLookupHandler creates a handler for
// GET /realms/:realm_name/personal-tokens/grantable-resources/-/lookup.
//
// It resolves names the user typed into one catalog row, which is how a resource
// the catalog cannot list gets granted: the listing upstream returns public
// objects only, so a private MCP server or API never appears on a page and has to
// be named outright.
//
// The response is a single entry of exactly the shape a page holds, so a client
// adds it to a selection beside the ticked rows without a second code path, and
// submits its audience the same way -- the token grammar stays the backend's.
//
// Nothing is written here, and the audience it hands back is not remembered: a
// caller may still submit an audience that never came from this endpoint, which
// create and update validate syntactically as they always have. That is the
// division on purpose. Existence is confirmed while the form is being filled in,
// where the user can act on the answer, and a token stays editable when its
// upstream is down or its object has since been removed.
func NewGrantableResourceLookupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		realm := oauth.GetRealm(util.GetRealmName(c))

		var req grantableResourceRefRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			util.WebBindError(c, err)
			return
		}

		names, err := req.names()
		if err != nil {
			util.WebInvalidArgumentError(c, err.Error())
			return
		}

		resource, err := realm.ResolveGrantableResource(
			c.Request.Context(), util.GetTenantID(c), oauth.GrantableResourceRef{
				Type:  req.Type,
				Names: names,
			})
		if err != nil {
			handleGrantableResourceRefError(c, err)
			return
		}

		util.WebSuccess(c, resource)
	}
}

// handleGrantableResourceRefError maps the realm's sentinels onto the web error
// envelope.
//
// The incomplete, not-found and not-grantable messages are passed through
// verbatim: each is built from the names the caller just submitted, and each is
// what the user needs in order to fix the form. Everything else is an upstream
// failure whose message carries query and argument context the client may not
// see.
func handleGrantableResourceRefError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, oauth.ErrUnknownGrantableResourceType):
		util.WebInvalidArgumentError(c,
			"type must be one of those returned by grantable-resource-types")

	case errors.Is(err, oauth.ErrUnknownGrantableResourceLevel):
		util.WebInvalidArgumentError(c, "each name must be keyed by one of this type's levels")

	case errors.Is(err, oauth.ErrIncompleteGrantableResourceRef):
		util.WebInvalidArgumentError(c, err.Error())

	case errors.Is(err, oauth.ErrGrantableResourceNotFound):
		util.WebNotFoundError(c, err.Error())

	// 403 rather than 404: the object is there and the name is right, so an
	// answer of "no such thing" would be a lie the user cannot act on. It is not
	// about who is asking either -- nobody may grant it until its owner opens it
	// -- but of the codes available this is the one that says "found, refused".
	case errors.Is(err, oauth.ErrGrantableResourceNotGrantable):
		util.WebNoPermissionError(c, err.Error())

	default:
		util.SetError(c, err)
		util.WebInternalError(c, "failed to look up the grantable resource")
	}
}
