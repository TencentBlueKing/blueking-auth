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

package oauth_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/oauth"
)

var _ = Describe("Extras", func() {
	It("should serialize an unset value as an object rather than null", func() {
		// The whole point of the named type: a realm that knows nothing about a
		// row leaves the field alone, and the row still carries {} so a client
		// only ever asks which keys are present.
		data, err := json.Marshal(oauth.GrantableResource{Name: "codecc"})

		require.NoError(GinkgoT(), err)
		assert.Contains(GinkgoT(), string(data), `"extras":{}`)
	})

	It("should always emit the field, so a client need not test for its absence", func() {
		data, err := json.Marshal(oauth.AudienceDisplay{Audience: "service:codecc"})

		require.NoError(GinkgoT(), err)
		assert.Contains(GinkgoT(), string(data), `"extras":{}`)
	})

	It("should serialize the keys it holds", func() {
		data, err := json.Marshal(oauth.Extras{"is_official": true})

		require.NoError(GinkgoT(), err)
		assert.JSONEq(GinkgoT(), `{"is_official": true}`, string(data))
	})
})

var _ = Describe("ValidateLevelNames", func() {
	It("should accept a query that filters nothing", func() {
		assert.NoError(GinkgoT(), oauth.ValidateLevelNames(nil, "gateway", "api"))
	})

	It("should accept every level the type declares, together or apart", func() {
		assert.NoError(GinkgoT(), oauth.ValidateLevelNames(
			map[string]string{"api": "query"}, "gateway", "api"))
		assert.NoError(GinkgoT(), oauth.ValidateLevelNames(
			map[string]string{"gateway": "bk-", "api": "query"}, "gateway", "api"))
	})

	It("should reject a level the type does not have, naming it", func() {
		err := oauth.ValidateLevelNames(map[string]string{"mcp": "log"}, "gateway", "api")

		require.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceLevel)
		assert.Contains(GinkgoT(), err.Error(), "mcp",
			"the caller cannot fix the query unless told which level was wrong")
	})

	It("should reject a level of another type, not just an invented one", func() {
		// The levels of the sibling type are the likeliest mistake a client
		// makes, and they are as wrong here as a typo.
		assert.ErrorIs(GinkgoT(),
			oauth.ValidateLevelNames(map[string]string{"gateway": "bk-"}, "service"),
			oauth.ErrUnknownGrantableResourceLevel)
	})
})

