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

package errorx

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
)

type NoIsWrapError struct {
	message string
	err     error
}

func (e NoIsWrapError) Error() string {
	return e.message
}

var _ = Describe("BKAuth Error", func() {
	e1 := errors.New("a")

	e2 := Error{
		message: "bkauth_e2",
		err:     e1,
	}

	e3 := Error{
		message: "bkauth_e3",
		err:     e2,
	}

	e4 := NoIsWrapError{
		message: "no_is_wrap",
		err:     e1,
	}
	e5 := Error{
		message: "bkauth_e5",
		err:     e4,
	}

	It("err vs bkauth error", func() {
		assert.False(GinkgoT(), errors.Is(e1, e2))
		assert.True(GinkgoT(), errors.Is(e2, e1))
	})

	It("bkauth error vs bkauth error", func() {
		assert.True(GinkgoT(), errors.Is(e3, e1))
		assert.True(GinkgoT(), errors.Is(e3, e2))

		assert.False(GinkgoT(), errors.Is(e2, e3))
		assert.False(GinkgoT(), errors.Is(e2, e3))
	})

	It("noIsWrapError vs bkauth error", func() {
		assert.True(GinkgoT(), errors.Is(e5, e4))
		assert.False(GinkgoT(), errors.Is(e4, e5))
	})
})
