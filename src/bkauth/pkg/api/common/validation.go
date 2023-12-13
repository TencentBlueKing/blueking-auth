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

package common

import (
	"errors"
	"regexp"
)

const (
	// 小写字母开头, 可以包含小写字母/数字/下划线/连字符
	validIDString = "^[a-z]+[a-z0-9_-]*$"
)

// ValidIDRegex ...
var (
	ValidIDRegex = regexp.MustCompile(validIDString)

	ErrInvalidID = errors.New("invalid id: id should begin with a lowercase letter, " +
		"contains lowercase letters(a-z), numbers(0-9), underline(_) or hyphen(-)")
)
