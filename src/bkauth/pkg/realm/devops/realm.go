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

package devops

import (
	"context"
	"fmt"
	"strings"

	"bkauth/pkg/oauth"
	"bkauth/pkg/util"
)

// ServiceDisplay represents a single service entry for the devops realm.
type ServiceDisplay struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// ResourceDisplay is the devops-specific resource display structure.
type ResourceDisplay struct {
	Type        string           `json:"type"`
	DisplayName string           `json:"display_name"`
	Items       []ServiceDisplay `json:"items"`
}

const (
	Name = "bk-devops"

	// Shared by the display path, the parser and the grantable resource catalog
	// in grantable_resource.go, so the prefix is spelled once.
	typeService     = "service"
	typeDisplayName = "蓝盾"

	// The catalog is one level deep. levelService spells the same string as
	// typeService and is deliberately not defined in terms of it: the type is a
	// division of what can be granted and is labelled by the product, the level
	// is a rung of that division's tree and is labelled by what the rung holds.
	// Here they collapse onto one string, which is not a reason to make a rename
	// of one rewrite the other.
	levelService            = "service"
	levelServiceDisplayName = "服务"

	servicePrefix = typeService + ":"
)

// key: lowercase service name -> display name
//
// This is a display lookup, not an allowlist: parseServiceItem accepts any
// service:<name>, so an unlisted service is still a valid audience and simply
// renders under its bare name. The grantable resource catalog can therefore
// only offer the services known here, which is a subset of what validation
// admits -- a caller must not read the catalog as the set of legal values.
var serviceDisplayNames = map[string]string{
	"codecc": "CodeCC",
}

func resolveServiceDisplayName(name string) string {
	if displayName, ok := serviceDisplayNames[strings.ToLower(name)]; ok {
		return displayName
	}
	return name
}

type devopsRealm struct{}

// New creates the devops Realm implementation.
func New() oauth.Realm {
	return &devopsRealm{}
}

func (r *devopsRealm) Name() string { return Name }

func parseServiceItem(item string) (string, error) {
	if !strings.HasPrefix(item, servicePrefix) {
		return "", fmt.Errorf("invalid resource: must be in service:<name> format, got %q", item)
	}
	name := strings.TrimPrefix(item, servicePrefix)
	if name == "" {
		return "", fmt.Errorf("invalid resource: empty name in %q", item)
	}
	return name, nil
}

func (r *devopsRealm) ValidateResource(_ context.Context, resource string) error {
	items := util.SplitCommaList(resource)
	if len(items) == 0 {
		return fmt.Errorf("empty resource string")
	}
	for _, item := range items {
		if _, err := parseServiceItem(item); err != nil {
			return err
		}
	}
	return nil
}

func (r *devopsRealm) ExtractAudiences(_ context.Context, resource string) ([]string, error) {
	items := util.SplitCommaList(resource)
	if len(items) == 0 {
		return nil, fmt.Errorf("empty resource string")
	}

	seen := make(map[string]bool)
	var audiences []string

	for _, item := range items {
		name, err := parseServiceItem(item)
		if err != nil {
			return nil, err
		}
		aud := servicePrefix + name
		if !seen[aud] {
			seen[aud] = true
			audiences = append(audiences, aud)
		}
	}

	return audiences, nil
}

func (r *devopsRealm) ResolveResourceDisplay(_ context.Context, resource string) (any, error) {
	items := util.SplitCommaList(resource)
	if len(items) == 0 {
		return nil, fmt.Errorf("empty resource string")
	}

	names := make([]string, 0, len(items))
	for _, item := range items {
		name, err := parseServiceItem(item)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}

	return serviceDisplay(names), nil
}

// ValidateAudiences accepts the same tokens as ValidateResource. This realm
// grants whole services, so a client asking for one and a user picking one name
// the same thing; the two are still checked through separate entry points
// because nothing guarantees that stays true.
func (r *devopsRealm) ValidateAudiences(_ context.Context, audiences []string) error {
	if len(audiences) == 0 {
		return fmt.Errorf("empty audience list")
	}
	for _, aud := range audiences {
		if _, err := parseServiceItem(aud); err != nil {
			return err
		}
	}
	return nil
}

// ResolveAudienceDisplays ignores tenantID and gains nothing from being handed
// many tokens: the service list is compiled in, so there is no upstream to scope
// and nothing to batch.
//
// Its entries carry the same name, display name and token as the catalog entry
// for the same service, there being one source for all three.
func (r *devopsRealm) ResolveAudienceDisplays(
	_ context.Context,
	_ string,
	audiences []string,
) (map[string]oauth.AudienceDisplay, error) {
	displays := make(map[string]oauth.AudienceDisplay, len(audiences))
	for _, aud := range audiences {
		name, err := parseServiceItem(aud)
		if err != nil {
			return nil, err
		}
		displays[aud] = oauth.AudienceDisplay{
			Type:  typeService,
			Level: levelService,
			// The catalog's only level, and every grant is at it: there is no
			// level above, and no all-services token for the type-wide box to
			// contribute either.
			Name:        name,
			DisplayName: resolveServiceDisplayName(name),
			Audience:    aud,
		}
	}
	return displays, nil
}

func serviceDisplay(names []string) []ResourceDisplay {
	items := make([]ServiceDisplay, 0, len(names))
	for _, name := range names {
		items = append(items, ServiceDisplay{Name: name, DisplayName: resolveServiceDisplayName(name)})
	}

	return []ResourceDisplay{{
		Type:        typeService,
		DisplayName: typeDisplayName,
		Items:       items,
	}}
}
