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

package middleware

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormBodyToJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantOK      bool
		wantFields  map[string]string
	}{
		{
			name:        "typical OAuth token request",
			body:        "grant_type=authorization_code&client_id=my_app&client_secret=super_secret&code=abc123&redirect_uri=https%3A%2F%2Fexample.com%2Fcallback",
			contentType: "application/x-www-form-urlencoded",
			wantOK:      true,
			wantFields: map[string]string{
				"grant_type":    "authorization_code",
				"client_id":     "my_app",
				"client_secret": "super_secret",
				"code":          "abc123",
				"redirect_uri":  "https://example.com/callback",
			},
		},
		{
			name:        "revoke request",
			body:        "token=eyJhbGciOiJSUzI1NiJ9.test_token_value",
			contentType: "application/x-www-form-urlencoded",
			wantOK:      true,
			wantFields: map[string]string{
				"token": "eyJhbGciOiJSUzI1NiJ9.test_token_value",
			},
		},
		{
			name:        "JSON content type is skipped",
			body:        `{"token":"abc"}`,
			contentType: "application/json",
			wantOK:      false,
		},
		{
			name:        "empty content type is skipped",
			body:        "key=value",
			contentType: "",
			wantOK:      false,
		},
		{
			name:        "empty form body returns empty object",
			body:        "",
			contentType: "application/x-www-form-urlencoded",
			wantOK:      true,
			wantFields:  map[string]string{},
		},
		{
			name:        "multi-value key keeps first value",
			body:        "scope=read&scope=write",
			contentType: "application/x-www-form-urlencoded",
			wantOK:      true,
			wantFields: map[string]string{
				"scope": "read",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := formBodyToJSON([]byte(tt.body), tt.contentType)
			assert.Equal(t, tt.wantOK, ok)

			if !tt.wantOK {
				assert.Nil(t, got)
				return
			}

			var parsed map[string]string
			err := json.Unmarshal(got, &parsed)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantFields, parsed)
		})
	}
}
