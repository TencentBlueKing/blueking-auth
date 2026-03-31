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
	"time"

	"bkauth/pkg/errorx"
	"bkauth/pkg/util"
)

// consentTTL must be shorter than the Cache-level TTL, otherwise the per-key
// TTL set via Set() will be silently capped to the Cache TTL, making this value ineffective.
const consentTTL = 600

// Consent holds the OAuth authorization request parameters stored in Redis.
// All fields are pre-validated by the /authorize endpoint before storage;
// consumers can trust the data directly.
type Consent struct {
	RealmName           string `msgpack:"realm_name"`
	ClientID            string `msgpack:"client_id"`
	RedirectURI         string `msgpack:"redirect_uri"`
	State               string `msgpack:"state,omitempty"`
	CodeChallenge       string `msgpack:"code_challenge"`
	CodeChallengeMethod string `msgpack:"code_challenge_method,omitempty"`
	Resource            string `msgpack:"resource"`
}

type consentKey struct {
	challenge string
}

// Key returns the Redis cache key for the consent challenge.
func (k consentKey) Key() string {
	return k.challenge
}

// CreateConsent stores a consent record in Redis and returns the consent challenge.
func CreateConsent(ctx context.Context, consent Consent) (string, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(CacheLayer, "CreateConsent")

	challenge, err := util.RandHex(8)
	if err != nil {
		return "", errorWrapf(err, "generate consent challenge fail")
	}

	key := consentKey{challenge: challenge}
	ttl := time.Duration(consentTTL) * time.Second
	if err := ConsentCache.Set(ctx, key, consent, ttl); err != nil {
		return "", errorWrapf(err, "ConsentCache.Set fail")
	}

	return challenge, nil
}

// GetConsent retrieves a consent record by consent challenge.
func GetConsent(ctx context.Context, challenge string) (Consent, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(CacheLayer, "GetConsent")

	key := consentKey{challenge: challenge}
	var consent Consent
	if err := ConsentCache.Get(ctx, key, &consent); err != nil {
		return Consent{}, errorWrapf(err, "ConsentCache.Get challenge=`%s` fail", challenge)
	}

	return consent, nil
}

// DeleteConsent deletes the consent record from Redis.
func DeleteConsent(ctx context.Context, challenge string) error {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(CacheLayer, "DeleteConsent")

	key := consentKey{challenge: challenge}
	if err := ConsentCache.Delete(ctx, key); err != nil {
		return errorWrapf(err, "ConsentCache.Delete challenge=`%s` fail", challenge)
	}

	return nil
}
