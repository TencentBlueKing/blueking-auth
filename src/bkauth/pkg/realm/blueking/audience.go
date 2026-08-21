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
	"strings"

	"go.uber.org/zap"

	"bkauth/pkg/external/bkapigateway"
	"bkauth/pkg/logging"
	"bkauth/pkg/oauth"
)

const (
	// wildcard stands for "every member, including ones created later". It is
	// the only wildcard the gateway matches, and only in the positions listed
	// with parseBluekingAudience.
	wildcard = "*"

	// The two select-all tokens, spelled as constants because a const cannot call
	// the builders below. Keep them in step with those: mcpAudience(wildcard) and
	// gatewayAPIAudience(wildcard, wildcard) must produce exactly these.
	allMCPServersAudience  = "mcp:*"
	allGatewayAPIsAudience = "gateway:*/api:*"

	displayAllMCPServers = "所有 MCP Server"
	displayAllAPIs       = "所有 API"
)

// mcpAudience and gatewayAPIAudience are the only places an audience is spelled.
// The gateway compares stored tokens byte for byte, so a second spelling of one
// grant is a grant that silently never matches -- and the catalog, the consent
// path and the resolve path each build tokens. parseBluekingAudience is their
// inverse; the three are meant to be read together.
func mcpAudience(serverName string) string {
	return "mcp:" + serverName
}

func gatewayAPIAudience(gatewayName, apiName string) string {
	return "gateway:" + gatewayName + "/api:" + apiName
}

// parseBluekingAudience accepts exactly the five token shapes the gateway
// matches against:
//
//	mcp:<name>              a single MCP server
//	mcp:*                   every MCP server
//	gateway:<gw>/api:<api>  a single API of one gateway
//	gateway:<gw>/api:*      every API of one gateway
//	gateway:*/api:*         every API of every gateway
//
// It is stricter than parseBluekingResource on purpose. That one also takes MCP
// URLs, because a client may ask for a resource in whatever form it has at
// hand, and normalises them on the way to an audience. What lands in the
// audience column, on the other hand, is compared byte for byte by the gateway,
// so a second spelling of the same grant would be a grant that silently never
// matches.
func parseBluekingAudience(aud string) (resType, name, apiName string, err error) {
	if rest, ok := strings.CutPrefix(aud, "mcp:"); ok {
		if rest == "" {
			return "", "", "", fmt.Errorf("invalid audience: empty MCP server name in %q", aud)
		}
		return typeMCP, rest, "", nil
	}

	if rest, ok := strings.CutPrefix(aud, "gateway:"); ok {
		gwName, api, found := strings.Cut(rest, "/api:")
		switch {
		case !found:
			return "", "", "", fmt.Errorf("invalid audience: gateway token must contain /api: segment in %q", aud)
		case gwName == "":
			return "", "", "", fmt.Errorf("invalid audience: empty gateway name in %q", aud)
		case api == "":
			return "", "", "", fmt.Errorf("invalid audience: empty api name in %q", aud)
		case gwName == wildcard && api != wildcard:
			// There is no "this API across every gateway" grant: an API name is
			// only unique within its gateway, so such a token would mean
			// something different on every gateway that happens to reuse it.
			return "", "", "", fmt.Errorf("invalid audience: %q must be %q to span all gateways", aud, allGatewayAPIsAudience)
		}
		return typeAPI, gwName, api, nil
	}

	return "", "", "", fmt.Errorf("invalid audience: unrecognized format %q", aud)
}

func (r *bluekingRealm) ValidateAudiences(_ context.Context, audiences []string) error {
	if len(audiences) == 0 {
		return fmt.Errorf("empty audience list")
	}
	for _, aud := range audiences {
		if _, _, _, err := parseBluekingAudience(aud); err != nil {
			return err
		}
	}
	return nil
}

// parsedAudience is one token split into its parts, kept alongside the token it
// came from so rendering never has to spell it back together.
type parsedAudience struct {
	audience string

	resType string
	// name is the MCP server for an mcp token and the gateway for a gateway one.
	name    string
	apiName string
}

