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

package types

import "time"

// Personal access token status values. Status is derived, never stored: a stored
// column would have to be maintained on four write paths, and a single missed
// update produces the worst kind of inconsistency ("list says active, introspect
// says inactive").
const (
	PersonalTokenStatusActive  = "active"
	PersonalTokenStatusExpired = "expired"
	PersonalTokenStatusRevoked = "revoked"

	// PersonalTokenStateActive / PersonalTokenStateInactive are the list filter
	// values (GitLab-style). inactive == expired OR revoked.
	PersonalTokenStateActive   = "active"
	PersonalTokenStateInactive = "inactive"
)

// PersonalAccessToken is the management view of a personal access token, returned
// by the list and detail endpoints. It never carries the plaintext or the hash.
type PersonalAccessToken struct {
	ID          int64
	TokenMask   string
	RealmName   string
	Name        string
	Description string
	Audience    []string
	ExpiresAt   time.Time
	Revoked     bool
	RevokedAt   *time.Time
	LastUsedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Status derives the lifecycle state from revoked + expires_at. Revoked is
// terminal and wins over expiry.
func (t PersonalAccessToken) Status() string {
	if t.Revoked {
		return PersonalTokenStatusRevoked
	}
	if !t.ExpiresAt.After(time.Now()) {
		return PersonalTokenStatusExpired
	}
	return PersonalTokenStatusActive
}

// CreatedPersonalAccessToken is returned by Create. The Token field holds the
// plaintext and is returned exactly once, at creation; it is never persisted and
// cannot be recovered afterwards.
type CreatedPersonalAccessToken struct {
	PersonalAccessToken
	Token string
}

// CreatePersonalAccessTokenInput carries the caller-provided fields for creating
// a personal access token. Identity fields (Sub / Username / TenantID) come from
// the login session, never from the request body.
type CreatePersonalAccessTokenInput struct {
	RealmName   string
	TenantID    string
	Sub         string
	Username    string
	Name        string
	Description string
	Audience    []string
	// ExpiresIn is the lifetime in seconds; expires_at = now + ExpiresIn. Using a
	// duration rather than an absolute timestamp avoids client-clock / time-zone
	// boundary disputes.
	ExpiresIn int64
}

// PersonalTokenPolicy carries the per-request policy knobs the service enforces.
// It is passed in by the handler (built from config.PersonalToken) so the service
// stays free of config, mirroring how TokenIssuancePolicy is threaded through the
// OAuth token service.
type PersonalTokenPolicy struct {
	// MaxTTL caps expires_in on both creation and renewal (seconds).
	MaxTTL int64
	// MaxActivePerUser bounds active tokens per (realm, sub).
	MaxActivePerUser int
}

// UpdatePersonalAccessTokenInput carries an edit of the mutable profile fields
// (name / description / audience). Ownership is enforced by (RealmName, Sub).
type UpdatePersonalAccessTokenInput struct {
	RealmName   string
	Sub         string
	ID          int64
	Name        string
	Description string
	Audience    []string
}
