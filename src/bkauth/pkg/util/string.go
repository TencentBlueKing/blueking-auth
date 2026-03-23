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

import "strings"

// SplitCommaList splits a comma-separated string into trimmed, non-empty items.
// Leading/trailing whitespace around each item is stripped, and empty items
// (including those produced by trailing commas) are discarded.
//
// Examples:
//
//	SplitCommaList("a, b, c")        => ["a", "b", "c"]
//	SplitCommaList(" a , b , ")      => ["a", "b"]
//	SplitCommaList("")               => []  (empty, length 0)
//	SplitCommaList("  ,  , ")        => []  (all items empty after trim)
//	SplitCommaList("single")         => ["single"]
func SplitCommaList(s string) []string {
	raw := strings.Split(s, ",")
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

// TruncateBytes truncate []byte to specific length
func TruncateBytes(content []byte, length int) []byte {
	if len(content) > length {
		return content[:length]
	}
	return content
}

// TruncateBytesToString ...
func TruncateBytesToString(content []byte, length int) string {
	s := TruncateBytes(content, length)
	return string(s)
}

// TruncateString truncate string to specific length
func TruncateString(s string, n int) string {
	if n > len(s) {
		return s
	}
	return s[:n]
}
