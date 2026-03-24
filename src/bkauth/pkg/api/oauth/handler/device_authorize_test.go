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

package handler

import (
	"errors"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bkauth/pkg/oauth"
	"bkauth/pkg/realm/blueking"
	"bkauth/pkg/service/mock"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

var _ = Describe("DeviceAuthorizeRequest.Validate", func() {
	var (
		ctl       *gomock.Controller
		clientSvc *mock.MockOAuthClientService
		c         *gin.Context
		validReq  DeviceAuthorizeRequest
	)

	const clientID = "test-device-client"

	validFlowSpec := types.OAuthClientFlowSpec{
		ID:         clientID,
		GrantTypes: []string{oauth.GrantTypeDeviceCode},
	}

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		clientSvc = mock.NewMockOAuthClientService(ctl)

		w := httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/device/authorize", nil)
		util.SetRealmName(c, blueking.Name)
		util.SetClientID(c, clientID)

		validReq = DeviceAuthorizeRequest{
			ClientID: clientID,
			Resource: "gateway:bk-paas:api:get_users",
		}
	})

	AfterEach(func() {
		ctl.Finish()
	})

	It("should propagate service error from GetFlowSpec", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), clientID).Return(
			types.OAuthClientFlowSpec{}, errors.New("db connection failed"),
		)

		err := validReq.Validate(c, clientSvc)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("db connection failed"))
	})

	It("should reject unknown client (empty flowSpec.ID)", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), clientID).Return(
			types.OAuthClientFlowSpec{}, nil,
		)

		err := validReq.Validate(c, clientSvc)

		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidClient))
	})

	It("should reject unsupported grant type", func() {
		spec := validFlowSpec
		spec.GrantTypes = []string{oauth.GrantTypeAuthorizationCode}
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), clientID).Return(spec, nil)

		err := validReq.Validate(c, clientSvc)

		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeUnauthorizedClient))
	})

	It("should reject empty resource", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), clientID).Return(validFlowSpec, nil)
		validReq.Resource = ""

		err := validReq.Validate(c, clientSvc)

		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
		Expect(oauthErr.Description).To(ContainSubstring("resource"))
	})

	It("should reject invalid resource format", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), clientID).Return(validFlowSpec, nil)
		validReq.Resource = ":::invalid"

		err := validReq.Validate(c, clientSvc)

		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
		Expect(oauthErr.Description).To(ContainSubstring("resource"))
	})

	It("should pass with all valid parameters", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), clientID).Return(validFlowSpec, nil)

		err := validReq.Validate(c, clientSvc)

		Expect(err).NotTo(HaveOccurred())
	})
})
