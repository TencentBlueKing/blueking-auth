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
	"bkauth/pkg/oauth"
	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
)

// ExternalAccessTokenKey keys the exat cache by realm and token hash. The realm
// is part of the key because the adapter answers per realm.
//
// raw is only carried for a cache miss to forward to the adapter; it stays out
// of Key(), so the raw token never reaches Redis, logs or errors.
type ExternalAccessTokenKey struct {
	RealmName string
	TokenHash string

	raw string
}

func (k ExternalAccessTokenKey) Key() string {
	return k.RealmName + ":" + k.TokenHash
}

var retrieveExternalAccessToken = func(ctx context.Context, key cache.Key) (interface{}, error) {
	k := key.(ExternalAccessTokenKey)

	svc := service.NewExternalAccessTokenService()
	return svc.Resolve(ctx, k.RealmName, k.raw)
}

// GetExternalAccessToken resolves a third-party token for introspection in
// realmName.
//
// There is no active invalidation: bkauth never learns of an upstream
// revocation, so the cache TTL is exactly how long one can go unnoticed.
func GetExternalAccessToken(ctx context.Context, realmName, raw string) (token types.ResolvedAccessToken, err error) {
	key := ExternalAccessTokenKey{
		RealmName: realmName,
		TokenHash: oauth.HashToken(raw),
		raw:       raw,
	}

	err = ExternalAccessTokenCache.GetInto(ctx, key, &token, retrieveExternalAccessToken)
	if err != nil {
		err = errorx.Wrapf(err, CacheLayer, "GetExternalAccessToken",
			"ExternalAccessTokenCache.GetInto key=`%s` fail", key.Key())
		return token, err
	}
	return token, nil
}
