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
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/util"
)

var _ = Describe("Error", func() {
	Describe("ValidationErrorWrap", func() {
		It("should wrap an error into ValidationError", func() {
			inner := errors.New("field is required")
			ve := util.ValidationErrorWrap(inner)
			assert.NotNil(GinkgoT(), ve)
			assert.Equal(GinkgoT(), "field is required", ve.Error())
		})
	})

	Describe("ValidationError.Error", func() {
		It("should return the inner error message", func() {
			inner := errors.New("some validation failure")
			ve := util.ValidationErrorWrap(inner)
			assert.Equal(GinkgoT(), inner.Error(), ve.Error())
		})
	})

	Describe("IsValidationError", func() {
		It("should return true for a ValidationError", func() {
			ve := util.ValidationErrorWrap(errors.New("bad input"))
			assert.True(GinkgoT(), util.IsValidationError(ve))
		})

		It("should return true for a wrapped ValidationError", func() {
			ve := util.ValidationErrorWrap(errors.New("bad input"))
			wrapped := fmt.Errorf("outer: %w", ve)
			assert.True(GinkgoT(), util.IsValidationError(wrapped))
		})

		It("should return false for a plain error", func() {
			assert.False(GinkgoT(), util.IsValidationError(errors.New("not a validation error")))
		})

		It("should return false for nil", func() {
			assert.False(GinkgoT(), util.IsValidationError(nil))
		})
	})
})
