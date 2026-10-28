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
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/service"
	"bkauth/pkg/service/types"
	"bkauth/pkg/util"
)

// The name carries a uniqueness constraint, so the whitespace binding lets
// through is not cosmetic: " ci" would be a second spelling of a name the owner
// already holds, indistinguishable from the first in the list it identifies a
// token in.
func TestNormalizeNameOrAbort(t *testing.T) {
	t.Run("trims surrounding whitespace", func(t *testing.T) {
		c, w := testGinContext(t)
		name, ok := normalizeNameOrAbort(c, "  ci  ")

		assert.True(t, ok)
		assert.Equal(t, "ci", name)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("passes an already-clean name through untouched", func(t *testing.T) {
		c, _ := testGinContext(t)
		name, ok := normalizeNameOrAbort(c, "ci")

		assert.True(t, ok)
		assert.Equal(t, "ci", name)
	})

	// binding's `required` only refuses the empty string, so this reaches the
	// handler intact and has to be rejected here.
	t.Run("rejects a name of pure whitespace", func(t *testing.T) {
		for _, raw := range []string{" ", "\t", "\n", "   "} {
			c, w := testGinContext(t)
			_, ok := normalizeNameOrAbort(c, raw)

			assert.False(t, ok)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			errObj := decodeBody(t, w)["error"].(map[string]any)
			assert.Equal(t, "INVALID_ARGUMENT", errObj["code"])
			assert.Equal(t, "name must not be blank", errObj["message"])
		}
	})

	// Trimming can only shorten, so the max=64 tag still holds afterwards and an
	// inner space is not the serializer's business.
	t.Run("leaves inner whitespace alone", func(t *testing.T) {
		c, _ := testGinContext(t)
		name, ok := normalizeNameOrAbort(c, "  ci deploy  ")

		assert.True(t, ok)
		assert.Equal(t, "ci deploy", name)
	})
}

func TestHandlePersonalAccessTokenError(t *testing.T) {
	policy := types.PersonalTokenPolicy{MaxTTL: 3600, MaxActivePerUser: 5}

	t.Run("not found", func(t *testing.T) {
		c, w := testGinContext(t)
		handlePersonalAccessTokenError(c, service.ErrPersonalTokenNotFound, policy)

		assert.Equal(t, http.StatusNotFound, w.Code)
		errObj := decodeBody(t, w)["error"].(map[string]any)
		assert.Equal(t, "NOT_FOUND", errObj["code"])
		assert.Equal(t, "personal access token not found", errObj["message"])
		assert.NotContains(t, errObj, "details")
	})

	t.Run("invalid expires_at names the ceiling in the message", func(t *testing.T) {
		before := time.Now().UTC()
		c, w := testGinContext(t)
		handlePersonalAccessTokenError(c, service.ErrPersonalTokenInvalidExpiresAt, policy)
		after := time.Now().UTC()

		assert.Equal(t, http.StatusBadRequest, w.Code)
		errObj := decodeBody(t, w)["error"].(map[string]any)
		assert.Equal(t, "INVALID_ARGUMENT", errObj["code"])
		message, ok := errObj["message"].(string)
		require.True(t, ok)

		const prefix = "expires_at must be in Unix seconds, after now and no later than "
		require.True(t, strings.HasPrefix(message, prefix), "message %q", message)

		rest := strings.TrimPrefix(message, prefix)
		unixStr, rfc3339, ok := strings.Cut(rest, " (")
		require.True(t, ok, "message %q", message)
		rfc3339 = strings.TrimSuffix(rfc3339, ")")

		gotUnix, err := strconv.ParseInt(unixStr, 10, 64)
		require.NoError(t, err)
		minUnix := before.Add(time.Duration(policy.MaxTTL) * time.Second).Unix()
		maxUnix := after.Add(time.Duration(policy.MaxTTL) * time.Second).Unix()
		assert.GreaterOrEqual(t, gotUnix, minUnix)
		assert.LessOrEqual(t, gotUnix, maxUnix)

		parsed, err := time.Parse(time.RFC3339, rfc3339)
		require.NoError(t, err)
		assert.Equal(t, gotUnix, parsed.Unix())

		assert.NotContains(t, errObj, "data")
		assert.NotContains(t, errObj, "details")
	})

	// 409 rather than the quota's 429: this one the caller can fix by editing the
	// request. The message names the states the rule spans, because the token in
	// the way may be one the user considers finished with.
	t.Run("name conflict", func(t *testing.T) {
		c, w := testGinContext(t)
		handlePersonalAccessTokenError(c, service.ErrPersonalTokenNameConflict, policy)

		assert.Equal(t, http.StatusConflict, w.Code)
		errObj := decodeBody(t, w)["error"].(map[string]any)
		assert.Equal(t, "ALREADY_EXISTS", errObj["code"])
		assert.Equal(t,
			"a personal access token with this name already exists, expired and revoked ones included",
			errObj["message"])
		assert.NotContains(t, errObj, "data")
	})

	t.Run("quota exceeded names the ceiling in the message", func(t *testing.T) {
		c, w := testGinContext(t)
		handlePersonalAccessTokenError(c, service.ErrPersonalTokenQuotaExceeded, policy)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		errObj := decodeBody(t, w)["error"].(map[string]any)
		assert.Equal(t, "RESOURCE_EXHAUSTED", errObj["code"])
		assert.Equal(t,
			"too many active personal access tokens (at most 5), "+
				"revoke an existing token before creating another",
			errObj["message"])
		assert.NotContains(t, errObj, "data")
	})

	t.Run("unrecognised error becomes a fixed 500 and is stored for logging", func(t *testing.T) {
		c, w := testGinContext(t)
		unrecognised := errors.New("dao: get by id: connection refused")
		handlePersonalAccessTokenError(c, unrecognised, policy)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		errObj := decodeBody(t, w)["error"].(map[string]any)
		assert.Equal(t, "INTERNAL", errObj["code"])
		assert.Equal(t, "internal server error", errObj["message"])
		assert.NotContains(t, errObj, "data")

		stored, ok := util.GetError(c)
		require.True(t, ok)
		assert.Equal(t, unrecognised, stored)
	})
}
