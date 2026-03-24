/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth 服务 (BlueKing - Auth) available.
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

package oauth

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/config"
	pkgoauth "bkauth/pkg/oauth"
)

func TestMiddleware(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OAuth Middleware Suite")
}

var _ = Describe("authenticateConfidentialClient", func() {
	// Non-exempt paths only; exemption lookup is tested in config_test.go
	// via IsClientSecretExempt.
	It("should fail when secret is empty and no exemptions", func() {
		oauthCfg := &config.OAuth{}
		err := authenticateConfidentialClient(nil, "strict_app", "", oauthCfg, "blueking")
		assert.ErrorIs(GinkgoT(), err, pkgoauth.ErrMissingClientSecret)
	})

	It("should fail when secret is empty with nil exemption map", func() {
		oauthCfg := &config.OAuth{}
		err := authenticateConfidentialClient(nil, "any_app", "", oauthCfg, "bk-devops")
		assert.ErrorIs(GinkgoT(), err, pkgoauth.ErrMissingClientSecret)
	})
})
