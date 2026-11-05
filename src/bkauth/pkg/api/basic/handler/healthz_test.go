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
	"os"
	"path/filepath"
	"time"

	"github.com/alicebob/miniredis"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bkauth/pkg/config"
	pkgredis "bkauth/pkg/redis"
)

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

var _ = Describe("checkRedisInstances", func() {
	var mr *miniredis.Miniredis

	BeforeEach(func() {
		var err error
		mr, err = miniredis.Run()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		mr.Close()
	})

	It("passes when no redis is configured", func() {
		Expect(checkRedisInstances(map[string]config.Redis{})).To(Succeed())
	})

	It("passes for a reachable standalone redis without TLS", func() {
		redisMap := map[string]config.Redis{
			pkgredis.ModeStandalone: {ID: pkgredis.ModeStandalone, Addr: mr.Addr()},
		}

		Expect(checkRedisInstances(redisMap)).To(Succeed())
	})

	It("negotiates TLS when TLS is enabled", func() {
		// miniredis speaks plaintext only, so a successful ping here would mean
		// the check ignored the TLS settings.
		redisMap := map[string]config.Redis{
			pkgredis.ModeStandalone: {
				ID:   pkgredis.ModeStandalone,
				Addr: mr.Addr(),
				TLS:  config.TLS{Enabled: true, CertCaFile: writeTestCA(GinkgoT().TempDir())},
			},
		}

		err := checkRedisInstances(redisMap)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("redis(mode=standalone) connect fail"))
		Expect(err.Error()).To(ContainSubstring("[addr=" + mr.Addr() + "]"))
	})

	It("returns an error instead of exiting when the TLS files are invalid", func() {
		redisMap := map[string]config.Redis{
			pkgredis.ModeStandalone: {
				ID:   pkgredis.ModeStandalone,
				Addr: mr.Addr(),
				TLS:  config.TLS{Enabled: true, CertCaFile: "/not/exists/ca.pem"},
			},
		}

		err := checkRedisInstances(redisMap)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("redis tls config init"))
	})

	It("reports the failing mode even when another mode is configured after it", func() {
		redisMap := map[string]config.Redis{
			pkgredis.ModeStandalone: {
				ID:          pkgredis.ModeStandalone,
				Addr:        "127.0.0.1:1",
				DialTimeout: 1,
			},
			pkgredis.ModeSentinel: {
				ID:           pkgredis.ModeSentinel,
				SentinelAddr: mr.Addr(),
				MasterName:   "mymaster",
			},
		}

		err := checkRedisInstances(redisMap)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("redis(mode=standalone) connect fail"))
		Expect(err.Error()).To(ContainSubstring("[addr=127.0.0.1:1]"))
	})
})