// parseAudiences splits the given tokens, preserving order and duplicates. It is
// the only parse: the renderer works from the result rather than parsing a
// second time, so a token cannot be read one way for the upstream lookup and
// another way for display.
func parseAudiences(audiences []string) ([]parsedAudience, error) {
	parsed := make([]parsedAudience, 0, len(audiences))
	for _, aud := range audiences {
		resType, name, apiName, err := parseBluekingAudience(aud)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, parsedAudience{
			audience: aud,
			resType:  resType,
			name:     name,
			apiName:  apiName,
		})
	}
	return parsed, nil
}

// ResolveAudienceDisplays renders stored tokens for the token list and detail
// pages.
//
// Unlike ResolveResourceDisplay it queries the by-name lookup endpoints rather
// than the search endpoints, because a grant outlives the object's visibility: a
// server that has since been made private must still show up, marked as such,
// instead of vanishing from a token the user can see is there.
//
// Every token is parsed first and the upstream is queried once for the union of
// what they name, so a page of ten tokens naming the same three gateways costs
// three lookups rather than thirty. Batching is the only thing keeping this page
// cheap: there is no cache in front of the upstream yet.
//
// A gateway wildcard does not swallow the specific APIs listed beside it. The
// user ticked both boxes and both boxes are still ticked when they reopen the
// form, so hiding one here would make the two pages disagree.
func (r *bluekingRealm) ResolveAudienceDisplays(
	ctx context.Context,
	tenantID string,
	audiences []string,
) (map[string]oauth.AudienceDisplay, error) {
	parsed, err := parseAudiences(audiences)
	if err != nil {
		return nil, err
	}

	displays := r.lookupDisplays(ctx, tenantID, parsed)

	return audienceDisplays(parsed, displays), nil
}

// audienceDisplays renders one entry per token, keyed by that token.
//
// The rule that a gateway wildcard does not swallow the specific APIs beside it
// needs no code here: the two are different tokens and so different keys.
func audienceDisplays(
	parsed []parsedAudience,
	displays displayIndex,
) map[string]oauth.AudienceDisplay {
	entries := make(map[string]oauth.AudienceDisplay, len(parsed))
	for _, p := range parsed {
		switch p.resType {
		case typeMCP:
			entries[p.audience] = mcpAudienceDisplay(p, displays)
		case typeAPI:
			entries[p.audience] = gatewayAudienceDisplay(p, displays)
		}
	}
	return entries
}

// An MCP grant is never at the gateway level: those rows are grouping handles
// with no token of their own, so what is stored can only have come from a server
// row or from the type-wide box.
func mcpAudienceDisplay(p parsedAudience, displays displayIndex) oauth.AudienceDisplay {
	entry := oauth.AudienceDisplay{
		Type:        typeMCP,
		Level:       levelMCP,
		Name:        p.name,
		DisplayName: p.name,
		Audience:    p.audience,
	}
	if p.name == wildcard {
		entry.Level = oauth.AudienceLevelNone
		entry.DisplayName = displayAllMCPServers
		return entry
	}

	if server, ok := displays.mcpServers[p.name]; ok {
		if server.Title != "" {
			entry.DisplayName = server.Title
		}
		// Present means known. Absent means the lookup came back without this
		// server, and claiming it is public would be worse than saying nothing.
		entry.Extras = oauth.Extras{"is_public": server.IsPublic}
	}
	return entry
}

// A per-gateway wildcard is the gateway row said back, so it is named by the
// gateway rather than by what it covers. That is what keeps the three levels
// legible without qualifiers: "所有 API" belongs to the type-wide grant alone,
// and a gateway-wide grant reads as the gateway, exactly as it was picked.
func gatewayAudienceDisplay(p parsedAudience, displays displayIndex) oauth.AudienceDisplay {
	entry := oauth.AudienceDisplay{
		Type:     typeAPI,
		Audience: p.audience,
	}
	// Built up rather than assigned, because the two facts below come from two
	// lookups that fail independently, and a key is only allowed to appear once
	// its value is known.
	extras := oauth.Extras{}

	switch {
	case p.name == wildcard:
		entry.Level = oauth.AudienceLevelNone
		entry.Name = wildcard
		entry.DisplayName = displayAllAPIs
	case p.apiName == wildcard:
		entry.Level = levelGateway
		entry.Name = p.name
		entry.DisplayName = p.name
	default:
		entry.Level = levelAPI
		entry.Name = p.apiName
		entry.DisplayName = p.apiName
		if resource, ok := displays.resources[p.name][p.apiName]; ok {
			if resource.Description != "" {
				entry.DisplayName = resource.Description
			}
			// Only on this level: the two above stand for sets, and a set is not
			// public or private the way one API is.
			extras["is_public"] = resource.IsPublic
		}
	}

	if gateway, ok := displays.gateways[p.name]; ok {
		// Describes the gateway, so an item-level entry repeats what its group
		// says about itself.
		extras["is_official"] = gateway.IsOfficial
	}

	if len(extras) > 0 {
		entry.Extras = extras
	}
	return entry
}

