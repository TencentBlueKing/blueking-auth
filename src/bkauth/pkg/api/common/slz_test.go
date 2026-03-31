/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth 服务 (BlueKing - Auth) available.
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsReservedAppCode(t *testing.T) {
	tests := []struct {
		name    string
		appCode string
		want    bool
	}{
		{
			name:    "normal app_code",
			appCode: "valid_app_code",
			want:    false,
		},
		{
			name:    "reserved exact match: public",
			appCode: "public",
			want:    true,
		},
		{
			name:    "reserved exact match: private",
			appCode: "private",
			want:    true,
		},
		{
			name:    "reserved exact match: dcr",
			appCode: "dcr",
			want:    true,
		},
		{
			name:    "reserved exact match: cimd",
			appCode: "cimd",
			want:    true,
		},
		{
			name:    "reserved prefix with underscore: public_xxx",
			appCode: "public_xxx",
			want:    true,
		},
		{
			name:    "reserved prefix with hyphen: dcr-xxx",
			appCode: "dcr-xxx",
			want:    true,
		},
		{
			name:    "reserved prefix with hyphen: cimd-foo",
			appCode: "cimd-foo",
			want:    true,
		},
		{
			name:    "reserved prefix with underscore: private_key",
			appCode: "private_key",
			want:    true,
		},
		{
			name:    "not reserved: publicapp (no delimiter)",
			appCode: "publicapp",
			want:    false,
		},
		{
			name:    "not reserved: dcraft (no delimiter)",
			appCode: "dcraft",
			want:    false,
		},
		{
			name:    "not reserved: privateer (no delimiter)",
			appCode: "privateer",
			want:    false,
		},
		{
			name:    "not reserved: empty string",
			appCode: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsReservedAppCode(tt.appCode)
			assert.Equal(t, tt.want, got)
		})
	}
}
