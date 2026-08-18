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
	"sort"

	"bkauth/pkg/oauth"
)

// GrantableResourceTypes returns the one type this realm has. Its services are
// compiled in, so there is nothing upstream to enumerate.
func (r *devopsRealm) GrantableResourceTypes() []oauth.GrantableResourceType {
	return []oauth.GrantableResourceType{{
		Name:        typeService,
		DisplayName: typeDisplayName,
		// Empty because the audience grammar has no all-services form:
		// ValidateAudiences takes service:<name> and nothing else, so there is
		// no token a select-all box could contribute.
		Audience: "",
		Levels: []oauth.GrantableResourceLevel{
			{Name: levelService, DisplayName: levelServiceDisplayName},
		},
	}}
}

// ListGrantableResource returns the services this realm knows how to name.
//
// The set is narrower than what ValidateAudiences accepts, which admits any
// service:<name> -- see serviceDisplayNames. That is deliberate: an unnamed
// service is still grantable, it just cannot be offered as an entry with a
// display name attached.
func (r *devopsRealm) ListGrantableResource(
	_ context.Context,
	_ string,
	q oauth.GrantableResourceQuery,
) (oauth.GrantableResourcePage, error) {
	if q.Type != typeService {
		return oauth.GrantableResourcePage{}, fmt.Errorf(
			"%w: %q", oauth.ErrUnknownGrantableResourceType, q.Type)
	}

	// Sorted because the source is a map, whose iteration order would otherwise
	// shuffle the entries between requests -- which paging cannot survive.
	names := make([]string, 0, len(serviceDisplayNames))
	for name := range serviceDisplayNames {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]oauth.GrantableResource, 0, len(names))
	for _, name := range names {
		entries = append(entries, oauth.GrantableResource{
			Name:        name,
			DisplayName: resolveServiceDisplayName(name),
			Audience:    servicePrefix + name,
		})
	}

	return oauth.PageFlatGrantableResources(entries, levelService, q)
}
