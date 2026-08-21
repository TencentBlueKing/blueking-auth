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
	"regexp"
	"strings"

	"go.uber.org/zap"

	"bkauth/pkg/external/bkapigateway"
	"bkauth/pkg/logging"
	"bkauth/pkg/oauth"
	"bkauth/pkg/util"
)

// ResourceItem represents a single resource entry with optional nested children.
type ResourceItem struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	Items       []ResourceItem `json:"items,omitempty"`
}

// ResourceGroup represents a group of resources sharing the same type.
type ResourceGroup struct {
	Type        string         `json:"type"`
	DisplayName string         `json:"display_name"`
	Items       []ResourceItem `json:"items"`
}

const (
	Name = "blueking"

	// typeMCP and typeAPI are this realm's two grantable resource types.
	// They live here rather than beside the enumeration because all three paths
	// speak them: the consent display puts them in ResourceGroup.Type, the
	// audience display in AudienceDisplay.Type, and the enumeration in
	// GrantableResourceType.Name. A client matching one against another needs
	// them to be one vocabulary, not three literals that happen to agree.
	// A type is named after what it grants, not after the prefix its audiences
	// carry: the API type grants APIs, even though every one of its audiences is
	// written "gateway:<gw>/api:<api>". The prefix is grammar, and gateways are
	// only a level within this type.
	typeMCP = "mcp"
	typeAPI = "api"

	typeMCPDisplayName = "MCP"
	typeAPIDisplayName = "API"

	// The levels of the two catalogs, named after what each level holds. Both
	// nest under gateways, but only the API catalog is grantable there, which is
	// a fact about the rows rather than about the levels.
	//
	// levelMCP and levelAPI spell the same strings as typeMCP and typeAPI, and
	// are deliberately not defined in terms of them: a level and a type are two
	// vocabularies that happen to agree here, and tying them together would let
	// a rename of one silently rewrite the other.
	levelGateway = "gateway"
	levelMCP     = "mcp"
	levelAPI     = "api"

	// The innermost level of each catalog is labelled the same as its type, which
	// is a coincidence of this realm rather than a rule: a type is a division of
	// what can be granted, a level is a rung of one division's tree. Other realms
	// label a product ("蓝盾") and a level ("服务") quite differently.
	levelGatewayDisplayName = "网关"
	levelMCPDisplayName     = "MCP"
	levelAPIDisplayName     = "API"
)

type bluekingRealm struct {
	// mcpServerClient serves the consent page over the open API, which still
	// works on gateway deployments that predate the v2 inner APIs. The two
	// clients converge once the open API is retired.
	mcpServerClient bkapigateway.MCPServerClient

	// The v2 inner clients are bound to a tenant at construction, and the tenant
	// arrives per request, so the realm -- registered once at startup -- holds
	// their constructors instead of instances. Both carry nothing but that
	// tenant, so building one per call costs nothing.
	newLookupClient func(tenantID string) bkapigateway.LookupClient
	newSearchClient func(tenantID string) bkapigateway.SearchClient
}

// New creates the blueking Realm implementation.
func New() oauth.Realm {
	return &bluekingRealm{
		mcpServerClient: bkapigateway.NewMCPServerClient(),
		newLookupClient: bkapigateway.NewLookupClient,
		newSearchClient: bkapigateway.NewSearchClient,
	}
}

func (r *bluekingRealm) Name() string { return Name }

var mcpServerNameRegex = regexp.MustCompile(`/mcp-servers/([^/]+)`)

