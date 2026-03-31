/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
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

type AccessTokenHashKey struct {
	TokenHash string
}

// Key returns the cache key for the access token hash lookup.
func (k AccessTokenHashKey) Key() string {
	return k.TokenHash
}

var retrieveAccessTokenByTokenHash = func(ctx context.Context, key cache.Key) (any, error) {
	k := key.(AccessTokenHashKey)

	svc := service.NewOAuthTokenService()
	return svc.GetAccessTokenByTokenHash(ctx, k.TokenHash)
}

// GetAccessTokenByTokenHash resolves an access token by token hash using the cache layer.
func GetAccessTokenByTokenHash(ctx context.Context, tokenHash string) (token types.ResolvedAccessToken, err error) {
	key := AccessTokenHashKey{
		TokenHash: tokenHash,
	}

	err = AccessTokenCache.GetInto(ctx, key, &token, retrieveAccessTokenByTokenHash)
	if err != nil {
		err = errorx.Wrapf(err, CacheLayer, "GetAccessTokenByTokenHash",
			"AccessTokenCache.GetInto tokenHash=`%s` fail", tokenHash)
		return token, err
	}

	return token, nil
}

// DeleteAccessTokenCache removes the cached access token entry for the given token hash.
func DeleteAccessTokenCache(ctx context.Context, tokenHash string) error {
	key := AccessTokenHashKey{
		TokenHash: tokenHash,
	}
	return AccessTokenCache.Delete(ctx, key)
}
