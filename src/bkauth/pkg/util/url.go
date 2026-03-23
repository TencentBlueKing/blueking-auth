/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
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

package util

import "net/url"

// URLJoin joins a base URL with path segments via url.JoinPath.
// Returns "" if the base URL is malformed; callers that rely on a valid
// base (e.g. from startup config) will surface the empty string quickly
// in downstream logic rather than panicking at runtime.
func URLJoin(base string, elem ...string) string {
	result, err := url.JoinPath(base, elem...)
	if err != nil {
		return ""
	}
	return result
}

// URLSetQuery parses rawURL, merges params into its existing query string
// (overwriting per key), and returns the resulting URL string.
// If rawURL is malformed, it is returned as-is.
func URLSetQuery(rawURL string, params url.Values) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	for k, vs := range params {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
