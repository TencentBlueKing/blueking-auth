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
	"errors"
	"fmt"
	"strings"

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
	if err := oauth.ValidateLevelNames(q.Keywords, levelGateway, levelMCP); err != nil {
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
				Audience:    mcpAudience(server.Name),
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
	if err := oauth.ValidateLevelNames(q.Keywords, levelGateway, levelAPI); err != nil {
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
				Audience:    gatewayAPIAudience(gateway.Name, resource.Name),
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
			Audience: gatewayAPIAudience(gateway.Name, wildcard),
			Items:    items,
		})
	}

	return oauth.GrantableResourcePage{Count: page.Count, Results: results}, nil
}

// ResolveGrantableResource resolves an entry the user named outright, which is
// how anything the catalog does not list gets granted. Both catalogs are served
// by search endpoints that return public objects exclusively, so a private MCP
// server or API is grantable and yet unreachable by browsing.
//
// Being private is what the catalog withholds; being closed to personal access
// tokens is what it filters, by asking upstream for OAuthClientTypePersonal. The
// lookup endpoints take no such parameter, so each resolve below re-applies that
// filter itself, on the field upstream returns. Doing it here rather than in the
// client is deliberate: the same lookups render stored tokens, and a grant made
// before the switch was turned off must still be shown, if only to be revoked.
//
// Both of a type's levels must be named. The API type has a per-gateway wildcard
// but does not offer it here: that row is in the catalog, ticking it is how it is
// meant to be granted, and accepting a gateway on its own would quietly turn a
// half-filled form into the broadest grant the type has.
func (r *bluekingRealm) ResolveGrantableResource(
	ctx context.Context,
	tenantID string,
	ref oauth.GrantableResourceRef,
) (oauth.GrantableResource, error) {
	switch ref.Type {
	case typeMCP:
		return r.resolveMCPResource(ctx, tenantID, ref)
	case typeAPI:
		return r.resolveGatewayResource(ctx, tenantID, ref)
	default:
		return oauth.GrantableResource{}, fmt.Errorf(
			"%w: %q", oauth.ErrUnknownGrantableResourceType, ref.Type)
	}
}

func (r *bluekingRealm) resolveMCPResource(
	ctx context.Context,
	tenantID string,
	ref oauth.GrantableResourceRef,
) (oauth.GrantableResource, error) {
	if err := oauth.ValidateLevelNames(ref.Names, levelGateway, levelMCP); err != nil {
		return oauth.GrantableResource{}, err
	}

	gatewayName, serverName := ref.Names[levelGateway], ref.Names[levelMCP]
	if gatewayName == "" || serverName == "" {
		return oauth.GrantableResource{}, fmt.Errorf(
			"%w: both %q and %q must be named to identify an MCP server",
			oauth.ErrIncompleteGrantableResourceRef, levelGateway, levelMCP)
	}

	client, err := r.lookupClient(tenantID)
	if err != nil {
		return oauth.GrantableResource{}, err
	}

	servers, err := client.LookupMCPServer(ctx, []string{serverName})
	if err != nil {
		return oauth.GrantableResource{}, err
	}

	server, found := servers[serverName]
	if !found || !mcpServerBelongsTo(serverName, gatewayName) {
		// One error for both, because from where the user sits the pair they typed
		// is what does not exist. Reporting "that server lives on another gateway"
		// would answer a question they did not ask, about an object the form gives
		// them no way to see.
		return oauth.GrantableResource{}, fmt.Errorf(
			"%w: gateway %q has no MCP server named %q",
			oauth.ErrGrantableResourceNotFound, gatewayName, serverName)
	}

	if !server.OAuth2PersonalClientEnabled {
		return oauth.GrantableResource{}, fmt.Errorf(
			"%w: MCP server %q is not open to personal access tokens",
			oauth.ErrGrantableResourceNotGrantable, serverName)
	}

	return oauth.GrantableResource{
		Name:        server.Name,
		DisplayName: displayNameOrFallback(server.Title, server.Name),
		Audience:    mcpAudience(server.Name),
		// Carried so the form can mark the entry private. It is display only: on
		// this gateway public is about being listed, not about who may call it.
		Extras: oauth.Extras{"is_public": server.IsPublic},
	}, nil
}

