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
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/database"
	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

func useMockDefaultDB(db *sqlx.DB) func() {
	old := database.DefaultDBClient
	database.DefaultDBClient = &database.DBClient{DB: db}
	return func() {
		database.DefaultDBClient = old
	}
}

func newValidRefreshTokenDAO() dao.OAuthRefreshToken {
	return dao.OAuthRefreshToken{
		ID:            1,
		GrantID:       "grant-1",
		AccessTokenID: 101,
		ClientID:      "client-1",
		RealmName:     "blueking",
		Sub:           "sub-1",
		Username:      "user-1",
		Audience:      `["aud-1","aud-2"]`,
		ExpiresAt:     time.Now().Add(time.Hour),
		UpdatedAt:     time.Now(),
		RotationCount: 0,
	}
}

var _ = Describe("oauthTokenService", func() {
	var policy types.TokenIssuancePolicy

	BeforeEach(func() {
		policy = types.TokenIssuancePolicy{
			Prefix:          "bk_",
			AccessTokenTTL:  300,
			RefreshTokenTTL: 3600,
		}
	})

	Describe("prepareTokenPair", func() {
		It("should build a token pair with hashed tokens and encoded audience", func() {
			svc := oauthTokenService{}

			start := time.Now()
			prepared, err := svc.prepareTokenPair(
				"blueking",
				"grant-1",
				"client-1",
				"default",
				"sub-1",
				"user-1",
				[]string{"aud-1", "aud-2"},
				3,
				policy,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(prepared.accessToken).To(HavePrefix(policy.Prefix))
			Expect(prepared.refreshToken).To(HavePrefix(policy.Prefix))
			Expect(prepared.accessToken).To(HaveLen(oauth.TokenLength))
			Expect(prepared.refreshToken).To(HaveLen(oauth.TokenLength))
			Expect(prepared.expiresIn).To(Equal(policy.AccessTokenTTL))

			Expect(prepared.daoAccessToken.GrantID).To(Equal("grant-1"))
			Expect(prepared.daoAccessToken.ClientID).To(Equal("client-1"))
			Expect(prepared.daoAccessToken.RealmName).To(Equal("blueking"))
			Expect(prepared.daoAccessToken.TenantID).To(Equal("default"))
			Expect(prepared.daoAccessToken.Sub).To(Equal("sub-1"))
			Expect(prepared.daoAccessToken.Username).To(Equal("user-1"))
			Expect(prepared.daoAccessToken.TokenHash).To(Equal(oauth.HashToken(prepared.accessToken)))
			Expect(prepared.daoAccessToken.TokenMask).To(Equal(oauth.MaskToken(prepared.accessToken)))
			Expect(prepared.daoAccessToken.Revoked).To(BeFalse())
			Expect(prepared.daoAccessToken.ExpiresAt).To(BeTemporally("~", start.Add(5*time.Minute), 2*time.Second))

			Expect(prepared.daoRefreshToken.GrantID).To(Equal("grant-1"))
			Expect(prepared.daoRefreshToken.ClientID).To(Equal("client-1"))
			Expect(prepared.daoRefreshToken.RealmName).To(Equal("blueking"))
			Expect(prepared.daoRefreshToken.TenantID).To(Equal("default"))
			Expect(prepared.daoRefreshToken.Sub).To(Equal("sub-1"))
			Expect(prepared.daoRefreshToken.Username).To(Equal("user-1"))
			Expect(prepared.daoRefreshToken.TokenHash).To(Equal(oauth.HashToken(prepared.refreshToken)))
			Expect(prepared.daoRefreshToken.TokenMask).To(Equal(oauth.MaskToken(prepared.refreshToken)))
			Expect(prepared.daoRefreshToken.Revoked).To(BeFalse())
			Expect(prepared.daoRefreshToken.RotationCount).To(Equal(int64(3)))
			Expect(prepared.daoRefreshToken.ExpiresAt).To(BeTemporally("~", start.Add(time.Hour), 2*time.Second))

			Expect(prepared.daoAccessToken.Audience).To(Equal(`["aud-1","aud-2"]`))
			Expect(prepared.daoRefreshToken.Audience).To(Equal(`["aud-1","aud-2"]`))
		})
	})

	Describe("GetAccessTokenByTokenHash", func() {
		var (
			ctl                *gomock.Controller
			mockAccessManager  *mock.MockOAuthAccessTokenManager
			mockRefreshManager *mock.MockOAuthRefreshTokenManager
		)

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
			mockAccessManager = mock.NewMockOAuthAccessTokenManager(ctl)
			mockRefreshManager = mock.NewMockOAuthRefreshTokenManager(ctl)
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("should return zero value when token does not exist", func() {
			mockAccessManager.EXPECT().GetByTokenHash(gomock.Any(), "hash-1").
				Return(dao.OAuthAccessToken{}, nil)
			svc := oauthTokenService{accessTokenManager: mockAccessManager}

			token, err := svc.GetAccessTokenByTokenHash(context.Background(), "hash-1")

			Expect(err).NotTo(HaveOccurred())
			Expect(token).To(Equal(types.ResolvedAccessToken{}))
		})

		It("should decode audience and map fields", func() {
			expiresAt := time.Now().Add(10 * time.Minute)
			mockAccessManager.EXPECT().GetByTokenHash(gomock.Any(), "hash-1").
				Return(dao.OAuthAccessToken{
					ID:        1,
					ClientID:  "client-1",
					RealmName: "blueking",
					Sub:       "sub-1",
					Username:  "user-1",
					Audience:  `["aud-1","aud-2"]`,
					ExpiresAt: expiresAt,
					Revoked:   true,
				}, nil)
			svc := oauthTokenService{accessTokenManager: mockAccessManager}

			token, err := svc.GetAccessTokenByTokenHash(context.Background(), "hash-1")

			Expect(err).NotTo(HaveOccurred())
			Expect(token).To(Equal(types.ResolvedAccessToken{
				ClientID:  "client-1",
				RealmName: "blueking",
				Sub:       "sub-1",
				Username:  "user-1",
				Audience:  []string{"aud-1", "aud-2"},
				ExpiresAt: expiresAt.Unix(),
				Revoked:   true,
			}))
		})

		It("should return wrapped error when audience is invalid json", func() {
			mockAccessManager.EXPECT().GetByTokenHash(gomock.Any(), "hash-1").
				Return(dao.OAuthAccessToken{ID: 1, Audience: "{invalid-json}"}, nil)
			svc := oauthTokenService{accessTokenManager: mockAccessManager}

			_, err := svc.GetAccessTokenByTokenHash(context.Background(), "hash-1")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("json.Unmarshal audience fail"))
		})

		_ = mockRefreshManager
	})

	Describe("RefreshAccessToken", func() {
		var (
			ctl                *gomock.Controller
			mockAccessManager  *mock.MockOAuthAccessTokenManager
			mockRefreshManager *mock.MockOAuthRefreshTokenManager
			svc                oauthTokenService
		)

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
			mockAccessManager = mock.NewMockOAuthAccessTokenManager(ctl)
			mockRefreshManager = mock.NewMockOAuthRefreshTokenManager(ctl)
			svc = oauthTokenService{
				accessTokenManager:  mockAccessManager,
				refreshTokenManager: mockRefreshManager,
			}
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("should reject unknown refresh tokens", func() {
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).
				Return(dao.OAuthRefreshToken{}, nil)

			_, err := svc.RefreshAccessToken(context.Background(), "blueking", "refresh-1", "client-1", policy)

			Expect(err).To(MatchError(oauth.ErrInvalidRefreshToken))
		})

		It("should reject refresh tokens from a different realm", func() {
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).
				Return(newValidRefreshTokenDAO(), nil)

			_, err := svc.RefreshAccessToken(context.Background(), "bk-devops", "refresh-1", "client-1", policy)

			Expect(err).To(MatchError(oauth.ErrRealmMismatch))
		})

		It("should reject refresh tokens owned by another client", func() {
			rt := newValidRefreshTokenDAO()
			rt.ClientID = "another-client"
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).Return(rt, nil)

			_, err := svc.RefreshAccessToken(context.Background(), "blueking", "refresh-1", "client-1", policy)

			Expect(err).To(MatchError(oauth.ErrClientMismatch))
		})

		It("should reject revoked refresh tokens inside grace period without revoking the family", func() {
			rt := newValidRefreshTokenDAO()
			rt.Revoked = true
			rt.UpdatedAt = time.Now()
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).Return(rt, nil)

			_, err := svc.RefreshAccessToken(context.Background(), "blueking", "refresh-1", "client-1", policy)

			Expect(err).To(MatchError(oauth.ErrRefreshTokenRevoked))
		})

		It("should revoke the token family when a revoked token is replayed after grace period", func() {
			rt := newValidRefreshTokenDAO()
			rt.Revoked = true
			rt.UpdatedAt = time.Now().Add(-oauth.ReplayDetectionGracePeriod - time.Second)
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).Return(rt, nil)

			mockRefreshManager.EXPECT().
				RevokeByGrantIDWithTx(gomock.Any(), gomock.Any(), "grant-1").
				Return(int64(1), nil)
			mockAccessManager.EXPECT().
				RevokeByGrantIDWithTx(gomock.Any(), gomock.Any(), "grant-1").
				Return(int64(1), nil)

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()
			restore := useMockDefaultDB(db)
			defer restore()

			_, err := svc.RefreshAccessToken(context.Background(), "blueking", "refresh-1", "client-1", policy)

			Expect(err).To(MatchError(oauth.ErrRefreshTokenRevoked))
			Expect(dbMock.ExpectationsWereMet()).To(Succeed())
		})

		It("should reject expired refresh tokens", func() {
			rt := newValidRefreshTokenDAO()
			rt.ExpiresAt = time.Now().Add(-time.Second)
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).Return(rt, nil)

			_, err := svc.RefreshAccessToken(context.Background(), "blueking", "refresh-1", "client-1", policy)

			Expect(err).To(MatchError(oauth.ErrRefreshTokenExpired))
		})

		It("should revoke the family when rotation limit is exceeded", func() {
			rt := newValidRefreshTokenDAO()
			rt.RotationCount = oauth.MaxRefreshTokenRotations
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).Return(rt, nil)

			mockRefreshManager.EXPECT().
				RevokeByGrantIDWithTx(gomock.Any(), gomock.Any(), "grant-1").
				Return(int64(1), nil)
			mockAccessManager.EXPECT().
				RevokeByGrantIDWithTx(gomock.Any(), gomock.Any(), "grant-1").
				Return(int64(1), nil)

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()
			restore := useMockDefaultDB(db)
			defer restore()

			_, err := svc.RefreshAccessToken(context.Background(), "blueking", "refresh-1", "client-1", policy)

			Expect(err).To(MatchError(oauth.ErrRotationLimitExceeded))
			Expect(dbMock.ExpectationsWereMet()).To(Succeed())
		})

		It("should return wrapped error when stored audience is invalid", func() {
			rt := newValidRefreshTokenDAO()
			rt.Audience = "{invalid-json}"
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).Return(rt, nil)

			_, err := svc.RefreshAccessToken(context.Background(), "blueking", "refresh-1", "client-1", policy)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("json.Unmarshal audience fail"))
		})

		It("should return refresh token revoked when the CAS revoke loses the race", func() {
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).
				Return(newValidRefreshTokenDAO(), nil)
			mockRefreshManager.EXPECT().
				RevokeIfNotRevokedWithTx(gomock.Any(), gomock.Any(), int64(1)).
				Return(int64(0), nil)

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectRollback()
			restore := useMockDefaultDB(db)
			defer restore()

			_, err := svc.RefreshAccessToken(context.Background(), "blueking", "refresh-1", "client-1", policy)

			Expect(err).To(MatchError(oauth.ErrRefreshTokenRevoked))
			Expect(dbMock.ExpectationsWereMet()).To(Succeed())
		})

		It("should revoke old tokens and persist a rotated pair", func() {
			rt := newValidRefreshTokenDAO()
			rt.RotationCount = 7
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), gomock.Any()).Return(rt, nil)

			mockRefreshManager.EXPECT().
				RevokeIfNotRevokedWithTx(gomock.Any(), gomock.Any(), int64(1)).
				Return(int64(1), nil)
			mockAccessManager.EXPECT().
				RevokeWithTx(gomock.Any(), gomock.Any(), int64(101)).
				Return(int64(1), nil)
			mockAccessManager.EXPECT().
				CreateWithTx(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthAccessToken{})).
				DoAndReturn(func(_ context.Context, _ *sqlx.Tx, token dao.OAuthAccessToken) (int64, error) {
					Expect(token.GrantID).To(Equal("grant-1"))
					Expect(token.ClientID).To(Equal("client-1"))
					Expect(token.RealmName).To(Equal("blueking"))
					return int64(1001), nil
				})
			mockRefreshManager.EXPECT().
				CreateWithTx(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthRefreshToken{})).
				DoAndReturn(func(_ context.Context, _ *sqlx.Tx, token dao.OAuthRefreshToken) (int64, error) {
					Expect(token.AccessTokenID).To(Equal(int64(1001)))
					Expect(token.GrantID).To(Equal("grant-1"))
					Expect(token.RealmName).To(Equal("blueking"))
					Expect(token.RotationCount).To(Equal(int64(8)))
					return int64(2002), nil
				})

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()
			restore := useMockDefaultDB(db)
			defer restore()

			pair, err := svc.RefreshAccessToken(context.Background(), "blueking", "refresh-1", "client-1", policy)

			Expect(err).NotTo(HaveOccurred())
			Expect(pair.AccessToken).To(HavePrefix(policy.Prefix))
			Expect(pair.RefreshToken).To(HavePrefix(policy.Prefix))
			Expect(pair.ExpiresIn).To(Equal(policy.AccessTokenTTL))
			Expect(dbMock.ExpectationsWereMet()).To(Succeed())
		})
	})

	Describe("RevokeToken", func() {
		var (
			ctl                *gomock.Controller
			mockAccessManager  *mock.MockOAuthAccessTokenManager
			mockRefreshManager *mock.MockOAuthRefreshTokenManager
			svc                oauthTokenService
		)

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
			mockAccessManager = mock.NewMockOAuthAccessTokenManager(ctl)
			mockRefreshManager = mock.NewMockOAuthRefreshTokenManager(ctl)
			svc = oauthTokenService{
				accessTokenManager:  mockAccessManager,
				refreshTokenManager: mockRefreshManager,
			}
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("should revoke an access token when it exists", func() {
			mockAccessManager.EXPECT().GetByTokenHash(gomock.Any(), "hash-1").
				Return(dao.OAuthAccessToken{ID: 101, ClientID: "client-1"}, nil)
			mockAccessManager.EXPECT().Revoke(gomock.Any(), int64(101)).Return(int64(1), nil)

			err := svc.RevokeToken(context.Background(), "hash-1", "client-1")

			Expect(err).NotTo(HaveOccurred())
		})

		It("should revoke refresh token and linked access token in one transaction", func() {
			mockAccessManager.EXPECT().GetByTokenHash(gomock.Any(), "hash-1").
				Return(dao.OAuthAccessToken{}, nil)
			mockRefreshManager.EXPECT().GetByTokenHash(gomock.Any(), "hash-1").
				Return(dao.OAuthRefreshToken{ID: 1, AccessTokenID: 101, ClientID: "client-1"}, nil)

			mockRefreshManager.EXPECT().
				RevokeWithTx(gomock.Any(), gomock.Any(), int64(1)).Return(int64(1), nil)
			mockAccessManager.EXPECT().
				RevokeWithTx(gomock.Any(), gomock.Any(), int64(101)).Return(int64(1), nil)

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()
			restore := useMockDefaultDB(db)
			defer restore()

			err := svc.RevokeToken(context.Background(), "hash-1", "client-1")

			Expect(err).NotTo(HaveOccurred())
			Expect(dbMock.ExpectationsWereMet()).To(Succeed())
		})
	})

	Describe("RevokeByGrantID", func() {
		var (
			ctl                *gomock.Controller
			mockAccessManager  *mock.MockOAuthAccessTokenManager
			mockRefreshManager *mock.MockOAuthRefreshTokenManager
			svc                oauthTokenService
		)

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
			mockAccessManager = mock.NewMockOAuthAccessTokenManager(ctl)
			mockRefreshManager = mock.NewMockOAuthRefreshTokenManager(ctl)
			svc = oauthTokenService{
				accessTokenManager:  mockAccessManager,
				refreshTokenManager: mockRefreshManager,
			}
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("should revoke refresh tokens before access tokens", func() {
			first := mockRefreshManager.EXPECT().
				RevokeByGrantIDWithTx(gomock.Any(), gomock.Any(), "grant-1").
				Return(int64(1), nil)
			mockAccessManager.EXPECT().
				RevokeByGrantIDWithTx(gomock.Any(), gomock.Any(), "grant-1").
				Return(int64(1), nil).
				After(first)

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()
			restore := useMockDefaultDB(db)
			defer restore()

			err := svc.RevokeByGrantID(context.Background(), "grant-1")

			Expect(err).NotTo(HaveOccurred())
			Expect(dbMock.ExpectationsWereMet()).To(Succeed())
		})
	})
})

