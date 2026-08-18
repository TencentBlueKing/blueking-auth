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

package gpu

import (
	"context"
	"fmt"

	"bkauth/pkg/oauth"
)

// ResourceDisplay is the gpu-specific resource display structure.
type ResourceDisplay struct {
	Type        string        `json:"type"`
	DisplayName string        `json:"display_name"`
	Items       []ItemDisplay `json:"items"`
}

// ItemDisplay represents a single resource item.
type ItemDisplay struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

const (
	Name = "bk-gpu"

	// These are shared by the display path below and the grantable resource
	// catalog in grantable_resource.go. Spelling them once is what keeps the
	// entry offered in the creation form identical to the one the detail page
	// renders back.
	typeResource            = "resource"
	typeDisplayName         = "IEG GPU 管理平台"
	resourceItemName        = "all"
	resourceItemDisplayName = "所有"

	// The catalog is one level deep. levelResource spells the same string as
	// typeResource and is deliberately not defined in terms of it: the type is a
	// division of what can be granted and is labelled by the product, the level
	// is a rung of that division's tree and is labelled by what the rung holds.
	//
	// Note that resourceItemName above is also "all". That one is the name of an
	// entry, this one names a level, and nothing follows from them being spelled
	// alike.
	levelResource            = "resource"
	levelResourceDisplayName = "资源"

	validResource = typeResource + ":" + resourceItemName
)

type gpuRealm struct{}

// New creates the gpu Realm implementation.
func New() oauth.Realm {
	return &gpuRealm{}
}

func (r *gpuRealm) Name() string { return Name }

func (r *gpuRealm) ValidateResource(_ context.Context, resource string) error {
	if resource != validResource {
		return fmt.Errorf("invalid resource: must be %q, got %q", validResource, resource)
	}
	return nil
}

func (r *gpuRealm) ExtractAudiences(_ context.Context, resource string) ([]string, error) {
	if resource != validResource {
		return nil, fmt.Errorf("invalid resource: must be %q, got %q", validResource, resource)
	}
	return []string{validResource}, nil
}

func (r *gpuRealm) ResolveResourceDisplay(_ context.Context, resource string) (any, error) {
	if resource != validResource {
		return nil, fmt.Errorf("invalid resource: must be %q, got %q", validResource, resource)
	}
	return resourceDisplay(), nil
}

// ValidateAudiences accepts the single token this realm knows about. The realm
// is all-or-nothing, so a token list is either exactly that one entry or a
// mistake.
func (r *gpuRealm) ValidateAudiences(_ context.Context, audiences []string) error {
	if len(audiences) != 1 || audiences[0] != validResource {
		return fmt.Errorf("invalid audience: must be exactly [%q], got %v", validResource, audiences)
	}
	return nil
}

// ResolveAudienceDisplays ignores tenantID and gains nothing from being handed
// many tokens: there is nothing upstream to scope and nothing to batch.
//
// It checks each token but not how many there are, unlike ValidateAudiences,
// which demands exactly the one. That constraint governs what may be written,
// and re-imposing it here would turn a stored grant list that predates it into a
// page that will not render at all.
//
// "所有" says nothing on its own, but there is only ever this one grant, so the
// type's label above it is the whole story. The entry is the same name, display
// name and token as the catalog's sole entry, there being one source for all
// three.
func (r *gpuRealm) ResolveAudienceDisplays(
	_ context.Context,
	_ string,
	audiences []string,
) (map[string]oauth.AudienceDisplay, error) {
	displays := make(map[string]oauth.AudienceDisplay, len(audiences))
	for _, aud := range audiences {
		if aud != validResource {
			return nil, fmt.Errorf("invalid audience: must be %q, got %q", validResource, aud)
		}
		displays[aud] = oauth.AudienceDisplay{
			Type: typeResource,
			// The sole entry of a one-level catalog, so the grant is at that
			// level. resource:all reads like a select-all token but is not one:
			// GrantableResourceTypes leaves the type-wide box empty so the token
			// is offered in one place only.
			Level:       levelResource,
			Name:        resourceItemName,
			DisplayName: resourceItemDisplayName,
			Audience:    validResource,
		}
	}
	return displays, nil
}

func resourceDisplay() []ResourceDisplay {
	return []ResourceDisplay{{
		Type:        typeResource,
		DisplayName: typeDisplayName,
		Items: []ItemDisplay{{
			Name:        resourceItemName,
			DisplayName: resourceItemDisplayName,
		}},
	}}
}
