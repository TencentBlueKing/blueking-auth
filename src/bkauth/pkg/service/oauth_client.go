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

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"context"
	"encoding/json"
	"strings"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

const OAuthClientSVC = "OAuthClientSVC"

// OAuthClientService defines the interface for OAuth client operations
type OAuthClientService interface {
	DynamicRegister(ctx context.Context, input types.OAuthClientDynamicRegistrationInput) (types.OAuthClient, error)
	Get(ctx context.Context, clientID string) (types.OAuthClient, error)
	Exists(ctx context.Context, clientID string) (bool, error)
	GetFlowSpec(ctx context.Context, clientID string) (types.OAuthClientFlowSpec, error)
	GetProfile(ctx context.Context, clientID string) (types.OAuthClientProfile, error)
}

type oauthClientService struct {
	manager dao.OAuthClientManager
}

// NewOAuthClientService creates a new OAuthClientService
func NewOAuthClientService() OAuthClientService {
	return &oauthClientService{
		manager: dao.NewOAuthClientManager(),
	}
}

// DynamicRegister registers a new OAuth client via Dynamic Client Registration (RFC 7591).
func (s *oauthClientService) DynamicRegister(
	ctx context.Context,
	input types.OAuthClientDynamicRegistrationInput,
) (types.OAuthClient, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthClientSVC, "DynamicRegister")

	clientID, err := oauth.GenerateDynamicClientID()
	if err != nil {
		return types.OAuthClient{}, errorWrapf(err, "GenerateDynamicClientID fail")
	}

	redirectURIsJSON, err := json.Marshal(input.RedirectURIs)
	if err != nil {
		return types.OAuthClient{}, errorWrapf(err, "json.Marshal redirectURIs fail")
	}

	daoClient := dao.OAuthClient{
		ID:           clientID,
		Name:         input.Name,
		Type:         oauth.ClientTypePublic,
		RedirectURIs: string(redirectURIsJSON),
		GrantTypes:   strings.Join(input.GrantTypes, ","),
		LogoURI:      input.LogoURI,
	}

	if err := s.manager.Create(ctx, daoClient); err != nil {
		return types.OAuthClient{}, errorWrapf(err, "manager.Create fail")
	}

	return s.Get(ctx, clientID)
}

// Get retrieves an OAuth client by client ID.
// Returns a zero-value OAuthClient (ID == "") with nil error when the client does not exist.
func (s *oauthClientService) Get(ctx context.Context, clientID string) (types.OAuthClient, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthClientSVC, "Get")

	daoClient, err := s.manager.Get(ctx, clientID)
	if err != nil {
		return types.OAuthClient{}, errorWrapf(err, "manager.Get clientID=`%s` fail", clientID)
	}

	if daoClient.ID == "" {
		return types.OAuthClient{}, nil
	}

	return s.convertToTypes(daoClient)
}

// Exists reports whether a client with the given ID is registered.
func (s *oauthClientService) Exists(ctx context.Context, clientID string) (bool, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthClientSVC, "Exists")

	exists, err := s.manager.Exists(ctx, clientID)
	if err != nil {
		return false, errorWrapf(err, "manager.Exists clientID=`%s` fail", clientID)
	}
	return exists, nil
}

// GetFlowSpec retrieves the OAuth protocol parameters for authorization flow validation.
// Returns a zero-value OAuthClientFlowSpec (ID == "") with nil error when the client does not exist.
func (s *oauthClientService) GetFlowSpec(ctx context.Context, clientID string) (types.OAuthClientFlowSpec, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthClientSVC, "GetFlowSpec")

	daoGrants, err := s.manager.GetGrants(ctx, clientID)
	if err != nil {
		return types.OAuthClientFlowSpec{}, errorWrapf(err, "manager.GetGrants clientID=`%s` fail", clientID)
	}

	if daoGrants.ID == "" {
		return types.OAuthClientFlowSpec{}, nil
	}

	var redirectURIs []string
	if err := json.Unmarshal([]byte(daoGrants.RedirectURIs), &redirectURIs); err != nil {
		return types.OAuthClientFlowSpec{}, errorWrapf(err, "json.Unmarshal redirectURIs fail")
	}

	return types.OAuthClientFlowSpec{
		ID:           daoGrants.ID,
		GrantTypes:   strings.Split(daoGrants.GrantTypes, ","),
		RedirectURIs: redirectURIs,
	}, nil
}

// GetProfile retrieves the display-oriented fields for consent / device pages.
// Returns a zero-value OAuthClientProfile (ID == "") with nil error when the client does not exist.
func (s *oauthClientService) GetProfile(ctx context.Context, clientID string) (types.OAuthClientProfile, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthClientSVC, "GetProfile")

	daoDisplay, err := s.manager.GetDisplay(ctx, clientID)
	if err != nil {
		return types.OAuthClientProfile{}, errorWrapf(err, "manager.GetDisplay clientID=`%s` fail", clientID)
	}

	if daoDisplay.ID == "" {
		return types.OAuthClientProfile{}, nil
	}

	return types.OAuthClientProfile{
		ID:      daoDisplay.ID,
		Name:    daoDisplay.Name,
		LogoURI: daoDisplay.LogoURI,
	}, nil
}

// convertToTypes converts a DAO client to a service types client
func (s *oauthClientService) convertToTypes(daoClient dao.OAuthClient) (types.OAuthClient, error) {
	var redirectURIs []string
	if err := json.Unmarshal([]byte(daoClient.RedirectURIs), &redirectURIs); err != nil {
		return types.OAuthClient{}, err
	}

	return types.OAuthClient{
		ID:           daoClient.ID,
		Name:         daoClient.Name,
		Type:         daoClient.Type,
		RedirectURIs: redirectURIs,
		GrantTypes:   strings.Split(daoClient.GrantTypes, ","),
		LogoURI:      daoClient.LogoURI,
		CreatedAt:    daoClient.CreatedAt.Unix(),
	}, nil
}
