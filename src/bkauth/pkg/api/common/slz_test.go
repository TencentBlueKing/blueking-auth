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

func TestAppCodeSerializer_ValidateAppCode(t *testing.T) {
	tests := []struct {
		name    string
		appCode string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid app_code",
			appCode: "valid_app_code",
			wantErr: false,
		},
		{
			name:    "invalid app_code: starts with special char",
			appCode: "==1",
			wantErr: true,
			errMsg:  ErrInvalidAppCode.Error(),
		},
		{
			name:    "reserved prefix: public_xxx",
			appCode: "public_xxx",
			wantErr: true,
			errMsg:  ErrReservedAppCode.Error(),
		},
		{
			name:    "reserved prefix: dcr-xxx",
			appCode: "dcr-xxx",
			wantErr: true,
			errMsg:  ErrReservedAppCode.Error(),
		},
		{
			name:    "reserved exact match: public",
			appCode: "public",
			wantErr: true,
			errMsg:  ErrReservedAppCode.Error(),
		},
		{
			name:    "not reserved: publicapp (no delimiter)",
			appCode: "publicapp",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AppCodeSerializer{AppCode: tt.appCode}
			err := s.ValidateAppCode()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