var _ = Describe("oauthTokenService.IssueTokensForAuthorizationCode", func() {
	var (
		ctl                *gomock.Controller
		mockAccessManager  *mock.MockOAuthAccessTokenManager
		mockRefreshManager *mock.MockOAuthRefreshTokenManager
		svc                oauthTokenService
		policy             types.TokenIssuancePolicy
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		mockAccessManager = mock.NewMockOAuthAccessTokenManager(ctl)
		mockRefreshManager = mock.NewMockOAuthRefreshTokenManager(ctl)
		svc = oauthTokenService{
			accessTokenManager:  mockAccessManager,
			refreshTokenManager: mockRefreshManager,
		}
		policy = types.TokenIssuancePolicy{
			Prefix:          "bk_",
			AccessTokenTTL:  300,
			RefreshTokenTTL: 3600,
		}
	})

	AfterEach(func() {
		ctl.Finish()
	})

	It("ok", func() {
		mockAccessManager.EXPECT().
			CreateWithTx(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthAccessToken{})).
			DoAndReturn(func(_ context.Context, _ *sqlx.Tx, token dao.OAuthAccessToken) (int64, error) {
				Expect(token.ClientID).To(Equal("client-1"))
				Expect(token.RealmName).To(Equal("blueking"))
				Expect(token.Sub).To(Equal("sub-1"))
				Expect(token.Username).To(Equal("user-1"))
				Expect(token.Audience).To(Equal(`["aud-1","aud-2"]`))
				Expect(token.GrantID).NotTo(BeEmpty())
				Expect(token.Revoked).To(BeFalse())
				return int64(1001), nil
			})
		mockRefreshManager.EXPECT().
			CreateWithTx(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthRefreshToken{})).
			DoAndReturn(func(_ context.Context, _ *sqlx.Tx, token dao.OAuthRefreshToken) (int64, error) {
				Expect(token.AccessTokenID).To(Equal(int64(1001)))
				Expect(token.ClientID).To(Equal("client-1"))
				Expect(token.RealmName).To(Equal("blueking"))
				Expect(token.RotationCount).To(Equal(oauth.InitialRotationCount))
				return int64(2001), nil
			})

		db, dbMock := database.NewMockSqlxDB()
		dbMock.ExpectBegin()
		dbMock.ExpectCommit()
		restore := useMockDefaultDB(db)
		defer restore()

		pair, err := svc.IssueTokensForAuthorizationCode(
			context.Background(), "blueking", "client-1", "default", "sub-1", "user-1",
			[]string{"aud-1", "aud-2"}, policy,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(pair.AccessToken).To(HavePrefix(policy.Prefix))
		Expect(pair.RefreshToken).To(HavePrefix(policy.Prefix))
		Expect(pair.ExpiresIn).To(Equal(policy.AccessTokenTTL))
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	It("tx error", func() {
		db, dbMock := database.NewMockSqlxDB()
		dbMock.ExpectBegin().WillReturnError(errors.New("begin failed"))
		restore := useMockDefaultDB(db)
		defer restore()

		_, err := svc.IssueTokensForAuthorizationCode(
			context.Background(), "blueking", "client-1", "default", "sub-1", "user-1",
			[]string{"aud-1"}, policy,
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("GenerateDefaultDBTx fail"))
	})
})

