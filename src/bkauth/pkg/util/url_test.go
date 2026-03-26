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

package util_test

import (
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/util"
)

var _ = Describe("URL", func() {
	Describe("URLJoin", func() {
		DescribeTable("should join base and path segments",
			func(expected, base string, elem ...string) {
				assert.Equal(GinkgoT(), expected, util.URLJoin(base, elem...))
			},
			Entry("base only", "http://example.com/", "http://example.com"),
			Entry("single segment", "http://example.com/api", "http://example.com", "api"),
			Entry("multiple segments", "http://example.com/api/v1/users", "http://example.com", "api", "v1", "users"),
			Entry("base with trailing slash", "http://example.com/api", "http://example.com/", "api"),
			Entry("segment with leading slash", "http://example.com/api", "http://example.com", "/api"),
			Entry("base with path", "http://example.com/v1/api/users", "http://example.com/v1", "api", "users"),
			Entry("segments needing encoding", "http://example.com/path%20with%20spaces", "http://example.com", "path with spaces"),
		)

		It("should return empty string for malformed base", func() {
			result := util.URLJoin("://invalid", "path")
			assert.Equal(GinkgoT(), "", result)
		})

		It("should return base as-is when no segments provided", func() {
			result := util.URLJoin("http://example.com/base")
			assert.Equal(GinkgoT(), "http://example.com/base", result)
		})
	})

	Describe("URLSetQuery", func() {
		It("should set query params on URL without existing query", func() {
			result := util.URLSetQuery("http://example.com/path", url.Values{
				"key": {"value"},
			})
			assert.Equal(GinkgoT(), "http://example.com/path?key=value", result)
		})

		It("should overwrite existing query params", func() {
			result := util.URLSetQuery("http://example.com/path?key=old", url.Values{
				"key": {"new"},
			})
			assert.Equal(GinkgoT(), "http://example.com/path?key=new", result)
		})

		It("should merge with existing query params", func() {
			result := util.URLSetQuery("http://example.com/path?existing=1", url.Values{
				"added": {"2"},
			})
			parsed, err := url.Parse(result)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "1", parsed.Query().Get("existing"))
			assert.Equal(GinkgoT(), "2", parsed.Query().Get("added"))
		})

		It("should handle multiple values for same key (last wins via Set)", func() {
			result := util.URLSetQuery("http://example.com", url.Values{
				"k": {"v1", "v2"},
			})
			parsed, err := url.Parse(result)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "v2", parsed.Query().Get("k"))
		})

		It("should handle empty params", func() {
			result := util.URLSetQuery("http://example.com/path?a=1", url.Values{})
			assert.Equal(GinkgoT(), "http://example.com/path?a=1", result)
		})

		It("should return raw URL as-is for malformed input", func() {
			malformed := "://\x7f"
			result := util.URLSetQuery(malformed, url.Values{"k": {"v"}})
			assert.Equal(GinkgoT(), malformed, result)
		})
	})
})
