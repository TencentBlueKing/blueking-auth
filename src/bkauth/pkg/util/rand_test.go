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

package util_test

import (
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/util"
)

var _ = Describe("Rand", func() {
	Describe("NewUUID", func() {
		It("should return a valid UUID v4 format (8-4-4-4-12 hex with hyphens)", func() {
			id := util.NewUUID()
			assert.Regexp(GinkgoT(), regexp.MustCompile(
				`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
			), id)
		})

		It("should return 36 characters", func() {
			id := util.NewUUID()
			assert.Equal(GinkgoT(), 36, len(id))
		})

		It("should produce different results on successive calls", func() {
			id1 := util.NewUUID()
			id2 := util.NewUUID()
			assert.NotEqual(GinkgoT(), id1, id2)
		})
	})

	Describe("NewUUIDHex", func() {
		It("should return a 32-char hex string (no hyphens)", func() {
			id := util.NewUUIDHex()
			assert.Equal(GinkgoT(), 32, len(id))
			assert.Regexp(GinkgoT(), regexp.MustCompile(`^[0-9a-f]{32}$`), id)
		})

		It("should produce different results on successive calls", func() {
			id1 := util.NewUUIDHex()
			id2 := util.NewUUIDHex()
			assert.NotEqual(GinkgoT(), id1, id2)
		})
	})

	Describe("RandString", func() {
		letterBytes := "abcdefghijklmnopqrstuvwxyz0123456789"

		DescribeTable("should return correct length", func(length int) {
			result, err := util.RandString(letterBytes, length)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), length, len(result))
		},
			Entry("length 0", 0),
			Entry("length 1", 1),
			Entry("length 10", 10),
			Entry("length 64", 64),
		)

		It("should only contain characters from the given charset", func() {
			result, err := util.RandString(letterBytes, 1000)
			assert.NoError(GinkgoT(), err)
			for _, c := range result {
				assert.Contains(GinkgoT(), letterBytes, string(c))
			}
		})

		It("should produce different results on successive calls", func() {
			r1, err := util.RandString(letterBytes, 32)
			assert.NoError(GinkgoT(), err)
			r2, err := util.RandString(letterBytes, 32)
			assert.NoError(GinkgoT(), err)
			assert.NotEqual(GinkgoT(), r1, r2)
		})
	})

	Describe("RandHex", func() {
		DescribeTable("should return correct length", func(byteLen int) {
			result, err := util.RandHex(byteLen)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), byteLen*2, len(result))
		},
			Entry("8 bytes", 8),
			Entry("16 bytes", 16),
			Entry("32 bytes", 32),
		)

		It("should only contain hex characters", func() {
			result, err := util.RandHex(32)
			assert.NoError(GinkgoT(), err)
			assert.Regexp(GinkgoT(), regexp.MustCompile(`^[0-9a-f]+$`), result)
		})

		It("should produce different results on successive calls", func() {
			r1, err := util.RandHex(16)
			assert.NoError(GinkgoT(), err)
			r2, err := util.RandHex(16)
			assert.NoError(GinkgoT(), err)
			assert.NotEqual(GinkgoT(), r1, r2)
		})
	})
})
