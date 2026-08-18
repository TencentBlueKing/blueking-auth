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

package bkapigateway

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"context"
	"net/url"
	"strconv"
)

const (
	// MaxPageSize is the largest Limit the search endpoints accept; asking for
	// more is answered 400. It is stated here because a caller has to decide its
	// own page ceiling before it can build a query, and this package is where
	// that number comes from.
	//
	// Nothing below enforces it. The upstream rejection is the guard, so a check
	// here would only trade one error for another, and clamping would be worse
	// still: Count reports the matching gateways, so a silently shortened page
	// would page against a total it does not belong to.
	MaxPageSize = 20
)

// SearchClient searches gateway objects under query constraints, as one tenant.
// Both endpoints page by gateway rather than by item, and every gateway on a
// page carries all of its matching items. OAuthClientType is required: the
// upstream has no unscoped listing of these collections.
type SearchClient interface {
	SearchMCPServer(ctx context.Context, q MCPServerQuery) (MCPServerPage, error)
	SearchResource(ctx context.Context, q ResourceQuery) (ResourcePage, error)
}

// MCPServerQuery mirrors the upstream query vocabulary: gateway name and MCP
// server name are AND-ed substring filters, and limit/offset count gateways.
//
// The tenant is not in here: it identifies who is asking rather than what is
// asked for, and belongs to the client (see NewSearchClient).
type MCPServerQuery struct {
	OAuthClientType string

	GatewayName   string
	MCPServerName string

	// Limit counts gateways and may not exceed MaxPageSize. Left at zero it
	// takes the upstream default rather than asking for nothing.
	Limit  int
	Offset int
}

// ResourceQuery is the resource-side counterpart of MCPServerQuery. The two
// are kept apart because their item filters name different things, and a
// shared field would leave callers guessing which one is in effect.
type ResourceQuery struct {
	OAuthClientType string

	GatewayName  string
	ResourceName string

	// Same contract as MCPServerQuery.Limit, MaxPageSize included.
	Limit  int
	Offset int
}

// MCPServerPage holds one page of gateways. Count is the number of matching
// gateways, not of MCP servers.
type MCPServerPage struct {
	Count   int                 `json:"count"`
	Results []GatewayMCPServers `json:"results"`
}

// GatewayMCPServers is a gateway together with all of its MCP servers that
// match the query.
type GatewayMCPServers struct {
	Name       string          `json:"name"`
	IsOfficial bool            `json:"is_official"`
	MCPServers []MCPServerItem `json:"mcp_servers"`
}

// MCPServerItem carries no is_public: the search endpoints only ever return
// public objects, so a field for it could only ever hold one value.
type MCPServerItem struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

// ResourcePage holds one page of gateways. Count is the number of matching
// gateways, not of resources.
type ResourcePage struct {
	Count   int                `json:"count"`
	Results []GatewayResources `json:"results"`
}

// GatewayResources is a gateway together with all of its API resources that
// match the query.
type GatewayResources struct {
	Name       string         `json:"name"`
	IsOfficial bool           `json:"is_official"`
	Resources  []ResourceItem `json:"resources"`
}

// ResourceItem is a single API resource of a gateway.
type ResourceItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type searchClient struct {
	tenantID string
}

// NewSearchClient creates a SearchClient acting as tenantID, backed by the BK
// API Gateway. Same contract as NewLookupClient: the tenant is identity, fixed
// for the life of the instance, and an empty one asks as no tenant.
func NewSearchClient(tenantID string) SearchClient {
	return &searchClient{tenantID: tenantID}
}

func (c *searchClient) SearchMCPServer(
	ctx context.Context,
	q MCPServerQuery,
) (MCPServerPage, error) {
	query := searchPagingQuery(q.OAuthClientType, q.Limit, q.Offset)
	setIfNotEmpty(query, "gateway_name", q.GatewayName)
	setIfNotEmpty(query, "mcp_server_name", q.MCPServerName)

	var page MCPServerPage
	err := innerGet(ctx, c.tenantID, "api/v2/inner/oauth2/client-scopes/mcp-servers/", query, &page)
	if err != nil {
		return MCPServerPage{}, err
	}
	return page, nil
}

func (c *searchClient) SearchResource(
	ctx context.Context,
	q ResourceQuery,
) (ResourcePage, error) {
	query := searchPagingQuery(q.OAuthClientType, q.Limit, q.Offset)
	setIfNotEmpty(query, "gateway_name", q.GatewayName)
	setIfNotEmpty(query, "resource_name", q.ResourceName)

	var page ResourcePage
	err := innerGet(ctx, c.tenantID, "api/v2/inner/oauth2/client-scopes/resources/", query, &page)
	if err != nil {
		return ResourcePage{}, err
	}
	return page, nil
}

// searchPagingQuery leaves limit and offset out when unset so the upstream
// defaults apply, rather than sending a zero limit the upstream rejects.
func searchPagingQuery(oauthClientType string, limit, offset int) url.Values {
	query := url.Values{}
	query.Set("oauth_client_type", oauthClientType)
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		query.Set("offset", strconv.Itoa(offset))
	}
	return query
}
