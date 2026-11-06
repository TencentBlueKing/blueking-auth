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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/TencentBlueKing/bk-kms-sdk/go/decrypt"
)

const (
	// defaultKMSEnvelopeFile is where the Helm chart mounts the envelope.
	defaultKMSEnvelopeFile = "/etc/secrets/bkauth-kms"

	// Must match redis.ModeSentinel; pkg/redis imports this package, so it cannot be referenced here.
	redisSentinelID = "sentinel"
)

// kmsBinding copies the string found at path in the decrypted credentials into target.
type kmsBinding struct {
	target *string
	path   []string
}

// applyKMS overrides the credentials in cfg with the ones decrypted from the KMS envelope.
// Everything it needs comes from cfg.KMS, which is read from the config file.
// Once KMS is enabled every bound credential must be present: there is no fallback to the
// values in the config file, so a stale password left there can never be used by accident.
// Only cfg in memory is changed; neither the environment nor the config file is touched.
func (cfg *Config) applyKMS() error {
	if !cfg.KMS.Enabled {
		return nil
	}

	credentials, err := cfg.loadKMSCredentials()
	if err != nil {
		return err
	}
	return bindKMSCredentials(credentials, cfg.kmsBindings())
}

func (cfg *Config) kmsBindings() []kmsBinding {
	bindings := []kmsBinding{
		{target: &cfg.AppSecret, path: []string{"bkapp_id_secret", "default", "app_secret"}},
		{target: &cfg.Crypto.Key, path: []string{"encryption", "cryptoKey"}},
	}

	for i := range cfg.Databases {
		db := &cfg.Databases[i]
		bindings = append(bindings,
			kmsBinding{target: &db.User, path: []string{"mysql", db.ID, "username"}},
			kmsBinding{target: &db.Password, path: []string{"mysql", db.ID, "password"}},
		)
	}

	for i := range cfg.Redis {
		rds := &cfg.Redis[i]
		bindings = append(bindings, kmsBinding{target: &rds.Password, path: []string{"redis", "default", "password"}})
		if rds.ID == redisSentinelID {
			bindings = append(bindings, kmsBinding{
				target: &rds.SentinelPassword,
				path:   []string{"redis", "default", "sentinel_password"},
			})
		}
	}

	return bindings
}

// loadKMSCredentials decrypts the envelope into its JSON object. Errors name the failing
// stage only and never carry credential material.
func (cfg *Config) loadKMSCredentials() (map[string]any, error) {
	privateKey := strings.TrimSpace(cfg.KMS.PrivateKey)
	if privateKey == "" {
		return nil, errors.New("KMS is enabled but kms.privateKey is empty")
	}

	envelopeFile := cfg.KMS.EnvelopeFile
	if strings.TrimSpace(envelopeFile) == "" {
		envelopeFile = defaultKMSEnvelopeFile
	}
	envelope, err := os.ReadFile(envelopeFile)
	if err != nil {
		return nil, fmt.Errorf("read KMS envelope %s: %w", envelopeFile, err)
	}

	plaintext, err := decrypt.Decrypt(strings.TrimSpace(string(envelope)), privateKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt KMS envelope %s: %w", envelopeFile, err)
	}

	// The JSON error is dropped on purpose: its message quotes bytes of the plaintext.
	var credentials map[string]any
	if !utf8.ValidString(plaintext) || json.Unmarshal([]byte(plaintext), &credentials) != nil || credentials == nil {
		return nil, errors.New("KMS envelope plaintext must be a JSON object")
	}
	return credentials, nil
}

func bindKMSCredentials(credentials map[string]any, bindings []kmsBinding) error {
	for _, binding := range bindings {
		var value any = credentials
		for _, key := range binding.path {
			object, _ := value.(map[string]any)
			value = object[key]
		}

		// The value is kept verbatim; only an all-whitespace value is rejected.
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return fmt.Errorf("KMS envelope requires a non-empty string at %s", strings.Join(binding.path, "."))
		}
		*binding.target = text
	}
	return nil
}
