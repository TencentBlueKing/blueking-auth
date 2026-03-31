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

// Package bkapigateway provides HTTP client functions for calling
// the BlueKing API Gateway service APIs.
package bkapigateway

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"bkauth/pkg/util"
)

const (
	gatewayName  = "bk-apigateway"
	gatewayStage = "prod"
)

var (
	baseURL         string
	authCredentials string
)

// Init resolves the gateway base URL from the URL template and pre-serializes
// the authentication header. Must be called once during startup before any
// client is used.
func Init(bkApiURLTmpl, appCode, appSecret string) {
	apiURL := strings.Replace(bkApiURLTmpl, "{api_name}", gatewayName, 1)
	baseURL = util.URLJoin(apiURL, gatewayStage)
	data, err := json.Marshal(map[string]string{
		"bk_app_code":   appCode,
		"bk_app_secret": appSecret,
	})
	if err != nil {
		panic("bkapigateway: failed to marshal auth credentials: " + err.Error())
	}
	authCredentials = string(data)
}

var defaultHTTPClient = &http.Client{
	Transport: otelhttp.NewTransport(http.DefaultTransport),
	Timeout:   5 * time.Second,
}
