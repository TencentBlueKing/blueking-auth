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
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"bkauth/pkg/logging"
	"bkauth/pkg/util"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write ...
func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// APILogger ...
func APILogger() gin.HandlerFunc {
	logger := logging.GetAPILogger()

	return func(c *gin.Context) {
		fields := logContextFields(c, false)
		logger.Info("-", fields...)
	}
}

// WebLogger ...
func WebLogger() gin.HandlerFunc {
	logger := logging.GetWebLogger()

	return func(c *gin.Context) {
		fields := logContextFields(c, false)
		logger.Info("-", fields...)
	}
}

// AuditRequestMethod : only record `change` method audit log
var AuditRequestMethod = map[string]bool{
	"POST":   true,
	"PUT":    true,
	"DELETE": true,
	"PATCH":  true,
}

func AuditLogger() gin.HandlerFunc {
	auditLogger := logging.GetAuditLogger()
	apiLogger := logging.GetAPILogger()

	return func(c *gin.Context) {
		fields := logContextFields(c, true)
		_, willAudit := AuditRequestMethod[c.Request.Method]
		// 非审计需要的请求，则直接记录到API流水日志里
		if willAudit {
			auditLogger.Info("-", fields...)
		} else {
			apiLogger.Info("-", fields...)
		}
	}
}

func logContextFields(c *gin.Context, logFullRequestBody bool) []zap.Field {
	start := time.Now()

	// request body
	var body string
	requestBody, err := util.ReadRequestBody(c.Request)
	if err != nil {
		body = ""
	} else {
		// Normalize form-urlencoded body to JSON so gjson-based desensitization can match field paths.
		if jsonBody, ok := formBodyToJSON(requestBody, c.ContentType()); ok {
			requestBody = jsonBody
		}

		if logFullRequestBody {
			body = string(requestBody)
		} else {
			body = util.TruncateBytesToString(requestBody, 1024)
		}
	}

	newWriter := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
	c.Writer = newWriter

	c.Next()

	duration := time.Since(start)
	// always add 1ms, in case the 0ms in log
	latency := float64(duration/time.Millisecond) + 1

	e, hasError := util.GetError(c)
	if !hasError {
		e = ""
	}

	params := util.TruncateString(c.Request.URL.RawQuery, 1024)
	fields := []zap.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("params", params),
		zap.String("body", body),
		zap.Int("status", c.Writer.Status()),
		zap.Float64("latency", latency),
		zap.String("request_id", util.GetRequestID(c)),
		zap.String("access_app_code", util.GetAccessAppCode(c)),
		zap.String("client_ip", c.ClientIP()),
		zap.Any("error", e),
	}

	if hasError {
		fields = append(fields, zap.String("response_body", newWriter.body.String()))
	} else {
		fields = append(fields, zap.String("response_body", util.TruncateString(newWriter.body.String(), 1024)))
	}

	if hasError && e != nil {
		util.ReportToSentry(
			fmt.Sprintf("%s %s error", c.Request.Method, c.Request.URL.Path),
			map[string]interface{}{
				"fields": fields,
			},
		)
	}

	return fields
}

const contentTypeFormURLEncoded = "application/x-www-form-urlencoded"

// formBodyToJSON converts a URL-encoded form body to a JSON object ([]byte),
// so that gjson-based log desensitization can match field paths like "client_secret".
// Returns (nil, false) when contentType is not form-urlencoded or parsing fails.
func formBodyToJSON(body []byte, contentType string) ([]byte, bool) {
	if contentType != contentTypeFormURLEncoded {
		return nil, false
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, false
	}

	m := make(map[string]string, len(values))
	for k, v := range values {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return nil, false
	}

	return jsonBytes, true
}
