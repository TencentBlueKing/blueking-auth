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

package config

const (
	defaultPersonalTokenMaxTTL int64 = 94608000 // 3 years
	// A count rather than a duration, so it stays a plain int like every other
	// length and quota in the codebase.
	defaultPersonalTokenMaxActivePerUser int = 20
)

// PersonalToken holds the policy knobs for personal access tokens (PAT).
//
// This phase does not gate the feature by realm on the backend: which realm the
// PAT management API is used from is constrained purely by the frontend routing,
// and the selectable-resource catalog is hardcoded in the frontend. Only the
// numeric policy knobs below stay in config.
type PersonalToken struct {
	// MaxTTL is how far past now an expires_at may be set, in seconds. It bounds
	// creation, renewal and the update path that carries an expiry.
	MaxTTL int64
	// MaxActivePerUser bounds active tokens per (realm, sub).
	MaxActivePerUser int
}

// load fills the defaults.
func (p *PersonalToken) load() {
	if p.MaxTTL == 0 {
		p.MaxTTL = defaultPersonalTokenMaxTTL
	}
	if p.MaxActivePerUser == 0 {
		p.MaxActivePerUser = defaultPersonalTokenMaxActivePerUser
	}
}
