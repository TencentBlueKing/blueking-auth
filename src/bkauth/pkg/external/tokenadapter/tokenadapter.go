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

// Package tokenadapter is the client side of the external token adapter
// protocol: an RFC 7662 style introspection whose answer is already in BlueKing
// terms. Vendor integrations live in the adapter service, not here.
package tokenadapter

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// A sane answer is a few hundred bytes; the cap only keeps a misbehaving
// adapter from making us buffer an unbounded body.
const maxResponseBodySize = 1 << 20

var (
	endpoint     string
	clientID     string
	clientSecret string
	timeout      time.Duration

	// The timeout is applied per call through the context.
	defaultHTTPClient = &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		// Protocol v1 forbids redirects. Following a 307/308 would re-send the body,
		// raw token included, to wherever Location points; following a 301-303 would
		// turn the call into a bodiless GET whose answer would pass for the
		// adapter's. Handing back the 3xx sends it down the non-200 failure path.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

// Init sets where the adapter is reached and how bkauth authenticates to it,
// with HTTP Basic. Must be called once during startup before any client is used.
func Init(adapterEndpoint, adapterClientID, adapterClientSecret string, adapterTimeout time.Duration) {
	endpoint = adapterEndpoint
	clientID = adapterClientID
	clientSecret = adapterClientSecret
	timeout = adapterTimeout
}

// describeTransportError says what failed without saying where. The standard
// library names the target at every layer: url.Error quotes the request URL,
// net.OpError the resolved address, net.DNSError the host name.
func describeTransportError(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}

	var dnsErr *net.DNSError
	var opErr *net.OpError
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timed out"
	case errors.As(err, &dnsErr):
		return "name resolution failed: " + dnsErr.Err
	case errors.As(err, &opErr):
		return opErr.Op + " failed: " + opErr.Err.Error()
	default:
		return err.Error()
	}
}
