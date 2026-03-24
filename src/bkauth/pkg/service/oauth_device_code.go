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

package service

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"context"
	"encoding/json"
	"time"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

const OAuthDeviceCodeSVC = "OAuthDeviceCodeSVC"

// OAuthDeviceCodeService defines the interface for device code operations
type OAuthDeviceCodeService interface {
	CreateDeviceCode(ctx context.Context, realmName, clientID, resource string) (types.CreatedDeviceCode, error)
	GetByUserCode(ctx context.Context, userCode string) (types.PendingDeviceCode, error)
	ApproveByUserCode(ctx context.Context, tenantID, userCode, sub, username string, audience []string) error
	DenyByUserCode(ctx context.Context, userCode string) error
	PollAndConsumeDeviceCode(ctx context.Context, realmName, deviceCode, clientID string) (types.ApprovedDeviceCode, error)
}

type oauthDeviceCodeService struct {
	deviceCodeManager dao.OAuthDeviceCodeManager
}

// NewOAuthDeviceCodeService creates a new OAuthDeviceCodeService
func NewOAuthDeviceCodeService() OAuthDeviceCodeService {
	return &oauthDeviceCodeService{
		deviceCodeManager: dao.NewOAuthDeviceCodeManager(),
	}
}

func (s *oauthDeviceCodeService) CreateDeviceCode(
	ctx context.Context,
	realmName, clientID, resource string,
) (types.CreatedDeviceCode, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthDeviceCodeSVC, "CreateDeviceCode")

	deviceCode, err := oauth.GenerateDeviceCode()
	if err != nil {
		return types.CreatedDeviceCode{}, errorWrapf(err, "GenerateDeviceCode fail")
	}

	userCode, err := oauth.GenerateUserCode()
	if err != nil {
		return types.CreatedDeviceCode{}, errorWrapf(err, "GenerateUserCode fail")
	}

	expiresAt := time.Now().Add(time.Duration(oauth.DeviceCodeTTL) * time.Second)

	daoDeviceCode := dao.OAuthDeviceCode{
		DeviceCode:   deviceCode,
		UserCode:     userCode,
		ClientID:     clientID,
		Resource:     resource,
		RealmName:    realmName,
		Status:       oauth.DeviceCodeStatusPending,
		PollInterval: oauth.DeviceCodeInterval,
		ExpiresAt:    expiresAt,
	}

	if _, err := s.deviceCodeManager.Create(ctx, daoDeviceCode); err != nil {
		return types.CreatedDeviceCode{}, errorWrapf(err, "deviceCodeManager.Create fail")
	}

	return types.CreatedDeviceCode{
		DeviceCode:   deviceCode,
		UserCode:     userCode,
		PollInterval: oauth.DeviceCodeInterval,
	}, nil
}

func (s *oauthDeviceCodeService) GetByUserCode(ctx context.Context, userCode string) (types.PendingDeviceCode, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthDeviceCodeSVC, "GetByUserCode")

	userCode = oauth.NormalizeUserCode(userCode)

	dc, err := s.deviceCodeManager.GetByUserCode(ctx, userCode)
	if err != nil {
		return types.PendingDeviceCode{}, errorWrapf(err, "deviceCodeManager.GetByUserCode fail")
	}

	if dc.ID == 0 {
		return types.PendingDeviceCode{}, oauth.ErrInvalidUserCode
	}

	if time.Now().After(dc.ExpiresAt) {
		return types.PendingDeviceCode{}, oauth.ErrUserCodeExpired
	}

	if dc.Status != oauth.DeviceCodeStatusPending {
		return types.PendingDeviceCode{}, oauth.ErrUserCodeAlreadyUsed
	}

	return types.PendingDeviceCode{
		ClientID:  dc.ClientID,
		RealmName: dc.RealmName,
		Resource:  dc.Resource,
	}, nil
}

func (s *oauthDeviceCodeService) ApproveByUserCode(
	ctx context.Context,
	tenantID, userCode, sub, username string, audience []string,
) error {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthDeviceCodeSVC, "ApproveByUserCode")

	userCode = oauth.NormalizeUserCode(userCode)

	dc, err := s.deviceCodeManager.GetByUserCode(ctx, userCode)
	if err != nil {
		return errorWrapf(err, "deviceCodeManager.GetByUserCode fail")
	}

	if dc.ID == 0 {
		return oauth.ErrInvalidUserCode
	}

	if time.Now().After(dc.ExpiresAt) {
		return oauth.ErrUserCodeExpired
	}

	if dc.Status != oauth.DeviceCodeStatusPending {
		return oauth.ErrUserCodeAlreadyUsed
	}

	audienceJSON, err := json.Marshal(audience)
	if err != nil {
		return errorWrapf(err, "json.Marshal audience fail")
	}

	if _, err := s.deviceCodeManager.Approve(ctx, dc.ID, tenantID, sub, username, string(audienceJSON)); err != nil {
		return errorWrapf(err, "deviceCodeManager.Approve fail")
	}

	return nil
}

