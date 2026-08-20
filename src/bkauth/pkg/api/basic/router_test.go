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

package basic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"bkauth/pkg/config"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}

	r := gin.Default()
	Register(cfg, r)

	assert.NotNil(t, r)
}

func TestRegisterIndexRedirectsToFrontend(t *testing.T) {
	t.Parallel()

	r := gin.New()
	Register(&config.Config{BKAuthURL: "https://bkauth.example.com"}, r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://bkauth.example.com/web/dashboard", w.Header().Get("Location"))
}

// The index redirect must stay on the exact path: an unmatched API path is a
// 404, not an invitation to the frontend.
func TestRegisterLeavesUnmatchedPathsUnredirected(t *testing.T) {
	t.Parallel()

	r := gin.New()
	Register(&config.Config{BKAuthURL: "https://bkauth.example.com"}, r)

	for _, path := range []string{"/api/v1/not-exist", "/.well-known/unknown"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))

		assert.Equal(t, http.StatusNotFound, w.Code, path)
	}
}
