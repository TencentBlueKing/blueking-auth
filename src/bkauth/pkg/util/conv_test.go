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
	"strconv"
	"sync"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/util"
)

var _ = Describe("Conv", func() {
	Describe("ToSlice", func() {
		intSlice := []int{1}
		strSlice := []string{"abc"}

		DescribeTable("ToSlice cases", func(expected int, willError bool, input interface{}) {
			data, err := util.ToSlice(input)

			if willError {
				assert.Error(GinkgoT(), err)
			} else {
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), expected, len(data))
			}
		},
			Entry("not a slice", 0, true, ""),
			Entry("a []int{1}", 1, false, intSlice),
			Entry("a []string{abc}", 1, false, strSlice),
		)
	})

	Describe("Int64SliceToString", func() {
		DescribeTable("Int64SliceToString cases", func(expected string, input []int64, sep string) {
			assert.Equal(GinkgoT(), expected, util.Int64SliceToString(input, sep))
		},
			Entry("empty slice", "", []int64{}, ","),
			Entry("slice with 1 value", "1", []int64{1}, ","),
			Entry("slice with 3 values", "1,2,3", []int64{1, 2, 3}, ","),
		)
	})

	Describe("StringToInt64Slice", func() {
		DescribeTable("StringToInt64Slice cases", func(expected []int64, willError bool, input string, sep string) {
			data, err := util.StringToInt64Slice(input, sep)
			if willError {
				assert.Error(GinkgoT(), err)
			} else {
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), expected, data)
			}
		},
			Entry("empty string", []int64{}, false, "", ","),
			Entry("string with 1 value", []int64{1}, false, "1", ","),
			Entry("string with 3 values", []int64{1, 2, 3}, false, "1,2,3", ","),
			Entry("string with invalid values", []int64{}, true, "1,a,3", ","),
		)
	})
})

func BenchmarkInt64ToStringFormat(b *testing.B) {
	a := []int64{1, 1, 11, 11, 111, 1111, 11111, 1111111111}
	for i := 0; i < b.N; i++ {
		for _, x := range a {
			_ = strconv.FormatInt(x, 10)
		}
	}
}

func BenchmarkInt64ToStringFormatWithCache(b *testing.B) {
	a := []int64{1, 1, 11, 11, 111, 1111, 11111, 1111111111}

	m := sync.Map{}
	for _, x := range a {
		m.Store(x, strconv.FormatInt(x, 10))
	}

	for i := 0; i < b.N; i++ {
		for _, x := range a {
			if value, ok := m.Load(x); ok {
				_ = value.(string)
			}
		}
	}
}
