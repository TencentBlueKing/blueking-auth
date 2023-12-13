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
	"time"

	"bkauth/pkg/cache"
)

// RetrieveFunc ...
type RetrieveFunc func(key cache.Key) (interface{}, error)

// Cache ...
type Cache interface {
	Get(key cache.Key) (interface{}, error)
	Set(key cache.Key, data interface{})

	GetString(key cache.Key) (string, error)
	GetBool(key cache.Key) (bool, error)
	GetTime(key cache.Key) (time.Time, error)
	GetInt64(key cache.Key) (int64, error)

	Delete(key cache.Key) error
	Exists(key cache.Key) bool

	DirectGet(key cache.Key) (interface{}, bool)

	Disabled() bool
}
