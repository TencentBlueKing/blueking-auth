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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bkauth/pkg/oauth"
)

var _ = Describe("ClientRegistrationRequest.Validate", func() {
	It("should reject blank client_name (whitespace only)", func() {
		req := ClientRegistrationRequest{
			ClientName:   "   ",
			RedirectURIs: []string{"https://example.com/cb"},
		}

		err := req.Validate()

		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRequest))
	})

	It("should default grant_types to authorization_code + refresh_token", func() {
		req := ClientRegistrationRequest{
			ClientName:   "My App",
			RedirectURIs: []string{"https://example.com/cb"},
		}

		err := req.Validate()

		Expect(err).NotTo(HaveOccurred())
		Expect(req.GrantTypes).To(Equal([]string{
			oauth.GrantTypeAuthorizationCode,
			oauth.GrantTypeRefreshToken,
		}))
	})

	It("should reject unsupported grant_types", func() {
		req := ClientRegistrationRequest{
			ClientName:   "My App",
			RedirectURIs: []string{"https://example.com/cb"},
			GrantTypes:   []string{"implicit"},
		}

		err := req.Validate()

		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidClientMetadata))
	})

	It("should deduplicate grant_types", func() {
		req := ClientRegistrationRequest{
			ClientName:   "My App",
			RedirectURIs: []string{"https://example.com/cb"},
			GrantTypes:   []string{oauth.GrantTypeAuthorizationCode, oauth.GrantTypeAuthorizationCode},
		}

		err := req.Validate()

		Expect(err).NotTo(HaveOccurred())
		Expect(req.GrantTypes).To(Equal([]string{oauth.GrantTypeAuthorizationCode}))
	})

	It("should reject invalid logo_uri", func() {
		req := ClientRegistrationRequest{
			ClientName:   "My App",
			RedirectURIs: []string{"https://example.com/cb"},
			LogoURI:      "ftp://invalid.com/logo.png",
		}

		err := req.Validate()

		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidClientMetadata))
	})

	It("should reject invalid redirect_uris", func() {
		req := ClientRegistrationRequest{
			ClientName:   "My App",
			RedirectURIs: []string{"not-a-url"},
		}

		err := req.Validate()

		Expect(err).To(HaveOccurred())
		oauthErr, ok := oauth.AsOAuthError(err)
		Expect(ok).To(BeTrue())
		Expect(oauthErr.Code).To(Equal(oauth.ErrorCodeInvalidRedirectURI))
	})

	It("should deduplicate redirect_uris", func() {
		req := ClientRegistrationRequest{
			ClientName:   "My App",
			RedirectURIs: []string{"https://example.com/cb", "https://example.com/cb"},
		}

		err := req.Validate()

		Expect(err).NotTo(HaveOccurred())
		Expect(req.RedirectURIs).To(Equal([]string{"https://example.com/cb"}))
	})

	It("should pass with valid input", func() {
		req := ClientRegistrationRequest{
			ClientName:   "My App",
			RedirectURIs: []string{"https://example.com/cb"},
			GrantTypes:   []string{oauth.GrantTypeAuthorizationCode},
			LogoURI:      "https://example.com/logo.png",
		}

		err := req.Validate()

		Expect(err).NotTo(HaveOccurred())
	})
})
