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

package util

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrNilRequestBody ...
var ErrNilRequestBody = errors.New("request Body is nil")

// ReadRequestBody will return the body in []byte, without change the origin body
func ReadRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, ErrNilRequestBody
	}

	body, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, err
}

// GetRequestID ...
func GetRequestID(c *gin.Context) string {
	return c.GetString(RequestIDKey)
}

// SetRequestID ...
func SetRequestID(c *gin.Context, requestID string) {
	c.Set(RequestIDKey, requestID)
}

// GetAccessAppCode ...
func GetAccessAppCode(c *gin.Context) string {
	return c.GetString(AccessAppCodeKey)
}

// SetAccessAppCode ...
func SetAccessAppCode(c *gin.Context, clientID string) {
	c.Set(AccessAppCodeKey, clientID)
}

// GetError ...
func GetError(c *gin.Context) (interface{}, bool) {
	return c.Get(ErrorIDKey)
}

// SetError ...
func SetError(c *gin.Context, err error) {
	c.Set(ErrorIDKey, err)
}

func SetEnableMultiTenantMode(c *gin.Context, enableMultiTenantMode bool) {
	c.Set(EnableMultiTenantModeKey, enableMultiTenantMode)
}

func GetEnableMultiTenantMode(c *gin.Context) bool {
	return c.GetBool(EnableMultiTenantModeKey)
}
