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

const bkTicketSVC = "bklogin.BKTicketVerifier"

type bkTicketResponse struct {
	Ret  int    `json:"ret"`
	Msg  string `json:"msg"`
	Data struct {
		Username string `json:"username"`
	} `json:"data"`
}

// BKTicketVerifier verifies a bk_ticket by calling the BK Login direct API (user/get_info).
type BKTicketVerifier struct {
	baseURL string
}

// NewBKTicketVerifier creates a BKTicketVerifier bound to the given BK Login base URL.
func NewBKTicketVerifier(baseURL string) *BKTicketVerifier {
	return &BKTicketVerifier{baseURL: baseURL}
}

func (v *BKTicketVerifier) Verify(ctx context.Context, ticket string) (VerifyResult, error) {
	logger := logging.GetWebLogger()
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(bkTicketSVC, "")

	api := util.URLJoin(v.baseURL, "user/get_info")
	checkURL := util.URLSetQuery(api, url.Values{"bk_ticket": {ticket}})

	tokenPreview := ticket
	if len(tokenPreview) > 12 {
		tokenPreview = tokenPreview[:12] + "..."
	}

	logger.Info("bk_ticket verify: sending request",
		zap.String("api", api),
		zap.String("token_preview", tokenPreview),
		zap.Int("token_len", len(ticket)),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		return VerifyResult{
			Message: "failed to build request",
		}, errorWrapf(err, "http.NewRequestWithContext url=`%s` fail", checkURL)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		logger.Error("bk_ticket verify: http request failed", zap.Error(err))
		return VerifyResult{
			Message: "failed to connect to login service",
		}, errorWrapf(err, "http.Do url=`%s` fail", checkURL)
	}
	defer resp.Body.Close()

	logger.Info("bk_ticket verify: response received",
		zap.Int("status_code", resp.StatusCode),
	)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("bk_ticket verify: failed to read response body", zap.Error(err))
		return VerifyResult{
			Message: "failed to read login response",
		}, errorWrapf(err, "io.ReadAll fail")
	}

	bodyStr := string(body)
	if len(bodyStr) > 512 {
		bodyStr = bodyStr[:512] + "...(truncated)"
	}
	logger.Info("bk_ticket verify: response body",
		zap.String("body", bodyStr),
		zap.Int("body_len", len(body)),
	)

	var loginResp bkTicketResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		logger.Error("bk_ticket verify: failed to parse response JSON",
			zap.Error(err),
			zap.String("body", bodyStr),
		)
		return VerifyResult{
			Message: "failed to parse login response",
		}, errorWrapf(err, "json.Unmarshal fail")
	}

	if loginResp.Ret != 0 {
		logger.Warn("bk_ticket verify: verification returned non-zero ret",
			zap.Int("ret", loginResp.Ret),
			zap.String("msg", loginResp.Msg),
		)
		return VerifyResult{Message: loginResp.Msg}, nil
	}

	logger.Info("bk_ticket verify: login verified successfully",
		zap.String("username", loginResp.Data.Username),
	)

	return VerifyResult{
		Success:  true,
		Sub:      loginResp.Data.Username,
		Username: loginResp.Data.Username,
	}, nil
}
