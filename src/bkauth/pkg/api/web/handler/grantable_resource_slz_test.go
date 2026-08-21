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

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/api/common"
)

func TestGrantableResourceQueryPaging(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "an omitted page and size fall back to the shared defaults",
			wantLimit:  common.DefaultPageSize,
			wantOffset: 0,
		},
		{
			name:       "the first page starts at zero",
			page:       1,
			pageSize:   20,
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "later pages offset by whole pages",
			page:       3,
			pageSize:   20,
			wantLimit:  20,
			wantOffset: 40,
		},
		{
			name:       "an omitted size still pages by the default",
			page:       2,
			wantLimit:  common.DefaultPageSize,
			wantOffset: common.DefaultPageSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := grantableResourceQueryRequest{Page: tt.page, PageSize: tt.pageSize}

			assert.Equal(t, tt.wantLimit, q.limit())
			assert.Equal(t, tt.wantOffset, q.offset())
		})
	}
}

func TestGrantableResourceQueryBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bind := func(query string) (grantableResourceQueryRequest, error) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)

		var req grantableResourceQueryRequest
		return req, c.ShouldBindQuery(&req)
	}

	t.Run("accepts a type alone", func(t *testing.T) {
		req, err := bind("type=mcp")
		require.NoError(t, err)
		assert.Equal(t, "mcp", req.Type)
	})

	t.Run("rejects a missing type", func(t *testing.T) {
		_, err := bind("page=1")
		assert.Error(t, err)
	})

	t.Run("rejects a page size past what the upstream can forward", func(t *testing.T) {
		_, err := bind("type=mcp&page_size=21")
		assert.Error(t, err)
	})

	t.Run("accepts the upstream ceiling itself", func(t *testing.T) {
		_, err := bind("type=mcp&page_size=20")
		assert.NoError(t, err)
	})

	t.Run("rejects a negative page", func(t *testing.T) {
		_, err := bind("type=mcp&page=-1")
		assert.Error(t, err)
	})

	t.Run("reads page=0 as the first page, as the shared page serializer does", func(t *testing.T) {
		// omitempty skips min=1 for the zero value, so an explicit 0 binds and
		// offset() folds it into page 1. Matching common.PageParamSerializer here
		// matters more than rejecting it: the two would otherwise answer the same
		// query differently.
		req, err := bind("type=mcp&page=0")
		require.NoError(t, err)
		assert.Equal(t, 0, req.offset())
	})

	t.Run("rejects a page far enough out to overflow the offset", func(t *testing.T) {
		_, err := bind("type=mcp&page=9223372036854775807")
		assert.Error(t, err)
	})

	t.Run("rejects a keyword past what the parser should be asked to walk", func(t *testing.T) {
		long := make([]byte, 513)
		for i := range long {
			long[i] = 'a'
		}

		_, err := bind("type=mcp&keyword=" + string(long))
		assert.Error(t, err)
	})
}

func TestGrantableResourceQueryKeywords(t *testing.T) {
	tests := []struct {
		name    string
		keyword string
		want    map[string]string
	}{
		{
			name:    "an absent keyword filters nothing, rather than filtering on empty",
			keyword: "",
			want:    nil,
		},
		{
			name:    "one level is named on its own",
			keyword: "api:query",
			want:    map[string]string{"api": "query"},
		},
		{
			name:    "both levels are AND-ed in one parameter",
			keyword: "gateway:bk-,api:query",
			want:    map[string]string{"gateway": "bk-", "api": "query"},
		},
		{
			name: "only the first separator splits, so a colon in the text survives",
			// Not a name any upstream object has, but the grammar should not be
			// the thing that decides that.
			keyword: "api:a:b",
			want:    map[string]string{"api": "a:b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := grantableResourceQueryRequest{Keyword: tt.keyword}

			keywords, err := q.keywords()
			require.NoError(t, err)
			assert.Equal(t, tt.want, keywords)
		})
	}
}

func TestGrantableResourceQueryMalformedKeywords(t *testing.T) {
	// Every case here widens the search if it is forgiven, and answering a
	// narrowed query with extra rows is worse than answering with a 400.
	tests := []struct {
		name    string
		keyword string
	}{
		{
			name:    "text with no level, which a single search box would send",
			keyword: "query",
		},
		{
			name:    "a comma in the text, which splits into a pair naming no level",
			keyword: "api:a,b",
		},
		{
			name:    "an empty level",
			keyword: ":query",
		},
		{
			name:    "a level with nothing to match",
			keyword: "api:",
		},
		{
			name:    "one level filtered twice, where honouring either drops the other",
			keyword: "api:query,api:log",
		},
		{
			name:    "a trailing separator",
			keyword: "api:query,",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := grantableResourceQueryRequest{Keyword: tt.keyword}

			_, err := q.keywords()
			assert.ErrorIs(t, err, errMalformedLevelPairs)
		})
	}
}

func TestGrantableResourceRefBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bind := func(query string) (grantableResourceRefRequest, error) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)

		var req grantableResourceRefRequest
		return req, c.ShouldBindQuery(&req)
	}

	t.Run("accepts a type and a name", func(t *testing.T) {
		req, err := bind("type=mcp&name=gateway:bk-log,mcp:bk-log-prod-query")
		require.NoError(t, err)
		assert.Equal(t, "mcp", req.Type)
	})

	t.Run("rejects a missing name, which has no sensible answer", func(t *testing.T) {
		// Unlike keyword on the listing endpoint, where an absent filter means
		// "everything" rather than "nothing to look up".
		_, err := bind("type=mcp")
		assert.Error(t, err)
	})

	t.Run("rejects a missing type", func(t *testing.T) {
		_, err := bind("name=gateway:bk-log,mcp:bk-log-prod-query")
		assert.Error(t, err)
	})

	t.Run("rejects a name past what the parser should be asked to walk", func(t *testing.T) {
		long := make([]byte, 513)
		for i := range long {
			long[i] = 'a'
		}

		_, err := bind("type=mcp&name=" + string(long))
		assert.Error(t, err)
	})
}

func TestGrantableResourceRefNames(t *testing.T) {
	t.Run("reads a name per level, the levels being the realm's to check", func(t *testing.T) {
		q := grantableResourceRefRequest{Name: "gateway:bk-log,mcp:bk-log-prod-query"}

		names, err := q.names()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"gateway": "bk-log",
			"mcp":     "bk-log-prod-query",
		}, names)
	})

	t.Run("shares the listing endpoint's grammar, so one syntax serves both", func(t *testing.T) {
		const raw = "gateway:bk-,api:query"

		fromKeyword, err := (&grantableResourceQueryRequest{Keyword: raw}).keywords()
		require.NoError(t, err)
		fromName, err := (&grantableResourceRefRequest{Name: raw}).names()
		require.NoError(t, err)

		assert.Equal(t, fromKeyword, fromName)
	})

	t.Run("rejects a bare name with no level to key it by", func(t *testing.T) {
		// The pairs are what single out the entry, so a dropped one cannot be
		// forgiven the way a dropped filter merely widens a page.
		_, err := (&grantableResourceRefRequest{Name: "bk-log-prod-query"}).names()

		assert.ErrorIs(t, err, errMalformedLevelPairs)
	})

	t.Run("errors on an empty name rather than resolving nothing", func(t *testing.T) {
		// The opposite of keywords(), which reads empty as "filter nothing". Here
		// there is nothing an empty ref could name.
		_, err := (&grantableResourceRefRequest{}).names()

		assert.ErrorIs(t, err, errMalformedLevelPairs)
	})
}
