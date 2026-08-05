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

package impls

import (
	"context"

	"bkauth/pkg/cache"
	"bkauth/pkg/errorx"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

// PersonalAccessTokenHashKey keys the pat cache by the token hash.
type PersonalAccessTokenHashKey struct {
	TokenHash string
}

func (k PersonalAccessTokenHashKey) Key() string {
	return k.TokenHash
}

// retrievePersonalAccessTokenByTokenHash is a package-level var so tests can
// inject a mock service without gomonkey (AGENTS.md 4.3, Cache/impls layer).
var retrievePersonalAccessTokenByTokenHash = func(ctx context.Context, key cache.Key) (interface{}, error) {
	k := key.(PersonalAccessTokenHashKey)

	svc := service.NewPersonalAccessTokenService()
	return svc.GetByTokenHash(ctx, k.TokenHash)
}

// GetPersonalAccessToken resolves a personal access token for introspection.
//
// The pat cache uses a 60s TTL — far shorter than oct's 5 minutes — because a PAT
// is mutable: active invalidation on every write is the first line of defence, and
// the short TTL is the necessary backstop for the two accepted races (a failed
// Redis delete, and a delete-after-write that a concurrent introspect refills).
// Both leave a stale value alive for at most the 60s TTL (design 5.4 / 13.14).
func GetPersonalAccessToken(
	ctx context.Context, tokenHash string,
) (token types.ResolvedAccessToken, err error) {
	key := PersonalAccessTokenHashKey{
		TokenHash: tokenHash,
	}

	err = PersonalAccessTokenCache.GetInto(ctx, key, &token, retrievePersonalAccessTokenByTokenHash)
	if err != nil {
		err = errorx.Wrapf(err, CacheLayer, "GetPersonalAccessToken",
			"PersonalAccessTokenCache.GetInto tokenHash=`%s` fail", tokenHash)
		return token, err
	}
	return token, nil
}

// DeletePersonalAccessTokenCache invalidates the cached entry after a write
// (revoke / renew / edit). Called by the handler, per the Handler -> cache/impls
// convention.
func DeletePersonalAccessTokenCache(ctx context.Context, tokenHash string) error {
	key := PersonalAccessTokenHashKey{
		TokenHash: tokenHash,
	}
	return PersonalAccessTokenCache.Delete(ctx, key)
}
