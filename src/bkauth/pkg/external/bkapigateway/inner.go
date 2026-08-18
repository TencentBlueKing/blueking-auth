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

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go.uber.org/zap"

	"bkauth/pkg/logging"
	"bkauth/pkg/util"
)

const (
	// OAuthClientTypePublic selects the scopes a confidential OAuth2 client may
	// be granted; OAuthClientTypePersonal selects the ones a personal access
	// token may. Upstream keeps a separate switch per object for each.
	OAuthClientTypePublic   = "public"
	OAuthClientTypePersonal = "personal"

	// namesPerRequest is the upstream cap on every comma separated name filter
	// of the v2 inner APIs. Callers hand over a whole name set and the split is
	// done here, because how many round trips it takes is nobody else's concern.
	namesPerRequest = 50
)

// ErrNotInitialized is returned when the package is used before Init, which
// happens when the gateway URL template is left empty in the config.
var ErrNotInitialized = errors.New("bkapigateway: base url is not initialized")

// APIError is a non-2xx answer from the v2 inner APIs. It is kept as a distinct
// type so callers can tell "this gateway does not exist" (404, expected while
// rendering a token that references a deleted gateway) from a transport or
// server failure, which should not be swallowed.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf(
		"bkapigateway: request failed, status=%d code=%s message=%s",
		e.StatusCode, e.Code, e.Message,
	)
}

// IsNotFound reports whether err is an upstream 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// innerErrorEnvelope is the v2 error shape. It differs from the open APIs,
// which carry a numeric code alongside the payload; here a failure is signalled
// by the status code and the body only names it.
type innerErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// innerGet issues a GET against path under the v2 inner API and decodes the
// envelope's data member into out.
func innerGet(ctx context.Context, tenantID, path string, query url.Values, out any) error {
	if baseURL == "" {
		return ErrNotInitialized
	}

	api := util.URLJoin(baseURL, path)
	if encoded := query.Encode(); encoded != "" {
		api += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return fmt.Errorf("bkapigateway: build request for %s: %w", path, err)
	}
	req.Header.Set("X-Bkapi-Authorization", authCredentials)
	// Multi-tenant deployments scope four of the five inner APIs by this header;
	// without it the caller silently gets only the tenant-neutral subset.
	if tenantID != "" {
		req.Header.Set("X-Bk-Tenant-Id", tenantID)
	}

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("bkapigateway: call %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("bkapigateway: read response of %s: %w", path, err)
	}

	if resp.StatusCode != http.StatusOK {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		var envelope innerErrorEnvelope
		if json.Unmarshal(body, &envelope) == nil {
			apiErr.Code = envelope.Error.Code
			apiErr.Message = envelope.Error.Message
		}

		logging.GetWebLogger().Error("bkapigateway: request failed",
			zap.String("api", api),
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", truncateBody(body)),
		)
		return apiErr
	}

	// The data member is decoded on its own so each caller can name the shape it
	// expects, which differs between the paged and the plain-array endpoints.
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("bkapigateway: decode envelope of %s: %w", path, err)
	}
	if len(envelope.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("bkapigateway: decode data of %s: %w", path, err)
	}
	return nil
}

func truncateBody(body []byte) string {
	const maxLoggedBody = 512
	if len(body) > maxLoggedBody {
		return string(body[:maxLoggedBody]) + "...(truncated)"
	}
	return string(body)
}

// chunkNames splits names into slices the upstream name filters accept. Blank
// and duplicate entries are dropped first: upstream ignores them anyway, and
// letting them through would waste room in a chunk.
func chunkNames(names []string) [][]string {
	seen := make(map[string]struct{}, len(names))
	deduped := make([]string, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		deduped = append(deduped, name)
	}

	chunks := make([][]string, 0, (len(deduped)+namesPerRequest-1)/namesPerRequest)
	for start := 0; start < len(deduped); start += namesPerRequest {
		end := min(start+namesPerRequest, len(deduped))
		chunks = append(chunks, deduped[start:end])
	}
	return chunks
}

func setIfNotEmpty(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}
