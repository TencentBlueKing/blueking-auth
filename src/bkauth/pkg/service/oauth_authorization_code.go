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
	"time"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

const OAuthAuthorizationCodeSVC = "OAuthAuthorizationCodeSVC"

// OAuthAuthorizationCodeService defines the interface for authorization code lifecycle operations.
type OAuthAuthorizationCodeService interface {
	CreateAuthorizationCode(ctx context.Context, input types.CreateAuthorizationCodeInput) error
	ValidateAndConsume(
		ctx context.Context, realmName, code, clientID, redirectURI, codeVerifier string,
	) (types.ConsumedAuthorizationCode, error)
}

type oauthAuthorizationCodeService struct {
	authCodeManager dao.OAuthAuthorizationCodeManager
}

// NewOAuthAuthorizationCodeService creates a new OAuthAuthorizationCodeService.
func NewOAuthAuthorizationCodeService() OAuthAuthorizationCodeService {
	return &oauthAuthorizationCodeService{
		authCodeManager: dao.NewOAuthAuthorizationCodeManager(),
	}
}

func (s *oauthAuthorizationCodeService) CreateAuthorizationCode(
	ctx context.Context, input types.CreateAuthorizationCodeInput,
) error {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthAuthorizationCodeSVC, "CreateAuthorizationCode")

	audienceJSON, err := json.Marshal(input.Audience)
	if err != nil {
		return errorWrapf(err, "json.Marshal audience fail")
	}

	daoCode := dao.OAuthAuthorizationCode{
		Code:                input.Code,
		ClientID:            input.ClientID,
		RealmName:           input.RealmName,
		TenantID:            input.TenantID,
		Sub:                 input.Sub,
		Username:            input.Username,
		RedirectURI:         input.RedirectURI,
		Audience:            string(audienceJSON),
		CodeChallenge:       input.CodeChallenge,
		CodeChallengeMethod: input.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(time.Duration(oauth.AuthorizationCodeTTL) * time.Second),
		Used:                false,
	}

	if err := s.authCodeManager.Create(ctx, daoCode); err != nil {
		return errorWrapf(err, "authCodeManager.Create fail")
	}

	return nil
}

// ValidateAndConsume validates an authorization code (ownership, expiry, PKCE) and atomically
// marks it as used. Returns the decoded authorization code data on success.
func (s *oauthAuthorizationCodeService) ValidateAndConsume(
	ctx context.Context,
	realmName, code, clientID, redirectURI, codeVerifier string,
) (types.ConsumedAuthorizationCode, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthAuthorizationCodeSVC, "ValidateAndConsume")

	authCode, err := s.authCodeManager.Get(ctx, code)
	if err != nil {
		return types.ConsumedAuthorizationCode{}, errorWrapf(err, "authCodeManager.Get fail")
	}

	if authCode.Code == "" {
		return types.ConsumedAuthorizationCode{}, oauth.ErrInvalidAuthorizationCode
	}

	if authCode.Used {
		return types.ConsumedAuthorizationCode{}, oauth.ErrAuthorizationCodeUsed
	}

	if time.Now().After(authCode.ExpiresAt) {
		return types.ConsumedAuthorizationCode{}, oauth.ErrAuthorizationCodeExpired
	}

	if authCode.RealmName != realmName {
		return types.ConsumedAuthorizationCode{}, oauth.ErrRealmMismatch
	}

	if authCode.ClientID != clientID {
		return types.ConsumedAuthorizationCode{}, oauth.ErrClientMismatch
	}

	if authCode.RedirectURI != redirectURI {
		return types.ConsumedAuthorizationCode{}, oauth.ErrRedirectURIMismatch
	}

	if authCode.CodeChallenge != "" {
		method := oauth.CodeChallengeMethodS256
		if authCode.CodeChallengeMethod != "" {
			method = authCode.CodeChallengeMethod
		}
		if !oauth.VerifyPKCE(codeVerifier, authCode.CodeChallenge, method) {
			return types.ConsumedAuthorizationCode{}, oauth.ErrInvalidCodeVerifier
		}
	}

	var audience []string
	if err := json.Unmarshal([]byte(authCode.Audience), &audience); err != nil {
		return types.ConsumedAuthorizationCode{}, errorWrapf(err, "json.Unmarshal audience fail")
	}

	// Optimistic lock: only the first request to CAS (used=0 -> used=1) succeeds.
	// rowsAffected==0 means another concurrent request already consumed this code.
	//
	// Other theoretically possible causes (code deleted by TTL cleanup, or code
	// expired between Get and UPDATE) are not handled here because:
	//   - TTL cleanup runs at coarse intervals (minutes/hours), never races with in-flight requests.
	//   - Expiry was already validated above; the sub-millisecond window is negligible.
	rowsAffected, err := s.authCodeManager.MarkAsUsed(ctx, code)
	if err != nil {
		return types.ConsumedAuthorizationCode{}, errorWrapf(err, "authCodeManager.MarkAsUsed fail")
	}
	if rowsAffected == 0 {
		return types.ConsumedAuthorizationCode{}, oauth.ErrAuthorizationCodeUsed
	}

	return types.ConsumedAuthorizationCode{
		TenantID: authCode.TenantID,
		Sub:      authCode.Sub,
		Username: authCode.Username,
		Audience: audience,
	}, nil
}