func (s *oauthDeviceCodeService) DenyByUserCode(ctx context.Context, userCode string) error {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthDeviceCodeSVC, "DenyByUserCode")

	userCode = oauth.NormalizeUserCode(userCode)

	dc, err := s.deviceCodeManager.GetByUserCode(ctx, userCode)
	if err != nil {
		return errorWrapf(err, "deviceCodeManager.GetByUserCode fail")
	}

	if dc.ID == 0 {
		return oauth.ErrInvalidUserCode
	}

	if time.Now().After(dc.ExpiresAt) {
		return oauth.ErrUserCodeExpired
	}

	if dc.Status != oauth.DeviceCodeStatusPending {
		return oauth.ErrUserCodeAlreadyUsed
	}

	if _, err := s.deviceCodeManager.UpdateStatus(ctx, dc.ID, oauth.DeviceCodeStatusDenied); err != nil {
		return errorWrapf(err, "deviceCodeManager.UpdateStatus fail")
	}

	return nil
}

// PollAndConsumeDeviceCode validates a device code (ownership, expiry, polling rate),
// and atomically marks it as consumed when approved. Returns the decoded identity
// claims on success. Analogous to OAuthAuthorizationCodeService.ValidateAndConsume.
func (s *oauthDeviceCodeService) PollAndConsumeDeviceCode(
	ctx context.Context, realmName, deviceCode, clientID string,
) (types.ApprovedDeviceCode, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthDeviceCodeSVC, "PollAndConsumeDeviceCode")

	dc, err := s.deviceCodeManager.GetByDeviceCode(ctx, deviceCode)
	if err != nil {
		return types.ApprovedDeviceCode{}, errorWrapf(err, "deviceCodeManager.GetByDeviceCode fail")
	}

	if dc.ID == 0 {
		return types.ApprovedDeviceCode{}, oauth.ErrInvalidDeviceCode
	}

	if dc.RealmName != realmName {
		return types.ApprovedDeviceCode{}, oauth.ErrRealmMismatch
	}

	if dc.ClientID != clientID {
		return types.ApprovedDeviceCode{}, oauth.ErrDeviceCodeClientMatch
	}

	if time.Now().After(dc.ExpiresAt) {
		return types.ApprovedDeviceCode{}, oauth.ErrDeviceCodeExpired
	}

	// RFC 8628 §3.5: if the client polls before the current interval elapses,
	// increase the poll interval to throttle it and return slow_down.
	if dc.LastPolledAt != nil {
		elapsed := time.Since(*dc.LastPolledAt)
		if elapsed < time.Duration(dc.PollInterval)*time.Second {
			// SlowDown failure is intentionally not propagated: the client is already
			// polling too fast, so it must receive slow_down regardless of DB errors.
			// If the interval fails to increase, the next poll will trigger slow_down again.
			// TODO: log the error once the logging convention is established
			_, _ = s.deviceCodeManager.SlowDown(ctx, dc.ID, oauth.SlowDownIncrement)
			return types.ApprovedDeviceCode{}, oauth.ErrSlowDown
		}
	}

	// Best-effort update; failure does not affect the correctness of the current response.
	// TODO: log the error once the logging convention is established
	_, _ = s.deviceCodeManager.UpdateLastPolledAt(ctx, dc.ID)

	switch dc.Status {
	case oauth.DeviceCodeStatusPending:
		return types.ApprovedDeviceCode{}, oauth.ErrAuthorizationPending
	case oauth.DeviceCodeStatusDenied:
		return types.ApprovedDeviceCode{}, oauth.ErrDeviceCodeDenied
	case oauth.DeviceCodeStatusConsumed:
		return types.ApprovedDeviceCode{}, oauth.ErrDeviceCodeConsumed
	case oauth.DeviceCodeStatusApproved:
		// fall through to consume
	default:
		return types.ApprovedDeviceCode{}, oauth.ErrInvalidDeviceCode
	}

	// Optimistic lock: only the first request to CAS (approved -> consumed) succeeds.
	// rowsAffected==0 means another concurrent request already consumed this device code.
	//
	// Other theoretically possible causes (code deleted by TTL cleanup, or code
	// expired between Get and UPDATE) are not handled here because:
	//   - TTL cleanup runs at coarse intervals (minutes/hours), never races with in-flight requests.
	//   - Expiry was already validated above; the sub-millisecond window is negligible.
	rowsAffected, err := s.deviceCodeManager.ConsumeApproved(ctx, deviceCode, clientID)
	if err != nil {
		return types.ApprovedDeviceCode{}, errorWrapf(err, "deviceCodeManager.ConsumeApproved fail")
	}
	if rowsAffected == 0 {
		return types.ApprovedDeviceCode{}, oauth.ErrDeviceCodeConsumed
	}

	approved := types.ApprovedDeviceCode{
		TenantID: dc.TenantID,
		Sub:      dc.Sub,
		Username: dc.Username,
	}
	if dc.Audience != nil {
		if err := json.Unmarshal([]byte(*dc.Audience), &approved.Audience); err != nil {
			return types.ApprovedDeviceCode{}, errorWrapf(err, "json.Unmarshal audience fail")
		}
	}
	return approved, nil
}
