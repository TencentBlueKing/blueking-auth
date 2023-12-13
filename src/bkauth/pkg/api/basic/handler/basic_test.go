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

package handler_test

import (
	"net/http"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/api/basic"
	"bkauth/pkg/config"
	"bkauth/pkg/util"
)

var _ = Describe("Basic", func() {
	var (
		t GinkgoTInterface
		r *gin.Engine
	)

	BeforeEach(func() {
		t = GinkgoT()
		r = util.SetupRouter()
		basic.Register(&config.Config{Debug: false}, r)
	})

	It("Ping", func() {
		apitest.New().
			Handler(r).
			Get("/ping").
			Expect(t).
			Body(`{"message":"pong"}`).
			Status(http.StatusOK).
			End()
	})

	It("version", func() {
		apitest.New().
			Handler(r).
			Get("/version").
			Expect(t).
			Assert(
				util.NewJSONAssertFunc(t, func(m map[string]interface{}) error {
					assert.Contains(t, m, "version")
					assert.Contains(t, m, "commit")
					assert.Contains(t, m, "buildTime")
					assert.Contains(t, m, "goVersion")
					assert.Contains(t, m, "env")
					return nil
				})).
			Status(http.StatusOK).
			End()
	})
})
