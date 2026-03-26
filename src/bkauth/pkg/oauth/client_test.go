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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/oauth"
)

var _ = Describe("Client", func() {
	Describe("IsPublicClient", func() {
		DescribeTable("cases",
			func(clientID string, want bool) {
				assert.Equal(GinkgoT(), want, oauth.IsPublicClient(clientID))
			},
			Entry("dcr prefix", "dcr_abc123def456", true),
			Entry("dcr prefix only", "dcr_", true),
			Entry("confidential app", "my-app", false),
			Entry("empty", "", false),
			Entry("partial prefix", "dcr", false),
			Entry("uppercase prefix", "DCR_abc123", false),
		)
	})

	Describe("ResolveAppCode", func() {
		DescribeTable("cases",
			func(clientID, expected string) {
				assert.Equal(GinkgoT(), expected, oauth.ResolveAppCode(clientID))
			},
			Entry("confidential client returns client_id as app_code",
				"my-app", "my-app"),
			Entry("public client returns sentinel",
				"dcr_abc123", "public"),
		)
	})

	Describe("GenerateDynamicClientID", func() {
		It("has dcr_ prefix and correct length", func() {
			id, err := oauth.GenerateDynamicClientID()
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), strings.HasPrefix(id, "dcr_"))
			// dcr_ (4) + 16 hex chars = 20
			assert.Equal(GinkgoT(), 20, len(id))
		})

		It("generates unique values", func() {
			id1, _ := oauth.GenerateDynamicClientID()
			id2, _ := oauth.GenerateDynamicClientID()
			assert.NotEqual(GinkgoT(), id1, id2)
		})
	})
})
