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

var _ = Describe("AuthorizeRequest.Validate", func() {
	var (
		ctl       *gomock.Controller
		clientSvc *mock.MockOAuthClientService
		c         *gin.Context
		validReq  AuthorizeRequest
	)

	validFlowSpec := types.OAuthClientFlowSpec{
		ID:           "test-client",
		GrantTypes:   []string{oauth.GrantTypeAuthorizationCode},
		RedirectURIs: []string{"https://example.com/callback"},
	}

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		clientSvc = mock.NewMockOAuthClientService(ctl)

		w := httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/authorize", nil)
		util.SetRealmName(c, blueking.Name)

		validReq = AuthorizeRequest{
			ClientID:            "test-client",
			RedirectURI:         "https://example.com/callback",
			ResponseType:        oauth.ResponseTypeCode,
			State:               "random-state",
			CodeChallenge:       "challenge123",
			CodeChallengeMethod: oauth.CodeChallengeMethodS256,
			Resource:            "gateway:bk-paas:api:get_users",
		}
	})

	AfterEach(func() {
		ctl.Finish()
	})

	// --- Phase 1: canRedirect=false ---

	It("should reject empty client_id (before calling service)", func() {
		validReq.ClientID = ""
		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeFalse())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
	})

	It("should propagate service error from GetFlowSpec", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(
			types.OAuthClientFlowSpec{}, errors.New("db connection failed"),
		)

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeFalse())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("db connection failed"))
	})

	It("should reject unknown client (empty flowSpec.ID)", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(
			types.OAuthClientFlowSpec{}, nil,
		)

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeFalse())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidClient))
	})

	It("should reject empty redirect_uri", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.RedirectURI = ""

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeFalse())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
	})

	It("should reject unregistered redirect_uri", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.RedirectURI = "https://evil.com/callback"

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeFalse())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRedirectURI))
	})

	// --- Phase 2: canRedirect=true ---

	It("should reject unsupported grant type", func() {
		spec := validFlowSpec
		spec.GrantTypes = []string{oauth.GrantTypeDeviceCode}
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(spec, nil)

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeUnauthorizedClient))
	})

	It("should reject non-code response_type", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.ResponseType = "token"

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeUnsupportedResponseType))
	})

	It("should reject empty state for confidential client", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.State = ""

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
		Expect(oauthErr.Description).To(ContainSubstring("state"))
	})

	It("should accept empty state for public client", func() {
		publicClientID := "dcr_abc123def456"
		publicFlowSpec := types.OAuthClientFlowSpec{
			ID:           publicClientID,
			GrantTypes:   []string{oauth.GrantTypeAuthorizationCode},
			RedirectURIs: []string{"https://example.com/callback"},
		}
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), publicClientID).Return(publicFlowSpec, nil)

		validReq.ClientID = publicClientID
		validReq.State = ""

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject empty code_challenge", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.CodeChallenge = ""

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
		Expect(oauthErr.Description).To(ContainSubstring("code_challenge"))
	})

	It("should reject empty code_challenge_method", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.CodeChallengeMethod = ""

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
		Expect(oauthErr.Description).To(ContainSubstring("code_challenge_method"))
	})

	It("should reject unsupported code_challenge_method", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.CodeChallengeMethod = "RS256"

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
		Expect(oauthErr.Description).To(ContainSubstring("code_challenge_method"))
	})

	It("should accept plain code_challenge_method", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.CodeChallengeMethod = oauth.CodeChallengeMethodPlain

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject empty resource", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.Resource = ""

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
		Expect(oauthErr.Description).To(ContainSubstring("resource"))
	})

	It("should reject invalid resource format", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)
		validReq.Resource = ":::invalid"

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
		Expect(oauthErr.Description).To(ContainSubstring("resource"))
	})

	It("should pass with all valid parameters", func() {
		clientSvc.EXPECT().GetFlowSpec(gomock.Any(), "test-client").Return(validFlowSpec, nil)

		canRedirect, err := validReq.Validate(c, clientSvc)

		Expect(canRedirect).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())
	})
})
