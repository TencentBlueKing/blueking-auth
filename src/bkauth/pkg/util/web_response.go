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

package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	webSystemName = "bkauth"

	// Protocol error codes. Unexported: each one is reachable only through the
	// shortcut that pairs it with its HTTP status, so a caller cannot invent a
	// combination the protocol does not define.
	webErrCodeInvalidArgument   = "INVALID_ARGUMENT"
	webErrCodeUnauthenticated   = "UNAUTHENTICATED"
	webErrCodeNoPermission      = "NO_PERMISSION"
	webErrCodeNotFound          = "NOT_FOUND"
	webErrCodeAlreadyExists     = "ALREADY_EXISTS"
	webErrCodeResourceExhausted = "RESOURCE_EXHAUSTED"
	webErrCodeInternal          = "INTERNAL"
)

type webSuccessResponse struct {
	Data interface{} `json:"data"`
}

// webErrorObject is the protocol Error object. message is what a user should
// read; data is optional structured input for the caller (login_url, policy
// bounds). details is omitted: user-facing text belongs in message, not a side
// channel.
type webErrorObject struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	System  string      `json:"system,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type webErrorResponse struct {
	Error webErrorObject `json:"error"`
}

// WebSuccess writes {"data": ...} for a 200 web response.
func WebSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, webSuccessResponse{Data: data})
}

// webError writes the error envelope. It does not stop the handler chain: that
// is the caller's business, so handlers return after it and middleware pairs it
// with c.Abort.
//
// The message is the whole user-facing explanation. Carrying nothing else is the
// normal case, which is why this signature has no data parameter.
func webError(c *gin.Context, httpStatus int, code string, message string) {
	webErrorWithData(c, httpStatus, code, message, nil)
}

// webErrorWithData is webError for the rare rejection a client has to act on
// programmatically rather than merely display.
func webErrorWithData(c *gin.Context, httpStatus int, code string, message string, data interface{}) {
	c.JSON(httpStatus, webErrorResponse{
		Error: webErrorObject{
			Code:    code,
			Message: message,
			System:  webSystemName,
			Data:    data,
		},
	})
}

// WebInvalidArgumentError answers 400 for unusable input, whether it is
// malformed or merely breaks a business rule: the caller has to fix the request
// either way, so the protocol does not split the two.
func WebInvalidArgumentError(c *gin.Context, message string) {
	webError(c, http.StatusBadRequest, webErrCodeInvalidArgument, message)
}

// WebBindError answers 400 for a gin binding / JSON-decode failure, carrying the
// validator's first field message so the user sees which input to fix.
func WebBindError(c *gin.Context, err error) {
	WebInvalidArgumentError(c, ValidationErrorMessage(err))
}

// WebUnauthenticatedErrorWithData answers 401. The data carries where to log in;
// without it the frontend has a rejection it cannot act on.
func WebUnauthenticatedErrorWithData(c *gin.Context, message string, data interface{}) {
	webErrorWithData(c, http.StatusUnauthorized, webErrCodeUnauthenticated, message, data)
}

// WebNoPermissionError answers 403 for a caller this service refuses, as opposed
// to one the permission center refuses.
func WebNoPermissionError(c *gin.Context, message string) {
	webError(c, http.StatusForbidden, webErrCodeNoPermission, message)
}

// WebNotFoundError answers 404.
func WebNotFoundError(c *gin.Context, message string) {
	webError(c, http.StatusNotFound, webErrCodeNotFound, message)
}

// WebAlreadyExistsError answers 409 for a resource that is already in the state
// the request would create.
func WebAlreadyExistsError(c *gin.Context, message string) {
	webError(c, http.StatusConflict, webErrCodeAlreadyExists, message)
}

// WebResourceExhaustedError answers 429 for a spent quota, as opposed to input
// the caller can correct.
func WebResourceExhaustedError(c *gin.Context, message string) {
	webError(c, http.StatusTooManyRequests, webErrCodeResourceExhausted, message)
}

// WebInternalError answers 500. The message stays fixed: the underlying error
// belongs in the logs, not in the response.
func WebInternalError(c *gin.Context, message string) {
	webError(c, http.StatusInternalServerError, webErrCodeInternal, message)
}
