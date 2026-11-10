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

	"go.uber.org/zap"

	"bkauth/pkg/external/bkapigateway"
	"bkauth/pkg/logging"
	"bkauth/pkg/oauth"
)

// unavailableMessage stands in for an upstream failure. The failure's own text
// carries the request it was built from, which the client is not shown.
const unavailableMessage = "the gateway could not be asked about this grant; try again later"

// ResolveGrantableAudiences confirms tokens the way ResolveGrantableResource
// confirms names: a token resolves only once upstream has said its object exists
// and is open to personal access tokens. It differs in taking the token itself,
// and in answering a failed lookup on the tokens it left unconfirmed rather than
// on the whole call.
//
// The lookups are those ResolveAudienceDisplays makes, merged across the tokens
// the same way, so a link naming ten APIs of one gateway costs one call for them.
// What is done with the answers differs: there a missing object still renders,
// here it is refused.
//
// The two wildcard levels need different amounts of checking. A type-wide token
// names nothing that could be missing. A gateway-wide one resolves once the
// gateway exists, with no claim about what it holds: whether an object is open
// to personal access tokens is a per-API switch, and the lookup endpoints cannot
// enumerate a gateway's APIs to ask.
func (r *bluekingRealm) ResolveGrantableAudiences(
	ctx context.Context,
	tenantID string,
	audiences []string,
) ([]oauth.AudienceResolution, error) {
	client, err := r.lookupClient(tenantID)
	if err != nil {
		return nil, err
	}

	resolutions := make([]oauth.AudienceResolution, len(audiences))
	parsed := make([]parsedAudience, 0, len(audiences))
	positions := make([]int, 0, len(audiences))
	for i, aud := range audiences {
		resType, name, apiName, err := parseBluekingAudience(aud)
		if err != nil {
			resolutions[i] = oauth.RejectedAudience(aud, oauth.AudienceRejectInvalid, err.Error())
			continue
		}
		parsed = append(parsed, parsedAudience{audience: aud, resType: resType, name: name, apiName: apiName})
		positions = append(positions, i)
	}

	index := lookupGrantables(ctx, client, parsed)
	for i, p := range parsed {
		resolutions[positions[i]] = index.resolve(p)
	}
	return resolutions, nil
}

// grantableIndex is a displayIndex that also remembers which lookups failed. The
// display path can forget, since a failed lookup only costs it a title; here a
// failure has to be told apart from an answer of "not there".
type grantableIndex struct {
	displays displayIndex

	mcpErr     error
	gatewayErr error
	// resourceErrs is keyed by gateway, the resource lookup being one call per
	// gateway, so a failure costs that gateway's tokens and no others.
	resourceErrs map[string]error
}

func lookupGrantables(
	ctx context.Context,
	client bkapigateway.LookupClient,
	parsed []parsedAudience,
) grantableIndex {
	logger := logging.GetWebLogger()
	index := grantableIndex{
		displays:     displayIndex{resources: map[string]map[string]bkapigateway.Resource{}},
		resourceErrs: map[string]error{},
	}

	if mcpNames := unionMCPNames(parsed); len(mcpNames) > 0 {
		index.displays.mcpServers, index.mcpErr = client.LookupMCPServer(ctx, mcpNames)
		if index.mcpErr != nil {
			logger.Warn("failed to look up MCP servers to resolve shared audiences",
				zap.Error(index.mcpErr), zap.Strings("names", mcpNames))
		}
	}

	gatewayAPIs := unionGatewayAPIs(parsed)
	if len(gatewayAPIs.order) > 0 {
		index.displays.gateways, index.gatewayErr = client.LookupGateway(ctx, gatewayAPIs.order)
		if index.gatewayErr != nil {
			logger.Warn("failed to look up gateways to resolve shared audiences",
				zap.Error(index.gatewayErr), zap.Strings("names", gatewayAPIs.order))
		}
	}

	for _, gwName := range gatewayAPIs.order {
		apiNames := gatewayAPIs.byGateway[gwName]
		if len(apiNames) == 0 {
			continue
		}

		resources, err := client.LookupReleasedResource(ctx, gwName, apiNames)
		if err != nil {
			// A 404 is upstream's answer that the gateway does not exist, so it
			// is kept to be read as one rather than logged as a fault.
			if !bkapigateway.IsNotFound(err) {
				logger.Warn("failed to look up released resources to resolve shared audiences",
					zap.Error(err), zap.String("gateway", gwName), zap.Strings("names", apiNames))
			}
			index.resourceErrs[gwName] = err
			continue
		}
		index.displays.resources[gwName] = resources
	}

	return index
}

