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

package blueking

import (
	"context"
	"fmt"

	"bkauth/pkg/external/bkapigateway"
	"bkauth/pkg/oauth"
)

// GrantableResourceTypes returns the two types this realm's catalog is divided
// into. Both are gateways over their members, but only one of them is grantable
// at the gateway row: an API grant has a per-gateway wildcard, so ticking the
// gateway covers APIs released later, whereas the MCP vocabulary has no "every
// MCP server of this gateway" form -- only one server or all of them.
//
// A frontend that wants a convenience tick on an MCP gateway row can expand it
// into one token per server, which is a bulk selection and, unlike a wildcard,
// does not cover servers added later. Which is also why that row is left with an
// empty Audience rather than given the all-servers token: it would claim a reach
// it does not have.
func (r *bluekingRealm) GrantableResourceTypes() []oauth.GrantableResourceType {
	return []oauth.GrantableResourceType{
		{
			Name:        typeMCP,
			DisplayName: typeMCPDisplayName,
			Audience:    allMCPServersAudience,
			Levels: []oauth.GrantableResourceLevel{
				{Name: levelGateway, DisplayName: levelGatewayDisplayName},
				{Name: levelMCP, DisplayName: levelMCPDisplayName},
			},
		},
		{
			Name:        typeAPI,
			DisplayName: typeAPIDisplayName,
			Audience:    allGatewayAPIsAudience,
			Levels: []oauth.GrantableResourceLevel{
				{Name: levelGateway, DisplayName: levelGatewayDisplayName},
				{Name: levelAPI, DisplayName: levelAPIDisplayName},
			},
		},
	}
}

func (r *bluekingRealm) ListGrantableResource(
	ctx context.Context,
	tenantID string,
	q oauth.GrantableResourceQuery,
) (oauth.GrantableResourcePage, error) {
	switch q.Type {
	case typeMCP:
		return r.listMCPResources(ctx, tenantID, q)
	case typeAPI:
		return r.listGatewayResources(ctx, tenantID, q)
	default:
		return oauth.GrantableResourcePage{}, fmt.Errorf(
			"%w: %q", oauth.ErrUnknownGrantableResourceType, q.Type)
	}
}

func (r *bluekingRealm) listMCPResources(
	ctx context.Context,
	tenantID string,
	q oauth.GrantableResourceQuery,
) (oauth.GrantableResourcePage, error) {
	if err := oauth.ValidateKeywordLevels(q.Keywords, levelGateway, levelMCP); err != nil {
		return oauth.GrantableResourcePage{}, err
	}

	page, err := r.newSearchClient(tenantID).SearchMCPServer(ctx, bkapigateway.MCPServerQuery{
		OAuthClientType: bkapigateway.OAuthClientTypePersonal,
		GatewayName:     q.Keywords[levelGateway],
		MCPServerName:   q.Keywords[levelMCP],
		Limit:           q.Limit,
		Offset:          q.Offset,
	})
	if err != nil {
		return oauth.GrantableResourcePage{}, err
	}

	results := make([]oauth.GrantableResource, 0, len(page.Results))
	for _, gateway := range page.Results {
		items := make([]oauth.GrantableResource, 0, len(gateway.MCPServers))
		for _, server := range gateway.MCPServers {
			items = append(items, oauth.GrantableResource{
				Name:        server.Name,
				DisplayName: displayNameOrFallback(server.Title, server.Name),
				Audience:    "mcp:" + server.Name,
			})
		}

		results = append(results, oauth.GrantableResource{
			Name:        gateway.Name,
			DisplayName: gateway.Name,
			Extras:      oauth.Extras{"is_official": gateway.IsOfficial},
			// Grouping handle only; see GrantableResourceTypes.
			Audience: "",
			Items:    items,
		})
	}

	return oauth.GrantableResourcePage{Count: page.Count, Results: results}, nil
}

func (r *bluekingRealm) listGatewayResources(
	ctx context.Context,
	tenantID string,
	q oauth.GrantableResourceQuery,
) (oauth.GrantableResourcePage, error) {
	if err := oauth.ValidateKeywordLevels(q.Keywords, levelGateway, levelAPI); err != nil {
		return oauth.GrantableResourcePage{}, err
	}

	page, err := r.newSearchClient(tenantID).SearchResource(ctx, bkapigateway.ResourceQuery{
		OAuthClientType: bkapigateway.OAuthClientTypePersonal,
		GatewayName:     q.Keywords[levelGateway],
		ResourceName:    q.Keywords[levelAPI],
		Limit:           q.Limit,
		Offset:          q.Offset,
	})
	if err != nil {
		return oauth.GrantableResourcePage{}, err
	}

	results := make([]oauth.GrantableResource, 0, len(page.Results))
	for _, gateway := range page.Results {
		items := make([]oauth.GrantableResource, 0, len(gateway.Resources))
		for _, resource := range gateway.Resources {
			items = append(items, oauth.GrantableResource{
				Name:        resource.Name,
				DisplayName: displayNameOrFallback(resource.Description, resource.Name),
				Audience:    "gateway:" + gateway.Name + "/api:" + resource.Name,
			})
		}

		results = append(results, oauth.GrantableResource{
			Name:        gateway.Name,
			DisplayName: gateway.Name,
			Extras:      oauth.Extras{"is_official": gateway.IsOfficial},
			// The gateway row is grantable itself, and means every API of it
			// including ones released later -- which ticking each row below
			// would not cover. Carrying it here rather than as an extra
			// all-APIs child keeps one row per grant, so a grant read back has
			// exactly one row it could have come from.
			Audience: "gateway:" + gateway.Name + "/api:" + wildcard,
			Items:    items,
		})
	}

	return oauth.GrantableResourcePage{Count: page.Count, Results: results}, nil
}

func displayNameOrFallback(displayName, name string) string {
	if displayName != "" {
		return displayName
	}
	return name
}
