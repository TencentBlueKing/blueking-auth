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
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/util"
)

var _ = Describe("Slice", func() {
	Describe("Deduplicate", func() {
		DescribeTable("string cases",
			func(input, expected []string) {
				result := util.Deduplicate(input)
				assert.Equal(GinkgoT(), expected, result)
			},
			Entry("no duplicates",
				[]string{"a", "b", "c"},
				[]string{"a", "b", "c"}),
			Entry("with duplicates",
				[]string{"a", "b", "a"},
				[]string{"a", "b"}),
			Entry("all same",
				[]string{"x", "x", "x"},
				[]string{"x"}),
			Entry("single",
				[]string{"a"},
				[]string{"a"}),
			Entry("empty",
				[]string{},
				[]string{}),
		)

		DescribeTable("int cases",
			func(input, expected []int) {
				result := util.Deduplicate(input)
				assert.Equal(GinkgoT(), expected, result)
			},
			Entry("no duplicates",
				[]int{1, 2, 3},
				[]int{1, 2, 3}),
			Entry("with duplicates",
				[]int{1, 2, 1, 3, 2},
				[]int{1, 2, 3}),
			Entry("all same",
				[]int{5, 5, 5},
				[]int{5}),
		)
	})
})
