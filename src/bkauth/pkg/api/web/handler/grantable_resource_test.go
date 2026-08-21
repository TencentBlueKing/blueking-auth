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
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"bkauth/pkg/oauth"
)

// The status codes are the lookup endpoint's contract with the form: each one
// sends the user somewhere different, so collapsing any two of them would be a
// change in behaviour rather than in wording.
//
// Every sentinel is wrapped here, the way the realms return it, because the
// mapping matches with errors.Is and would silently stop working if it were
// ever rewritten as an equality check.
func TestHandleGrantableResourceRefError(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		// The caller's word is not echoed back: an unknown type is a client bug,
		// and the answer that helps is where the vocabulary comes from.
		{
			name:    "unknown type names the endpoint that lists them",
			err:     fmt.Errorf("%w: %q", oauth.ErrUnknownGrantableResourceType, "vm"),
			status:  http.StatusBadRequest,
			code:    "INVALID_ARGUMENT",
			message: "type must be one of those returned by grantable-resource-types",
		},
		{
			name:    "unknown level",
			err:     fmt.Errorf("%w: %q", oauth.ErrUnknownGrantableResourceLevel, "cluster"),
			status:  http.StatusBadRequest,
			code:    "INVALID_ARGUMENT",
			message: "each name must be keyed by one of this type's levels",
		},
		// Verbatim from here on: the message is built from the names just
		// submitted, and naming the level still missing is the whole point --
		// "incomplete" on its own tells a half-filled form nothing.
		{
			name:    "incomplete ref passes the missing level through",
			err:     fmt.Errorf("%w: %q must be named", oauth.ErrIncompleteGrantableResourceRef, "mcp"),
			status:  http.StatusBadRequest,
			code:    "INVALID_ARGUMENT",
			message: `incomplete grantable resource ref: "mcp" must be named`,
		},
		{
			name: "not found passes the names through",
			err: fmt.Errorf("%w: gateway %q has no MCP server named %q",
				oauth.ErrGrantableResourceNotFound, "bk-log", "bk-log-prod-query"),
			status: http.StatusNotFound,
			code:   "NOT_FOUND",
			message: `grantable resource not found: gateway "bk-log" ` +
				`has no MCP server named "bk-log-prod-query"`,
		},
		// 403 rather than 404: the object is there and the name is right, so
		// "no such thing" would be a lie the user cannot act on.
		{
			name: "not grantable is refused rather than reported missing",
			err: fmt.Errorf("%w: MCP server %q is not open to personal access tokens",
				oauth.ErrGrantableResourceNotGrantable, "bk-log-prod-query"),
			status: http.StatusForbidden,
			code:   "NO_PERMISSION",
			message: `grantable resource not grantable: MCP server "bk-log-prod-query" ` +
				`is not open to personal access tokens`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := testGinContext(t)
			handleGrantableResourceRefError(c, tc.err)

			assert.Equal(t, tc.status, w.Code)
			errObj := decodeBody(t, w)["error"].(map[string]any)
			assert.Equal(t, tc.code, errObj["code"])
			assert.Equal(t, tc.message, errObj["message"])
		})
	}

	// An upstream failure carries the query and the credentials-bearing path it
	// was built from, so the message is replaced rather than passed through.
	t.Run("an upstream failure is answered without its own message", func(t *testing.T) {
		c, w := testGinContext(t)
		handleGrantableResourceRefError(c, errors.New(
			"bkapigateway: request failed, status=400 code=INVALID_ARGUMENT message=unsupported field"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		errObj := decodeBody(t, w)["error"].(map[string]any)
		assert.Equal(t, "INTERNAL", errObj["code"])
		assert.Equal(t, "failed to look up the grantable resource", errObj["message"])
		assert.NotContains(t, errObj["message"], "bkapigateway")
	})
}
