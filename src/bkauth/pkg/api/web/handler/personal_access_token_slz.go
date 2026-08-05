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

package handler

import (
	"errors"
	"time"

	"bkauth/pkg/service/types"
)

const (
	// audience write constraints (design 13.5), hard-coded on purpose: an empty
	// aud has undefined meaning at the gateway (could be read as "unrestricted",
	// i.e. an all-powerful token), and a JSON column has no length ceiling, so an
	// unbounded aud would inflate both the introspect response and the Redis value.
	personalTokenAudienceMinItems  = 1
	personalTokenAudienceMaxItems  = 50
	personalTokenAudienceMaxLength = 255
)

var (
	errPersonalTokenEmptyAudience   = errors.New("resource must resolve to at least one audience")
	errPersonalTokenTooManyAudience = errors.New("resource resolves to too many audiences")
	errPersonalTokenAudienceTooLong = errors.New("a resolved audience exceeds the maximum length")
)

// createPersonalTokenSerializer is the POST body. Identity (sub/username/tenant)
// is taken from the login session and any same-named body field is ignored.
//
// It is bound with ShouldBindJSON, never ShouldBind: a cross-site HTML form cannot
// send application/json without tripping a CORS preflight, which is half of this
// endpoint's CSRF defence (design 6.2). Do NOT copy the form+json dual-tag style
// of IntrospectRequest here.
type createPersonalTokenSerializer struct {
	Name        string `json:"name" binding:"required,max=64"`
	Description string `json:"description" binding:"omitempty,max=255"`
	// Resource is the same comma-separated selector string as /authorize and
	// /device/authorize; the realm validates and extracts audiences from it.
	Resource string `json:"resource" binding:"required"`
	// ExpiresIn is the lifetime in seconds. Zero means "use the configured
	// default TTL". The maximum is enforced by the service against MaxTTL.
	ExpiresIn int64 `json:"expires_in" binding:"omitempty,min=0"`
}

// updatePersonalTokenSerializer is the PUT body: edits name / description /
// audience together.
type updatePersonalTokenSerializer struct {
	Name        string `json:"name" binding:"required,max=64"`
	Description string `json:"description" binding:"omitempty,max=255"`
	Resource    string `json:"resource" binding:"required"`
}

// renewPersonalTokenSerializer is the renew body. Zero ExpiresIn means the
// configured default TTL. expires_in (a duration) is used rather than expires_at
// (an absolute time) to avoid client-clock / time-zone boundary disputes.
type renewPersonalTokenSerializer struct {
	ExpiresIn int64 `json:"expires_in" binding:"omitempty,min=0"`
}

// listPersonalTokenQuery is the list filter. state=active|inactive follows the
// GitLab API; empty means all.
type listPersonalTokenQuery struct {
	State string `form:"state" binding:"omitempty,oneof=active inactive"`
}

// validateAudiences enforces the write constraints on the extracted audiences.
func validateAudiences(audiences []string) error {
	if len(audiences) < personalTokenAudienceMinItems {
		return errPersonalTokenEmptyAudience
	}
	if len(audiences) > personalTokenAudienceMaxItems {
		return errPersonalTokenTooManyAudience
	}
	for _, aud := range audiences {
		if len(aud) > personalTokenAudienceMaxLength {
			return errPersonalTokenAudienceTooLong
		}
	}
	return nil
}

// personalTokenResponse is the management view returned by list / detail / edit /
// renew / revoke. It never carries the plaintext.
//
// Times are rendered as RFC3339 (UTC); the frontend is responsible for showing
// them in the viewer's time zone.
type personalTokenResponse struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	RealmName   string     `json:"realm_name"`
	Audience    []string   `json:"audience"`
	TokenMask   string     `json:"token_mask"`
	Status      string     `json:"status"`
	ExpiresAt   time.Time  `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// createPersonalTokenResponse embeds the management view and adds the plaintext
// token, which is returned exactly once here and never again. The field name is
// fixed to "token"; log desensitization for logger.web / logger.audit targets the
// jsonPath data.token (design 13.1.1).
type createPersonalTokenResponse struct {
	personalTokenResponse
	Token string `json:"token"`
}

func newPersonalTokenResponse(t types.PersonalAccessToken) personalTokenResponse {
	aud := t.Audience
	if aud == nil {
		aud = []string{}
	}
	return personalTokenResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		RealmName:   t.RealmName,
		Audience:    aud,
		TokenMask:   t.TokenMask,
		Status:      t.Status(),
		ExpiresAt:   t.ExpiresAt,
		RevokedAt:   t.RevokedAt,
		LastUsedAt:  t.LastUsedAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
