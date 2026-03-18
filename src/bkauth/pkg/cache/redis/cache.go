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

// Package redis provides Redis-backed cache implementations.
package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	cache "github.com/go-redis/cache/v8"
	redis "github.com/go-redis/redis/v8"
	msgpack "github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	bkauthCache "bkauth/pkg/cache"
)

const (
	// while the go-redis/cache upgrade maybe not compatible with the previous version.
	// e.g. the object set by v7 can't read by v8
	// https://github.com/go-redis/cache/issues/52
	// NOTE: important!!! if upgrade the go-redis/cache version, should change the version

	// CacheVersion is loop in 00->99->00 => make sure will not conflict with previous version
	CacheVersion = "00"

	PipelineSizeThreshold = 100

	retrieveTimeout = 10 * time.Second
)

// RetrieveFunc ...
type RetrieveFunc func(ctx context.Context, key bkauthCache.Key) (any, error)

// Cache is a cache implements
type Cache struct {
	name              string
	keyPrefix         string
	codec             *cache.Cache
	cli               *redis.Client
	defaultExpiration time.Duration
	g                 singleflight.Group
}

// NewCache create a cache instance
func NewCache(cli *redis.Client, name string, expiration time.Duration) *Cache {
	// key format = iam:{version}:{cache_name}:{real_key}
	keyPrefix := fmt.Sprintf("bkauth:%s:%s", CacheVersion, name)

	codec := cache.New(&cache.Options{
		Redis: cli,
	})

	return &Cache{
		name:              name,
		keyPrefix:         keyPrefix,
		codec:             codec,
		cli:               cli,
		defaultExpiration: expiration,
	}
}

// NewMockCache will create a cache for mock
func NewMockCache(cli *redis.Client, name string, expiration time.Duration) *Cache {
	// key format = bkauth:{cache_name}:{real_key}
	keyPrefix := "bkauth:" + name

	codec := cache.New(&cache.Options{
		Redis: cli,
	})

	return &Cache{
		name:              name,
		keyPrefix:         keyPrefix,
		codec:             codec,
		cli:               cli,
		defaultExpiration: expiration,
	}
}

func (c *Cache) genKey(key string) string {
	return c.keyPrefix + ":" + key
}

func (c *Cache) copyTo(source, dest any) error {
	b, err := msgpack.Marshal(source)
	if err != nil {
		return err
	}

	err = msgpack.Unmarshal(b, dest)
	return err
}

// Set execute `set`
func (c *Cache) Set(ctx context.Context, key bkauthCache.Key, value any, duration time.Duration) error {
	if duration == time.Duration(0) {
		duration = c.defaultExpiration
	}

	k := c.genKey(key.Key())
	return c.codec.Set(&cache.Item{
		Ctx:   ctx,
		Key:   k,
		Value: value,
		TTL:   duration,
	})
}

// Get execute `get`
func (c *Cache) Get(ctx context.Context, key bkauthCache.Key, value any) error {
	k := c.genKey(key.Key())
	return c.codec.Get(ctx, k, value)
}

// Exists execute `exists`
func (c *Cache) Exists(ctx context.Context, key bkauthCache.Key) bool {
	k := c.genKey(key.Key())

	count, err := c.cli.Exists(ctx, k).Result()

	return err == nil && count == 1
}

// GetInto will retrieve the data from cache and unmarshal into the obj
func (c *Cache) GetInto(
	ctx context.Context,
	key bkauthCache.Key,
	obj any,
	retrieveFunc RetrieveFunc,
) (err error) {
	// 1. get from cache, hit, return
	err = c.Get(ctx, key, obj)
	if err == nil {
		return err
	}

	// 2. if missing
	// 2.1 check the guard
	// 2.2 do retrieve
	// 防止首个请求取消导致后续请求失败
	retrieveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), retrieveTimeout)
	defer cancel()

	data, err, _ := c.g.Do(key.Key(), func() (any, error) {
		return retrieveFunc(retrieveCtx, key)
	})
	// 2.3 do retrieve fail, make guard and return
	if err != nil {
		// if retrieve fail, should wait for few seconds for the missing-retrieve
		// c.makeGuard(key)
		return err
	}

	// 3. set to cache
	errNotImportant := c.Set(ctx, key, data, 0)
	if errNotImportant != nil {
		zap.S().Errorf("set to redis fail, key=%s, err=%s", key.Key(), errNotImportant)
	}

	// 注意, 这里基础类型无法通过 *obj = value 来赋值
	// 所以利用从缓存再次反序列化给对应指针赋值(相当于底层msgpack.unmarshal帮做了转换再次反序列化给对应指针赋值
	return c.copyTo(data, obj)
}

// Delete execute `del`
func (c *Cache) Delete(ctx context.Context, key bkauthCache.Key) (err error) {
	k := c.genKey(key.Key())

	_, err = c.cli.Del(ctx, k).Result()
	return err
}

// BatchDelete execute `del` with pipeline
func (c *Cache) BatchDelete(ctx context.Context, keys []bkauthCache.Key) error {
	newKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		newKeys = append(newKeys, c.genKey(key.Key()))
	}

	var err error
	if len(newKeys) < PipelineSizeThreshold {
		_, err = c.cli.Del(ctx, newKeys...).Result()
	} else {
		pipe := c.cli.Pipeline()

		for _, key := range newKeys {
			pipe.Del(ctx, key)
		}

		_, err = pipe.Exec(ctx)
	}
	return err
}

