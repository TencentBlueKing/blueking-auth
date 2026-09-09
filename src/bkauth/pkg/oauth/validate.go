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

package oauth

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// MaxResourceCount bounds how many resource indicators one request may carry.
	MaxResourceCount = 30

	// MaxResourceLength bounds one resource indicator. It is sized for the URL
	// form, which MCP requires a client to send as the canonical URI of the
	// server it wants, not for the shorter mcp: and gateway: spellings.
	MaxResourceLength = 200
)

// ValidateGrantTypes checks that every element is a server-supported grant type.
func ValidateGrantTypes(grantTypes []string) error {
	for _, gt := range grantTypes {
		if _, ok := SupportedGrantTypes[gt]; !ok {
			return fmt.Errorf("unsupported grant_type: %s", gt)
		}
	}
	return nil
}

// ValidateLogoURI checks that the URI is a valid http or https URL.
func ValidateLogoURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("logo_uri is not a valid URL: %s", raw)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("logo_uri must use http or https scheme: %s", raw)
	}

	if parsed.Host == "" {
		return fmt.Errorf("logo_uri must have a host: %s", raw)
	}

	return nil
}

// NormalizeResources cleans up the resource indicators of one request, which
// RFC 8707 Section 2 carries as a repeated resource parameter, and checks the
// bounds that hold for every realm. What an indicator may say is the realm's
// grammar, checked by Realm.ValidateResources.
//
// Surrounding whitespace is stripped, an empty indicator is dropped, and an
// indicator repeated byte for byte is kept once, in the position it first
// appeared: none of the three is something a realm could act on, and all three
// are what a client produces by splitting a configured list. The bounds are
// checked against what survives, so neither a stray "resource=" nor a resource
// asked for twice costs a slot of the count bound.
//
// Two different spellings of one resource are a realm's to collapse, not this
// function's: only the realm knows that an MCP server URL and its mcp: token
// name the same thing. Comparison here is byte for byte, so whether case
// matters is likewise the realm's call.
//
// Nothing else is repaired: an indicator reaches its realm as the client
// spelled it, so an unparseable one is the realm's rejection to make.
func NormalizeResources(resources []string) ([]string, error) {
	seen := make(map[string]bool, len(resources))
	normalized := make([]string, 0, len(resources))
	for _, resource := range resources {
		trimmed := strings.TrimSpace(resource)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("resource is required")
	}

	if len(normalized) > MaxResourceCount {
		return nil, fmt.Errorf("at most %d resource values are allowed, got %d", MaxResourceCount, len(normalized))
	}

	for _, resource := range normalized {
		if len(resource) > MaxResourceLength {
			return nil, fmt.Errorf(
				"resource value must be at most %d characters, got %d", MaxResourceLength, len(resource),
			)
		}
	}

	return normalized, nil
}
