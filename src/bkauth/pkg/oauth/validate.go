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
