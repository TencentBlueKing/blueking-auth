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

package config

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

func encodeKMSPrivateKey(key *rsa.PrivateKey) string {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	Expect(err).NotTo(HaveOccurred())
	return base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// sealKMSEnvelope builds an envelope in the bk-kms-sdk hybrid format (RSA-OAEP-SHA256
// wrapping a 16-byte AES key, AES-CBC with PKCS#7 padding), as the KMS service issues it.
func sealKMSEnvelope(pub *rsa.PublicKey, plaintext string) string {
	raw := make([]byte, 8)
	_, err := rand.Read(raw)
	Expect(err).NotTo(HaveOccurred())
	symmetricKey := hex.EncodeToString(raw)

	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, []byte(symmetricKey), nil)
	Expect(err).NotTo(HaveOccurred())

	block, err := aes.NewCipher([]byte(symmetricKey))
	Expect(err).NotTo(HaveOccurred())
	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := append([]byte(plaintext), bytes.Repeat([]byte{byte(padding)}, padding)...)
	iv := make([]byte, aes.BlockSize)
	_, err = rand.Read(iv)
	Expect(err).NotTo(HaveOccurred())
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	envelope, err := json.Marshal(map[string]string{
		"asymmetric_type": "RSA",
		"symmetric_type":  "AES",
		"symmetric_mode":  "CBC",
		"encrypted_key":   base64.StdEncoding.EncodeToString(encryptedKey),
		"ciphertext":      base64.StdEncoding.EncodeToString(append(iv, ciphertext...)),
	})
	Expect(err).NotTo(HaveOccurred())
	return base64.StdEncoding.EncodeToString(envelope)
}

func sealKMSCredentials(pub *rsa.PublicKey, credentials map[string]any) string {
	plaintext, err := json.Marshal(credentials)
	Expect(err).NotTo(HaveOccurred())
	return sealKMSEnvelope(pub, string(plaintext))
}

// writeKMSFile stores content in a temporary file and returns its path.
// name keeps two files written in the same spec from overwriting each other.
func writeKMSFile(name, content string) string {
	path := filepath.Join(GinkgoT().TempDir(), name)
	Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())
	return path
}

func newKMSTestConfig() *Config {
	return &Config{
		AppSecret: "file-app-secret",
		Crypto:    Crypto{Key: "file-crypto-key", Nonce: "fileNonce123"},
		Databases: []Database{{ID: "bkauth", Host: "db.example.com", User: "file-user", Password: "file-password"}},
		Redis:     []Redis{{ID: "standalone", Addr: "redis.example.com:6379", Password: "file-redis-password"}},
	}
}

func newKMSTestCredentials() map[string]any {
	return map[string]any{
		"bkapp_id_secret": map[string]any{
			"default": map[string]any{"app_code": "bk_auth", "app_secret": "kms-app-secret"},
		},
		"encryption": map[string]any{"cryptoKey": "kmsCryptoKey0123456789abcdefghij"},
		"mysql": map[string]any{
			"bkauth": map[string]any{"username": "kms-user", "password": " kms $db password "},
		},
		"redis": map[string]any{
			"default": map[string]any{"password": "kms-redis-password", "sentinel_password": "kms-sentinel-password"},
		},
		// Credentials of other components sharing the envelope must be ignored.
		"etcd": map[string]any{"default": map[string]any{"username": "etcd-user", "password": "etcd-password"}},
	}
}

// nestedKMSObject returns the object reached by following keys from credentials.
func nestedKMSObject(credentials map[string]any, keys ...string) map[string]any {
	object := credentials
	for _, key := range keys {
		object = object[key].(map[string]any)
	}
	return object
}

