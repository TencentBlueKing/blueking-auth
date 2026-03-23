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
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	systemName = "bkauth"

	webErrCodeInvalidArgument = "INVALID_ARGUMENT"
	webErrCodeUnauthenticated = "UNAUTHENTICATED"
	webErrCodeNotFound        = "NOT_FOUND"
	webErrCodeInternal        = "INTERNAL"
)

type webSuccessResponse struct {
	Data interface{} `json:"data"`
}

type webErrorDetail struct {
	Field   string      `json:"field,omitempty"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type webError struct {
	Code       string           `json:"code"`
	Message    string           `json:"message"`
	SystemName string           `json:"system_name"`
	Details    []webErrorDetail `json:"details,omitempty"`
	Data       interface{}      `json:"data,omitempty"`
}

type webErrorResponse struct {
	Error webError `json:"error"`
}

func webJSONSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, webSuccessResponse{Data: data})
}

func webJSONError(c *gin.Context, httpStatus int, code string, message string) {
	c.JSON(httpStatus, webErrorResponse{
		Error: webError{
			Code:       code,
			Message:    message,
			SystemName: systemName,
		},
	})
}

func webJSONErrorWithDetails(c *gin.Context, httpStatus int, code string, message string, details []webErrorDetail) {
	c.JSON(httpStatus, webErrorResponse{
		Error: webError{
			Code:       code,
			Message:    message,
			SystemName: systemName,
			Details:    details,
		},
	})
}
