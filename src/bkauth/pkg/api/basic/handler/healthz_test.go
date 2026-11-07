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

package handler

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/alicebob/miniredis/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bkauth/pkg/config"
)

const testMasterName = "mymaster"

// writeTestCA writes a self-signed CA certificate to dir and returns its path.
func writeTestCA(dir string) string {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "healthz-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	Expect(err).NotTo(HaveOccurred())

	path := filepath.Join(dir, "ca.pem")
	Expect(os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600)).To(Succeed())
	return path
}

// startFakeSentinel serves just enough of the sentinel protocol for a go-redis
// failover client: master discovery points at masterAddr, sentinel discovery
// finds no peers, and the +switch-master subscription is acknowledged.
func startFakeSentinel(masterAddr string) *server.Server {
	host, port, err := net.SplitHostPort(masterAddr)
	Expect(err).NotTo(HaveOccurred())

	s, err := server.NewServer("127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(s.Close)

	Expect(s.Register("SENTINEL", func(c *server.Peer, _ string, args []string) {
		switch {
		case len(args) == 2 && strings.EqualFold(args[0], "get-master-addr-by-name") && args[1] == testMasterName:
			c.WriteLen(2)
			c.WriteBulk(host)
			c.WriteBulk(port)
		case len(args) == 2 && strings.EqualFold(args[0], "sentinels"):
			c.WriteLen(0)
		default:
			c.WriteError("ERR unsupported sentinel command")
		}
	})).To(Succeed())

	Expect(s.Register("SUBSCRIBE", func(c *server.Peer, _ string, channels []string) {
		for i, channel := range channels {
			c.WriteLen(3)
			c.WriteBulk("subscribe")
			c.WriteBulk(channel)
			c.WriteInt(i + 1)
		}
	})).To(Succeed())

	return s
}

var _ = Describe("checkRedisInstances", func() {
	var mr *miniredis.Miniredis

	BeforeEach(func() {
		var err error
		mr, err = miniredis.Run()
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(mr.Close)
	})

	It("passes when no redis is configured", func() {
		Expect(checkRedisInstances(nil)).To(Succeed())
	})

	Context("standalone", func() {
		It("passes for a reachable redis without TLS", func() {
			redisConfigs := []config.Redis{{ID: "default", Addr: mr.Addr()}}

			Expect(checkRedisInstances(redisConfigs)).To(Succeed())
		})

		It("negotiates TLS when TLS is enabled", func() {
			// miniredis speaks plaintext only, so a successful ping here would mean
			// the check ignored the TLS settings.
			redisConfigs := []config.Redis{{
				ID:   "default",
				Addr: mr.Addr(),
				TLS:  config.TLS{Enabled: true, CertCaFile: writeTestCA(GinkgoT().TempDir())},
			}}

			err := checkRedisInstances(redisConfigs)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("redis connect fail"))
			Expect(err.Error()).To(ContainSubstring("[id=default sentinelEnabled=false addr=" + mr.Addr() + "]"))
		})

		It("returns an error instead of exiting when the TLS files are invalid", func() {
			redisConfigs := []config.Redis{{
				ID:   "default",
				Addr: mr.Addr(),
				TLS:  config.TLS{Enabled: true, CertCaFile: "/not/exists/ca.pem"},
			}}

			err := checkRedisInstances(redisConfigs)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("redis tls config init"))
		})
	})

	Context("sentinel", func() {
		var sentinelAddr string

		BeforeEach(func() {
			sentinelAddr = startFakeSentinel(mr.Addr()).Addr().String()
		})

		It("passes for a reachable sentinel without TLS", func() {
			redisConfigs := []config.Redis{{
				ID:              "default",
				SentinelEnabled: true,
				SentinelAddr:    sentinelAddr,
				MasterName:      testMasterName,
			}}

			Expect(checkRedisInstances(redisConfigs)).To(Succeed())
		})

		It("negotiates TLS when TLS is enabled", func() {
			// The same plaintext sentinel passes above, so the failure here can
			// only come from the TLS handshake.
			redisConfigs := []config.Redis{{
				ID:              "default",
				SentinelEnabled: true,
				SentinelAddr:    sentinelAddr,
				MasterName:      testMasterName,
				TLS:             config.TLS{Enabled: true, CertCaFile: writeTestCA(GinkgoT().TempDir())},
			}}

			err := checkRedisInstances(redisConfigs)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("redis connect fail"))
			Expect(err.Error()).To(ContainSubstring("[id=default sentinelEnabled=true addr=" + sentinelAddr + "]"))
		})

		It("uses the sentinel even when addr is also set", func() {
			// addr points nowhere, so passing proves the check followed SentinelEnabled.
			redisConfigs := []config.Redis{{
				ID:              "default",
				SentinelEnabled: true,
				Addr:            "127.0.0.1:1",
				SentinelAddr:    sentinelAddr,
				MasterName:      testMasterName,
			}}

			Expect(checkRedisInstances(redisConfigs)).To(Succeed())
		})

		It("still reports a failing redis listed after a healthy one", func() {
			redisConfigs := []config.Redis{
				{
					ID:              "default",
					SentinelEnabled: true,
					SentinelAddr:    sentinelAddr,
					MasterName:      testMasterName,
				},
				{ID: "cache", Addr: "127.0.0.1:1"},
			}

			err := checkRedisInstances(redisConfigs)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("redis connect fail"))
			Expect(err.Error()).To(ContainSubstring("[id=cache sentinelEnabled=false addr=127.0.0.1:1]"))
		})
	})
})
