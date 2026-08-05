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
	"errors"
	"time"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

const PersonalAccessTokenSVC = "PersonalAccessTokenSVC"

// Domain errors. The handler maps every one of these to a client-facing status;
// they are returned unwrapped so callers can match them with errors.Is.
//
// "not found" is deliberately overloaded: a token that does not exist, belongs to
// someone else, or is in a state the operation forbids all collapse to it, so the
// API never reveals whether a given id exists under another user's account.
var (
	ErrPersonalTokenNotFound      = errors.New("personal access token not found")
	ErrPersonalTokenQuotaExceeded = errors.New("personal access token active quota exceeded")
	ErrPersonalTokenInvalidTTL    = errors.New("personal access token expires_in is invalid or exceeds the maximum")
)

// PersonalAccessTokenService defines personal access token operations.
type PersonalAccessTokenService interface {
	Create(
		ctx context.Context, input types.CreatePersonalAccessTokenInput, policy types.PersonalTokenPolicy,
	) (types.CreatedPersonalAccessToken, error)
	ListByOwner(ctx context.Context, realmName, sub, state string) ([]types.PersonalAccessToken, error)
	GetByIDAndSub(ctx context.Context, realmName string, id int64, sub string) (types.PersonalAccessToken, error)
	// The three mutating methods return the affected token_hash so the handler can
	// invalidate the pat cache (Handler -> cache/impls is an allowed call).
	Update(ctx context.Context, input types.UpdatePersonalAccessTokenInput) (string, error)
	Renew(
		ctx context.Context, realmName string, id int64, sub string,
		expiresIn int64, policy types.PersonalTokenPolicy,
	) (string, error)
	Revoke(ctx context.Context, realmName string, id int64, sub string) (string, error)
	// GetByTokenHash resolves a token for introspection. Callers hash the raw
	// token first so it never enters this layer. A miss returns a zero-value
	// ResolvedAccessToken (ClientID == "") which is cacheable as a negative entry.
	GetByTokenHash(ctx context.Context, tokenHash string) (types.ResolvedAccessToken, error)
}

type personalAccessTokenService struct {
	manager dao.PersonalAccessTokenManager
}

// NewPersonalAccessTokenService creates a new PersonalAccessTokenService.
func NewPersonalAccessTokenService() PersonalAccessTokenService {
	return &personalAccessTokenService{
		manager: dao.NewPersonalAccessTokenManager(),
	}
}

func (s *personalAccessTokenService) Create(
	ctx context.Context, input types.CreatePersonalAccessTokenInput, policy types.PersonalTokenPolicy,
) (types.CreatedPersonalAccessToken, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "Create")

	if input.ExpiresIn <= 0 || input.ExpiresIn > policy.MaxTTL {
		return types.CreatedPersonalAccessToken{}, ErrPersonalTokenInvalidTTL
	}

	// Quota is a soft anti-abuse bound, not a security boundary: the COUNT and the
	// INSERT below are not serialized, so concurrent creates may overshoot by one
	// or two. This is intentional and MUST NOT be "fixed" with a row lock or a
	// counter table (design 4.3).
	count, err := s.manager.CountActiveByOwner(ctx, input.RealmName, input.Sub)
	if err != nil {
		return types.CreatedPersonalAccessToken{}, errorWrapf(err, "manager.CountActiveByOwner fail")
	}
	if count >= policy.MaxActivePerUser {
		return types.CreatedPersonalAccessToken{}, ErrPersonalTokenQuotaExceeded
	}

	raw, err := oauth.GeneratePersonalToken()
	if err != nil {
		return types.CreatedPersonalAccessToken{}, errorWrapf(err, "oauth.GeneratePersonalToken fail")
	}

	audienceJSON, err := json.Marshal(input.Audience)
	if err != nil {
		return types.CreatedPersonalAccessToken{}, errorWrapf(err, "json.Marshal audience fail")
	}

	expiresAt := time.Now().UTC().Add(time.Duration(input.ExpiresIn) * time.Second)

	daoToken := dao.PersonalAccessToken{
		TokenHash:   oauth.HashToken(raw),
		TokenMask:   oauth.MaskToken(raw),
		RealmName:   input.RealmName,
		TenantID:    input.TenantID,
		Sub:         input.Sub,
		Username:    input.Username,
		Name:        input.Name,
		Description: input.Description,
		Audience:    string(audienceJSON),
		ExpiresAt:   expiresAt,
		Revoked:     false,
	}

	id, err := s.manager.Create(ctx, daoToken)
	if err != nil {
		return types.CreatedPersonalAccessToken{}, errorWrapf(err, "manager.Create fail")
	}

	return types.CreatedPersonalAccessToken{
		PersonalAccessToken: types.PersonalAccessToken{
			ID:          id,
			TokenMask:   daoToken.TokenMask,
			RealmName:   input.RealmName,
			Name:        input.Name,
			Description: input.Description,
			Audience:    input.Audience,
			ExpiresAt:   expiresAt,
			Revoked:     false,
		},
		Token: raw,
	}, nil
}

