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
	"encoding/json"
	"errors"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/util"
)

func readResponse(w *httptest.ResponseRecorder) util.Response {
	var got util.Response
	err := json.Unmarshal(w.Body.Bytes(), &got)
	assert.NoError(GinkgoT(), err)
	return got
}

var _ = Describe("Response", func() {
	var c *gin.Context
	// var r *gin.Engine
	var w *httptest.ResponseRecorder
	BeforeEach(func() {
		w = httptest.NewRecorder()
		gin.SetMode(gin.ReleaseMode)
		// gin.DefaultWriter = ioutil.Discard
		c, _ = gin.CreateTestContext(w)
		// c, r = gin.CreateTestContext(w)
		// r.Use(gin.Recovery())
	})

	It("BaseJSONResponse", func() {
		util.BaseJSONResponse(c, 200, 10000, "ok", nil)

		assert.Equal(GinkgoT(), 200, w.Code)

		got := readResponse(w)
		assert.Equal(GinkgoT(), 10000, got.Code)
		assert.Equal(GinkgoT(), "ok", got.Message)
	})

	It("BaseErrorJSONResponse", func() {
		util.BaseErrorJSONResponse(c, 400, 1901000, "error")
		assert.Equal(GinkgoT(), 400, c.Writer.Status())

		got := readResponse(w)
		assert.Equal(GinkgoT(), 1901000, got.Code)
		assert.Equal(GinkgoT(), "error", got.Message)
	})

	It("SuccessJSONResponse", func() {
		util.SuccessJSONResponse(c, "ok", nil)
		assert.Equal(GinkgoT(), 200, c.Writer.Status())

		got := readResponse(w)
		assert.Equal(GinkgoT(), 0, got.Code)
		assert.Equal(GinkgoT(), "ok", got.Message)
	})

	Context("SuccessJSONResponseWithDebug", func() {
		It("debug is nil", func() {
			util.SuccessJSONResponseWithDebug(c, "ok", nil, nil)
			assert.Equal(GinkgoT(), 200, c.Writer.Status())

			got := readResponse(w)
			assert.Equal(GinkgoT(), util.NoError, got.Code)
		})

		It("debug is not nil", func() {
			util.SuccessJSONResponseWithDebug(c, "ok", nil, map[string]interface{}{"hello": "world"})
			assert.Equal(GinkgoT(), 200, c.Writer.Status())

			got := readResponse(w)
			assert.Equal(GinkgoT(), util.NoError, got.Code)
		})
	})

	Describe("NewErrorJSONResponse", func() {
		It("should use default message when message is empty", func() {
			handler := util.NewErrorJSONResponse(418, 999, "teapot")
			handler(c, "")
			assert.Equal(GinkgoT(), 418, c.Writer.Status())

			got := readResponse(w)
			assert.Equal(GinkgoT(), 999, got.Code)
			assert.Equal(GinkgoT(), "teapot", got.Message)
		})

		It("should append custom message after default message", func() {
			handler := util.NewErrorJSONResponse(418, 999, "teapot")
			handler(c, "extra detail")
			assert.Equal(GinkgoT(), 418, c.Writer.Status())

			got := readResponse(w)
			assert.Equal(GinkgoT(), 999, got.Code)
			assert.Equal(GinkgoT(), "teapot:extra detail", got.Message)
		})
	})

	It("BadRequestErrorJSONResponse", func() {
		util.BadRequestErrorJSONResponse(c, "error")
		assert.Equal(GinkgoT(), 400, c.Writer.Status())

		got := readResponse(w)
		assert.Equal(GinkgoT(), util.BadRequestError, got.Code)
		assert.Equal(GinkgoT(), "bad request:error", got.Message)
	})

	It("ForbiddenJSONResponse", func() {
		util.ForbiddenJSONResponse(c, "denied")
		assert.Equal(GinkgoT(), 403, c.Writer.Status())

		got := readResponse(w)
		assert.Equal(GinkgoT(), util.ForbiddenError, got.Code)
		assert.Equal(GinkgoT(), "no permission:denied", got.Message)
	})

	It("UnauthorizedJSONResponse", func() {
		util.UnauthorizedJSONResponse(c, "invalid token")
		assert.Equal(GinkgoT(), 401, c.Writer.Status())

		got := readResponse(w)
		assert.Equal(GinkgoT(), util.UnauthorizedError, got.Code)
		assert.Equal(GinkgoT(), "unauthorized:invalid token", got.Message)
	})

	It("NotFoundJSONResponse", func() {
		util.NotFoundJSONResponse(c, "resource missing")
		assert.Equal(GinkgoT(), 404, c.Writer.Status())

		got := readResponse(w)
		assert.Equal(GinkgoT(), util.NotFoundError, got.Code)
		assert.Equal(GinkgoT(), "not found:resource missing", got.Message)
	})

	It("ConflictJSONResponse", func() {
		util.ConflictJSONResponse(c, "duplicate")
		assert.Equal(GinkgoT(), 409, c.Writer.Status())

		got := readResponse(w)
		assert.Equal(GinkgoT(), util.ConflictError, got.Code)
		assert.Equal(GinkgoT(), "conflict:duplicate", got.Message)
	})

	It("TooManyRequestsJSONResponse", func() {
		util.TooManyRequestsJSONResponse(c, "rate limited")
		assert.Equal(GinkgoT(), 429, c.Writer.Status())

		got := readResponse(w)
		assert.Equal(GinkgoT(), util.TooManyRequests, got.Code)
		assert.Equal(GinkgoT(), "too many requests:rate limited", got.Message)
	})

	It("SystemErrorJSONResponse", func() {
		util.SystemErrorJSONResponse(c, errors.New("anError"))
		assert.Equal(GinkgoT(), 500, c.Writer.Status())

		got := readResponse(w)
		assert.Equal(GinkgoT(), util.SystemError, got.Code)
		assert.Contains(GinkgoT(), got.Message, "system error")
	})

	Context("SystemErrorJSONResponseWithDebug", func() {
		It("debug is nil", func() {
			util.SystemErrorJSONResponseWithDebug(c, errors.New("anError"), nil)
			assert.Equal(GinkgoT(), 500, c.Writer.Status())

			got := readResponse(w)
			assert.Equal(GinkgoT(), util.SystemError, got.Code)
		})

		It("debug is not nil", func() {
			util.SystemErrorJSONResponseWithDebug(c, errors.New("anError"), map[string]interface{}{"hello": "world"})
			assert.Equal(GinkgoT(), 500, c.Writer.Status())

			got := readResponse(w)
			assert.Equal(GinkgoT(), util.SystemError, got.Code)
		})
	})
})
