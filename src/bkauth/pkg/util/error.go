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
	"errors"
	"time"

	sentry "github.com/getsentry/sentry-go"

	"bkauth/pkg/errorx"
)

// Error Codes
const (
	NoError           = 0
	BadRequestError   = 1903400
	UnauthorizedError = 1903401
	ForbiddenError    = 1903403
	NotFoundError     = 1903404
	ConflictError     = 1903409
	SystemError       = 1903500
	TooManyRequests   = 1903429
)

// ReportToSentry is a shortcut to build and send an event to sentry
func ReportToSentry(message string, extra map[string]interface{}) {
	// report to sentry
	ev := sentry.NewEvent()
	ev.Message = message
	ev.Level = "error"
	ev.Timestamp = time.Now()
	ev.Extra = extra
	errorx.ReportEvent(ev)
}

type ValidationError struct {
	err error
}

func (e *ValidationError) Error() string {
	return e.err.Error()
}

func ValidationErrorWrap(err error) *ValidationError {
	return &ValidationError{
		err: err,
	}
}

func IsValidationError(err error) bool {
	var validationError *ValidationError
	return errors.As(err, &validationError)
}
