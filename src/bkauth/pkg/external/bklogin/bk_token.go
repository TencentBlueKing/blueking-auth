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

	"go.uber.org/zap"

	"bkauth/pkg/errorx"
	"bkauth/pkg/logging"
	"bkauth/pkg/util"
)

const bkTokenSVC = "bklogin.BKTokenVerifier"

type bkTokenResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Data    struct {
		Username string `json:"username"`
	} `json:"data"`
	Result bool `json:"result"`
}

// BKTokenVerifier verifies a bk_token by calling the BK Login direct API (accounts/is_login/).
type BKTokenVerifier struct {
	baseURL string
}

// NewBKTokenVerifier creates a BKTokenVerifier bound to the given BK Login base URL.
func NewBKTokenVerifier(baseURL string) *BKTokenVerifier {
	return &BKTokenVerifier{baseURL: baseURL}
}

func (v *BKTokenVerifier) Verify(ctx context.Context, token string) (VerifyResult, error) {
	logger := logging.GetWebLogger()
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(bkTokenSVC, "")

	api := util.URLJoin(v.baseURL, "accounts/is_login/")
	checkURL := util.URLSetQuery(api, url.Values{"bk_token": {token}})

	tokenPreview := token
	if len(tokenPreview) > 12 {
		tokenPreview = tokenPreview[:12] + "..."
	}

	logger.Info("bk_token verify: sending request",
		zap.String("api", api),
		zap.String("token_preview", tokenPreview),
		zap.Int("token_len", len(token)),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		return VerifyResult{
			Message: "failed to build request",
		}, errorWrapf(err, "http.NewRequestWithContext url=`%s` fail", checkURL)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		logger.Error("bk_token verify: http request failed",
			zap.Error(err),
			zap.String("api", api),
		)
		return VerifyResult{
			Message: "failed to connect to login service",
		}, errorWrapf(err, "http.Do url=`%s` fail", checkURL)
	}
	defer resp.Body.Close()

	logger.Info("bk_token verify: response received",
		zap.Int("status_code", resp.StatusCode),
		zap.String("status", resp.Status),
	)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("bk_token verify: failed to read response body", zap.Error(err))
		return VerifyResult{
			Message: "failed to read login response",
		}, errorWrapf(err, "io.ReadAll fail")
	}

	bodyStr := string(body)
	if len(bodyStr) > 512 {
		bodyStr = bodyStr[:512] + "...(truncated)"
	}
	logger.Info("bk_token verify: response body",
		zap.String("body", bodyStr),
		zap.Int("body_len", len(body)),
	)

	var loginResp bkTokenResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		logger.Error("bk_token verify: failed to parse response JSON",
			zap.Error(err),
			zap.String("body", bodyStr),
		)
		return VerifyResult{
			Message: "failed to parse login response",
		}, errorWrapf(err, "json.Unmarshal fail")
	}

	if !loginResp.Result {
		logger.Warn("bk_token verify: login verification returned false",
			zap.String("message", loginResp.Message),
			zap.String("code", loginResp.Code),
		)
		return VerifyResult{Message: loginResp.Message}, nil
	}

	logger.Info("bk_token verify: login verified successfully",
		zap.String("username", loginResp.Data.Username),
	)

	return VerifyResult{
		Success:  true,
		Username: loginResp.Data.Username,
	}, nil
}
