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

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// serve starts an adapter for the duration of one spec, points the package at
// it through Init, and returns an introspector plus the requests it received.
func serve(adapterTimeout time.Duration, handler http.HandlerFunc) (Introspector, *[]*http.Request) {
	var received []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		received = append(received, r)
		handler(w, r)
	}))
	DeferCleanup(server.Close)

	pointAt(server.URL+"/v1/introspect", adapterTimeout)
	return NewIntrospector(), &received
}

// pointAt configures the package through Init for the duration of one spec.
func pointAt(adapterEndpoint string, adapterTimeout time.Duration) {
	savedEndpoint, savedClientID, savedClientSecret, savedTimeout := endpoint, clientID, clientSecret, timeout
	DeferCleanup(func() {
		endpoint, clientID, clientSecret, timeout = savedEndpoint, savedClientID, savedClientSecret, savedTimeout
	})
	Init(adapterEndpoint, "bkauth", "secret", adapterTimeout)
}

func writeJSON(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

var _ = Describe("Introspector.Introspect", func() {
	ctx := context.Background()

	It("posts token and realm in the body, credentials in Basic auth", func() {
		client, received := serve(time.Second, writeJSON(http.StatusOK, `{"active": false}`))

		_, err := client.Introspect(ctx, "blueking", "demo_pat_xxx")

		Expect(err).NotTo(HaveOccurred())
		Expect(*received).To(HaveLen(1))
		req := (*received)[0]
		Expect(req.Method).To(Equal(http.MethodPost))
		Expect(req.URL.Path).To(Equal("/v1/introspect"))
		Expect(req.URL.RawQuery).To(BeEmpty())
		Expect(req.Header.Get("Content-Type")).To(Equal("application/x-www-form-urlencoded"))
		Expect(req.PostForm.Get("token")).To(Equal("demo_pat_xxx"))
		Expect(req.PostForm.Get("realm")).To(Equal("blueking"))
		Expect(req.PostForm.Has("client_secret")).To(BeFalse())
		user, pass, ok := req.BasicAuth()
		Expect(ok).To(BeTrue())
		Expect(user).To(Equal("bkauth"))
		Expect(pass).To(Equal("secret"))
	})

	It("decodes an active answer", func() {
		client, _ := serve(time.Second, writeJSON(http.StatusOK, `{
			"active": true,
			"client_id": "personal",
			"sub": "u1",
			"username": "alice",
			"tenant_id": "default",
			"aud": ["gateway:devops/api:*", "mcp:devops-prod-codecc"],
			"exp": 1819561386,
			"unknown_member": "ignored"
		}`))

		resp, err := client.Introspect(ctx, "blueking", "demo_pat_xxx")

		Expect(err).NotTo(HaveOccurred())
		Expect(resp).To(Equal(Response{
			Active:    true,
			ClientID:  "personal",
			Sub:       "u1",
			Username:  "alice",
			TenantID:  "default",
			Audience:  []string{"gateway:devops/api:*", "mcp:devops-prod-codecc"},
			ExpiresAt: 1819561386,
		}))
	})

	It("decodes an inactive answer", func() {
		client, _ := serve(time.Second, writeJSON(http.StatusOK, `{"active": false}`))

		resp, err := client.Introspect(ctx, "blueking", "demo_pat_xxx")

		Expect(err).NotTo(HaveOccurred())
		Expect(resp).To(Equal(Response{}))
	})

	// A 200 that is not a protocol answer is the adapter failing; reading it as
	// inactive would cache a lockout for the whole TTL.
	DescribeTable("fails on a 200 that is not a protocol answer",
		func(body string) {
			client, _ := serve(time.Second, writeJSON(http.StatusOK, body))

			_, err := client.Introspect(ctx, "blueking", "demo_pat_secret_value")

			Expect(err).To(MatchError(ErrInvalidResponse))
			Expect(err.Error()).NotTo(ContainSubstring("secret_value"))
			Expect(err.Error()).NotTo(ContainSubstring("127.0.0.1"))
		},
		Entry("active missing, error body", `{"error": "upstream down"}`),
		Entry("active missing, empty object", `{}`),
		Entry("active null", `{"active": null}`),
		Entry("malformed JSON", `{"active": tru`),
		Entry("aud of the wrong type", `{"active": true, "aud": "mcp:x"}`),
		// Cut at the cap, this body would decode as a valid inactive answer.
		Entry("body over the cap", `{"active": false}`+strings.Repeat(" ", maxResponseBodySize)),
	)

	It("accepts a body exactly at the cap", func() {
		answer := `{"active": false}`
		client, _ := serve(time.Second, writeJSON(http.StatusOK,
			answer+strings.Repeat(" ", maxResponseBodySize-len(answer))))

		resp, err := client.Introspect(ctx, "blueking", "demo_pat_xxx")

		Expect(err).NotTo(HaveOccurred())
		Expect(resp).To(Equal(Response{}))
	})

	DescribeTable("fails on a redirect without following it",
		func(status int) {
			var followed []*http.Request
			elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = r.ParseForm()
				followed = append(followed, r)
				writeJSON(http.StatusOK, `{"active": true}`)(w, r)
			}))
			DeferCleanup(elsewhere.Close)

			client, received := serve(time.Second, func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, elsewhere.URL+"/v1/introspect", status)
			})

			_, err := client.Introspect(ctx, "blueking", "demo_pat_xxx")

			Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("status %d", status))))
			Expect(*received).To(HaveLen(1))
			Expect(followed).To(BeEmpty())
		},
		Entry("307 would re-send the token to Location", http.StatusTemporaryRedirect),
		Entry("308 would re-send the token to Location", http.StatusPermanentRedirect),
		Entry("302 would turn into a bodiless GET", http.StatusFound),
	)

	// The error reaches introspect's 500 answer: it must say why, but neither
	// carry the token nor name the adapter's address.
	DescribeTable("fails when the adapter cannot answer",
		func(handler http.HandlerFunc, cause string) {
			client, _ := serve(50*time.Millisecond, handler)

			_, err := client.Introspect(ctx, "blueking", "demo_pat_secret_value")

			Expect(err).To(MatchError(ContainSubstring(cause)))
			Expect(err.Error()).NotTo(ContainSubstring("secret_value"))
			Expect(err.Error()).NotTo(ContainSubstring("127.0.0.1"))
		},
		Entry("5xx", writeJSON(http.StatusBadGateway, `{"error": "bad gateway"}`), "status 502"),
		Entry("4xx", writeJSON(http.StatusUnauthorized, `{"error": "invalid_client"}`), "status 401"),
		Entry("timeout", func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-time.After(time.Second):
			}
		}, "timed out"),
	)

	It("keeps the cause but not the address when the adapter is unreachable", func() {
		closed := httptest.NewServer(http.NotFoundHandler())
		hostPort := strings.TrimPrefix(closed.URL, "http://")
		closed.Close()
		pointAt(closed.URL+"/v1/introspect", time.Second)

		_, err := NewIntrospector().Introspect(ctx, "blueking", "demo_pat_xxx")

		Expect(err).To(MatchError(ContainSubstring("dial failed")))
		Expect(err.Error()).To(ContainSubstring("connection refused"))
		Expect(err.Error()).NotTo(ContainSubstring(hostPort))
		Expect(err.Error()).NotTo(ContainSubstring("127.0.0.1"))
	})
})
