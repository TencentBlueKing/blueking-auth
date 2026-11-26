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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// adapterAddr stands for the address a dial error would name.
var adapterAddr = &net.TCPAddr{IP: net.IPv4(10, 0, 0, 5), Port: 8080}

var _ = DescribeTable("describeTransportError keeps what failed but not where",
	func(inner error, want string) {
		err := &url.Error{Op: "Post", URL: "http://adapter.internal:8080/v1/introspect", Err: inner}

		got := describeTransportError(err)

		Expect(got).To(Equal(want))
		Expect(got).NotTo(ContainSubstring("adapter.internal"))
		Expect(got).NotTo(ContainSubstring(adapterAddr.IP.String()))
	},
	Entry("name resolution",
		&net.DNSError{Err: "no such host", Name: "adapter.internal"}, "name resolution failed: no such host"),
	Entry("dial",
		&net.OpError{Op: "dial", Net: "tcp", Addr: adapterAddr, Err: errors.New("connect: connection refused")},
		"dial failed: connect: connection refused"),
	// The shape a real handshake yields: the hostname error wrapped by tls,
	// whose message would quote both the cert's names and the dialled host.
	Entry("TLS host mismatch",
		&tls.CertificateVerificationError{Err: x509.HostnameError{
			Certificate: &x509.Certificate{DNSNames: []string{"other.example"}},
			Host:        "adapter.internal",
		}},
		"TLS verification failed: certificate does not match the adapter host"),
	Entry("timeout", context.DeadlineExceeded, "timed out"),
	Entry("anything else, with the URL stripped", errors.New("tls: handshake failure"), "tls: handshake failure"),
)
