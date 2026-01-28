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

package util

import (
	"fmt"
	"strconv"
	"time"
)

// UnixTime 自定义时间类型，JSON 序列化时输出为 Unix 时间戳
type UnixTime time.Time

// MarshalJSON 实现 JSON 序列化，输出 Unix 时间戳
func (ut UnixTime) MarshalJSON() ([]byte, error) {
	var buf []byte
	buf = fmt.Appendf(buf, "%d", time.Time(ut).Unix())
	return buf, nil
}

// UnmarshalJSON 实现 JSON 反序列化，从 Unix 时间戳解析
func (ut *UnixTime) UnmarshalJSON(data []byte) error {
	timestamp, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	*ut = UnixTime(time.Unix(timestamp, 0))
	return nil
}

// String 实现 fmt.Stringer 接口，用于 CLI 输出
func (ut UnixTime) String() string {
	return time.Time(ut).String()
}

// ToTime 转换为标准 time.Time 类型
func (ut UnixTime) ToTime() time.Time {
	return time.Time(ut)
}

// FromTime 从标准 time.Time 类型创建 UnixTime
func FromTime(t time.Time) UnixTime {
	return UnixTime(t)
}
