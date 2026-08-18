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
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// serveInner points the package at an httptest server for the duration of one
// spec. handler receives every request the code under test makes.
func serveInner(handler http.HandlerFunc) *[]*http.Request {
	var received []*http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		received = append(received, r)
		handler(w, r)
	}))

	oldBaseURL, oldCredentials, oldClient := baseURL, authCredentials, defaultHTTPClient
	baseURL = server.URL
	authCredentials = `{"bk_app_code":"app","bk_app_secret":"secret"}`
	defaultHTTPClient = server.Client()

	DeferCleanup(func() {
		baseURL, authCredentials, defaultHTTPClient = oldBaseURL, oldCredentials, oldClient
		server.Close()
	})

	return &received
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

var _ = Describe("innerGet", func() {
	It("should send the auth and tenant headers", func() {
		received := serveInner(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, `{"data": []}`)
		})

		var out []Gateway
		err := innerGet(context.Background(), "tencent", "api/v2/inner/gateways/-/lookup/", url.Values{}, &out)
		Expect(err).NotTo(HaveOccurred())

		Expect(*received).To(HaveLen(1))
		req := (*received)[0]
		Expect(req.Header.Get("X-Bkapi-Authorization")).To(ContainSubstring("bk_app_code"))
		Expect(req.Header.Get("X-Bk-Tenant-Id")).To(Equal("tencent"))
		Expect(req.URL.Path).To(Equal("/api/v2/inner/gateways/-/lookup/"))
	})

	It("should omit the tenant header when no tenant is given", func() {
		received := serveInner(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, `{"data": []}`)
		})

		var out []Gateway
		Expect(innerGet(context.Background(), "", "api/v2/inner/gateways/-/lookup/", url.Values{}, &out)).To(Succeed())
		Expect((*received)[0].Header.Get("X-Bk-Tenant-Id")).To(BeEmpty())
	})

	It("should fail without calling out when not initialized", func() {
		var out []Gateway
		err := innerGet(context.Background(), "", "api/v2/inner/gateways/-/lookup/", url.Values{}, &out)
		Expect(errors.Is(err, ErrNotInitialized)).To(BeTrue())
	})

	It("should turn a 404 into a recognizable APIError", func() {
		serveInner(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusNotFound, `{"error": {"code": "NOT_FOUND", "message": "gateway not found"}}`)
		})

		var out []Gateway
		err := innerGet(context.Background(), "", "api/v2/inner/gateways/x/", url.Values{}, &out)
		Expect(err).To(HaveOccurred())
		Expect(IsNotFound(err)).To(BeTrue())

		var apiErr *APIError
		Expect(errors.As(err, &apiErr)).To(BeTrue())
		Expect(apiErr.Code).To(Equal("NOT_FOUND"))
		Expect(apiErr.Message).To(Equal("gateway not found"))
	})

	It("should report a non-404 failure without claiming it is a 404", func() {
		serveInner(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusInternalServerError, `{"error": {"code": "INTERNAL", "message": "boom"}}`)
		})

		var out []Gateway
		err := innerGet(context.Background(), "", "api/v2/inner/gateways/-/lookup/", url.Values{}, &out)
		Expect(err).To(HaveOccurred())
		Expect(IsNotFound(err)).To(BeFalse())
	})

	It("should still fail on an error status whose body is not the expected envelope", func() {
		serveInner(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusBadGateway, `<html>nginx</html>`)
		})

		var out []Gateway
		err := innerGet(context.Background(), "", "api/v2/inner/gateways/-/lookup/", url.Values{}, &out)

		var apiErr *APIError
		Expect(errors.As(err, &apiErr)).To(BeTrue())
		Expect(apiErr.StatusCode).To(Equal(http.StatusBadGateway))
	})

	It("should leave out untouched when data is absent", func() {
		serveInner(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, `{}`)
		})

		out := []Gateway{{Name: "kept"}}
		Expect(innerGet(context.Background(), "", "api/v2/inner/gateways/-/lookup/", url.Values{}, &out)).To(Succeed())
		Expect(out).To(HaveLen(1))
	})
})

var _ = Describe("chunkNames", func() {
	It("should drop blanks and duplicates", func() {
		Expect(chunkNames([]string{"a", "", "b", "a", ""})).To(Equal([][]string{{"a", "b"}}))
	})

	It("should return nothing for an empty input", func() {
		Expect(chunkNames(nil)).To(BeEmpty())
		Expect(chunkNames([]string{"", ""})).To(BeEmpty())
	})

	It("should not split at the upstream limit", func() {
		names := make([]string, namesPerRequest)
		for i := range names {
			names[i] = strconv.Itoa(i)
		}
		Expect(chunkNames(names)).To(HaveLen(1))
	})

	It("should split one past the upstream limit", func() {
		names := make([]string, namesPerRequest+1)
		for i := range names {
			names[i] = strconv.Itoa(i)
		}
		chunks := chunkNames(names)
		Expect(chunks).To(HaveLen(2))
		Expect(chunks[0]).To(HaveLen(namesPerRequest))
		Expect(chunks[1]).To(HaveLen(1))
	})
})
