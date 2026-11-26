/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
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
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// will error, no file available
	_, err := Load(viper.GetViper())
	assert.Error(t, err)
}

var _ = Describe("LoadFile", func() {
	It("loads the file without touching the global viper", func() {
		path := filepath.Join(GinkgoT().TempDir(), "config.yaml")
		content := "databases:\n  - {id: bkauth, host: db.example.com, name: bkauth}\n"
		assert.NoError(GinkgoT(), os.WriteFile(path, []byte(content), 0o600))

		cfg, err := LoadFile(path)

		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), "db.example.com", cfg.DatabaseMap["bkauth"].Host)
		assert.Empty(GinkgoT(), viper.ConfigFileUsed())
	})

	It("names the file it cannot read", func() {
		path := filepath.Join(GinkgoT().TempDir(), "missing.yaml")

		_, err := LoadFile(path)

		assert.ErrorContains(GinkgoT(), err, "read config file "+path)
	})
})
