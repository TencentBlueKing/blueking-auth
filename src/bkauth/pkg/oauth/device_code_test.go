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
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/oauth"
)

var _ = Describe("DeviceCode", func() {
	Describe("GenerateDeviceCode", func() {
		It("returns 32-char hex string", func() {
			code, err := oauth.GenerateDeviceCode()
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 32, len(code))
			assert.Regexp(GinkgoT(), regexp.MustCompile(`^[0-9a-f]{32}$`), code)
		})

		It("generates unique values", func() {
			code1, _ := oauth.GenerateDeviceCode()
			code2, _ := oauth.GenerateDeviceCode()
			assert.NotEqual(GinkgoT(), code1, code2)
		})
	})

	Describe("GenerateUserCode", func() {
		It("returns XXXX-XXXX format with restricted charset", func() {
			code, err := oauth.GenerateUserCode()
			assert.NoError(GinkgoT(), err)
			assert.Regexp(GinkgoT(),
				regexp.MustCompile(`^[BCDFGHJKLMNPQRSTVWXZ]{4}-[BCDFGHJKLMNPQRSTVWXZ]{4}$`), code)
		})

		It("generates unique values", func() {
			code1, _ := oauth.GenerateUserCode()
			code2, _ := oauth.GenerateUserCode()
			assert.NotEqual(GinkgoT(), code1, code2)
		})
	})

	Describe("NormalizeUserCode", func() {
		DescribeTable("cases",
			func(input, expected string) {
				assert.Equal(GinkgoT(), expected, oauth.NormalizeUserCode(input))
			},
			Entry("already normalized", "WDJB-MJHT", "WDJB-MJHT"),
			Entry("lowercase with hyphen", "wdjb-mjht", "WDJB-MJHT"),
			Entry("no hyphen 8 chars", "wdjbmjht", "WDJB-MJHT"),
			Entry("with spaces", " wdjb mjht ", "WDJB-MJHT"),
			Entry("mixed case no hyphen", "WdJbMjHt", "WDJB-MJHT"),
			Entry("9 chars no hyphen stays as-is", "ABCDEFGHI", "ABCDEFGHI"),
			Entry("7 chars no hyphen stays as-is", "ABCDEFG", "ABCDEFG"),
		)
	})
})
