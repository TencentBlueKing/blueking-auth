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

// Package devops provides the OAuth realm implementation for BK DevOps (BlueKing CI) service-scoped resources.
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

const Name = "bk-devops"

// key: lowercase service name -> display name
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

// Name returns the realm identifier used in configuration and issued tokens.
func (r *devopsRealm) Name() string { return Name }

// TokenPrefix returns the prefix for opaque access tokens in this realm.
func (r *devopsRealm) TokenPrefix() string { return "bkci_" }

func parseServiceItem(item string) (string, error) {
	if !strings.HasPrefix(item, "service:") {
		return "", fmt.Errorf("invalid resource: must be in service:<name> format, got %q", item)
	}
	name := strings.TrimPrefix(item, "service:")
	if name == "" {
		return "", fmt.Errorf("invalid resource: empty name in %q", item)
	}
	return name, nil
}

// ValidateResource checks that each comma-separated entry is a service:<name> resource.
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

// ExtractAudiences derives deduplicated OAuth audience strings from the devops service resource list.
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
		aud := "service:" + name
		if !seen[aud] {
			seen[aud] = true
			audiences = append(audiences, aud)
		}
	}

	return audiences, nil
}

// ResolveResourceDisplay builds structured display data for the consent UI from service resource entries.
func (r *devopsRealm) ResolveResourceDisplay(_ context.Context, resource string) (any, error) {
	items := util.SplitCommaList(resource)
	if len(items) == 0 {
		return nil, fmt.Errorf("empty resource string")
	}

	var serviceItems []ServiceDisplay
	for _, item := range items {
		name, err := parseServiceItem(item)
		if err != nil {
			return nil, err
		}
		serviceItems = append(
			serviceItems,
			ServiceDisplay{Name: name, DisplayName: resolveServiceDisplayName(name)},
		)
	}

	return []ResourceDisplay{{
		Type:        "service",
		DisplayName: "蓝盾",
		Items:       serviceItems,
	}}, nil
}