func extractMCPServerName(resourceURL string) string {
	matches := mcpServerNameRegex.FindStringSubmatch(resourceURL)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// parseBluekingResource parses a single resource item for the blueking realm.
// Returns (type, name, apiName) where apiName is only set for gateway resources.
func parseBluekingResource(item string) (resType, name, apiName string, err error) {
	if strings.HasPrefix(item, "http://") || strings.HasPrefix(item, "https://") {
		n := extractMCPServerName(item)
		if n == "" {
			return "", "", "", fmt.Errorf("invalid resource URL: cannot extract MCP server name from %q", item)
		}
		return typeMCP, n, "", nil
	}

	if strings.HasPrefix(item, "mcp:") {
		n := strings.TrimPrefix(item, "mcp:")
		if n == "" {
			return "", "", "", fmt.Errorf("invalid resource: empty name in %q", item)
		}
		return typeMCP, n, "", nil
	}

	if rest, ok := strings.CutPrefix(item, "gateway:"); ok {
		gwName, api, found := strings.Cut(rest, "/api:")
		if !found {
			return "", "", "", fmt.Errorf("invalid resource: gateway item must contain /api: segment in %q", item)
		}
		if gwName == "" {
			return "", "", "", fmt.Errorf("invalid resource: empty gateway name in %q", item)
		}
		if api == "" {
			return "", "", "", fmt.Errorf("invalid resource: empty api name in %q", item)
		}
		return typeAPI, gwName, api, nil
	}

	return "", "", "", fmt.Errorf("invalid resource: unrecognized format %q", item)
}

func (r *bluekingRealm) ValidateResource(_ context.Context, resource string) error {
	items := util.SplitCommaList(resource)
	if len(items) == 0 {
		return fmt.Errorf("empty resource string")
	}
	for _, item := range items {
		if _, _, _, err := parseBluekingResource(item); err != nil {
			return err
		}
	}
	return nil
}

func (r *bluekingRealm) ExtractAudiences(_ context.Context, resource string) ([]string, error) {
	items := util.SplitCommaList(resource)
	if len(items) == 0 {
		return nil, fmt.Errorf("empty resource string")
	}

	seen := make(map[string]bool)
	var audiences []string

	for _, item := range items {
		resType, name, apiName, err := parseBluekingResource(item)
		if err != nil {
			return nil, err
		}

		// The audience keeps the API segment. Collapsing it to "gateway:<name>"
		// would both widen the grant beyond what the user picked and produce a
		// value outside the five formats the gateway matches against.
		var aud string
		switch resType {
		case typeMCP:
			aud = mcpAudience(name)
		case typeAPI:
			aud = gatewayAPIAudience(name, apiName)
		}

		if !seen[aud] {
			seen[aud] = true
			audiences = append(audiences, aud)
		}
	}

	return audiences, nil
}

func (r *bluekingRealm) ResolveResourceDisplay(ctx context.Context, resource string) (any, error) {
	items := util.SplitCommaList(resource)
	if len(items) == 0 {
		return nil, fmt.Errorf("empty resource string")
	}

	// Deduplicated MCP names in encounter order.
	var mcpNames []string
	mcpSeen := make(map[string]bool)

	gwOrder := make([]string, 0)
	type gwState struct {
		allAPIs bool
		apis    []ResourceItem
		seen    map[string]bool
	}
	gwMap := make(map[string]*gwState)

	for _, item := range items {
		resType, name, apiName, err := parseBluekingResource(item)
		if err != nil {
			return nil, err
		}

		switch resType {
		case typeMCP:
			if !mcpSeen[name] {
				mcpSeen[name] = true
				mcpNames = append(mcpNames, name)
			}
		case typeAPI:
			gs, ok := gwMap[name]
			if !ok {
				gs = &gwState{seen: make(map[string]bool)}
				gwMap[name] = gs
				gwOrder = append(gwOrder, name)
			}
			if apiName == "*" {
				gs.allAPIs = true
			} else if !gs.seen[apiName] {
				gs.seen[apiName] = true
				gs.apis = append(gs.apis, ResourceItem{Name: apiName, DisplayName: apiName})
			}
		}
	}

	// Batch-fetch MCP display titles; fall back to name on failure.
	mcpTitles := r.fetchMCPTitles(ctx, mcpNames)

	var groups []ResourceGroup

	if len(mcpNames) > 0 {
		mcpItems := make([]ResourceItem, 0, len(mcpNames))
		for _, name := range mcpNames {
			displayName := name
			if title, ok := mcpTitles[name]; ok {
				displayName = title
			}
			mcpItems = append(mcpItems, ResourceItem{Name: name, DisplayName: displayName})
		}
		groups = append(groups, ResourceGroup{
			Type:        typeMCP,
			DisplayName: "网关 MCP Server",
			Items:       mcpItems,
		})
	}

	if len(gwOrder) > 0 {
		gwItems := make([]ResourceItem, 0, len(gwOrder))
		for _, gwName := range gwOrder {
			gs := gwMap[gwName]
			ri := ResourceItem{Name: gwName, DisplayName: gwName}
			if gs.allAPIs {
				ri.Items = []ResourceItem{{Name: "*", DisplayName: "所有 API"}}
			} else {
				ri.Items = gs.apis
			}
			gwItems = append(gwItems, ri)
		}
		groups = append(groups, ResourceGroup{
			Type:        typeAPI,
			DisplayName: "网关 API",
			Items:       gwItems,
		})
	}

	return groups, nil
}

func (r *bluekingRealm) fetchMCPTitles(ctx context.Context, names []string) map[string]string {
	if len(names) == 0 || r.mcpServerClient == nil {
		return nil
	}

	titles, err := r.mcpServerClient.BatchQueryTitles(ctx, names)
	if err != nil {
		logging.GetWebLogger().Warn("failed to fetch MCP server titles, falling back to names",
			zap.Error(err),
			zap.Strings("names", names),
		)
		return nil
	}
	return titles
}