func (s *personalAccessTokenService) ListByOwner(
	ctx context.Context, realmName, sub, state string,
) ([]types.PersonalAccessToken, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "ListByOwner")

	daoTokens, err := s.manager.ListByOwner(ctx, realmName, sub, state)
	if err != nil {
		return nil, errorWrapf(err, "manager.ListByOwner fail")
	}

	tokens := make([]types.PersonalAccessToken, 0, len(daoTokens))
	for _, t := range daoTokens {
		token, err := toPersonalAccessToken(t)
		if err != nil {
			return nil, errorWrapf(err, "toPersonalAccessToken fail")
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (s *personalAccessTokenService) GetByIDAndSub(
	ctx context.Context, realmName string, id int64, sub string,
) (types.PersonalAccessToken, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "GetByIDAndSub")

	daoToken, err := s.manager.GetByIDAndSub(ctx, realmName, id, sub)
	if err != nil {
		return types.PersonalAccessToken{}, errorWrapf(err, "manager.GetByIDAndSub fail")
	}
	if daoToken.ID == 0 {
		return types.PersonalAccessToken{}, ErrPersonalTokenNotFound
	}

	token, err := toPersonalAccessToken(daoToken)
	if err != nil {
		return types.PersonalAccessToken{}, errorWrapf(err, "toPersonalAccessToken fail")
	}
	return token, nil
}

func (s *personalAccessTokenService) Update(
	ctx context.Context, input types.UpdatePersonalAccessTokenInput,
) (string, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "Update")

	// Read the row only for its (immutable) token_hash and early existence check;
	// the state guard (revoked = 0) still lives in the UPDATE below, so this is not
	// a read-then-check-then-write.
	daoToken, err := s.manager.GetByIDAndSub(ctx, input.RealmName, input.ID, input.Sub)
	if err != nil {
		return "", errorWrapf(err, "manager.GetByIDAndSub fail")
	}
	if daoToken.ID == 0 {
		return "", ErrPersonalTokenNotFound
	}

	audienceJSON, err := json.Marshal(input.Audience)
	if err != nil {
		return "", errorWrapf(err, "json.Marshal audience fail")
	}

	rows, err := s.manager.UpdateByIDAndSub(
		ctx, input.RealmName, input.ID, input.Sub, input.Name, input.Description, string(audienceJSON),
	)
	if err != nil {
		return "", errorWrapf(err, "manager.UpdateByIDAndSub fail")
	}
	if rows == 0 {
		return "", ErrPersonalTokenNotFound
	}
	return daoToken.TokenHash, nil
}