// mcpServerBelongsTo reports whether the server named is one of the gateway's.
//
// Upstream names every MCP server "<gateway>-<stage>-<...>", and that convention
// is all there is to go on: the lookup endpoint does not say which gateway a
// server belongs to, and the audience carries the server name alone. Checking it
// still earns its keep -- without it a mistyped gateway silently yields a grant
// on a stranger's server -- but it is a prefix test, so a gateway whose name
// prefixes another's passes for it. That is a wrong label on a row the user
// typed themselves, not a grant they did not ask for: the audience names the
// server they spelled out either way.
func mcpServerBelongsTo(serverName, gatewayName string) bool {
	return strings.HasPrefix(serverName, gatewayName+"-")
}

func (r *bluekingRealm) resolveGatewayResource(
	ctx context.Context,
	tenantID string,
	ref oauth.GrantableResourceRef,
) (oauth.GrantableResource, error) {
	if err := oauth.ValidateLevelNames(ref.Names, levelGateway, levelAPI); err != nil {
		return oauth.GrantableResource{}, err
	}

	gatewayName, apiName := ref.Names[levelGateway], ref.Names[levelAPI]
	if gatewayName == "" || apiName == "" {
		return oauth.GrantableResource{}, fmt.Errorf(
			"%w: both %q and %q must be named to identify an API",
			oauth.ErrIncompleteGrantableResourceRef, levelGateway, levelAPI)
	}

	client, err := r.lookupClient(tenantID)
	if err != nil {
		return oauth.GrantableResource{}, err
	}

	resources, err := client.LookupReleasedResource(ctx, gatewayName, []string{apiName})
	if err != nil {
		// The gateway is on the path upstream, so a gateway nobody has heard of
		// comes back as a 404. That is a name to go back and fix, not a fault.
		if bkapigateway.IsNotFound(err) {
			return oauth.GrantableResource{}, fmt.Errorf(
				"%w: no gateway named %q", oauth.ErrGrantableResourceNotFound, gatewayName)
		}
		return oauth.GrantableResource{}, err
	}

	resource, found := resources[apiName]
	if !found {
		return oauth.GrantableResource{}, fmt.Errorf(
			"%w: gateway %q has no released API named %q",
			oauth.ErrGrantableResourceNotFound, gatewayName, apiName)
	}

	if !resource.OAuth2PersonalClientEnabled {
		return oauth.GrantableResource{}, fmt.Errorf(
			"%w: API %q of gateway %q is not open to personal access tokens",
			oauth.ErrGrantableResourceNotGrantable, apiName, gatewayName)
	}

	return oauth.GrantableResource{
		Name:        resource.Name,
		DisplayName: displayNameOrFallback(resource.Description, resource.Name),
		Audience:    gatewayAPIAudience(gatewayName, resource.Name),
		Extras:      oauth.Extras{"is_public": resource.IsPublic},
	}, nil
}

// lookupClient builds the by-name client for one tenant.
//
// A realm without the constructor is one built by hand in a test; New always
// sets it. It is an error rather than the empty fallback lookupDisplays takes,
// because resolving is asked to confirm the entry exists and there is no
// degraded way to do that.
func (r *bluekingRealm) lookupClient(tenantID string) (bkapigateway.LookupClient, error) {
	if r.newLookupClient == nil {
		return nil, errors.New("blueking realm was built without a lookup client")
	}
	return r.newLookupClient(tenantID), nil
}

func displayNameOrFallback(displayName, name string) string {
	if displayName != "" {
		return displayName
	}
	return name
}
