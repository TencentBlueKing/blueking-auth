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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
	"bkauth/pkg/service/types"
)

var _ = Describe("oauthClientService", func() {
	var (
		ctl         *gomock.Controller
		mockManager *mock.MockOAuthClientManager
		svc         oauthClientService
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		mockManager = mock.NewMockOAuthClientManager(ctl)
		svc = oauthClientService{manager: mockManager}
	})

	AfterEach(func() {
		ctl.Finish()
	})

	ctx := context.Background()

	Describe("Get", func() {
		It("should return zero-value when client does not exist", func() {
			mockManager.EXPECT().Get(gomock.Any(), "nonexistent").Return(dao.OAuthClient{}, nil)

			client, err := svc.Get(ctx, "nonexistent")

			Expect(err).NotTo(HaveOccurred())
			Expect(client).To(Equal(types.OAuthClient{}))
		})

		It("should convert DAO client to types client", func() {
			now := time.Now()
			mockManager.EXPECT().Get(gomock.Any(), "dcr_abc123").Return(dao.OAuthClient{
				ID:           "dcr_abc123",
				Name:         "My App",
				Type:         "public",
				RedirectURIs: `["https://example.com/callback","https://other.com/cb"]`,
				GrantTypes:   "authorization_code,refresh_token",
				LogoURI:      "https://example.com/logo.png",
				CreatedAt:    now,
			}, nil)

			client, err := svc.Get(ctx, "dcr_abc123")

			Expect(err).NotTo(HaveOccurred())
			Expect(client.ID).To(Equal("dcr_abc123"))
			Expect(client.Name).To(Equal("My App"))
			Expect(client.Type).To(Equal("public"))
			Expect(
				client.RedirectURIs,
			).To(
				Equal([]string{"https://example.com/callback", "https://other.com/cb"}),
			)
			Expect(client.GrantTypes).To(Equal([]string{"authorization_code", "refresh_token"}))
			Expect(client.LogoURI).To(Equal("https://example.com/logo.png"))
			Expect(client.CreatedAt).To(Equal(now.Unix()))
		})

		It("should return error when redirect_uris JSON is invalid", func() {
			mockManager.EXPECT().Get(gomock.Any(), "dcr_abc123").Return(dao.OAuthClient{
				ID:           "dcr_abc123",
				RedirectURIs: "{bad-json}",
			}, nil)

			_, err := svc.Get(ctx, "dcr_abc123")

			Expect(err).To(HaveOccurred())
		})

		It("should propagate manager errors", func() {
			mockManager.EXPECT().Get(gomock.Any(), "dcr_abc123").
				Return(dao.OAuthClient{}, errors.New("db connection failed"))

			_, err := svc.Get(ctx, "dcr_abc123")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("manager.Get"))
		})
	})

	Describe("Exists", func() {
		It("should return true when client exists", func() {
			mockManager.EXPECT().Exists(gomock.Any(), "client-1").Return(true, nil)

			exists, err := svc.Exists(ctx, "client-1")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should return false when client does not exist", func() {
			mockManager.EXPECT().Exists(gomock.Any(), "nonexistent").Return(false, nil)

			exists, err := svc.Exists(ctx, "nonexistent")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should propagate manager errors", func() {
			mockManager.EXPECT().Exists(gomock.Any(), "client-1").Return(false, errors.New("db error"))

			_, err := svc.Exists(ctx, "client-1")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetFlowSpec", func() {
		It("should return zero-value when client does not exist", func() {
			mockManager.EXPECT().GetGrants(gomock.Any(), "nonexistent").
				Return(dao.OAuthClientGrants{}, nil)

			spec, err := svc.GetFlowSpec(ctx, "nonexistent")

			Expect(err).NotTo(HaveOccurred())
			Expect(spec).To(Equal(types.OAuthClientFlowSpec{}))
		})

		It("should parse redirect_uris and split grant_types", func() {
			mockManager.EXPECT().GetGrants(gomock.Any(), "client-1").
				Return(dao.OAuthClientGrants{
					ID:           "client-1",
					RedirectURIs: `["https://a.com/cb","https://b.com/cb"]`,
					GrantTypes:   "authorization_code,refresh_token,urn:ietf:params:oauth:grant-type:device_code",
				}, nil)

			spec, err := svc.GetFlowSpec(ctx, "client-1")

			Expect(err).NotTo(HaveOccurred())
			Expect(spec.ID).To(Equal("client-1"))
			Expect(spec.RedirectURIs).To(Equal([]string{"https://a.com/cb", "https://b.com/cb"}))
			Expect(spec.GrantTypes).To(Equal([]string{
				"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code",
			}))
		})

		It("should return error when redirect_uris JSON is invalid", func() {
			mockManager.EXPECT().GetGrants(gomock.Any(), "client-1").
				Return(dao.OAuthClientGrants{ID: "client-1", RedirectURIs: "not-json"}, nil)

			_, err := svc.GetFlowSpec(ctx, "client-1")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("json.Unmarshal redirectURIs fail"))
		})

		It("should propagate manager errors", func() {
			mockManager.EXPECT().GetGrants(gomock.Any(), "client-1").
				Return(dao.OAuthClientGrants{}, errors.New("db error"))

			_, err := svc.GetFlowSpec(ctx, "client-1")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetProfile", func() {
		It("should return zero-value when client does not exist", func() {
			mockManager.EXPECT().GetDisplay(gomock.Any(), "nonexistent").
				Return(dao.OAuthClientDisplay{}, nil)

			profile, err := svc.GetProfile(ctx, "nonexistent")

			Expect(err).NotTo(HaveOccurred())
			Expect(profile).To(Equal(types.OAuthClientProfile{}))
		})

		It("should map DAO display fields to types profile", func() {
			mockManager.EXPECT().GetDisplay(gomock.Any(), "client-1").
				Return(dao.OAuthClientDisplay{
					ID:      "client-1",
					Name:    "My App",
					LogoURI: "https://example.com/logo.png",
				}, nil)

			profile, err := svc.GetProfile(ctx, "client-1")

			Expect(err).NotTo(HaveOccurred())
			Expect(profile).To(Equal(types.OAuthClientProfile{
				ID:      "client-1",
				Name:    "My App",
				LogoURI: "https://example.com/logo.png",
			}))
		})

		It("should propagate manager errors", func() {
			mockManager.EXPECT().GetDisplay(gomock.Any(), "client-1").
				Return(dao.OAuthClientDisplay{}, errors.New("db error"))

			_, err := svc.GetProfile(ctx, "client-1")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("DynamicRegister", func() {
		It("should create client and return the registered client", func() {
			now := time.Now()
			mockManager.EXPECT().
				Create(gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthClient{})).
				DoAndReturn(func(_ context.Context, client dao.OAuthClient) error {
					Expect(client.Type).To(Equal("public"))
					Expect(client.Name).To(Equal("Test App"))
					return nil
				})
			mockManager.EXPECT().Get(gomock.Any(), gomock.Any()).
				Return(dao.OAuthClient{
					ID:           "dcr_placeholder",
					Name:         "Test App",
					Type:         "public",
					RedirectURIs: `["https://example.com/cb"]`,
					GrantTypes:   "authorization_code",
					CreatedAt:    now,
				}, nil)

			client, err := svc.DynamicRegister(ctx, types.OAuthClientDynamicRegistrationInput{
				Name:         "Test App",
				RedirectURIs: []string{"https://example.com/cb"},
				GrantTypes:   []string{"authorization_code"},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(client.Name).To(Equal("Test App"))
		})

		It("should propagate create errors", func() {
			mockManager.EXPECT().
				Create(gomock.Any(), gomock.AssignableToTypeOf(dao.OAuthClient{})).
				Return(errors.New("duplicate key"))

			_, err := svc.DynamicRegister(ctx, types.OAuthClientDynamicRegistrationInput{
				Name:         "Test App",
				RedirectURIs: []string{"https://example.com/cb"},
				GrantTypes:   []string{"authorization_code"},
			})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("manager.Create fail"))
		})
	})
})