var _ = Describe("PageFlatGrantableResources", func() {
	// The level these entries sit at. Named as devops names it, since that is
	// the shape this function serves.
	const level = "service"

	// codecc and codecc-plugin share a prefix so a keyword can match more than
	// one entry without the test leaning on which incidental letters a name
	// happens to contain.
	entries := []oauth.GrantableResource{
		{Name: "codecc", DisplayName: "CodeCC", Audience: "service:codecc"},
		{Name: "codecc-plugin", DisplayName: "CodeCC 插件", Audience: "service:codecc-plugin"},
		{Name: "turbo", DisplayName: "编译加速", Audience: "service:turbo"},
		{Name: "stream", DisplayName: "Stream", Audience: "service:stream"},
	}

	// pageOf keeps the cases below about what was matched and paged. The error
	// return has one cause, an unknown level, and its own case.
	pageOf := func(
		all []oauth.GrantableResource,
		q oauth.GrantableResourceQuery,
	) oauth.GrantableResourcePage {
		page, err := oauth.PageFlatGrantableResources(all, level, q)
		require.NoError(GinkgoT(), err)
		return page
	}
	filterBy := func(text string) map[string]string {
		return map[string]string{level: text}
	}

	It("should return everything when nothing is asked of it", func() {
		page := pageOf(entries, oauth.GrantableResourceQuery{})

		assert.Equal(GinkgoT(), 4, page.Count)
		assert.Len(GinkgoT(), page.Results, 4)
	})

	It("should preserve the order it was given", func() {
		page := pageOf(entries, oauth.GrantableResourceQuery{})

		require.Len(GinkgoT(), page.Results, 4)
		assert.Equal(GinkgoT(), "codecc", page.Results[0].Name)
		assert.Equal(GinkgoT(), "codecc-plugin", page.Results[1].Name)
		assert.Equal(GinkgoT(), "turbo", page.Results[2].Name)
		assert.Equal(GinkgoT(), "stream", page.Results[3].Name)
	})

	Describe("keywords", func() {
		It("should match the name case-insensitively", func() {
			page := pageOf(entries, oauth.GrantableResourceQuery{Keywords: filterBy("TURB")})

			require.Len(GinkgoT(), page.Results, 1)
			assert.Equal(GinkgoT(), "turbo", page.Results[0].Name)
		})

		It("should match the display name too, so a user can search what they see", func() {
			page := pageOf(entries, oauth.GrantableResourceQuery{Keywords: filterBy("编译")})

			require.Len(GinkgoT(), page.Results, 1)
			assert.Equal(GinkgoT(), "turbo", page.Results[0].Name)
		})

		It("should count only what matched", func() {
			page := pageOf(entries, oauth.GrantableResourceQuery{Keywords: filterBy("codecc")})

			assert.Equal(GinkgoT(), 2, page.Count)
			assert.Len(GinkgoT(), page.Results, 2)
		})

		It("should reject a keyword aimed at a level this catalog does not have", func() {
			// Answered rather than emptied: an empty page for a filter that was
			// never applied is indistinguishable from one that matched nothing.
			_, err := oauth.PageFlatGrantableResources(entries, level,
				oauth.GrantableResourceQuery{Keywords: map[string]string{"gateway": "codecc"}})

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceLevel)
		})
	})

	Describe("paging", func() {
		It("should slice by limit and offset", func() {
			page := pageOf(entries, oauth.GrantableResourceQuery{Limit: 1, Offset: 1})

			require.Len(GinkgoT(), page.Results, 1)
			assert.Equal(GinkgoT(), "codecc-plugin", page.Results[0].Name)
		})

		It("should report the unpaged total, which is what a client pages by", func() {
			page := pageOf(entries, oauth.GrantableResourceQuery{Limit: 1})

			assert.Equal(GinkgoT(), 4, page.Count)
			assert.Len(GinkgoT(), page.Results, 1)
		})

		It("should return an empty page past the end rather than the first entries again", func() {
			page := pageOf(entries, oauth.GrantableResourceQuery{Limit: 2, Offset: 10})

			assert.Equal(GinkgoT(), 4, page.Count)
			assert.Empty(GinkgoT(), page.Results)
		})

		It("should return a short last page", func() {
			page := pageOf(entries, oauth.GrantableResourceQuery{Limit: 3, Offset: 3})

			assert.Len(GinkgoT(), page.Results, 1)
		})

		It("should count matches, not entries, when paging a filtered set", func() {
			page := pageOf(entries,
				oauth.GrantableResourceQuery{Keywords: filterBy("codecc"), Limit: 1})

			assert.Equal(GinkgoT(), 2, page.Count)
			assert.Len(GinkgoT(), page.Results, 1)
		})

		It("should survive a negative offset no caller should send", func() {
			page := pageOf(entries, oauth.GrantableResourceQuery{Offset: -1})

			assert.Len(GinkgoT(), page.Results, 4)
		})
	})

	It("should serialize an empty result as a list rather than null", func() {
		page := pageOf(nil, oauth.GrantableResourceQuery{})

		assert.NotNil(GinkgoT(), page.Results)
		assert.Empty(GinkgoT(), page.Results)
	})
})

var _ = Describe("FindFlatGrantableResource", func() {
	const level = "service"

	entries := []oauth.GrantableResource{
		{Name: "codecc", DisplayName: "CodeCC", Audience: "service:codecc"},
		{Name: "codecc-plugin", DisplayName: "CodeCC 插件", Audience: "service:codecc-plugin"},
	}

	refTo := func(name string) oauth.GrantableResourceRef {
		return oauth.GrantableResourceRef{Names: map[string]string{level: name}}
	}

	It("should return the entry named", func() {
		entry, err := oauth.FindFlatGrantableResource(entries, level, refTo("codecc"))

		require.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), entries[0], entry,
			"the catalog row verbatim, so a resolved entry and a browsed one are one thing")
	})

	It("should match the whole name rather than a prefix of it", func() {
		// The paging helper would return both of these for "codecc". A ref names
		// one entry, so the near miss must not answer for the exact one.
		entry, err := oauth.FindFlatGrantableResource(entries, level, refTo("codecc-plugin"))

		require.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), "codecc-plugin", entry.Name)
	})

	It("should not match on the display name, which identifies nothing", func() {
		_, err := oauth.FindFlatGrantableResource(entries, level, refTo("CodeCC"))

		assert.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
	})

	It("should report a name nobody has as not found", func() {
		_, err := oauth.FindFlatGrantableResource(entries, level, refTo("turbo"))

		require.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
		assert.Contains(GinkgoT(), err.Error(), "turbo",
			"the user has to be told which name to go back and check")
	})

	It("should tell an unnamed level from one named nothing", func() {
		// A form the user has not finished is not a name to go and check, so the
		// two sentinels stay apart.
		_, err := oauth.FindFlatGrantableResource(entries, level, oauth.GrantableResourceRef{})
		assert.ErrorIs(GinkgoT(), err, oauth.ErrIncompleteGrantableResourceRef)

		_, err = oauth.FindFlatGrantableResource(entries, level, refTo(""))
		assert.ErrorIs(GinkgoT(), err, oauth.ErrIncompleteGrantableResourceRef)
	})

	It("should reject a name keyed by a level this catalog does not have", func() {
		_, err := oauth.FindFlatGrantableResource(entries, level,
			oauth.GrantableResourceRef{Names: map[string]string{"gateway": "codecc"}})

		assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceLevel)
	})
})
