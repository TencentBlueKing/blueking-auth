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

// Package memory provides in-memory cache implementations.
package memory

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"

	"bkauth/pkg/cache"
	"bkauth/pkg/cache/memory/backend"
)

const (
	// EmptyCacheExpiration ...
	EmptyCacheExpiration = 5 * time.Second

	retrieveTimeout = 10 * time.Second
)

// BaseCache ...
type BaseCache struct {
	backend backend.Backend

	disabled     bool
	retrieveFunc RetrieveFunc
	g            singleflight.Group
}

// EmptyCache is a place holder for the missing key
type EmptyCache struct {
	err error
}

// Exists ...
func (c *BaseCache) Exists(key cache.Key) bool {
	k := key.Key()
	_, ok := c.backend.Get(k)
	return ok
}

// Get will get the key from cache, if missing, will call the retrieveFunc to get the data, add to cache, then return
func (c *BaseCache) Get(ctx context.Context, key cache.Key) (any, error) {
	// 1. if cache is disabled, fetch and return
	if c.disabled {
		value, err := c.retrieveFunc(ctx, key)
		if err != nil {
			return nil, err
		}
		return value, nil
	}

	k := key.Key()

	// 2. get from cache
	value, ok := c.backend.Get(k)
	if ok {
		// if retrieve fail from retrieveFunc
		if emptyCache, isEmptyCache := value.(EmptyCache); isEmptyCache {
			return nil, emptyCache.err
		}
		return value, nil
	}

	// 3. if not exists in cache, retrieve it
	return c.doRetrieve(ctx, key)
}

func (c *BaseCache) doRetrieve(ctx context.Context, k cache.Key) (any, error) {
	key := k.Key()

	// 防止首个请求取消导致后续请求失败
	retrieveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), retrieveTimeout)
	defer cancel()

	value, err, _ := c.g.Do(key, func() (any, error) {
		return c.retrieveFunc(retrieveCtx, k)
	})

	if err != nil {
		// ! if error, cache it too, make it short enough(5s)
		c.backend.Set(key, EmptyCache{err: err}, EmptyCacheExpiration)
		return nil, err
	}

	// 4. set value to cache, use default expiration
	c.backend.Set(key, value, 0)

	return value, nil
}

// Set ...
func (c *BaseCache) Set(key cache.Key, data any) {
	k := key.Key()
	c.backend.Set(k, data, 0)
}

// TODO: 这里需要实现所有类型的 GetXXXX

// GetString returns the cached string value. If retrieve fails, will return ("", err) for expire time.
func (c *BaseCache) GetString(ctx context.Context, k cache.Key) (string, error) {
	value, err := c.Get(ctx, k)
	if err != nil {
		return "", err
	}

	v, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("not a string value. key=%s, value=%v(%T)", k.Key(), value, value)
	}
	return v, nil
}

// GetBool ...
func (c *BaseCache) GetBool(ctx context.Context, k cache.Key) (bool, error) {
	value, err := c.Get(ctx, k)
	if err != nil {
		return false, err
	}

	v, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("not a bool value. key=%s, value=%v(%T)", k.Key(), value, value)
	}
	return v, nil
}

// GetInt64 ...
func (c *BaseCache) GetInt64(ctx context.Context, k cache.Key) (int64, error) {
	value, err := c.Get(ctx, k)
	if err != nil {
		return 0, err
	}

	v, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("not a int64 value. key=%s, value=%v(%T)", k.Key(), value, value)
	}
	return v, nil
}

var defaultZeroTime = time.Time{}

// GetTime ...
func (c *BaseCache) GetTime(ctx context.Context, k cache.Key) (time.Time, error) {
	value, err := c.Get(ctx, k)
	if err != nil {
		return defaultZeroTime, err
	}

	v, ok := value.(time.Time)
	if !ok {
		return defaultZeroTime, fmt.Errorf("not a time.Time value. key=%s, value=%v(%T)", k.Key(), value, value)
	}
	return v, nil
}

// Delete ...
func (c *BaseCache) Delete(key cache.Key) error {
	k := key.Key()
	return c.backend.Delete(k)
}

// DirectGet will get key from cache, without calling the retrieveFunc
func (c *BaseCache) DirectGet(key cache.Key) (any, bool) {
	k := key.Key()
	return c.backend.Get(k)
}

// Disabled ...
func (c *BaseCache) Disabled() bool {
	return c.disabled
}

// NewBaseCache ...
func NewBaseCache(disabled bool, retrieveFunc RetrieveFunc, backend backend.Backend) Cache {
	return &BaseCache{
		backend:      backend,
		disabled:     disabled,
		retrieveFunc: retrieveFunc,
	}
}
