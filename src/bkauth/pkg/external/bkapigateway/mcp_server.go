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

package bkapigateway

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"

	"bkauth/pkg/errorx"
	"bkauth/pkg/logging"
	"bkauth/pkg/util"
)

// MCPServerClient queries MCP server display info from BK API Gateway.
type MCPServerClient interface {
	// BatchQueryTitles returns a map of mcp_name → title for the given names.
	// Names not found in the remote response are omitted from the result.
	BatchQueryTitles(ctx context.Context, names []string) (map[string]string, error)
}

type mcpServerClient struct{}

// NewMCPServerClient creates the default MCPServerClient backed by the BK API Gateway.
func NewMCPServerClient() MCPServerClient {
	return &mcpServerClient{}
}

const mcpServerSVC = "bkapigateway.MCPServer"

type batchQueryMCPRequest struct {
	Names  []string `json:"names"`
	Fields string   `json:"fields"`
}

type batchQueryMCPResponse struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    []mcpServerResult `json:"data"`
}

type mcpServerResult struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

func (c *mcpServerClient) BatchQueryTitles(ctx context.Context, names []string) (map[string]string, error) {
	if baseURL == "" || len(names) == 0 {
		return nil, nil
	}

	logger := logging.GetWebLogger()
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(mcpServerSVC, "BatchQueryTitles")

	api := util.URLJoin(baseURL, "api/v2/open/mcp-servers/batch-query/")

	reqBody := batchQueryMCPRequest{
		Names:  names,
		Fields: "name,title",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errorWrapf(err, "json.Marshal request body fail")
	}

	logger.Info("bkapigateway mcp batch query: sending request",
		zap.String("api", api),
		zap.Strings("names", names),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errorWrapf(err, "http.NewRequest url=`%s` fail", api)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bkapi-Authorization", authCredentials)

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		logger.Error("bkapigateway mcp batch query: http request failed",
			zap.Error(err),
			zap.String("api", api),
		)
		return nil, errorWrapf(err, "http.Do url=`%s` fail", api)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("bkapigateway mcp batch query: failed to read response body", zap.Error(err))
		return nil, errorWrapf(err, "io.ReadAll fail")
	}

	bodyStr := string(body)
	if len(bodyStr) > 512 {
		bodyStr = bodyStr[:512] + "...(truncated)"
	}
	logger.Info("bkapigateway mcp batch query: response",
		zap.Int("status_code", resp.StatusCode),
		zap.String("body", bodyStr),
	)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bkapigateway mcp batch query: unexpected status %d", resp.StatusCode)
	}

	var result batchQueryMCPResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errorWrapf(err, "json.Unmarshal fail")
	}

	if result.Code != 0 {
		return nil, fmt.Errorf(
			"bkapigateway mcp batch query error: code=%d, message=%s", result.Code, result.Message,
		)
	}

	titles := make(map[string]string, len(result.Data))
	for _, item := range result.Data {
		if item.Title != "" {
			titles[item.Name] = item.Title
		}
	}
	return titles, nil
}
