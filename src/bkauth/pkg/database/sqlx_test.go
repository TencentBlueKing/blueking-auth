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

package database

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("sqlx", func() {
	Describe("GetSetClause", func() {
		It("one parameters", func() {
			b := GetSetClause(map[string]interface{}{
				"user_name": "test",
			})
			assert.Equal(GinkgoT(), "user_name = :user_name", b)
		})

		It("multiple parameters", func() {
			b := GetSetClause(map[string]interface{}{
				"user_name": "test",
				"status":    true,
			})
			assert.LessOrEqual(GinkgoT(),
				"user_name = :user_name, status = :status",
				"user_name = :user_name, status = :status", b)
		})
	})
})
