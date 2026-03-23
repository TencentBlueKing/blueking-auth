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
	"net/http"

	"github.com/gin-gonic/gin"

	"bkauth/pkg/config"
	"bkauth/pkg/oauth"
	"bkauth/pkg/util"
)

// AuthorizationServerMetadata represents OAuth 2.0 Authorization Server Metadata (RFC 8414)
type AuthorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	DeviceAuthorizationEndpoint       string   `json:"device_authorization_endpoint,omitempty"`
	IntrospectionEndpoint             string   `json:"introspection_endpoint,omitempty"`
	RevocationEndpoint                string   `json:"revocation_endpoint,omitempty"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	ResponseModesSupported            []string `json:"response_modes_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// NewMetadataHandler creates a handler for authorization server metadata.
// Reads the realm from gin context (set by RealmMiddleware).
func NewMetadataHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		renderMetadata(c, cfg, util.GetRealmName(c))
	}
}

// NewDefaultRealmMetadataHandler creates a metadata handler that always uses the
// configured default realm. Used for backward-compatible well-known endpoint.
func NewDefaultRealmMetadataHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		renderMetadata(c, cfg, cfg.OAuth.DefaultRealmName)
	}
}

func renderMetadata(c *gin.Context, cfg *config.Config, realm string) {
	base := cfg.BKAuthURL

	metadata := AuthorizationServerMetadata{
		Issuer:                      oauth.IssuerURL(base, realm),
		AuthorizationEndpoint:       oauth.AuthorizationEndpointURL(base, realm),
		TokenEndpoint:               oauth.TokenEndpointURL(base, realm),
		DeviceAuthorizationEndpoint: oauth.DeviceAuthorizationEndpointURL(base, realm),
		IntrospectionEndpoint:       oauth.IntrospectionEndpointURL(base, realm),
		RevocationEndpoint:          oauth.RevocationEndpointURL(base, realm),
		ResponseTypesSupported:      []string{oauth.ResponseTypeCode},
		ResponseModesSupported:      []string{oauth.ResponseModeQuery},
		GrantTypesSupported: []string{
			oauth.GrantTypeAuthorizationCode,
			oauth.GrantTypeRefreshToken,
			// NOTE: device_code grant type is not exposed in well-known metadata for now
			// oauth.GrantTypeDeviceCode,
		},
		CodeChallengeMethodsSupported:     []string{oauth.CodeChallengeMethodS256},
		TokenEndpointAuthMethodsSupported: []string{
			oauth.AuthMethodNone, oauth.AuthMethodClientSecretBasic, oauth.AuthMethodClientSecretPost,
		},
	}

	if cfg.OAuth.DCREnabled {
		metadata.RegistrationEndpoint = oauth.RegistrationEndpointURL(base, realm)
	}

	c.JSON(http.StatusOK, metadata)
}
