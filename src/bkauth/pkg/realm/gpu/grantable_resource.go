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

// GrantableResourceTypes returns the one type this realm has. There is nothing
// upstream to enumerate: the realm is all-or-nothing, so its whole grantable set
// is the single constant below.
func (r *gpuRealm) GrantableResourceTypes() []oauth.GrantableResourceType {
	return []oauth.GrantableResourceType{{
		Name:        typeResource,
		DisplayName: typeDisplayName,
		// Empty even though resource:all reads like a select-all token, because
		// the sole entry already carries it. Offering the same token in two
		// places is how the two come to disagree.
		Audience: "",
		Levels: []oauth.GrantableResourceLevel{
			{Name: levelResource, DisplayName: levelResourceDisplayName},
		},
	}}
}

// ListGrantableResource returns the realm's single grantable entry.
//
// Note what the shape cannot say: ValidateAudiences demands *exactly* this one
// entry, so here a selection is not a choice at all. The protocol carries the
// entries but not how many of them a valid selection holds, which leaves a
// caller to discover the difference by being rejected.
func (r *gpuRealm) ListGrantableResource(
	_ context.Context,
	_ string,
	q oauth.GrantableResourceQuery,
) (oauth.GrantableResourcePage, error) {
	if q.Type != typeResource {
		return oauth.GrantableResourcePage{}, fmt.Errorf(
			"%w: %q", oauth.ErrUnknownGrantableResourceType, q.Type)
	}

	return oauth.PageFlatGrantableResources(grantableResources(), levelResource, q)
}

// ResolveGrantableResource answers by exact name. Nothing is withheld from the
// catalog here -- the set is the constant below -- so this only saves a caller
// that already knows the name from paging to find it.
func (r *gpuRealm) ResolveGrantableResource(
	_ context.Context,
	_ string,
	ref oauth.GrantableResourceRef,
) (oauth.GrantableResource, error) {
	if ref.Type != typeResource {
		return oauth.GrantableResource{}, fmt.Errorf(
			"%w: %q", oauth.ErrUnknownGrantableResourceType, ref.Type)
	}

	return oauth.FindFlatGrantableResource(grantableResources(), levelResource, ref)
}

// grantableResources is the realm's entire grantable set, read by both paths
// above so neither can offer an entry the other does not know.
func grantableResources() []oauth.GrantableResource {
	return []oauth.GrantableResource{{
		Name:        resourceItemName,
		DisplayName: resourceItemDisplayName,
		Audience:    validResource,
	}}
}
