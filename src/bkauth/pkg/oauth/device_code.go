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

package oauth

import (
	"strings"

	"bkauth/pkg/util"
)

const (
	// UserCodeCharset RFC 8628 Section 6.1: restricted character set to avoid ambiguity
	UserCodeCharset = "BCDFGHJKLMNPQRSTVWXZ"

	DeviceCodeStatusPending  = "pending"
	DeviceCodeStatusApproved = "approved"
	DeviceCodeStatusDenied   = "denied"
	DeviceCodeStatusConsumed = "consumed"

	// DeviceCodeTTL is the lifetime of device code in seconds (10 minutes per RFC 8628 Section 3.1)
	DeviceCodeTTL = 600
	// DeviceCodeInterval is the minimum polling interval in seconds (RFC 8628 Section 3.2)
	DeviceCodeInterval = 5
	// SlowDownIncrement is the number of seconds added per RFC 8628 Section 3.5
	SlowDownIncrement = 5
)

// GenerateDeviceCode generates a high-entropy device code.
// RFC 8628 Section 6.1 + RFC 6749 Section 10.10: MUST have >= 128 bits of entropy.
func GenerateDeviceCode() (string, error) {
	// 128 bit => 32-char hex string
	return util.RandHex(16)
}

// GenerateUserCode generates a human-readable user code (e.g., "WDJB-MJHT").
// RFC 8628 Section 6.1: SHOULD use a limited character set to avoid ambiguity,
// with at least 20 bits of entropy. Current: 20-char alphabet, 8 chars => ~2^34.6.
func GenerateUserCode() (string, error) {
	raw, err := util.RandString(UserCodeCharset, 8)
	if err != nil {
		return "", err
	}
	return raw[:4] + "-" + raw[4:], nil
}

// NormalizeUserCode uppercases and ensures the hyphen format (e.g. "WDJB-MJHT").
func NormalizeUserCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, " ", "")
	if len(code) == 8 && !strings.Contains(code, "-") {
		code = code[:4] + "-" + code[4:]
	}
	return code
}
