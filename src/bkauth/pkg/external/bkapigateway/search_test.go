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

package bkapigateway

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("searchClient", func() {
	ctx := context.Background()

	Describe("SearchMCPServer", func() {
		It("should forward the filters and parse the page", func() {
			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": {"count": 1, "results": [
					{"id": 12, "name": "bk-log", "is_official": true, "mcp_server_count": 1,
					 "mcp_servers": [{"id": 301, "name": "log-query", "title": "日志查询"}]}
				]}}`)
			})

			page, err := NewSearchClient("tencent").SearchMCPServer(ctx, MCPServerQuery{
				OAuthClientType: OAuthClientTypePersonal,
				GatewayName:     "bk-",
				MCPServerName:   "query",
				Limit:           10,
				Offset:          20,
			})
			Expect(err).NotTo(HaveOccurred())

			req := (*received)[0]
			Expect(req.URL.Path).To(Equal("/api/v2/inner/oauth2/client-scopes/mcp-servers/"))
			Expect(req.Header.Get("X-Bk-Tenant-Id")).To(Equal("tencent"),
				"the tenant the client was built with, sent on every call it makes")
			Expect(req.URL.Query().Get("oauth_client_type")).To(Equal("personal"))
			Expect(req.URL.Query().Get("gateway_name")).To(Equal("bk-"))
			Expect(req.URL.Query().Get("mcp_server_name")).To(Equal("query"))
			Expect(req.URL.Query().Get("limit")).To(Equal("10"))
			Expect(req.URL.Query().Get("offset")).To(Equal("20"))

			Expect(page.Count).To(Equal(1))
			Expect(page.Results).To(HaveLen(1))
			Expect(page.Results[0].Name).To(Equal("bk-log"))
			Expect(page.Results[0].IsOfficial).To(BeTrue())
			Expect(page.Results[0].MCPServers).To(Equal([]MCPServerItem{
				{Name: "log-query", Title: "日志查询"},
			}))
		})

		It("should leave paging to the upstream defaults when unset", func() {
			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": {"count": 0, "results": []}}`)
			})

			_, err := NewSearchClient("").SearchMCPServer(ctx, MCPServerQuery{
				OAuthClientType: OAuthClientTypePersonal,
			})
			Expect(err).NotTo(HaveOccurred())

			query := (*received)[0].URL.Query()
			Expect(query).NotTo(HaveKey("limit"))
			Expect(query).NotTo(HaveKey("offset"))
			Expect(query).NotTo(HaveKey("gateway_name"))
			Expect(query).NotTo(HaveKey("mcp_server_name"))
		})

		It("should return the error and an empty page on failure", func() {
			serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusInternalServerError, `{"error": {"code": "INTERNAL"}}`)
			})

			page, err := NewSearchClient("").SearchMCPServer(ctx, MCPServerQuery{
				OAuthClientType: OAuthClientTypePersonal,
			})
			Expect(err).To(HaveOccurred())
			Expect(page).To(Equal(MCPServerPage{}))
		})
	})

	Describe("SearchResource", func() {
		It("should forward the filters and parse the page", func() {
			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": {"count": 1, "results": [
					{"id": 12, "name": "bk-log", "is_official": false, "resource_count": 2,
					 "resources": [{"id": 201, "name": "query_log", "description": "查询日志"},
					               {"id": 202, "name": "query_index_set", "description": ""}]}
				]}}`)
			})

			page, err := NewSearchClient("").SearchResource(ctx, ResourceQuery{
				OAuthClientType: OAuthClientTypePersonal,
				ResourceName:    "query",
				Limit:           5,
			})
			Expect(err).NotTo(HaveOccurred())

			req := (*received)[0]
			Expect(req.URL.Path).To(Equal("/api/v2/inner/oauth2/client-scopes/resources/"))
			Expect(req.URL.Query().Get("resource_name")).To(Equal("query"))
			Expect(req.URL.Query().Get("limit")).To(Equal("5"))

			Expect(page.Count).To(Equal(1))
			Expect(page.Results[0].Resources).To(Equal([]ResourceItem{
				{Name: "query_log", Description: "查询日志"},
				{Name: "query_index_set", Description: ""},
			}))
		})
	})
})
