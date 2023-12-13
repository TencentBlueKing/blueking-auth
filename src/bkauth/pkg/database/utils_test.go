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
	"fmt"
	"testing"

	jsoniter "github.com/json-iterator/go"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/util"
)

var _ = Describe("Utils", func() {
	Describe("truncateArgs", func() {
		Context("a string", func() {
			var a string
			BeforeEach(func() {
				a = `abc`
			})
			It("less than", func() {
				b := truncateArgs(a, 10)
				assert.Equal(GinkgoT(), `"abc"`, b)
			})
			It("just equals", func() {
				b := truncateArgs(a, 5)
				assert.Equal(GinkgoT(), `"abc"`, b)
			})
			It("greater than", func() {
				b := truncateArgs(a, 2)
				assert.Equal(GinkgoT(), `"a`, b)
			})
		})

		Context("a interface", func() {
			var a []int64
			BeforeEach(func() {
				a = []int64{1, 2, 3, 4, 5, 6}
			})
			It("less than", func() {
				b := truncateArgs(a, 20)
				assert.Equal(GinkgoT(), `[1,2,3,4,5,6]`, b)
			})
			It("just equals", func() {
				b := truncateArgs(a, 22)
				assert.Equal(GinkgoT(), `[1,2,3,4,5,6]`, b)
			})
			It("greater than", func() {
				b := truncateArgs(a, 2)
				assert.Equal(GinkgoT(), `[1`, b)
			})
		})
	})
})

func truncateInterface(v interface{}) string {
	s := fmt.Sprintf("%v", v)
	return util.TruncateString(s, 10)
}

func truncateInterfaceViaJSON(v interface{}) string {
	s, err := jsoniter.MarshalToString(v)
	if err != nil {
		s = fmt.Sprintf("%v", v)
	}
	return util.TruncateString(s, 10)
}

func BenchmarkTruncateInterface(b *testing.B) {
	x := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateInterface(x)
	}
}

func BenchmarkTruncateInterfaceViaJson(b *testing.B) {
	x := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateInterfaceViaJSON(x)
	}
}
