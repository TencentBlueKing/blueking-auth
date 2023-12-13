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

package memory

import (
	"errors"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/singleflight"

	"bkauth/pkg/cache"
	"bkauth/pkg/cache/memory/backend"
)

var _ = Describe("Base Cache", func() {
	var be *backend.MemoryBackend
	BeforeEach(func() {
		expiration := 5 * time.Minute
		be = backend.NewMemoryBackend("test", expiration, nil)
	})

	Context("retrieve OK", func() {
		var retrieveTest RetrieveFunc
		var c Cache
		BeforeEach(func() {
			retrieveTest = func(k cache.Key) (interface{}, error) {
				kStr := k.Key()
				switch kStr {
				case "a":
					return "1", nil
				case "b":
					return "2", nil
				case "error":
					return nil, errors.New("error")
				case "bool":
					return true, nil
				case "time":
					return time.Time{}, nil
				default:
					return "", nil
				}
			}
			c = NewBaseCache(false, retrieveTest, be)
		})

		It("is disabled", func() {
			assert.False(GinkgoT(), c.Disabled())
		})

		It("get from cache", func() {
			aKey := cache.NewStringKey("a")
			x, err := c.Get(aKey)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "1", x.(string))

			x, err = c.Get(aKey)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "1", x.(string))

			assert.True(GinkgoT(), c.Exists(aKey))

			_, ok := c.DirectGet(aKey)
			assert.True(GinkgoT(), ok)
		})

		It("get string", func() {
			aKey := cache.NewStringKey("a")
			x, err := c.GetString(aKey)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "1", x)
		})

		It("get bool", func() {
			boolKey := cache.NewStringKey("bool")
			x, err := c.GetBool(boolKey)
			assert.NoError(GinkgoT(), err)
			assert.True(GinkgoT(), x)
		})

		It("get time", func() {
			timeKey := cache.NewStringKey("time")
			x, err := c.GetTime(timeKey)
			assert.NoError(GinkgoT(), err)
			assert.IsType(GinkgoT(), time.Time{}, x)
		})

		It("get fail", func() {
			errorKey := cache.NewStringKey("error")
			x, err := c.Get(errorKey)
			assert.Error(GinkgoT(), err)
			assert.Nil(GinkgoT(), x)

			err1 := err

			// get fail twice
			x, err = c.Get(errorKey)
			assert.Error(GinkgoT(), err)
			assert.Nil(GinkgoT(), x)

			err2 := err

			// the error should be the same
			assert.Equal(GinkgoT(), err1, err2)

			x, err = c.GetString(errorKey)
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "", x)
		})

		It("delete", func() {
			delKey := cache.NewStringKey("a")
			x, err := c.Get(delKey)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "1", x.(string))

			err = c.Delete(delKey)
			assert.NoError(GinkgoT(), err)
			assert.False(GinkgoT(), c.Exists(delKey))

			_, ok := c.DirectGet(delKey)
			assert.False(GinkgoT(), ok)
		})

		It("set", func() {
			setKey := cache.NewStringKey("s")
			c.Set(setKey, "1")
			x, err := c.GetString(setKey)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "1", x)
		})

		It("disabled", func() {
			c = NewBaseCache(true, retrieveTest, be)
			assert.NotNil(GinkgoT(), c)

			aKey := cache.NewStringKey("a")
			x, err := c.Get(aKey)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "1", x.(string))

			timeKey := cache.NewStringKey("time")
			_, err = c.GetString(timeKey)
			assert.Error(GinkgoT(), err)

			_, err = c.GetBool(aKey)
			assert.Error(GinkgoT(), err)

			_, err = c.GetTime(aKey)
			assert.Error(GinkgoT(), err)
		})
	})

	Context("retrieve Error", func() {
		var c Cache
		BeforeEach(func() {
			retrieveError := func(k cache.Key) (interface{}, error) {
				return nil, errors.New("test error")
			}
			c = NewBaseCache(true, retrieveError, be)
		})
		It("ok", func() {
			assert.NotNil(GinkgoT(), c)

			aKey := cache.NewStringKey("a")
			_, err := c.Get(aKey)
			assert.Error(GinkgoT(), err)

			timeKey := cache.NewStringKey("time")
			_, err = c.GetString(timeKey)
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "test error", err.Error())

			_, err = c.GetBool(aKey)
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "test error", err.Error())

			_, err = c.GetTime(aKey)
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), "test error", err.Error())
		})
	})
})

func retrieveBenchmark(k cache.Key) (interface{}, error) {
	return "", nil
}

func BenchmarkRawRetrieve(b *testing.B) {
	var keys []cache.StringKey
	for i := 0; i < 100000; i++ {
		// keys = append(keys, cache.NewStringKey(util.RandString(5)))
		keys = append(keys, cache.NewStringKey("aaa"))
	}

	b.ResetTimer()
	b.ReportAllocs()

	index := 0
	for i := 0; i < b.N; i++ {
		key := keys[index]
		index++
		if index > 99999 {
			index = 0
		}
		retrieveBenchmark(key)
	}
}

func BenchmarkSingleFlightRetrieve(b *testing.B) {
	var keys []cache.StringKey
	for i := 0; i < 100000; i++ {
		// keys = append(keys, cache.NewStringKey(util.RandString(5)))
		keys = append(keys, cache.NewStringKey("aaa"))
	}

	b.ResetTimer()
	b.ReportAllocs()

	var g singleflight.Group
	index := 0
	for i := 0; i < b.N; i++ {
		key := keys[index]
		index++
		if index == 99999 {
			index = 0
		}

		g.Do(key.Key(), func() (interface{}, error) {
			return retrieveBenchmark(key)
		})
	}
}
