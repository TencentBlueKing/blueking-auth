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

package util_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/util"
)

func webTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func decodeJSONBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestWebErrorEnvelope(t *testing.T) {
	c, w := webTestContext(t)

	util.WebInvalidArgumentError(c, "name is required")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := decodeJSONBody(t, w)
	assert.NotContains(t, body, "data")
	assert.NotContains(t, body, "system_name")

	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "INVALID_ARGUMENT", errObj["code"])
	assert.Equal(t, "name is required", errObj["message"])
	assert.Equal(t, "bkauth", errObj["system"])
	assert.NotContains(t, errObj, "details")
	assert.NotContains(t, errObj, "data")
	assert.NotContains(t, errObj, "system_name")
}

// The status/code pairing is the wire contract, so the expectations are spelled
// out as literals rather than read back from the constants under test.
func TestWebErrorShortcuts(t *testing.T) {
	cases := []struct {
		name   string
		write  func(c *gin.Context)
		status int
		code   string
	}{
		{
			name:   "invalid argument",
			write:  func(c *gin.Context) { util.WebInvalidArgumentError(c, "boom") },
			status: http.StatusBadRequest,
			code:   "INVALID_ARGUMENT",
		},
		{
			name:   "no permission",
			write:  func(c *gin.Context) { util.WebNoPermissionError(c, "boom") },
			status: http.StatusForbidden,
			code:   "NO_PERMISSION",
		},
		{
			name:   "not found",
			write:  func(c *gin.Context) { util.WebNotFoundError(c, "boom") },
			status: http.StatusNotFound,
			code:   "NOT_FOUND",
		},
		{
			name:   "already exists",
			write:  func(c *gin.Context) { util.WebAlreadyExistsError(c, "boom") },
			status: http.StatusConflict,
			code:   "ALREADY_EXISTS",
		},
		{
			name:   "resource exhausted",
			write:  func(c *gin.Context) { util.WebResourceExhaustedError(c, "boom") },
			status: http.StatusTooManyRequests,
			code:   "RESOURCE_EXHAUSTED",
		},
		{
			name:   "internal",
			write:  func(c *gin.Context) { util.WebInternalError(c, "boom") },
			status: http.StatusInternalServerError,
			code:   "INTERNAL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := webTestContext(t)
			tc.write(c)

			assert.Equal(t, tc.status, w.Code)
			errObj := decodeJSONBody(t, w)["error"].(map[string]any)
			assert.Equal(t, tc.code, errObj["code"])
			assert.Equal(t, "boom", errObj["message"])
			assert.NotContains(t, errObj, "data")
		})
	}
}

// The 401 is the one error that carries a data payload, because the frontend
// cannot act on the rejection without the login URL.
func TestWebUnauthenticatedErrorWithData(t *testing.T) {
	c, w := webTestContext(t)

	util.WebUnauthenticatedErrorWithData(c, "login required",
		gin.H{"login_url": "https://login.example.com"})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	assert.Equal(t, "UNAUTHENTICATED", errObj["code"])
	data, ok := errObj["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://login.example.com", data["login_url"])
}

func TestWebSuccess(t *testing.T) {
	c, w := webTestContext(t)

	util.WebSuccess(c, gin.H{"ok": true})

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeJSONBody(t, w)
	assert.NotContains(t, body, "error")
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, data["ok"])
}

// Writing the envelope must not stop the handler chain: aborting is the
// caller's decision, and middleware pairs it with c.Abort itself.
func TestWebErrorDoesNotAbort(t *testing.T) {
	c, _ := webTestContext(t)

	util.WebNoPermissionError(c, "cross-origin request rejected")

	assert.False(t, c.IsAborted())
}
