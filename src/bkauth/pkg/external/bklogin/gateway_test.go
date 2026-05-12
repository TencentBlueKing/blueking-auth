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

package bklogin

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BKTokenGatewayVerifier", func() {
	It("should map bk_username to sub and login_name to username", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.URL.Query().Get("bk_token")).To(Equal("token-1"))
			Expect(r.Header.Get("X-Bk-Tenant-Id")).To(Equal("system"))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {
					"bk_username": "nteuuhzxlh0jcanw",
					"tenant_id": "system",
					"login_name": "admin",
					"display_name": "admin",
					"language": "zh-cn",
					"time_zone": "Asia/Shanghai"
				}
			}`))
		}))
		defer server.Close()

		oldHTTPClient := defaultHTTPClient
		defaultHTTPClient = server.Client()
		DeferCleanup(func() {
			defaultHTTPClient = oldHTTPClient
		})

		verifier := &BKTokenGatewayVerifier{
			baseURL:         server.URL,
			authCredentials: `{"bk_app_code":"app","bk_app_secret":"secret"}`,
		}

		result, err := verifier.Verify(context.Background(), "token-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Success).To(BeTrue())
		Expect(result.Sub).To(Equal("nteuuhzxlh0jcanw"))
		Expect(result.Username).To(Equal("admin"))
		Expect(result.TenantID).To(Equal("system"))
	})
})
