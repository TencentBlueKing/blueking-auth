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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/oauth"
)

func testGinContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestHandleUserCodeError(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{
			name:    "expired",
			err:     oauth.ErrUserCodeExpired,
			status:  http.StatusBadRequest,
			code:    "INVALID_ARGUMENT",
			message: "code has expired, please request a new one on your device",
		},
		{
			name:    "already used",
			err:     oauth.ErrUserCodeAlreadyUsed,
			status:  http.StatusConflict,
			code:    "ALREADY_EXISTS",
			message: "device code has already been used",
		},
		{
			name:    "not found",
			err:     errors.New("no such code"),
			status:  http.StatusBadRequest,
			code:    "INVALID_ARGUMENT",
			message: "code not found, please check and re-enter",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := testGinContext(t)
			handleUserCodeError(c, tc.err)

			assert.Equal(t, tc.status, w.Code)
			errObj := decodeBody(t, w)["error"].(map[string]any)
			assert.Equal(t, tc.code, errObj["code"])
			assert.Equal(t, tc.message, errObj["message"])
			assert.NotContains(t, errObj, "details")
		})
	}
}
