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

package oauth

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"bkauth/pkg/util"
)

// IsLoopbackHost reports whether host is a loopback address (RFC 8252 Section 7.3).
func IsLoopbackHost(host string) bool {
	switch host {
	case "127.0.0.1", "::1", "[::1]", "localhost":
		return true
	}
	return false
}

// BuildAuthorizationRedirectURL builds the success redirect with authorization code (RFC 6749 Section 4.1.2).
func BuildAuthorizationRedirectURL(redirectURI, state, code string) string {
	return util.URLSetQuery(redirectURI, url.Values{
		"code":  {code},
		"state": {state},
	})
}

// BuildErrorRedirectURL builds the error redirect (RFC 6749 Section 4.1.2.1).
func BuildErrorRedirectURL(redirectURI, state, errorCode, errorDesc string) string {
	return util.URLSetQuery(redirectURI, url.Values{
		"error":             {errorCode},
		"error_description": {errorDesc},
		"state":             {state},
	})
}

// MatchRedirectURI checks if requestURI matches the registered registeredURI.
// For loopback addresses the port is ignored per RFC 8252 Section 7.3.
func MatchRedirectURI(registeredURI, requestURI string) bool {
	if registeredURI == requestURI {
		return true
	}

	reg, err := url.Parse(registeredURI)
	if err != nil {
		return false
	}

	if !IsLoopbackHost(reg.Hostname()) {
		return false
	}

	req, err := url.Parse(requestURI)
	if err != nil {
		return false
	}

	return reg.Scheme == req.Scheme &&
		reg.Hostname() == req.Hostname() &&
		reg.Path == req.Path
}

// MatchRegisteredRedirectURI checks if requestURI matches any of the registered URIs.
func MatchRegisteredRedirectURI(registeredURIs []string, requestURI string) bool {
	for _, uri := range registeredURIs {
		if MatchRedirectURI(uri, requestURI) {
			return true
		}
	}
	return false
}

var forbiddenSchemes = map[string]bool{
	"file":       true,
	"data":       true,
	"javascript": true,
	"ftp":        true,
}

// ValidateRedirectURI checks a single redirect URI for format correctness.
func ValidateRedirectURI(raw string) error {
	if raw == "" {
		return errors.New("redirect_uri cannot be empty")
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("redirect_uri is not a valid URL: %s", raw)
	}

	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		if parsed.Host == "" {
			return fmt.Errorf("redirect_uri with scheme %s must have a host: %s", parsed.Scheme, raw)
		}
	}

	if forbiddenSchemes[strings.ToLower(parsed.Scheme)] {
		return fmt.Errorf("redirect_uri scheme '%s' is not allowed: %s", parsed.Scheme, raw)
	}

	// RFC 6749 §3.1.2: redirect URI MUST NOT include a fragment component
	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		return fmt.Errorf("redirect_uri must not contain a fragment (#): %s", raw)
	}

	if parsed.User != nil {
		return fmt.Errorf("redirect_uri must not contain userinfo: %s", raw)
	}

	return nil
}
