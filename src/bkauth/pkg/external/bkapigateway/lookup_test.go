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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("lookupClient", func() {
	ctx := context.Background()

	Describe("LookupMCPServer", func() {
		It("should key the result by name and keep is_public", func() {
			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": [
					{"name": "log-query", "title": "日志查询", "is_public": true,
						"oauth2_personal_client_enabled": true,
						"oauth2_public_client_enabled": true},
					{"name": "secret-one", "title": "内部", "is_public": false,
						"oauth2_personal_client_enabled": false,
						"oauth2_public_client_enabled": true}
				]}`)
			})

			servers, err := NewLookupClient("tencent").LookupMCPServer(ctx, []string{"log-query", "secret-one"})
			Expect(err).NotTo(HaveOccurred())

			req := (*received)[0]
			Expect(req.URL.Path).To(Equal("/api/v2/inner/mcp-servers/-/lookup/"))
			Expect(req.Header.Get("X-Bk-Tenant-Id")).To(Equal("tencent"),
				"the tenant the client was built with, sent on every call it makes")
			Expect(req.URL.Query().Get("names")).To(Equal("log-query,secret-one"))
			Expect(req.URL.Query().Get("fields")).To(Equal(
				"name,title,is_public,oauth2_personal_client_enabled,oauth2_public_client_enabled"))

			Expect(servers).To(Equal(map[string]MCPServer{
				"log-query": {
					Name: "log-query", Title: "日志查询", IsPublic: true,
					OAuth2PersonalClientEnabled: true, OAuth2PublicClientEnabled: true,
				},
				"secret-one": {
					Name: "secret-one", Title: "内部", IsPublic: false,
					OAuth2PersonalClientEnabled: false, OAuth2PublicClientEnabled: true,
				},
			}))
		})

		It("should carry a private server that is open to personal tokens", func() {
			// The two facts are independent: unlisted says nothing about callable,
			// and this pair is the whole reason the resolve path exists.
			serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": [
					{"name": "secret-one", "is_public": false,
						"oauth2_personal_client_enabled": true}
				]}`)
			})

			servers, err := NewLookupClient("").LookupMCPServer(ctx, []string{"secret-one"})
			Expect(err).NotTo(HaveOccurred())
			Expect(servers["secret-one"].IsPublic).To(BeFalse())
			Expect(servers["secret-one"].OAuth2PersonalClientEnabled).To(BeTrue())
		})

		It("should not send the paged endpoint's retired name filter", func() {
			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": []}`)
			})

			_, err := NewLookupClient("").LookupMCPServer(ctx, []string{"s1"})
			Expect(err).NotTo(HaveOccurred())

			query := (*received)[0].URL.Query()
			Expect(query).NotTo(HaveKey("mcp_server_names"))
			Expect(query).NotTo(HaveKey("limit"))
			Expect(query).NotTo(HaveKey("offset"))
		})

		It("should omit names the upstream did not return", func() {
			serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": []}`)
			})

			servers, err := NewLookupClient("").LookupMCPServer(ctx, []string{"gone"})
			Expect(err).NotTo(HaveOccurred())
			Expect(servers).NotTo(HaveKey("gone"))
		})

		It("should not call out for an empty name set, which upstream answers 400", func() {
			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": []}`)
			})

			servers, err := NewLookupClient("").LookupMCPServer(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(servers).To(BeEmpty())
			Expect(*received).To(BeEmpty())
		})

		It("should split a name set larger than the upstream limit and merge the answers", func() {
			names := make([]string, namesPerRequest+2)
			for i := range names {
				names[i] = "s" + strconv.Itoa(i)
			}

			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				requested := strings.Split(r.URL.Query().Get("names"), ",")
				results := make([]string, 0, len(requested))
				for _, name := range requested {
					results = append(results, `{"name": "`+name+`", "title": "t-`+name+`", "is_public": true}`)
				}
				writeJSON(w, http.StatusOK, `{"data": [`+strings.Join(results, ",")+`]}`)
			})

			servers, err := NewLookupClient("").LookupMCPServer(ctx, names)
			Expect(err).NotTo(HaveOccurred())
			Expect(*received).To(HaveLen(2))
			Expect(servers).To(HaveLen(namesPerRequest + 2))
			Expect(servers["s51"].Title).To(Equal("t-s51"))
		})
	})

	Describe("LookupGateway", func() {
		It("should read the plain array this endpoint returns", func() {
			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": [
					{"id": 1, "name": "bk-log", "is_official": true},
					{"id": 2, "name": "custom", "is_official": false}
				]}`)
			})

			gateways, err := NewLookupClient("").LookupGateway(ctx, []string{"bk-log", "custom"})
			Expect(err).NotTo(HaveOccurred())

			req := (*received)[0]
			Expect(req.URL.Path).To(Equal("/api/v2/inner/gateways/-/lookup/"))
			Expect(req.URL.Query().Get("gateway_names")).To(Equal("bk-log,custom"))

			Expect(gateways).To(Equal(map[string]Gateway{
				"bk-log": {Name: "bk-log", IsOfficial: true},
				"custom": {Name: "custom", IsOfficial: false},
			}))
		})
	})

	Describe("LookupReleasedResource", func() {
		It("should put the gateway on the path and the names in the query", func() {
			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": [
					{"id": 1001, "name": "get_gateway", "description": "获取网关",
						"is_public": false, "oauth2_personal_client_enabled": true,
						"oauth2_public_client_enabled": false}
				]}`)
			})

			resources, err := NewLookupClient("").LookupReleasedResource(ctx, "bk-log", []string{"get_gateway"})
			Expect(err).NotTo(HaveOccurred())

			req := (*received)[0]
			Expect(req.URL.Path).To(Equal("/api/v2/inner/gateways/bk-log/released-resources/-/lookup/"))
			Expect(req.URL.Query().Get("names")).To(Equal("get_gateway"))
			Expect(req.URL.Query()).NotTo(HaveKey("resource_names"))
			Expect(req.URL.Query().Get("fields")).To(Equal(
				"name,description,is_public,oauth2_personal_client_enabled,oauth2_public_client_enabled"))

			Expect(resources).To(Equal(map[string]Resource{
				"get_gateway": {
					Name: "get_gateway", Description: "获取网关", IsPublic: false,
					OAuth2PersonalClientEnabled: true, OAuth2PublicClientEnabled: false,
				},
			}))
		})

		It("should not call out for an empty name set, which upstream answers 400", func() {
			received := serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, `{"data": []}`)
			})

			resources, err := NewLookupClient("").LookupReleasedResource(ctx, "bk-log", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(resources).To(BeEmpty())
			Expect(*received).To(BeEmpty())
		})

		It("should surface a missing gateway as a not-found error", func() {
			serveInner(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusNotFound, `{"error": {"code": "NOT_FOUND", "message": "no such gateway"}}`)
			})

			_, err := NewLookupClient("").LookupReleasedResource(ctx, "gone", []string{"a"})
			Expect(IsNotFound(err)).To(BeTrue())
		})
	})
})