func (index grantableIndex) resolve(p parsedAudience) oauth.AudienceResolution {
	if p.resType == typeMCP {
		return index.resolveMCP(p)
	}
	return index.resolveGatewayAPI(p)
}

func (index grantableIndex) resolveMCP(p parsedAudience) oauth.AudienceResolution {
	if p.name != wildcard {
		if index.mcpErr != nil {
			return unavailableAudience(p.audience)
		}

		server, found := index.displays.mcpServers[p.name]
		if !found {
			return rejectedAudience(p.audience, oauth.AudienceRejectNotFound, fmt.Errorf(
				"%w: no MCP server named %q", oauth.ErrGrantableResourceNotFound, p.name))
		}
		if !server.OAuth2PersonalClientEnabled {
			return rejectedAudience(p.audience, oauth.AudienceRejectNotGrantable, fmt.Errorf(
				"%w: MCP server %q is not open to personal access tokens",
				oauth.ErrGrantableResourceNotGrantable, p.name))
		}
	}

	return oauth.ResolvedAudience(mcpAudienceDisplay(p, index.displays))
}

func (index grantableIndex) resolveGatewayAPI(p parsedAudience) oauth.AudienceResolution {
	switch {
	case p.name == wildcard:
		// Every API of every gateway: nothing named, so nothing to confirm.

	case p.apiName == wildcard:
		if index.gatewayErr != nil {
			return unavailableAudience(p.audience)
		}
		if _, found := index.displays.gateways[p.name]; !found {
			return rejectedAudience(p.audience, oauth.AudienceRejectNotFound, errNoGateway(p.name))
		}

	default:
		if err, failed := index.resourceErrs[p.name]; failed {
			if bkapigateway.IsNotFound(err) {
				return rejectedAudience(p.audience, oauth.AudienceRejectNotFound, errNoGateway(p.name))
			}
			return unavailableAudience(p.audience)
		}

		resource, found := index.displays.resources[p.name][p.apiName]
		if !found {
			return rejectedAudience(p.audience, oauth.AudienceRejectNotFound, fmt.Errorf(
				"%w: gateway %q has no released API named %q",
				oauth.ErrGrantableResourceNotFound, p.name, p.apiName))
		}
		if !resource.OAuth2PersonalClientEnabled {
			return rejectedAudience(p.audience, oauth.AudienceRejectNotGrantable, fmt.Errorf(
				"%w: API %q of gateway %q is not open to personal access tokens",
				oauth.ErrGrantableResourceNotGrantable, p.apiName, p.name))
		}
	}

	return oauth.ResolvedAudience(gatewayAudienceDisplay(p, index.displays))
}

func errNoGateway(gatewayName string) error {
	return fmt.Errorf("%w: no gateway named %q", oauth.ErrGrantableResourceNotFound, gatewayName)
}

func rejectedAudience(aud, reason string, err error) oauth.AudienceResolution {
	return oauth.RejectedAudience(aud, reason, err.Error())
}

func unavailableAudience(aud string) oauth.AudienceResolution {
	return oauth.RejectedAudience(aud, oauth.AudienceRejectUnavailable, unavailableMessage)
}
