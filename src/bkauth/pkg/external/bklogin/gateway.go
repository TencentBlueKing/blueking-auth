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

package bklogin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"

	"bkauth/pkg/errorx"
	"bkauth/pkg/logging"
	"bkauth/pkg/util"
)

const (
	bkTokenGatewaySVC        = "bklogin.BKTokenGatewayVerifier"
	bkTokenGatewayName       = "bk-login"
	bkTokenGatewayStage      = "prod"
	bkTokenGatewayVerifyPath = "login/api/v3/open/bk-tokens/verify/"
)

type bkGatewayResponse struct {
	Data *struct {
		BKUsername string `json:"bk_username"`
		TenantID   string `json:"tenant_id"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// BKTokenGatewayVerifier verifies a bk_token by calling the BK Login API through the BK API Gateway.
type BKTokenGatewayVerifier struct {
	baseURL         string
	authCredentials string
}

// NewBKTokenGatewayVerifier creates a BKTokenGatewayVerifier.
// bkApiURLTmpl is the gateway URL template containing {api_name} placeholder.
func NewBKTokenGatewayVerifier(bkApiURLTmpl, appCode, appSecret string) *BKTokenGatewayVerifier {
	bkApiURL := strings.Replace(bkApiURLTmpl, "{api_name}", bkTokenGatewayName, 1)
	authData, _ := json.Marshal(map[string]string{
		"bk_app_code":   appCode,
		"bk_app_secret": appSecret,
	})
	return &BKTokenGatewayVerifier{
		baseURL:         util.URLJoin(bkApiURL, bkTokenGatewayStage),
		authCredentials: string(authData),
	}
}

func (v *BKTokenGatewayVerifier) Verify(ctx context.Context, token string) (VerifyResult, error) {
	logger := logging.GetWebLogger()
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(bkTokenGatewaySVC, "")

	api := util.URLJoin(v.baseURL, bkTokenGatewayVerifyPath)
	checkURL := util.URLSetQuery(api, url.Values{"bk_token": {token}})

	tokenPreview := token
	if len(tokenPreview) > 12 {
		tokenPreview = tokenPreview[:12] + "..."
	}

	logger.Info("gateway verify: sending request",
		zap.String("api", api),
		zap.String("token_preview", tokenPreview),
		zap.Int("token_len", len(token)),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		return VerifyResult{
			Message: "failed to build request",
		}, errorWrapf(err, "http.NewRequest url=`%s` fail", checkURL)
	}
	req.Header.Set("X-Bkapi-Authorization", v.authCredentials)

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		logger.Error("gateway verify: http request failed",
			zap.Error(err),
			zap.String("api", api),
		)
		return VerifyResult{
			Message: "failed to connect to login service via gateway",
		}, errorWrapf(err, "http.Do url=`%s` fail", checkURL)
	}
	defer resp.Body.Close()

	logger.Info("gateway verify: response received",
		zap.Int("status_code", resp.StatusCode),
		zap.String("status", resp.Status),
	)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("gateway verify: failed to read response body", zap.Error(err))
		return VerifyResult{
			Message: "failed to read login response",
		}, errorWrapf(err, "io.ReadAll fail")
	}

	bodyStr := string(body)
	if len(bodyStr) > 512 {
		bodyStr = bodyStr[:512] + "...(truncated)"
	}
	logger.Info("gateway verify: response body",
		zap.String("body", bodyStr),
		zap.Int("body_len", len(body)),
	)

	var gatewayResp bkGatewayResponse
	if err := json.Unmarshal(body, &gatewayResp); err != nil {
		logger.Error("gateway verify: failed to parse response JSON",
			zap.Error(err),
			zap.String("body", bodyStr),
		)
		return VerifyResult{
			Message: "failed to parse login response",
		}, errorWrapf(err, "json.Unmarshal fail")
	}

	if gatewayResp.Error != nil {
		logger.Warn("gateway verify: login verification failed",
			zap.String("error_code", gatewayResp.Error.Code),
			zap.String("error_message", gatewayResp.Error.Message),
		)
		return VerifyResult{Message: gatewayResp.Error.Message}, nil
	}

	if gatewayResp.Data == nil || gatewayResp.Data.BKUsername == "" {
		logger.Warn("gateway verify: empty username in response")
		return VerifyResult{Message: "empty username in login response"}, nil
	}

	logger.Info("gateway verify: login verified successfully",
		zap.String("username", gatewayResp.Data.BKUsername),
		zap.String("tenant_id", gatewayResp.Data.TenantID),
	)

	return VerifyResult{
		Success:  true,
		Username: gatewayResp.Data.BKUsername,
		TenantID: gatewayResp.Data.TenantID,
	}, nil
}