var _ = Describe("oauthTokenService.IssueTokensForDeviceCode", func() {
	var (
		ctl                *gomock.Controller
		mockAccessManager  *mock.MockOAuthAccessTokenManager
		mockRefreshManager *mock.MockOAuthRefreshTokenManager
		svc                oauthTokenService
		policy             types.TokenIssuancePolicy
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		mockAccessManager = mock.NewMockOAuthAccessTokenManager(ctl)
		mockRefreshManager = mock.NewMockOAuthRefreshTokenManager(ctl)
		svc = oauthTokenService{
			accessTokenManager:  mockAccessManager,
			refreshTokenManager: mockRefreshManager,
		}
		policy = types.TokenIssuancePolicy{
			Prefix:          "bk_",
			AccessTokenTTL:  300,
			RefreshTokenTTL: 3600,
		}
	})

	AfterEach(func() {
		ctl.Finish()
	})

	It("ok", func() {
		mockAccessManager.EXPECT().
			CreateWithTx(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthAccessToken{})).
			DoAndReturn(func(_ context.Context, _ *sqlx.Tx, token dao.OAuthAccessToken) (int64, error) {
				Expect(token.ClientID).To(Equal("client-1"))
				Expect(token.RealmName).To(Equal("blueking"))
				Expect(token.Sub).To(Equal("sub-1"))
				Expect(token.Username).To(Equal("user-1"))
				Expect(token.Audience).To(Equal(`["aud-1"]`))
				Expect(token.GrantID).NotTo(BeEmpty())
				Expect(token.Revoked).To(BeFalse())
				return int64(1001), nil
			})
		mockRefreshManager.EXPECT().
			CreateWithTx(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthRefreshToken{})).
			DoAndReturn(func(_ context.Context, _ *sqlx.Tx, token dao.OAuthRefreshToken) (int64, error) {
				Expect(token.AccessTokenID).To(Equal(int64(1001)))
				Expect(token.ClientID).To(Equal("client-1"))
				Expect(token.RealmName).To(Equal("blueking"))
				Expect(token.RotationCount).To(Equal(oauth.InitialRotationCount))
				return int64(2001), nil
			})

		db, dbMock := database.NewMockSqlxDB()
		dbMock.ExpectBegin()
		dbMock.ExpectCommit()
		restore := useMockDefaultDB(db)
		defer restore()

		pair, err := svc.IssueTokensForDeviceCode(
			context.Background(), "blueking", "client-1", "default", "sub-1", "user-1",
			[]string{"aud-1"}, policy,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(pair.AccessToken).To(HavePrefix(policy.Prefix))
		Expect(pair.RefreshToken).To(HavePrefix(policy.Prefix))
		Expect(pair.ExpiresIn).To(Equal(policy.AccessTokenTTL))
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	It("tx error", func() {
		db, dbMock := database.NewMockSqlxDB()
		dbMock.ExpectBegin().WillReturnError(errors.New("begin failed"))
		restore := useMockDefaultDB(db)
		defer restore()

		_, err := svc.IssueTokensForDeviceCode(
			context.Background(), "blueking", "client-1", "default", "sub-1", "user-1",
			[]string{"aud-1"}, policy,
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("GenerateDefaultDBTx fail"))
	})
})