var _ = Describe("KMS", Ordered, func() {
	var key, otherKey *rsa.PrivateKey

	BeforeAll(func() {
		var err error
		key, err = rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).NotTo(HaveOccurred())
		otherKey, err = rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).NotTo(HaveOccurred())
	})

	// enableKMS turns KMS on for cfg, taking the private key and envelope path from the config itself.
	enableKMS := func(cfg *Config, envelope string) {
		cfg.KMS = KMSConfig{
			Enabled:        true,
			PrivateKeyFile: writeKMSFile("private-key", encodeKMSPrivateKey(key)),
			EnvelopeFile:   writeKMSFile("envelope", envelope),
		}
	}

	Describe("applyKMS", func() {
		It("leaves the config untouched when KMS is disabled", func() {
			cfg := newKMSTestConfig()
			cfg.KMS.EnvelopeFile = "/not/exists/bkauth-kms"

			Expect(cfg.applyKMS()).To(Succeed())
			Expect(cfg.AppSecret).To(Equal("file-app-secret"))
			Expect(cfg.Databases[0].Password).To(Equal("file-password"))
		})

		It("overrides the credentials with the decrypted values verbatim", func() {
			cfg := newKMSTestConfig()
			enableKMS(cfg, sealKMSCredentials(&key.PublicKey, newKMSTestCredentials()))

			Expect(cfg.applyKMS()).To(Succeed())

			Expect(cfg.AppSecret).To(Equal("kms-app-secret"))
			Expect(cfg.Crypto.Key).To(Equal("kmsCryptoKey0123456789abcdefghij"))
			Expect(cfg.Databases[0].User).To(Equal("kms-user"))
			Expect(cfg.Databases[0].Password).To(Equal(" kms $db password "))
			Expect(cfg.Redis[0].Password).To(Equal("kms-redis-password"))
			// Non-credential settings still come from the config file.
			Expect(cfg.Crypto.Nonce).To(Equal("fileNonce123"))
			Expect(cfg.Databases[0].Host).To(Equal("db.example.com"))
			Expect(cfg.Redis[0].Addr).To(Equal("redis.example.com:6379"))
			Expect(cfg.Redis[0].SentinelPassword).To(BeEmpty())
		})

		It("binds the sentinel password for a sentinel redis", func() {
			cfg := newKMSTestConfig()
			enableKMS(cfg, sealKMSCredentials(&key.PublicKey, newKMSTestCredentials()))
			cfg.Redis = []Redis{{ID: "sentinel", SentinelAddr: "sentinel.example.com:26379", MasterName: "mymaster"}}

			Expect(cfg.applyKMS()).To(Succeed())

			Expect(cfg.Redis[0].Password).To(Equal("kms-redis-password"))
			Expect(cfg.Redis[0].SentinelPassword).To(Equal("kms-sentinel-password"))
		})

		It("requires the sentinel password for a sentinel redis", func() {
			credentials := newKMSTestCredentials()
			delete(nestedKMSObject(credentials, "redis", "default"), "sentinel_password")
			cfg := newKMSTestConfig()
			cfg.Redis = []Redis{{ID: "sentinel", SentinelPassword: "file-sentinel-password"}}
			enableKMS(cfg, sealKMSCredentials(&key.PublicKey, credentials))

			err := cfg.applyKMS()

			Expect(err).To(MatchError(ContainSubstring("redis.default.sentinel_password")))
		})

		It("does not modify the environment", func() {
			cfg := newKMSTestConfig()
			enableKMS(cfg, sealKMSCredentials(&key.PublicKey, newKMSTestCredentials()))
			before := os.Environ()

			Expect(cfg.applyKMS()).To(Succeed())

			Expect(os.Environ()).To(Equal(before))
		})

		DescribeTable("rejects a missing or invalid credential without falling back to the config file",
			func(mutate func(credentials map[string]any), path string) {
				credentials := newKMSTestCredentials()
				mutate(credentials)
				cfg := newKMSTestConfig()
				enableKMS(cfg, sealKMSCredentials(&key.PublicKey, credentials))

				err := cfg.applyKMS()

				Expect(err).To(MatchError("KMS envelope requires a non-empty string at " + path))
			},
			Entry("missing database password",
				func(c map[string]any) { delete(nestedKMSObject(c, "mysql", "bkauth"), "password") },
				"mysql.bkauth.password"),
			Entry("missing database section for the configured id",
				func(c map[string]any) { c["mysql"] = map[string]any{"other": map[string]any{}} },
				"mysql.bkauth.username"),
			Entry("non-string app secret",
				func(c map[string]any) { nestedKMSObject(c, "bkapp_id_secret", "default")["app_secret"] = 42 },
				"bkapp_id_secret.default.app_secret"),
			Entry("null crypto key",
				func(c map[string]any) { nestedKMSObject(c, "encryption")["cryptoKey"] = nil },
				"encryption.cryptoKey"),
			Entry("whitespace-only redis password",
				func(c map[string]any) { nestedKMSObject(c, "redis", "default")["password"] = "  \t" },
				"redis.default.password"),
			Entry("section of the wrong type",
				func(c map[string]any) { c["redis"] = "not-an-object" },
				"redis.default.password"),
		)

		It("fails when the private key file is empty", func() {
			cfg := newKMSTestConfig()
			enableKMS(cfg, sealKMSCredentials(&key.PublicKey, newKMSTestCredentials()))
			cfg.KMS.PrivateKeyFile = writeKMSFile("empty-private-key", " \n")

			Expect(cfg.applyKMS()).To(MatchError(ContainSubstring("KMS private key " + cfg.KMS.PrivateKeyFile + " is empty")))
		})

		It("fails when the private key file is missing", func() {
			cfg := newKMSTestConfig()
			enableKMS(cfg, sealKMSCredentials(&key.PublicKey, newKMSTestCredentials()))
			cfg.KMS.PrivateKeyFile = filepath.Join(GinkgoT().TempDir(), "missing")

			Expect(cfg.applyKMS()).To(MatchError(ContainSubstring("read KMS private key")))
		})

		It("fails when the envelope file is missing", func() {
			cfg := newKMSTestConfig()
			cfg.KMS = KMSConfig{
				Enabled:        true,
				PrivateKeyFile: writeKMSFile("private-key", encodeKMSPrivateKey(key)),
				EnvelopeFile:   filepath.Join(GinkgoT().TempDir(), "missing"),
			}

			Expect(cfg.applyKMS()).To(MatchError(ContainSubstring("read KMS envelope")))
		})

		It("uses the default file paths when none are configured", func() {
			cfg := newKMSTestConfig()
			cfg.KMS.Enabled = true

			Expect(cfg.applyKMS()).To(MatchError(ContainSubstring(defaultKMSPrivateKeyFile)))

			cfg.KMS.PrivateKeyFile = writeKMSFile("private-key", encodeKMSPrivateKey(key))
			Expect(cfg.applyKMS()).To(MatchError(ContainSubstring(defaultKMSEnvelopeFile)))
		})

		It("tolerates surrounding whitespace in the private key file", func() {
			cfg := newKMSTestConfig()
			enableKMS(cfg, sealKMSCredentials(&key.PublicKey, newKMSTestCredentials()))
			cfg.KMS.PrivateKeyFile = writeKMSFile("padded-private-key", "\n"+encodeKMSPrivateKey(key)+"\n")

			Expect(cfg.applyKMS()).To(Succeed())
		})

		It("fails to decrypt with a different private key", func() {
			cfg := newKMSTestConfig()
			enableKMS(cfg, sealKMSCredentials(&otherKey.PublicKey, newKMSTestCredentials()))

			Expect(cfg.applyKMS()).To(MatchError(ContainSubstring("decrypt KMS envelope")))
		})

		It("tolerates surrounding whitespace in the envelope file", func() {
			cfg := newKMSTestConfig()
			enableKMS(cfg, "\n"+sealKMSCredentials(&key.PublicKey, newKMSTestCredentials())+"\n")

			Expect(cfg.applyKMS()).To(Succeed())
		})

		DescribeTable("rejects a plaintext that is not a JSON object without echoing it",
			func(plaintext string) {
				cfg := newKMSTestConfig()
				enableKMS(cfg, sealKMSEnvelope(&key.PublicKey, plaintext))

				err := cfg.applyKMS()

				Expect(err).To(MatchError("KMS envelope plaintext must be a JSON object"))
			},
			Entry("array", `["secret"]`),
			Entry("string", `"secret"`),
			Entry("null", `null`),
			Entry("malformed", `{"secret": `),
			Entry("invalid utf-8", "{\"k\": \"\xff\"}"),
		)
	})

	Describe("Load", func() {
		It("applies KMS before building DatabaseMap and RedisMap", func() {
			privateKeyFile := writeKMSFile("private-key", encodeKMSPrivateKey(key))
			envelopeFile := writeKMSFile("envelope", sealKMSCredentials(&key.PublicKey, newKMSTestCredentials()))
			v := viper.New()
			v.SetConfigType("yaml")
			Expect(v.ReadConfig(strings.NewReader(fmt.Sprintf(`
kms:
  enabled: true
  privateKeyFile: %q
  envelopeFile: %q
appSecret: "file-app-secret"
crypto: {key: "file-crypto-key", nonce: "fileNonce123"}
databases:
  - {id: "bkauth", host: "db.example.com", user: "file-user", password: "file-password"}
redis:
  - {id: "standalone", addr: "redis.example.com:6379", password: "file-redis-password"}
`, privateKeyFile, envelopeFile)))).To(Succeed())

			cfg, err := Load(v)

			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.DatabaseMap["bkauth"].User).To(Equal("kms-user"))
			Expect(cfg.DatabaseMap["bkauth"].Password).To(Equal(" kms $db password "))
			Expect(cfg.RedisMap["standalone"].Password).To(Equal("kms-redis-password"))
			Expect(cfg.AppSecret).To(Equal("kms-app-secret"))
		})
	})
})
