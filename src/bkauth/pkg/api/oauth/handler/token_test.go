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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

var _ = Describe("makeTokenResponse", func() {
	It("should map TokenPair fields to TokenResponse", func() {
		pair := types.TokenPair{
			AccessToken:  "bk_abc123",
			RefreshToken: "bk_xyz789",
			ExpiresIn:    300,
			Scope:        "openid",
		}

		resp := makeTokenResponse(pair)

		Expect(resp.AccessToken).To(Equal("bk_abc123"))
		Expect(resp.TokenType).To(Equal(oauth.TokenTypeBearer))
		Expect(resp.ExpiresIn).To(Equal(int64(300)))
		Expect(resp.RefreshToken).To(Equal("bk_xyz789"))
		Expect(resp.Scope).To(Equal("openid"))
	})

	It("should leave refresh_token empty when not provided", func() {
		pair := types.TokenPair{
			AccessToken: "bk_abc123",
			ExpiresIn:   300,
		}

		resp := makeTokenResponse(pair)

		Expect(resp.RefreshToken).To(BeEmpty())
	})
})

var _ = Describe("handleTokenError", func() {
	gin.SetMode(gin.TestMode)

	type errorCase struct {
		inputErr     error
		expectedCode string
		httpStatus   int
	}

	DescribeTable("should map service errors to correct OAuth error codes",
		func(tc errorCase) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			handleTokenError(c, tc.inputErr)

			Expect(w.Code).To(Equal(tc.httpStatus))

			var body map[string]interface{}
			Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
			Expect(body["error"]).To(Equal(tc.expectedCode))
		},
		Entry("invalid authorization code",
			errorCase{oauth.ErrInvalidAuthorizationCode, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("authorization code expired",
			errorCase{oauth.ErrAuthorizationCodeExpired, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("authorization code used",
			errorCase{oauth.ErrAuthorizationCodeUsed, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("invalid code verifier",
			errorCase{oauth.ErrInvalidCodeVerifier, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("realm mismatch",
			errorCase{oauth.ErrRealmMismatch, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("client mismatch",
			errorCase{oauth.ErrClientMismatch, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("redirect URI mismatch",
			errorCase{oauth.ErrRedirectURIMismatch, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("invalid refresh token",
			errorCase{oauth.ErrInvalidRefreshToken, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("refresh token expired",
			errorCase{oauth.ErrRefreshTokenExpired, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("refresh token revoked",
			errorCase{oauth.ErrRefreshTokenRevoked, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("rotation limit exceeded",
			errorCase{oauth.ErrRotationLimitExceeded, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("unexpected error",
			errorCase{errors.New("something went wrong"), oauth.ErrorCodeServerError, http.StatusInternalServerError}),
	)
})

var _ = Describe("handleDeviceCodeError", func() {
	gin.SetMode(gin.TestMode)

	type errorCase struct {
		inputErr     error
		expectedCode string
		httpStatus   int
	}

	DescribeTable("should map device code errors to correct OAuth error codes",
		func(tc errorCase) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			handleDeviceCodeError(c, tc.inputErr)

			Expect(w.Code).To(Equal(tc.httpStatus))

			var body map[string]interface{}
			Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
			Expect(body["error"]).To(Equal(tc.expectedCode))
		},
		Entry("authorization pending",
			errorCase{oauth.ErrAuthorizationPending, oauth.ErrorCodeAuthorizationPending, http.StatusBadRequest}),
		Entry("slow down",
			errorCase{oauth.ErrSlowDown, oauth.ErrorCodeSlowDown, http.StatusBadRequest}),
		Entry("device code denied",
			errorCase{oauth.ErrDeviceCodeDenied, oauth.ErrorCodeAccessDenied, http.StatusBadRequest}),
		Entry("device code expired",
			errorCase{oauth.ErrDeviceCodeExpired, oauth.ErrorCodeExpiredToken, http.StatusBadRequest}),
		Entry("device code consumed",
			errorCase{oauth.ErrDeviceCodeConsumed, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("invalid device code",
			errorCase{oauth.ErrInvalidDeviceCode, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("realm mismatch",
			errorCase{oauth.ErrRealmMismatch, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("device code client mismatch",
			errorCase{oauth.ErrDeviceCodeClientMatch, oauth.ErrorCodeInvalidGrant, http.StatusBadRequest}),
		Entry("unexpected error",
			errorCase{errors.New("something went wrong"), oauth.ErrorCodeServerError, http.StatusInternalServerError}),
	)
})
