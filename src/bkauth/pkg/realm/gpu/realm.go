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

	validResource = "resource:all"
)

type gpuRealm struct{}

// New creates the gpu Realm implementation.
func New() oauth.Realm {
	return &gpuRealm{}
}

func (r *gpuRealm) Name() string        { return Name }
func (r *gpuRealm) TokenPrefix() string { return "bkgpu_" }

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
	return []ResourceDisplay{{
		Type:        "resource",
		DisplayName: "IEG GPU 管理平台",
		Items: []ItemDisplay{{
			Name:        "all",
			DisplayName: "所有",
		}},
	}}, nil
}