// displayIndex holds everything the upstream told us about one page of tokens,
// keyed by name so rendering is a map lookup rather than another round trip.
// A name absent from a map was not returned, which callers must render as
// unknown rather than as a false value.
type displayIndex struct {
	mcpServers map[string]bkapigateway.MCPServer
	gateways   map[string]bkapigateway.Gateway
	// resources is keyed by gateway name, then by API name, because an API name
	// is only unique within its gateway.
	resources map[string]map[string]bkapigateway.Resource
}

func (r *bluekingRealm) lookupDisplays(
	ctx context.Context,
	tenantID string,
	parsed []parsedAudience,
) displayIndex {
	displays := displayIndex{resources: map[string]map[string]bkapigateway.Resource{}}
	if r.newLookupClient == nil {
		return displays
	}

	logger := logging.GetWebLogger()
	client := r.newLookupClient(tenantID)

	mcpNames := unionMCPNames(parsed)
	if len(mcpNames) > 0 {
		servers, err := client.LookupMCPServer(ctx, mcpNames)
		if err != nil {
			logger.Warn("failed to look up MCP servers, falling back to names",
				zap.Error(err), zap.Strings("names", mcpNames))
		}
		displays.mcpServers = servers
	}

	gatewayAPIs := unionGatewayAPIs(parsed)
	if len(gatewayAPIs.order) > 0 {
		gateways, err := client.LookupGateway(ctx, gatewayAPIs.order)
		if err != nil {
			logger.Warn("failed to look up gateways, falling back to names",
				zap.Error(err), zap.Strings("names", gatewayAPIs.order))
		}
		displays.gateways = gateways
	}

	// One call per gateway, because the upstream takes the gateway on the path.
	// A gateway that no longer exists answers 404, which costs that one gateway
	// its descriptions and leaves the rest of the page intact.
	for _, gwName := range gatewayAPIs.order {
		apiNames := gatewayAPIs.byGateway[gwName]
		if len(apiNames) == 0 {
			continue
		}

		resources, err := client.LookupReleasedResource(ctx, gwName, apiNames)
		if err != nil {
			logger.Warn("failed to look up released resources, falling back to names",
				zap.Error(err), zap.String("gateway", gwName), zap.Strings("names", apiNames))
			continue
		}
		displays.resources[gwName] = resources
	}

	return displays
}

// unionMCPNames collects the MCP names across every token, dropping the
// wildcard, which no server is called and which needs no lookup.
func unionMCPNames(parsed []parsedAudience) []string {
	seen := map[string]bool{}
	names := make([]string, 0)
	for _, p := range parsed {
		if p.resType != typeMCP || p.name == wildcard || seen[p.name] {
			continue
		}
		seen[p.name] = true
		names = append(names, p.name)
	}
	return names
}

// gatewayAPIUnion is the merged set of gateways and, per gateway, the API names
// worth asking about. The wildcard is dropped from both levels.
type gatewayAPIUnion struct {
	order     []string
	byGateway map[string][]string
	seenAPI   map[string]map[string]bool
}

func unionGatewayAPIs(parsed []parsedAudience) gatewayAPIUnion {
	union := gatewayAPIUnion{
		byGateway: map[string][]string{},
		seenAPI:   map[string]map[string]bool{},
	}

	for _, p := range parsed {
		if p.resType != typeAPI || p.name == wildcard {
			continue
		}
		if _, ok := union.seenAPI[p.name]; !ok {
			union.seenAPI[p.name] = map[string]bool{}
			union.order = append(union.order, p.name)
		}
		if p.apiName == wildcard || union.seenAPI[p.name][p.apiName] {
			continue
		}
		union.seenAPI[p.name][p.apiName] = true
		union.byGateway[p.name] = append(union.byGateway[p.name], p.apiName)
	}

	return union
}