func (s *personalAccessTokenService) Renew(
	ctx context.Context, realmName string, id int64, sub string,
	expiresIn int64, policy types.PersonalTokenPolicy,
) (string, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "Renew")

	if expiresIn <= 0 || expiresIn > policy.MaxTTL {
		return "", ErrPersonalTokenInvalidTTL
	}

	daoToken, err := s.manager.GetByIDAndSub(ctx, realmName, id, sub)
	if err != nil {
		return "", errorWrapf(err, "manager.GetByIDAndSub fail")
	}
	if daoToken.ID == 0 {
		return "", ErrPersonalTokenNotFound
	}

	// Renew must run the same quota check as Create, otherwise the quota is
	// deterministically bypassable: create N, let them expire, create N more, then
	// renew the expired ones back to life for 2N active tokens (design 13.1.3).
	// Only resurrection counts: renewing an already-active token merely extends it
	// and is itself already in the active count, so checking it would wrongly block
	// a legitimate extension. Quota semantics stay "active counts, expired doesn't".
	isExpired := !daoToken.Revoked && !daoToken.ExpiresAt.After(time.Now().UTC())
	if isExpired {
		count, err := s.manager.CountActiveByOwner(ctx, realmName, sub)
		if err != nil {
			return "", errorWrapf(err, "manager.CountActiveByOwner fail")
		}
		if count >= policy.MaxActivePerUser {
			return "", ErrPersonalTokenQuotaExceeded
		}
	}

	expiresAt := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
	rows, err := s.manager.Renew(ctx, realmName, id, sub, expiresAt)
	if err != nil {
		return "", errorWrapf(err, "manager.Renew fail")
	}
	if rows == 0 {
		return "", ErrPersonalTokenNotFound
	}
	return daoToken.TokenHash, nil
}

func (s *personalAccessTokenService) Revoke(
	ctx context.Context, realmName string, id int64, sub string,
) (string, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "Revoke")

	daoToken, err := s.manager.GetByIDAndSub(ctx, realmName, id, sub)
	if err != nil {
		return "", errorWrapf(err, "manager.GetByIDAndSub fail")
	}
	if daoToken.ID == 0 {
		return "", ErrPersonalTokenNotFound
	}

	rows, err := s.manager.Revoke(ctx, realmName, id, sub)
	if err != nil {
		return "", errorWrapf(err, "manager.Revoke fail")
	}
	if rows == 0 {
		// Already revoked (terminal). Report not found; the cache was already
		// invalidated by the first revoke.
		return "", ErrPersonalTokenNotFound
	}
	return daoToken.TokenHash, nil
}

func (s *personalAccessTokenService) GetByTokenHash(
	ctx context.Context, tokenHash string,
) (types.ResolvedAccessToken, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "GetByTokenHash")

	daoToken, err := s.manager.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return types.ResolvedAccessToken{}, errorWrapf(err, "manager.GetByTokenHash fail")
	}
	if daoToken.ID == 0 {
		return types.ResolvedAccessToken{}, nil
	}

	var audience []string
	if err := json.Unmarshal([]byte(daoToken.Audience), &audience); err != nil {
		return types.ResolvedAccessToken{}, errorWrapf(err, "json.Unmarshal audience fail")
	}

	// ClientID == PersonalAppCode makes introspection report
	// client_id == bk_app_code == personal with zero changes to the response
	// assembly (ResolveAppCode returns any non-dcr id unchanged).
	return types.ResolvedAccessToken{
		ClientID:  oauth.PersonalAppCode,
		RealmName: daoToken.RealmName,
		TenantID:  daoToken.TenantID,
		Sub:       daoToken.Sub,
		Username:  daoToken.Username,
		Audience:  audience,
		ExpiresAt: daoToken.ExpiresAt.Unix(),
		Revoked:   daoToken.Revoked,
	}, nil
}

// toPersonalAccessToken maps a DAO row to the management view, decoding the JSON
// audience column and the nullable time columns.
func toPersonalAccessToken(t dao.PersonalAccessToken) (types.PersonalAccessToken, error) {
	var audience []string
	if err := json.Unmarshal([]byte(t.Audience), &audience); err != nil {
		return types.PersonalAccessToken{}, err
	}

	token := types.PersonalAccessToken{
		ID:          t.ID,
		TokenMask:   t.TokenMask,
		RealmName:   t.RealmName,
		Name:        t.Name,
		Description: t.Description,
		Audience:    audience,
		ExpiresAt:   t.ExpiresAt,
		Revoked:     t.Revoked,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if t.RevokedAt.Valid {
		token.RevokedAt = &t.RevokedAt.Time
	}
	if t.LastUsedAt.Valid {
		token.LastUsedAt = &t.LastUsedAt.Time
	}
	return token, nil
}
