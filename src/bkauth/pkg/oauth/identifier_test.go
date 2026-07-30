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

var _ = Describe("Identifier", func() {
	Describe("GenerateJTI", func() {
		It("returns UUID v4 format", func() {
			jti := oauth.GenerateJTI()
			assert.NotEmpty(GinkgoT(), jti)
			assert.Regexp(GinkgoT(),
				regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), jti)
		})

		It("generates unique values", func() {
			jti1 := oauth.GenerateJTI()
			jti2 := oauth.GenerateJTI()
			assert.NotEqual(GinkgoT(), jti1, jti2)
		})
	})

	Describe("GenerateGrantID", func() {
		It("returns UUID v4 format", func() {
			id := oauth.GenerateGrantID()
			assert.Regexp(GinkgoT(),
				regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), id)
		})

		It("generates unique values", func() {
			id1 := oauth.GenerateGrantID()
			id2 := oauth.GenerateGrantID()
			assert.NotEqual(GinkgoT(), id1, id2)
		})
	})
})
