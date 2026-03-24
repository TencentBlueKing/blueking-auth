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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

var _ = Describe("newActiveIntrospectionResponse", func() {
	It("should map all fields from ResolvedAccessToken", func() {
		token := types.ResolvedAccessToken{
			ClientID:  "my-app",
			RealmName: "blueking",
			Sub:       "sub-1",
			Username:  "admin",
			Audience:  []string{"aud-1", "aud-2"},
			ExpiresAt: 1700000000,
		}

		resp := newActiveIntrospectionResponse(token)

		Expect(resp.Active).To(BeTrue())
		Expect(resp.Username).To(Equal("admin"))
		Expect(resp.Sub).To(Equal("sub-1"))
		Expect(resp.Exp).To(Equal(int64(1700000000)))
		Expect(resp.Aud).To(Equal([]string{"aud-1", "aud-2"}))
		Expect(resp.ClientID).To(Equal("my-app"))
		Expect(resp.BkAppCode).To(Equal("my-app"))
	})

	It("should resolve BkAppCode to 'public' for DCR clients", func() {
		token := types.ResolvedAccessToken{
			ClientID: "dcr_abc123",
			Audience: []string{},
		}

		resp := newActiveIntrospectionResponse(token)

		Expect(resp.BkAppCode).To(Equal(oauth.PublicAppCode))
	})

	It("should default nil audience to empty slice", func() {
		token := types.ResolvedAccessToken{
			ClientID: "my-app",
		}

		resp := newActiveIntrospectionResponse(token)

		Expect(resp.Aud).To(Equal([]string{}))
	})
})

var _ = Describe("newInactiveIntrospectionResponse", func() {
	It("should return inactive response with error details", func() {
		resp := newInactiveIntrospectionResponse()

		Expect(resp.Active).To(BeFalse())
		Expect(resp.Aud).To(Equal([]string{}))
		Expect(resp.Error.Code).To(Equal("invalid_token"))
		Expect(resp.Error.Message).NotTo(BeEmpty())
	})
})
