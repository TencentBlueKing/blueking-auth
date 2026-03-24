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

package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

func newValidAuthCode() dao.OAuthAuthorizationCode {
	return dao.OAuthAuthorizationCode{
		Code:        "code-1",
		ClientID:    "client-1",
		RealmName:   "blueking",
		Sub:         "user-sub-1",
		Username:    "admin",
		RedirectURI: "https://example.com/callback",
		Audience:    `["aud-1"]`,
		ExpiresAt:   time.Now().Add(time.Minute),
		Used:        false,
	}
}

var _ = Describe("oauthAuthorizationCodeService", func() {
	var (
		ctl         *gomock.Controller
		mockManager *mock.MockOAuthAuthorizationCodeManager
		svc         oauthAuthorizationCodeService
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		mockManager = mock.NewMockOAuthAuthorizationCodeManager(ctl)
		svc = oauthAuthorizationCodeService{authCodeManager: mockManager}
	})

	AfterEach(func() {
		ctl.Finish()
	})

	Describe("ValidateAndConsume", func() {
		It("should reject when code does not exist", func() {
			mockManager.EXPECT().Get(gomock.Any(), "nonexistent").
				Return(dao.OAuthAuthorizationCode{}, nil)

			_, err := svc.ValidateAndConsume(
				context.Background(), "blueking", "nonexistent", "client-1", "https://example.com/callback", "",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrInvalidAuthorizationCode)
		})

		It("should reject when code is already used", func() {
			code := newValidAuthCode()
			code.Used = true
			mockManager.EXPECT().Get(gomock.Any(), "code-1").Return(code, nil)

			_, err := svc.ValidateAndConsume(
				context.Background(), "blueking", "code-1", "client-1", "https://example.com/callback", "",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrAuthorizationCodeUsed)
		})

		It("should reject when code is expired", func() {
			code := newValidAuthCode()
			code.ExpiresAt = time.Now().Add(-time.Second)
			mockManager.EXPECT().Get(gomock.Any(), "code-1").Return(code, nil)

			_, err := svc.ValidateAndConsume(
				context.Background(), "blueking", "code-1", "client-1", "https://example.com/callback", "",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrAuthorizationCodeExpired)
		})

		It("should reject when realm does not match", func() {
			mockManager.EXPECT().Get(gomock.Any(), "code-1").Return(newValidAuthCode(), nil)

			_, err := svc.ValidateAndConsume(
				context.Background(), "bk-devops", "code-1", "client-1", "https://example.com/callback", "",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrRealmMismatch)
		})

		It("should reject when client_id does not match", func() {
			mockManager.EXPECT().Get(gomock.Any(), "code-1").Return(newValidAuthCode(), nil)

			_, err := svc.ValidateAndConsume(
				context.Background(), "blueking", "code-1", "wrong-client", "https://example.com/callback", "",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrClientMismatch)
		})

		It("should reject when redirect_uri does not match", func() {
			mockManager.EXPECT().Get(gomock.Any(), "code-1").Return(newValidAuthCode(), nil)

			_, err := svc.ValidateAndConsume(
				context.Background(), "blueking", "code-1", "client-1", "https://evil.com/callback", "",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrRedirectURIMismatch)
		})

		It("should reject when PKCE verification fails (S256)", func() {
			hash := sha256.Sum256([]byte("correct-verifier"))
			challenge := base64.RawURLEncoding.EncodeToString(hash[:])

			code := newValidAuthCode()
			code.CodeChallenge = challenge
			code.CodeChallengeMethod = oauth.CodeChallengeMethodS256
			mockManager.EXPECT().Get(gomock.Any(), "code-1").Return(code, nil)

			_, err := svc.ValidateAndConsume(
				context.Background(), "blueking", "code-1", "client-1", "https://example.com/callback", "wrong-verifier",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrInvalidCodeVerifier)
		})

		It("should accept valid PKCE (S256) and consume code", func() {
			verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
			hash := sha256.Sum256([]byte(verifier))
			challenge := base64.RawURLEncoding.EncodeToString(hash[:])

			code := newValidAuthCode()
			code.CodeChallenge = challenge
			code.CodeChallengeMethod = oauth.CodeChallengeMethodS256
			mockManager.EXPECT().Get(gomock.Any(), "code-1").Return(code, nil)
			mockManager.EXPECT().MarkAsUsed(gomock.Any(), "code-1").Return(int64(1), nil)

			result, err := svc.ValidateAndConsume(
				context.Background(), "blueking", "code-1", "client-1", "https://example.com/callback", verifier,
			)

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "user-sub-1", result.Sub)
			assert.Equal(GinkgoT(), "admin", result.Username)
			assert.Equal(GinkgoT(), []string{"aud-1"}, result.Audience)
		})

		It("should return authorization code used when atomic mark loses the race", func() {
			mockManager.EXPECT().Get(gomock.Any(), "code-1").Return(newValidAuthCode(), nil)
			mockManager.EXPECT().MarkAsUsed(gomock.Any(), "code-1").Return(int64(0), nil)

			_, err := svc.ValidateAndConsume(
				context.Background(), "blueking", "code-1", "client-1", "https://example.com/callback", "",
			)

			assert.ErrorIs(GinkgoT(), err, oauth.ErrAuthorizationCodeUsed)
		})

		It("should succeed without PKCE when code_challenge is empty", func() {
			mockManager.EXPECT().Get(gomock.Any(), "code-1").Return(newValidAuthCode(), nil)
			mockManager.EXPECT().MarkAsUsed(gomock.Any(), "code-1").Return(int64(1), nil)

			result, err := svc.ValidateAndConsume(
				context.Background(), "blueking", "code-1", "client-1", "https://example.com/callback", "",
			)

			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "user-sub-1", result.Sub)
			assert.Equal(GinkgoT(), "admin", result.Username)
			assert.Equal(GinkgoT(), []string{"aud-1"}, result.Audience)
		})
	})
})

