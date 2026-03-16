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

package redis

import (
	"context"
	"fmt"
	"time"

	redis "github.com/go-redis/redis/v8"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	msgpack "github.com/vmihailenco/msgpack/v5"

	"bkauth/pkg/cache"
	"bkauth/pkg/util"
)

var _ = Describe("RedisCache", func() {
	var (
		cli *redis.Client
		c   *Cache
	)
	BeforeEach(func() {
		cli = util.NewTestRedisClient()
		c = NewMockCache(cli, "test", 5*time.Minute)
	})

	It("genKey", func() {
		assert.Equal(GinkgoT(), "bkauth:test:abc", c.genKey("abc"))
	})

	It("Set_Exists_Get", func() {
		key := cache.NewStringKey("abc")
		ctx := context.Background()
		// set
		err := c.Set(ctx, key, 1, 0)
		assert.NoError(GinkgoT(), err)

		// exists
		exists := c.Exists(ctx, key)
		assert.True(GinkgoT(), exists)

		// get
		var a int
		err = c.Get(ctx, key, &a)
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 1, a)
	})

	It("GetInto", func() {
		retrieveTest := func(ctx context.Context, key cache.Key) (interface{}, error) {
			return "ok", nil
		}

		key := cache.NewStringKey("akey")
		ctx := context.Background()

		var i string
		err := c.GetInto(ctx, key, &i, retrieveTest)
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), "ok", i)

		var i2 string
		err = c.GetInto(ctx, key, &i2, retrieveTest)
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), "ok", i2)
	})

	It("Delete", func() {
		key := cache.NewStringKey("dkey")
		ctx := context.Background()

		// do delete
		err := c.Delete(ctx, key)
		assert.NoError(GinkgoT(), err)

		// set
		err = c.Set(ctx, key, 1, 0)
		assert.NoError(GinkgoT(), err)

		// do it again
		err = c.Delete(ctx, key)
		assert.NoError(GinkgoT(), err)
	})

	It("BatchDelete", func() {
		key1 := cache.NewStringKey("d1key")
		key2 := cache.NewStringKey("d2key")
		ctx := context.Background()

		keys := []cache.Key{
			key1,
			key2,
		}

		// do delete 0 key
		err := c.BatchDelete(ctx, keys)
		// assert.Equal(t, int64(0), count)
		assert.NoError(GinkgoT(), err)

		// set
		err = c.Set(ctx, key1, 1, 0)
		assert.NoError(GinkgoT(), err)

		// do delete 1 key
		err = c.BatchDelete(ctx, keys)
		// assert.Equal(t, int64(1), count)
		assert.NoError(GinkgoT(), err)
	})

	It("BatchExpireWithTx", func() {
		key1 := cache.NewStringKey("d1key")
		key2 := cache.NewStringKey("d2key")
		ctx := context.Background()

		keys := []cache.Key{
			key1,
			key2,
		}

		err := c.BatchExpireWithTx(ctx, keys, 1*time.Minute)
		assert.NoError(GinkgoT(), err)
	})

	It("BatchSetWithTx_And_BatchGet", func() {
		ctx := context.Background()
		kvs := []KV{
			{
				Key:   "a",
				Value: "1",
			},
			{
				Key:   "b",
				Value: "2",
			},
		}

		err := c.BatchSetWithTx(ctx, kvs, 5*time.Minute)
		assert.NoError(GinkgoT(), err)

		keys := []cache.Key{
			cache.NewStringKey("a"),
			cache.NewStringKey("b"),
		}

		data, err := c.BatchGet(ctx, keys)
		assert.NoError(GinkgoT(), err)
		assert.Len(GinkgoT(), data, 2)

		akey := cache.NewStringKey("a")
		assert.Contains(GinkgoT(), data, akey)
		assert.Equal(GinkgoT(), "1", data[akey])
	})

	Context("SetOneAndBatchGet", func() {
		type Abc struct {
			X string
			Y int
			Z string
		}

		key := cache.NewStringKey("a")

		// compressionThreshold = 64
		It("less than compressionThreshold", func() {
			ctx := context.Background()
			c.Set(ctx, key, Abc{
				X: "hello",
				Y: 123,
				Z: "",
			}, 5*time.Minute)

			data, err := c.BatchGet(ctx, []cache.Key{key})
			assert.NoError(GinkgoT(), err)
			assert.Len(GinkgoT(), data, 1)

			// NOTE: the string is msgpack marshal and compress, so
			value := data[key]
			fmt.Println("value", value)
			var abc Abc

			err = c.Unmarshal(util.StringToBytes(value), &abc)
			fmt.Println("abc:", abc)
			assert.NoError(GinkgoT(), err)

			var def Abc
			err = msgpack.Unmarshal(util.StringToBytes(value), &def)
			fmt.Println("def:", abc)
			assert.NoError(GinkgoT(), err)
		})

		It("greater than compressThreshold", func() {
			ctx := context.Background()
			c.Set(ctx, key, Abc{
				X: "hello",
				Y: 123,
				Z: "123456789012345678901234567890123456789012345678901234567890",
			}, 5*time.Minute)

			data, err := c.BatchGet(ctx, []cache.Key{key})
			assert.NoError(GinkgoT(), err)
			assert.Len(GinkgoT(), data, 1)

			// NOTE: the string is msgpack marshal and compress, so
			value := data[key]
			fmt.Println("value", value)
			var abc Abc

			err = c.Unmarshal(util.StringToBytes(value), &abc)
			fmt.Println("abc:", abc)
			assert.NoError(GinkgoT(), err)

			var def Abc
			err = msgpack.Unmarshal(util.StringToBytes(value), &def)
			fmt.Println("def:", abc)
			assert.Error(GinkgoT(), err)
		})
	})

	It("BatchSetAndGet", func() {
		type Abc struct {
			X string
			Y int
			Z string
		}

		small, _ := c.Marshal(Abc{
			X: "hello",
			Y: 123,
			Z: "",
		})
		huge, _ := c.Marshal(Abc{
			X: "hello",
			Y: 123,
			Z: "123456789012345678901234567890123456789012345678901234567890",
		})

		kvs := []KV{
			{
				Key:   "a",
				Value: util.BytesToString(small),
			},
			{
				Key:   "b",
				Value: util.BytesToString(huge),
			},
		}

		ctx := context.Background()
		err := c.BatchSetWithTx(ctx, kvs, 5*time.Minute)
		assert.NoError(GinkgoT(), err)

		// get single: small without compress
		var v1 Abc
		err = c.Get(ctx, cache.NewStringKey("a"), &v1)
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), v1.X, "hello")
		assert.Equal(GinkgoT(), v1.Y, 123)
		assert.Equal(GinkgoT(), v1.Z, "")
		// get single: huge with compress
		var v2 Abc
		err = c.Get(ctx, cache.NewStringKey("b"), &v2)
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), v2.X, "hello")
		assert.Equal(GinkgoT(), v2.Y, 123)
		assert.Equal(GinkgoT(), v2.Z, "123456789012345678901234567890123456789012345678901234567890")
	})
})
