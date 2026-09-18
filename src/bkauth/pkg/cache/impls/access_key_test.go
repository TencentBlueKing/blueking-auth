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

package impls

import (
	"context"
	"errors"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/app"
	"bkauth/pkg/cache"
	"bkauth/pkg/cache/redis"
	"bkauth/pkg/cryptography"
	"bkauth/pkg/service/mock"
	"bkauth/pkg/service/types"
)

type deterministicCrypto struct{}

func (deterministicCrypto) Encrypt(plaintext []byte) ([]byte, error) {
	return []byte("enc:" + string(plaintext)), nil
}

func (deterministicCrypto) Decrypt(encryptedText []byte) ([]byte, error) {
	if !strings.HasPrefix(string(encryptedText), "enc:") {
		return nil, errors.New("invalid encrypted text")
	}
	return []byte(strings.TrimPrefix(string(encryptedText), "enc:")), nil
}

func (deterministicCrypto) EncryptToBase64(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (deterministicCrypto) DecryptFromBase64(encryptedTextB64 string) (string, error) {
	if !strings.HasPrefix(encryptedTextB64, "enc:") {
		return "", errors.New("invalid encrypted text")
	}
	return strings.TrimPrefix(encryptedTextB64, "enc:"), nil
}

func useDeterministicCrypto() func() {
	old := cryptography.AppSecretCrypto
	cryptography.AppSecretCrypto = deterministicCrypto{}
	return func() { cryptography.AppSecretCrypto = old }
}

var _ = Describe("AccessKeysCache", func() {
	BeforeEach(func() {
		expiration := 5 * time.Minute
		cli := newTestRedisClient()
		mockCache := redis.NewMockCache(cli, "mockCache", expiration)

		AccessKeysCache = mockCache
	})

	It("Key", func() {
		key := AccessKeysKey{
			AppCode: "test",
		}
		assert.Equal(GinkgoT(), key.Key(), "test")
	})

	Context("VerifyAccessKey", func() {
		var ctl *gomock.Controller
		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})
		AfterEach(func() {
			ctl.Finish()
		})

		// useMockRetrieve routes the cache miss path through mockService, mirroring
		// what the production retrieveAccessKeys does.
		useMockRetrieve := func(mockService *mock.MockAccessKeyService) func() {
			orig := retrieveAccessKeys
			retrieveAccessKeys = func(ctx context.Context, key cache.Key) (interface{}, error) {
				k := key.(AccessKeysKey)
				return mockService.ListEncryptedAccessKeyByAppCode(ctx, k.AppCode)
			}
			return func() { retrieveAccessKeys = orig }
		}

		It("AccessKeysCache Get ok", func() {
			restoreCrypto := useDeterministicCrypto()
			defer restoreCrypto()

			enc1, err := app.EncryptSecret("secret1")
			assert.NoError(GinkgoT(), err)
			enc2, err := app.EncryptSecret("secret2")
			assert.NoError(GinkgoT(), err)

			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().
				ListEncryptedAccessKeyByAppCode(gomock.Any(), "test").
				Return([]types.EncryptedAccessKey{
					{AppSecret: enc1, Enabled: true},
					{AppSecret: enc2, Enabled: true},
				}, nil).
				AnyTimes()

			defer useMockRetrieve(mockService)()

			exists, err := VerifyAccessKey(context.Background(), "test", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, true)

			exists, err = VerifyAccessKey(context.Background(), "test", "secret2")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, true)

			exists, err = VerifyAccessKey(context.Background(), "test", "secret3")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)
		})

		It("AccessKeysCache Get fail", func() {
			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().
				ListEncryptedAccessKeyByAppCode(gomock.Any(), "test").
				Return(nil, errors.New("error")).
				AnyTimes()

			defer useMockRetrieve(mockService)()

			exists, err := VerifyAccessKey(context.Background(), "test", "secret1")
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)

			exists, err = VerifyAccessKey(context.Background(), "test", "secret2")
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)
		})

		It("AccessKeysCache Get empty secret", func() {
			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().
				ListEncryptedAccessKeyByAppCode(gomock.Any(), "test").
				Return([]types.EncryptedAccessKey{}, nil).
				AnyTimes()

			defer useMockRetrieve(mockService)()

			exists, err := VerifyAccessKey(context.Background(), "test", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)

			exists, err = VerifyAccessKey(context.Background(), "test", "secret2")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)
		})

		It("AccessKeysCache Get disable secret", func() {
			restoreCrypto := useDeterministicCrypto()
			defer restoreCrypto()

			enc1, err := app.EncryptSecret("secret1")
			assert.NoError(GinkgoT(), err)
			enc2, err := app.EncryptSecret("secret2")
			assert.NoError(GinkgoT(), err)

			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().
				ListEncryptedAccessKeyByAppCode(gomock.Any(), "test").
				Return([]types.EncryptedAccessKey{
					{AppSecret: enc1, Enabled: false},
					{AppSecret: enc2, Enabled: true},
				}, nil).
				AnyTimes()

			defer useMockRetrieve(mockService)()

			exists, err := VerifyAccessKey(context.Background(), "test", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)

			exists, err = VerifyAccessKey(context.Background(), "test", "secret2")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, true)
		})

		It("AccessKeysCache Get skips undecryptable secret", func() {
			// One unreadable row must not lock the app out of its remaining keys.
			restoreCrypto := useDeterministicCrypto()
			defer restoreCrypto()

			enc2, err := app.EncryptSecret("secret2")
			assert.NoError(GinkgoT(), err)

			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().
				ListEncryptedAccessKeyByAppCode(gomock.Any(), "test").
				Return([]types.EncryptedAccessKey{
					{AppSecret: "corrupted", Enabled: true},
					{AppSecret: enc2, Enabled: true},
				}, nil).
				AnyTimes()

			defer useMockRetrieve(mockService)()

			exists, err := VerifyAccessKey(context.Background(), "test", "secret2")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, true)

			exists, err = VerifyAccessKey(context.Background(), "test", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)
		})

		It("AccessKeysCache Get all secrets undecryptable", func() {
			restoreCrypto := useDeterministicCrypto()
			defer restoreCrypto()

			mockService := mock.NewMockAccessKeyService(ctl)
			mockService.EXPECT().
				ListEncryptedAccessKeyByAppCode(gomock.Any(), "test").
				Return([]types.EncryptedAccessKey{
					{AppSecret: "corrupted", Enabled: true},
				}, nil).
				AnyTimes()

			defer useMockRetrieve(mockService)()

			exists, err := VerifyAccessKey(context.Background(), "test", "secret1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), exists, false)
		})
	})

	It("DeleteAccessKey", func() {
		err := DeleteAccessKey(context.Background(), "test")
		assert.NoError(GinkgoT(), err)
	})
})