var _ = Describe("oauthAuthorizationCodeService.CreateAuthorizationCode", func() {
	var (
		ctl         *gomock.Controller
		mockManager *mock.MockOAuthAuthorizationCodeManager
		svc         oauthAuthorizationCodeService
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		mockManager = mock.NewMockOAuthAuthorizationCodeManager(ctl)
		svc = oauthAuthorizationCodeService{authCodeManager: mockManager}
	})

	AfterEach(func() {
		ctl.Finish()
	})

	It("ok", func() {
		input := types.CreateAuthorizationCodeInput{
			Code:                "test-code",
			ClientID:            "client-1",
			RealmName:           "blueking",
			Sub:                 "user-sub-1",
			Username:            "admin",
			RedirectURI:         "https://example.com/callback",
			Audience:            []string{"aud-1"},
			CodeChallenge:       "challenge-abc",
			CodeChallengeMethod: oauth.CodeChallengeMethodS256,
		}

		start := time.Now()
		mockManager.EXPECT().
			Create(gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthAuthorizationCode{})).
			DoAndReturn(func(_ context.Context, code dao.OAuthAuthorizationCode) error {
				assert.Equal(GinkgoT(), "test-code", code.Code)
				assert.Equal(GinkgoT(), "client-1", code.ClientID)
				assert.Equal(GinkgoT(), "blueking", code.RealmName)
				assert.Equal(GinkgoT(), "user-sub-1", code.Sub)
				assert.Equal(GinkgoT(), "admin", code.Username)
				assert.Equal(GinkgoT(), "https://example.com/callback", code.RedirectURI)
				assert.Equal(GinkgoT(), `["aud-1"]`, code.Audience)
				assert.Equal(GinkgoT(), "challenge-abc", code.CodeChallenge)
				assert.Equal(GinkgoT(), oauth.CodeChallengeMethodS256, code.CodeChallengeMethod)
				assert.False(GinkgoT(), code.Used)

				expectedExpiry := start.Add(time.Duration(oauth.AuthorizationCodeTTL) * time.Second)
				assert.WithinDuration(GinkgoT(), expectedExpiry, code.ExpiresAt, 2*time.Second)
				return nil
			})

		err := svc.CreateAuthorizationCode(context.Background(), input)

		assert.NoError(GinkgoT(), err)
	})

	It("create error", func() {
		input := types.CreateAuthorizationCodeInput{
			Code:      "test-code",
			ClientID:  "client-1",
			RealmName: "blueking",
			Audience:  []string{"aud-1"},
		}

		mockManager.EXPECT().
			Create(gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthAuthorizationCode{})).
			Return(errors.New("db connection lost"))

		err := svc.CreateAuthorizationCode(context.Background(), input)

		assert.Error(GinkgoT(), err)
		assert.Contains(GinkgoT(), err.Error(), "authCodeManager.Create fail")
	})
})