// BatchExpireWithTx execute `expire` with tx pipeline
func (c *Cache) BatchExpireWithTx(ctx context.Context, keys []bkauthCache.Key, expiration time.Duration) error {
	pipe := c.cli.TxPipeline()

	for _, k := range keys {
		key := c.genKey(k.Key())
		pipe.Expire(ctx, key, expiration)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// KV is a key-value pair
type KV struct {
	Key   string
	Value string
}

// BatchGet execute `get` with pipeline
func (c *Cache) BatchGet(ctx context.Context, keys []bkauthCache.Key) (map[bkauthCache.Key]string, error) {
	pipe := c.cli.Pipeline()

	cmds := map[bkauthCache.Key]*redis.StringCmd{}
	for _, k := range keys {
		key := c.genKey(k.Key())
		cmd := pipe.Get(ctx, key)

		cmds[k] = cmd
	}

	_, err := pipe.Exec(ctx)
	// 当批量操作, 里面有个key不存在, err = redis.Nil; 但是不应该影响其他存在的key的获取
	// Nil reply returned by Redis when key does not exist.
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	values := make(map[bkauthCache.Key]string, len(cmds))
	for hkf, cmd := range cmds {
		// maybe err or key missing
		// only return the HashKeyField who get value success from redis
		val, err := cmd.Result()
		if err != nil {
			continue
		} else {
			values[hkf] = val
		}
	}
	return values, nil
}

// BatchSetWithTx execute `set` with tx pipeline
func (c *Cache) BatchSetWithTx(ctx context.Context, kvs []KV, expiration time.Duration) error {
	// tx, all success or all fail
	pipe := c.cli.TxPipeline()

	for _, kv := range kvs {
		key := c.genKey(kv.Key)
		pipe.Set(ctx, key, kv.Value, expiration)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// ZData is a sorted-set data for redis `key: {member: score}`
type ZData struct {
	Key string
	Zs  []*redis.Z
}

// BatchZAdd execute `zadd` with pipeline
func (c *Cache) BatchZAdd(ctx context.Context, zDataList []ZData) error {
	pipe := c.cli.TxPipeline()

	for _, zData := range zDataList {
		key := c.genKey(zData.Key)
		pipe.ZAdd(ctx, key, zData.Zs...)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// ZRevRangeByScore execute `zrevrangebyscorewithscores`
func (c *Cache) ZRevRangeByScore(
	ctx context.Context,
	k string,
	min int64,
	max int64,
	offset int64,
	count int64,
) ([]redis.Z, error) {
	// 时间戳, 从大到小排序
	key := c.genKey(k)
	// TODO: add limit, offset, count => to ignore the too large list size
	// LIMIT 0 -1 equals no args
	cmds := c.cli.ZRevRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min:    strconv.FormatInt(min, 10),
		Max:    strconv.FormatInt(max, 10),
		Offset: offset,
		Count:  count,
	})

	return cmds.Result()
}

// BatchZRemove execute `zremrangebyscore` with pipeline
func (c *Cache) BatchZRemove(ctx context.Context, keys []string, min, max int64) error {
	pipe := c.cli.TxPipeline()

	minStr := strconv.FormatInt(min, 10)
	maxStr := strconv.FormatInt(max, 10)

	for _, k := range keys {
		key := c.genKey(k)
		pipe.ZRemRangeByScore(ctx, key, minStr, maxStr)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// HashKeyField is a hash data for redis, `Key: field -> `
type HashKeyField struct {
	Key   string
	Field string
}

// Hash is a hash data  `Key: field->value`
type Hash struct {
	HashKeyField
	Value string
}

// BatchHSetWithTx execute `hset` with tx pipeline
func (c *Cache) BatchHSetWithTx(ctx context.Context, hashes []Hash) error {
	// tx, all success or all fail
	pipe := c.cli.TxPipeline()

	for _, h := range hashes {
		key := c.genKey(h.Key)
		pipe.HSet(ctx, key, h.Field, h.Value)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// BatchHGet execute `hget` with pipeline
func (c *Cache) BatchHGet(ctx context.Context, hashKeyFields []HashKeyField) (map[HashKeyField]string, error) {
	pipe := c.cli.Pipeline()

	cmds := make(map[HashKeyField]*redis.StringCmd, len(hashKeyFields))
	for _, h := range hashKeyFields {
		key := c.genKey(h.Key)
		cmd := pipe.HGet(ctx, key, h.Field)

		cmds[h] = cmd
	}

	_, err := pipe.Exec(ctx)
	// 当批量操作, 里面有个key不存在, err = redis.Nil; 但是不应该影响其他存在的key的获取
	// Nil reply returned by Redis when key does not exist.
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	values := make(map[HashKeyField]string, len(cmds))
	for hkf, cmd := range cmds {
		// maybe err or key missing
		// only return the HashKeyField who get value success from redis
		val, err := cmd.Result()
		if err != nil {
			continue
		} else {
			values[hkf] = val
		}
	}
	return values, nil
}

// HKeys execute `hkeys`
func (c *Cache) HKeys(ctx context.Context, hashKey string) ([]string, error) {
	key := c.genKey(hashKey)
	return c.cli.HKeys(ctx, key).Result()
}

// Unmarshal with compress, via go-redis/cache, use s2 compression
// Note: YOU SHOULD NOT USE THE RAW msgpack.Unmarshal directly! will panic with decode fail
func (c *Cache) Unmarshal(b []byte, value any) error {
	return c.codec.Unmarshal(b, value)
}

// Marshal with compress, via go-redis/cache, use s2 compression
// Note: YOU SHOULD NOT USE THE RAW msgpack.Marshal directly!
func (c *Cache) Marshal(value any) ([]byte, error) {
	return c.codec.Marshal(value)
}
