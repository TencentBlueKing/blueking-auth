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
	"strings"
)

// LookupClient looks up gateway objects by exact name, as one tenant.
//
// All three methods target the upstream's dedicated `-/lookup/` endpoints, which
// exist precisely for this: they take up to namesPerRequest names, are not
// paged, and return the array directly. Their paged siblings cannot serve this
// purpose -- they dropped their name filters, and a page of 20 could not answer
// a question about 50 names anyway.
//
// Unlike the search endpoints these also return objects that are no longer
// public, which is what makes them usable for rendering a grant that was made
// before the object was hidden.
//
// Every method keys its result by name and simply omits what upstream did not
// return. A missing key means "not found", which callers must be able to tell
// apart from an object whose title or description happens to be empty.
type LookupClient interface {
	LookupMCPServer(ctx context.Context, names []string) (map[string]MCPServer, error)
	LookupGateway(ctx context.Context, names []string) (map[string]Gateway, error)
	LookupReleasedResource(
		ctx context.Context, gatewayName string, resourceNames []string,
	) (map[string]Resource, error)
}

// MCPServer is the display-side view of an MCP server.
type MCPServer struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	IsPublic bool   `json:"is_public"`
}

// Gateway is the display-side view of a gateway.
type Gateway struct {
	Name       string `json:"name"`
	IsOfficial bool   `json:"is_official"`
}

// Resource is the display-side view of a released API resource.
type Resource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type lookupClient struct {
	tenantID string
}

// NewLookupClient creates a LookupClient acting as tenantID, backed by the BK
// API Gateway. An empty tenantID asks as no tenant, which the upstream answers
// with the tenant-neutral subset.
//
// The tenant is fixed here rather than passed per call because it scopes
// everything the instance can see, the same way the authentication header does;
// a caller switching tenants mid-instance would be switching identity, not
// filtering. Instances hold nothing but that identity, so a caller serving one
// request builds one and drops it.
func NewLookupClient(tenantID string) LookupClient {
	return &lookupClient{tenantID: tenantID}
}

// LookupMCPServer takes only names, not the ids the endpoint also accepts:
// audiences carry names, and an id parameter would be surface with no caller.
func (c *lookupClient) LookupMCPServer(
	ctx context.Context,
	names []string,
) (map[string]MCPServer, error) {
	result := map[string]MCPServer{}
	for _, chunk := range chunkNames(names) {
		query := url.Values{}
		query.Set("names", strings.Join(chunk, ","))
		query.Set("fields", "name,title,is_public")

		var servers []MCPServer
		if err := innerGet(ctx, c.tenantID, "api/v2/inner/mcp-servers/-/lookup/", query, &servers); err != nil {
			return nil, err
		}
		for _, server := range servers {
			result[server.Name] = server
		}
	}
	return result, nil
}

func (c *lookupClient) LookupGateway(
	ctx context.Context,
	names []string,
) (map[string]Gateway, error) {
	result := map[string]Gateway{}
	for _, chunk := range chunkNames(names) {
		query := url.Values{}
		query.Set("gateway_names", strings.Join(chunk, ","))
		query.Set("fields", "name,is_official")

		var gateways []Gateway
		if err := innerGet(ctx, c.tenantID, "api/v2/inner/gateways/-/lookup/", query, &gateways); err != nil {
			return nil, err
		}
		for _, gateway := range gateways {
			result[gateway.Name] = gateway
		}
	}
	return result, nil
}

// LookupReleasedResource returns the named resources of one gateway. An empty
// resourceNames asks for nothing and is answered without a round trip, because
// upstream requires at least one name and would answer 400.
//
// The gateway is on the path, so this is one call per gateway. A gateway that no
// longer exists answers 404, which IsNotFound distinguishes from a fault.
func (c *lookupClient) LookupReleasedResource(
	ctx context.Context,
	gatewayName string,
	resourceNames []string,
) (map[string]Resource, error) {
	result := map[string]Resource{}
	for _, chunk := range chunkNames(resourceNames) {
		query := url.Values{}
		query.Set("names", strings.Join(chunk, ","))
		query.Set("fields", "name,description")

		path := "api/v2/inner/gateways/" + url.PathEscape(gatewayName) + "/released-resources/-/lookup/"

		var resources []Resource
		if err := innerGet(ctx, c.tenantID, path, query, &resources); err != nil {
			return nil, err
		}
		for _, resource := range resources {
			result[resource.Name] = resource
		}
	}
	return result, nil
}
