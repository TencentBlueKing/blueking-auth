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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/service/types"
)

// bindJSON runs a body through gin's binding exactly as the handlers do.
func bindJSON(t *testing.T, body string, out any) error {
	t.Helper()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	return c.ShouldBindJSON(out)
}

// expires_at is optional on update and mandatory-by-policy on create, and the
// difference is not visible in the binding tags -- neither field carries a rule,
// so both bind a missing value to 0 and the meaning of that 0 is decided further
// down. These pin which endpoint reads it as "unchanged".
func TestPersonalAccessTokenExpiresAtBinding(t *testing.T) {
	t.Run("create binds an absolute timestamp", func(t *testing.T) {
		var req createPersonalAccessTokenRequest
		err := bindJSON(t, `{"name":"ci","audience":["mcp:demo"],"expires_at":1786521750}`, &req)

		require.NoError(t, err)
		assert.Equal(t, int64(1786521750), req.ExpiresAt)
	})

	// Binding accepts it; the service rejects the 0 as out of window. The rule
	// lives in one place rather than being half-stated by a tag.
	t.Run("create accepts a missing expires_at at the binding layer", func(t *testing.T) {
		var req createPersonalAccessTokenRequest
		err := bindJSON(t, `{"name":"ci","audience":["mcp:demo"]}`, &req)

		require.NoError(t, err)
		assert.Zero(t, req.ExpiresAt)
	})

	t.Run("update reads a missing expires_at as leave-unchanged", func(t *testing.T) {
		var req updatePersonalAccessTokenRequest
		err := bindJSON(t, `{"name":"ci","description":"","audience":["mcp:demo"]}`, &req)

		require.NoError(t, err)
		assert.Zero(t, req.ExpiresAt)
	})

	t.Run("update carries an expiry when one is supplied", func(t *testing.T) {
		var req updatePersonalAccessTokenRequest
		err := bindJSON(t, `{"name":"ci","audience":["mcp:demo"],"expires_at":1786521750}`, &req)

		require.NoError(t, err)
		assert.Equal(t, int64(1786521750), req.ExpiresAt)
	})

	// The profile fields keep replace semantics: omitting audience is an error,
	// not an instruction to clear it.
	t.Run("update still requires the profile fields", func(t *testing.T) {
		var req updatePersonalAccessTokenRequest
		err := bindJSON(t, `{"name":"ci","expires_at":1786521750}`, &req)

		assert.Error(t, err)
	})

	t.Run("renew takes the timestamp alone", func(t *testing.T) {
		var req renewPersonalAccessTokenRequest
		err := bindJSON(t, `{"expires_at":1786521750}`, &req)

		require.NoError(t, err)
		assert.Equal(t, int64(1786521750), req.ExpiresAt)
	})
}

func TestNormalizeAudience(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "trims surrounding whitespace",
			input: []string{"  mcp:demo  ", "\tgateway:bk-cmdb/api:v2\n"},
			want:  []string{"mcp:demo", "gateway:bk-cmdb/api:v2"},
		},
		{
			name:  "drops blank entries",
			input: []string{"mcp:demo", "   ", ""},
			want:  []string{"mcp:demo"},
		},
		{
			name:  "de-duplicates only after trimming",
			input: []string{"mcp:demo", " mcp:demo", "mcp:demo "},
			want:  []string{"mcp:demo"},
		},
		{
			name:  "preserves the caller's order",
			input: []string{"c", "a", "b"},
			want:  []string{"c", "a", "b"},
		},
		{
			// Binding allows this: min=1 counts entries, not their content.
			name:  "an all-blank list normalizes to empty, which the handler rejects",
			input: []string{" ", "\t", ""},
			want:  []string{},
		},
		{
			name:  "empty input yields an empty slice, never nil",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "nil input yields an empty slice, never nil",
			input: nil,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAudience(tt.input)

			assert.Equal(t, tt.want, got)
			// Guards the JSON shape: a nil slice would marshal to null.
			assert.NotNil(t, got)
		})
	}
}

func TestToPersonalAccessTokenResponse(t *testing.T) {
	t.Run("maps every field the wire format carries", func(t *testing.T) {
		got := toPersonalAccessTokenResponse(types.PersonalAccessToken{
			ID:          42,
			Name:        "ci",
			Description: "for CI",
			TokenMask:   "bkp_ab****yz",
			Audience:    []string{"mcp:demo"},
			ExpiresAt:   1786521750,
			Revoked:     true,
			RevokedAt:   1786521800,
			CreatedAt:   1786521700,
			UpdatedAt:   1786521800,
		})

		assert.Equal(t, personalAccessTokenResponse{
			ID:          42,
			Name:        "ci",
			Description: "for CI",
			TokenMask:   "bkp_ab****yz",
			Audience:    []string{"mcp:demo"},
			ExpiresAt:   1786521750,
			Revoked:     true,
			RevokedAt:   1786521800,
			CreatedAt:   1786521700,
			UpdatedAt:   1786521800,
		}, got)
	})

	t.Run("a nil audience marshals to [] rather than null", func(t *testing.T) {
		got := toPersonalAccessTokenResponse(types.PersonalAccessToken{})
		assert.NotNil(t, got.Audience)

		body, err := json.Marshal(got)
		assert.NoError(t, err)
		assert.Contains(t, string(body), `"audience":[]`)
	})

	t.Run("revoked_at is absent while the token is live", func(t *testing.T) {
		body, err := json.Marshal(toPersonalAccessTokenResponse(types.PersonalAccessToken{
			Revoked:   false,
			RevokedAt: 0,
		}))

		assert.NoError(t, err)
		assert.NotContains(t, string(body), "revoked_at")
	})

	t.Run("revoked_at appears once the token is revoked", func(t *testing.T) {
		body, err := json.Marshal(toPersonalAccessTokenResponse(types.PersonalAccessToken{
			Revoked:   true,
			RevokedAt: 1786521800,
		}))

		assert.NoError(t, err)
		assert.Contains(t, string(body), `"revoked_at":1786521800`)
	})

	t.Run("resources is left for the handler to fill and marshals to null until then", func(t *testing.T) {
		got := toPersonalAccessTokenResponse(types.PersonalAccessToken{Audience: []string{"mcp:demo"}})
		assert.Nil(t, got.Resources)

		// Null, not absent: a client can then tell "the realm could not render
		// this" from a tree that happens to be empty, and fall back to the raw
		// audience.
		body, err := json.Marshal(got)
		assert.NoError(t, err)
		assert.Contains(t, string(body), `"resources":null`)
	})

	t.Run("realm_name and scope never reach the client", func(t *testing.T) {
		body, err := json.Marshal(toPersonalAccessTokenResponse(types.PersonalAccessToken{
			RealmName: "blueking",
			Scope:     "read:all",
		}))

		assert.NoError(t, err)
		assert.NotContains(t, string(body), "realm_name")
		assert.NotContains(t, string(body), "blueking")
		assert.NotContains(t, string(body), "scope")
		assert.NotContains(t, string(body), "read:all")
	})
}
