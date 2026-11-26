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

package tokenadapter

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrInvalidResponse means the adapter answered 200 with a body that is not a
// protocol answer. It is a failure of the adapter, not a verdict on the token.
var ErrInvalidResponse = errors.New("tokenadapter: response is not a protocol answer")

// Introspector introspects a third-party token through the adapter.
type Introspector interface {
	// Introspect asks the adapter about token as seen from realm. An inactive
	// token is a normal answer, not an error; an error means the adapter could
	// not answer, and callers must not mistake it for "inactive".
	Introspect(ctx context.Context, realm, token string) (Response, error)
}

// Response is the adapter's answer. When Active is false nothing else is
// meaningful.
type Response struct {
	Active   bool     `json:"active"`
	ClientID string   `json:"client_id"`
	Sub      string   `json:"sub"`
	Username string   `json:"username"`
	TenantID string   `json:"tenant_id"`
	Audience []string `json:"aud"`
	// ExpiresAt is a Unix timestamp in seconds.
	ExpiresAt int64 `json:"exp"`
}

type introspector struct{}

// NewIntrospector creates the default Introspector backed by the adapter
// configured through Init.
func NewIntrospector() Introspector {
	return &introspector{}
}

func (i *introspector) Introspect(ctx context.Context, realm, token string) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The token travels in the body, never the URL, which ends up in far more
	// logs; the credentials travel in Authorization, the header every proxy and
	// log pipeline already treats as sensitive.
	form := url.Values{
		"token": {token},
		"realm": {realm},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Response{}, fmt.Errorf("tokenadapter: build request for %s: %w", endpoint, err)
	}
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		// url.Error repeats the URL but never the body, so the token stays out.
		return Response{}, fmt.Errorf("tokenadapter: call %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return Response{}, fmt.Errorf("tokenadapter: read response of %s: %w", endpoint, err)
	}

	// An inactive token is answered with 200 as well, so anything else is the
	// adapter failing rather than judging the token.
	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("tokenadapter: %s answered status %d", endpoint, resp.StatusCode)
	}

	var out Response
	if err := json.Unmarshal(body, &out); err != nil {
		return Response{}, fmt.Errorf("%w: decode response of %s: %w", ErrInvalidResponse, endpoint, err)
	}
	// Decoded into a bool, a missing active reads as false: a failure dressed as
	// a 200 (`{}`, `{"error": ...}`) would pass for an inactive token and be
	// cached as one, locking the user out for the whole TTL.
	var presence struct {
		Active *bool `json:"active"`
	}
	if err := json.Unmarshal(body, &presence); err != nil || presence.Active == nil {
		return Response{}, fmt.Errorf("%w: %s answered without active", ErrInvalidResponse, endpoint)
	}
	return out, nil
}
