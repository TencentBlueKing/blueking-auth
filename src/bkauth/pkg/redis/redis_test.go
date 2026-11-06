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

package redis

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"bkauth/pkg/config"
)

// writeTestCA writes a self-signed CA certificate to t.TempDir() and returns its path.
func writeTestCA(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "redis-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	assert.NoError(t, err)

	path := filepath.Join(t.TempDir(), "ca.pem")
	assert.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	return path
}

func TestNewTLSConfig(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		tlsConfig, err := NewTLSConfig(&config.Redis{})

		assert.NoError(t, err)
		assert.Nil(t, tlsConfig)
	})

	t.Run("enabled", func(t *testing.T) {
		cfg := &config.Redis{TLS: config.TLS{Enabled: true, CertCaFile: writeTestCA(t)}}

		tlsConfig, err := NewTLSConfig(cfg)

		assert.NoError(t, err)
		if assert.NotNil(t, tlsConfig) {
			assert.NotNil(t, tlsConfig.RootCAs)
			assert.False(t, tlsConfig.InsecureSkipVerify)
		}
	})

	t.Run("missing ca file", func(t *testing.T) {
		cfg := &config.Redis{TLS: config.TLS{Enabled: true, CertCaFile: "/not/exists/ca.pem"}}

		tlsConfig, err := NewTLSConfig(cfg)

		assert.ErrorContains(t, err, "redis tls config init")
		assert.Nil(t, tlsConfig)
	})
}
