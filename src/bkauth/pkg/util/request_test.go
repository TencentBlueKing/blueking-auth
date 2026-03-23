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
	"net/http"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/util"
)

var _ = Describe("Request", func() {
	Describe("ReadRequestBody", func() {
		It("nil body", func() {
			r := &http.Request{Body: nil}
			body, err := util.ReadRequestBody(r)
			assert.Error(GinkgoT(), err)
			assert.Nil(GinkgoT(), body)
		})

		It("Empty response", func() {
			// read empty body
			r, _ := util.NewRequestEmptyResponse()
			body, err := util.ReadRequestBody(r)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []byte(""), body)
		})

		It("Error response", func() {
			// read error body, will error
			r, _ := util.NewRequestErrorResponse()
			_, err := util.ReadRequestBody(r)
			assert.Error(GinkgoT(), err)
		})

		It("read test content from body", func() {
			// read test content from body
			r, _ := util.NewRequestResponse()
			body, err := util.ReadRequestBody(r)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), util.TestingContent, body)
		})

		It("read twice", func() {
			// test read twice
			r, _ := util.NewRequestResponse()
			body, err := util.ReadRequestBody(r)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), util.TestingContent, body)

			body, err = util.ReadRequestBody(r)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), util.TestingContent, body)
		})
	})

	Describe("RequestID", func() {
		var c *gin.Context
		BeforeEach(func() {
			c = &gin.Context{}
		})

		It("GetRequestID", func() {
			id := util.GetRequestID(c)
			assert.Equal(GinkgoT(), "", id)
		})

		It("SetRequestID", func() {
			util.SetRequestID(c, "123")

			id := util.GetRequestID(c)
			assert.Equal(GinkgoT(), "123", id)
		})
	})

	Describe("AccessAppCode", func() {
		var c *gin.Context
		BeforeEach(func() {
			c = &gin.Context{}
		})

		It("GetAccessAppCode", func() {
			id := util.GetAccessAppCode(c)
			assert.Equal(GinkgoT(), "", id)
		})

		It("SetAccessAppCode", func() {
			util.SetAccessAppCode(c, "test")

			id := util.GetAccessAppCode(c)
			assert.Equal(GinkgoT(), "test", id)
		})
	})

	Describe("Error", func() {
		var c *gin.Context
		BeforeEach(func() {
			c = &gin.Context{}
		})

		It("GetError", func() {
			_, ok := util.GetError(c)
			assert.False(GinkgoT(), ok)
		})

		It("SetError", func() {
			expected := errors.New("test")
			util.SetError(c, expected)
			err, ok := util.GetError(c)
			assert.True(GinkgoT(), ok)
			assert.Equal(GinkgoT(), expected, err)
		})
	})

	Describe("EnableMultiTenantMode", func() {
		var c *gin.Context
		BeforeEach(func() {
			c = &gin.Context{}
		})

		It("GetEnableMultiTenantMode default", func() {
			assert.False(GinkgoT(), util.GetEnableMultiTenantMode(c))
		})

		It("SetEnableMultiTenantMode true", func() {
			util.SetEnableMultiTenantMode(c, true)
			assert.True(GinkgoT(), util.GetEnableMultiTenantMode(c))
		})

		It("SetEnableMultiTenantMode false", func() {
			util.SetEnableMultiTenantMode(c, false)
			assert.False(GinkgoT(), util.GetEnableMultiTenantMode(c))
		})
	})

	Describe("Username", func() {
		var c *gin.Context
		BeforeEach(func() {
			c = &gin.Context{}
		})

		It("GetUsername default", func() {
			assert.Equal(GinkgoT(), "", util.GetUsername(c))
		})

		It("SetUsername", func() {
			util.SetUsername(c, "admin")
			assert.Equal(GinkgoT(), "admin", util.GetUsername(c))
		})
	})

	Describe("RealmName", func() {
		var c *gin.Context
		BeforeEach(func() {
			c = &gin.Context{}
		})

		It("GetRealmName default", func() {
			assert.Equal(GinkgoT(), "", util.GetRealmName(c))
		})

		It("SetRealmName", func() {
			util.SetRealmName(c, "bk")
			assert.Equal(GinkgoT(), "bk", util.GetRealmName(c))
		})
	})

	Describe("ClientID", func() {
		var c *gin.Context
		BeforeEach(func() {
			c = &gin.Context{}
		})

		It("GetClientID default", func() {
			assert.Equal(GinkgoT(), "", util.GetClientID(c))
		})

		It("SetClientID", func() {
			util.SetClientID(c, "my-client")
			assert.Equal(GinkgoT(), "my-client", util.GetClientID(c))
		})
	})
})
