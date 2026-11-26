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
	"errors"
	"fmt"

	"go.uber.org/zap"

	"bkauth/pkg/errorx"
	"bkauth/pkg/external/tokenadapter"
	"bkauth/pkg/logging"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

const ExternalAccessTokenSVC = "ExternalAccessTokenSVC"

// ErrExternalTokenContractViolation means the adapter called a token active but
// answered outside the protocol. The adapter is bkauth's own component, so this
// is its bug rather than a verdict on the token: it surfaces as a server error
// and is never cached, instead of passing for an inactive token.
var ErrExternalTokenContractViolation = errors.New("external token adapter violated the protocol")

// ExternalAccessTokenService resolves third-party tokens through the adapter.
type ExternalAccessTokenService interface {
	// Resolve returns the token as seen from realmName. An inactive answer
	// carries nothing meaningful, so it yields a zero-value ResolvedAccessToken
	// (ClientID == ""), like an unknown native token, cacheable as a negative
	// entry. An expired token is returned in full and left to IsActive. An error
	// means no verdict was reached and must not be cached.
	Resolve(ctx context.Context, realmName, raw string) (types.ResolvedAccessToken, error)
}

type externalAccessTokenService struct {
	introspector tokenadapter.Introspector
}

// NewExternalAccessTokenService creates a service backed by the adapter.
func NewExternalAccessTokenService() ExternalAccessTokenService {
	return &externalAccessTokenService{
		introspector: tokenadapter.NewIntrospector(),
	}
}

func (s *externalAccessTokenService) Resolve(
	ctx context.Context, realmName, raw string,
) (token types.ResolvedAccessToken, err error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(ExternalAccessTokenSVC, "Resolve")

	realm := oauth.GetRealm(realmName)
	if realm == nil {
		return token, errorWrapf(fmt.Errorf("realm %q is not registered", realmName), "oauth.GetRealm fail")
	}

	resp, err := s.introspector.Introspect(ctx, realmName, raw)
	if err != nil {
		return token, errorWrapf(err, "introspector.Introspect realm=`%s` fail", realmName)
	}
	if !resp.Active {
		return token, nil
	}

	if err := checkExternalTokenContract(ctx, realm, resp); err != nil {
		logging.GetSystemLogger().Error("external token adapter violated the protocol",
			zap.Error(err),
			zap.String("realm_name", realmName),
			zap.String("client_id", resp.ClientID),
			zap.String("sub", resp.Sub),
		)
		return token, errorWrapf(err, "checkContract realm=`%s` fail", realmName)
	}

	// Native tokens decode their audience from a JSON column and so never carry
	// a nil one; callers can rely on the same shape here.
	audience := resp.Audience
	if audience == nil {
		audience = []string{}
	}

	// Expiry is not judged here: as with native tokens, the caller's IsActive
	// does, so a token that expired between the adapter's clock and ours is
	// reported inactive with its details intact rather than as unknown.
	return types.ResolvedAccessToken{
		ClientID:  resp.ClientID,
		TenantID:  resp.TenantID,
		RealmName: realmName,
		Sub:       resp.Sub,
		Username:  resp.Username,
		Audience:  audience,
		ExpiresAt: resp.ExpiresAt,
	}, nil
}

// checkExternalTokenContract holds an active answer to the protocol. It checks
// shape, not meaning: which user, tenant and audiences a token maps to is the
// adapter's call. What is checked is what downstream would otherwise trip over
// silently: identity fields it reads, an expiry it caches by, and audiences it
// compares byte for byte, so a malformed one would match nothing rather than
// fail.
func checkExternalTokenContract(ctx context.Context, realm oauth.Realm, resp tokenadapter.Response) error {
	violation := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrExternalTokenContractViolation, fmt.Sprintf(format, args...))
	}

	switch {
	case resp.ClientID == "":
		return violation("client_id is empty")
	case resp.Sub == "":
		return violation("sub is empty")
	case resp.Username == "":
		return violation("username is empty")
	case resp.TenantID == "":
		return violation("tenant_id is empty")
	case resp.ExpiresAt == 0:
		return violation("exp is missing")
	}

	// An empty aud is legitimate: third-party audiences only become BlueKing
	// ones through configured mappings, and a token none of whose audiences map
	// is valid yet grants nothing in this realm. The realm grammar rejects an
	// empty list, so it only judges a non-empty one.
	if len(resp.Audience) == 0 {
		return nil
	}
	if err := realm.ValidateAudiences(ctx, resp.Audience); err != nil {
		return violation("aud is invalid in realm %q: %s", realm.Name(), err)
	}
	return nil
}
