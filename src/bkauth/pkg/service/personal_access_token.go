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

	"go.uber.org/zap"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/logging"
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
	ErrPersonalTokenNotFound         = errors.New("personal access token not found")
	ErrPersonalTokenQuotaExceeded    = errors.New("personal access token active quota exceeded")
	ErrPersonalTokenInvalidExpiresAt = errors.New(
		"personal access token expires_at is outside the (now, now+max_ttl] window")
)

// PersonalAccessTokenService defines personal access token operations.
type PersonalAccessTokenService interface {
	Create(
		ctx context.Context, input types.CreatePersonalAccessTokenInput, policy types.PersonalTokenPolicy,
	) (types.CreatedPersonalAccessToken, error)
	// ListByOwner returns all of the owner's tokens in every lifecycle state. There
	// is no server-side state filter: retention caps the list at a size that is
	// cheap to return whole, and Revoked / ExpiresAt let the client group the rows
	// itself — including telling expired from revoked, a distinction a binary
	// active/inactive filter would have thrown away.
	ListByOwner(ctx context.Context, realmName, sub string) ([]types.PersonalAccessToken, error)
	GetByIDAndSub(ctx context.Context, realmName string, id int64, sub string) (types.PersonalAccessToken, error)
	// The three mutating methods return the affected token_hash so the handler can
	// invalidate the pat cache (Handler -> cache/impls is an allowed call).
	// Update takes the policy because it replaces the expiry as well, which puts
	// it under the same window and resurrection checks as Renew.
	Update(
		ctx context.Context, realmName string, id int64, sub string,
		input types.UpdatePersonalAccessTokenInput, policy types.PersonalTokenPolicy,
	) (string, error)
	// Renew overlaps Update, which replaces the expiry too. It stays because it
	// needs nothing but the new expiry: renewing through the replace-semantics
	// Update would make the caller resend name, description and audience, and a
	// caller working from a stale list would revert whatever changed meanwhile.
	Renew(
		ctx context.Context, realmName string, id int64, sub string,
		expiresAt int64, policy types.PersonalTokenPolicy,
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

// resolveExpiresAt checks a client-supplied absolute expiry against the policy
// window and converts it for the DAO.
//
// The window is (now, now+MaxTTL] measured by the server's clock, never the
// client's. A timestamp at or before now is rejected rather than clamped: it
// would mint a token that is already dead, and a client that sent one has a
// broken clock or a bug, both of which are better surfaced than absorbed.
//
// Callers return the error unwrapped so the handler can match it with errors.Is.
func resolveExpiresAt(expiresAt int64, policy types.PersonalTokenPolicy) (time.Time, error) {
	now := time.Now().UTC()
	if expiresAt <= now.Unix() || expiresAt > now.Unix()+policy.MaxTTL {
		return time.Time{}, ErrPersonalTokenInvalidExpiresAt
	}
	return time.Unix(expiresAt, 0).UTC(), nil
}

// guardResurrection runs the active quota check when a write would bring an
// expired token back to life.
//
// Without it the quota is deterministically bypassable: create N, let them
// expire, create N more, then push the expired ones back into the future for 2N
// active tokens. Both writes that can carry an expiry have to call this, or the
// one that does not becomes the bypass.
//
// Extending an already-active token is not resurrection -- it is already counted
// as active, so checking it would block a legitimate extension at the quota. A
// revoked token is not resurrection either: no write path can revive one, since
// every UPDATE carries a revoked = 0 guard. Both callers already refuse a revoked
// token before reaching this point; that arm stays so the function holds on its
// own rather than on the order its callers happen to check things in.
func (s *personalAccessTokenService) guardResurrection(
	ctx context.Context, realmName, sub string,
	token dao.PersonalAccessToken, policy types.PersonalTokenPolicy,
) error {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "guardResurrection")

	if token.Revoked || token.ExpiresAt.After(time.Now().UTC()) {
		return nil
	}

	count, err := s.manager.CountActiveByOwner(ctx, realmName, sub)
	if err != nil {
		return errorWrapf(err, "manager.CountActiveByOwner fail")
	}
	if count >= policy.MaxActivePerUser {
		return ErrPersonalTokenQuotaExceeded
	}
	return nil
}

func (s *personalAccessTokenService) Create(
	ctx context.Context, input types.CreatePersonalAccessTokenInput, policy types.PersonalTokenPolicy,
) (types.CreatedPersonalAccessToken, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "Create")

	expiresAt, err := resolveExpiresAt(input.ExpiresAt, policy)
	if err != nil {
		return types.CreatedPersonalAccessToken{}, err
	}

	// Quota is a soft anti-abuse bound, not a security boundary: the COUNT and the
	// INSERT below are not serialized, so concurrent creates may overshoot by one
	// or two. This is intentional and MUST NOT be "fixed" with a row lock or a
	// counter table.
	count, err := s.manager.CountActiveByOwner(ctx, input.RealmName, input.Sub)
	if err != nil {
		return types.CreatedPersonalAccessToken{}, errorWrapf(err, "manager.CountActiveByOwner fail")
	}
	if count >= policy.MaxActivePerUser {
		return types.CreatedPersonalAccessToken{}, ErrPersonalTokenQuotaExceeded
	}

	// Runs after the quota check on purpose: at most MaxActivePerUser rows are
	// active at this point, so an owner at the retention target always has
	// inactive rows to give up and can never be locked out of creating a token.
	s.evictOldestInactive(ctx, input.RealmName, input.Sub)

	raw, err := oauth.GeneratePersonalToken()
	if err != nil {
		return types.CreatedPersonalAccessToken{}, errorWrapf(err, "oauth.GeneratePersonalToken fail")
	}

	audienceJSON, err := json.Marshal(input.Audience)
	if err != nil {
		return types.CreatedPersonalAccessToken{}, errorWrapf(err, "json.Marshal audience fail")
	}

	daoToken := dao.PersonalAccessToken{
		Name:        input.Name,
		Description: input.Description,
		TokenHash:   oauth.HashToken(raw),
		TokenMask:   oauth.MaskToken(raw),
		RealmName:   input.RealmName,
		TenantID:    input.TenantID,
		Sub:         input.Sub,
		Username:    input.Username,
		Audience:    string(audienceJSON),
		ExpiresAt:   expiresAt,
		Revoked:     false,
	}

	id, err := s.manager.Create(ctx, daoToken)
	if err != nil {
		return types.CreatedPersonalAccessToken{}, errorWrapf(err, "manager.Create fail")
	}

	return types.CreatedPersonalAccessToken{
		ID:    id,
		Token: raw,
	}, nil
}

// evictOldestInactive trims an owner's oldest expired or revoked rows so the row
// count stays under the DAO's list cap. That is what lets ListByOwner return a
// whole list rather than a silently truncated window of one.
//
// Errors are logged and swallowed: this is housekeeping, and failing a token
// creation over it would be a far worse outcome than carrying a few extra rows.
// A transient failure heals on the next create, since the excess is recomputed
// from the current count every time.
func (s *personalAccessTokenService) evictOldestInactive(ctx context.Context, realmName, sub string) {
	logger := logging.GetSystemLogger()

	total, err := s.manager.CountByOwner(ctx, realmName, sub)
	if err != nil {
		logger.Warn("personal access token retention: count fail",
			zap.Error(err), zap.String("realm_name", realmName), zap.String("sub", sub))
		return
	}

	// +1 clears space for the row about to be inserted. Deriving the count from
	// the live total (rather than deleting a fixed number) also lets a single
	// sweep catch up after concurrent creates overshot the target.
	excess := total - dao.PersonalAccessTokenRetentionTarget + 1
	if excess <= 0 {
		return
	}

	if _, err := s.manager.DeleteOldestInactiveByOwner(ctx, realmName, sub, excess); err != nil {
		logger.Warn("personal access token retention: evict fail",
			zap.Error(err), zap.String("realm_name", realmName), zap.String("sub", sub))
	}
}

func (s *personalAccessTokenService) ListByOwner(
	ctx context.Context, realmName, sub string,
) ([]types.PersonalAccessToken, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "ListByOwner")

	daoTokens, err := s.manager.ListByOwner(ctx, realmName, sub)
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

// Update replaces every mutable field, the expiry included: PUT has no partial
// form, so there is no "unspecified" value to detect here.
//
// The expiry the row already holds is the one value exempt from the policy
// window. It has to be: an expired token's own expiry is in the past, and a
// client that echoes back the object it was handed must not be told the field it
// did not touch is now illegal -- an expired token would then be impossible to
// rename, which is exactly when fixing its name or audience before reviving it
// matters most.
func (s *personalAccessTokenService) Update(
	ctx context.Context, realmName string, id int64, sub string,
	input types.UpdatePersonalAccessTokenInput, policy types.PersonalTokenPolicy,
) (string, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "Update")

	// Read the row for its (immutable) token_hash, the expiry the resurrection
	// guard needs, and both rejections below.
	daoToken, err := s.manager.GetByIDAndSub(ctx, realmName, id, sub)
	if err != nil {
		return "", errorWrapf(err, "manager.GetByIDAndSub fail")
	}
	if daoToken.ID == 0 {
		return "", ErrPersonalTokenNotFound
	}
	// Revocation is terminal, and this is now where an edit of a revoked token is
	// refused. The UPDATE's own revoked = 0 guard still runs -- it, not this
	// check, is what makes the state precondition impossible to escape -- but its
	// affected row count can no longer report the rejection (see below).
	if daoToken.Revoked {
		return "", ErrPersonalTokenNotFound
	}

	// Compared in Unix seconds, the precision the client was given. On the
	// unchanged path the stored time is written back verbatim rather than
	// reconstructed from the client's seconds, so a rename cannot round away the
	// sub-second remainder the column carries.
	expiresAt := daoToken.ExpiresAt
	if input.ExpiresAt != daoToken.ExpiresAt.Unix() {
		if expiresAt, err = resolveExpiresAt(input.ExpiresAt, policy); err != nil {
			return "", err
		}
		if err = s.guardResurrection(ctx, realmName, sub, daoToken, policy); err != nil {
			return "", err
		}
	}

	audienceJSON, err := json.Marshal(input.Audience)
	if err != nil {
		return "", errorWrapf(err, "json.Marshal audience fail")
	}

	// The affected row count is deliberately not inspected. MySQL counts changed
	// rows, not matched ones, so a PUT that submits the stored values back --
	// opening the edit form and saving it untouched -- reports zero, which read as
	// "no such row" and answered an ordinary save with a 404.
	//
	// The two rejections above cover what that count used to stand for. What is
	// left is the row being revoked or evicted between the read and this
	// statement, a window of one round trip whose only cost is reporting success
	// for a write that did not land.
	if _, err = s.manager.UpdateByIDAndSub(
		ctx, realmName, id, sub, input.Name, input.Description, string(audienceJSON), expiresAt,
	); err != nil {
		return "", errorWrapf(err, "manager.UpdateByIDAndSub fail")
	}
	return daoToken.TokenHash, nil
}

func (s *personalAccessTokenService) Renew(
	ctx context.Context, realmName string, id int64, sub string,
	expiresAt int64, policy types.PersonalTokenPolicy,
) (string, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(PersonalAccessTokenSVC, "Renew")

	newExpiresAt, err := resolveExpiresAt(expiresAt, policy)
	if err != nil {
		return "", err
	}

	daoToken, err := s.manager.GetByIDAndSub(ctx, realmName, id, sub)
	if err != nil {
		return "", errorWrapf(err, "manager.GetByIDAndSub fail")
	}
	if daoToken.ID == 0 {
		return "", ErrPersonalTokenNotFound
	}
	// Same reasoning as in Update: the UPDATE still carries revoked = 0, but the
	// rejection has to be reported from here.
	if daoToken.Revoked {
		return "", ErrPersonalTokenNotFound
	}

	if err = s.guardResurrection(ctx, realmName, sub, daoToken, policy); err != nil {
		return "", err
	}

	// Renewing to the expiry the row already holds changes no column and so
	// reports zero affected rows, the same ambiguity Update has; the count is
	// discarded for the same reason.
	if _, err = s.manager.Renew(ctx, realmName, id, sub, newExpiresAt); err != nil {
		return "", errorWrapf(err, "manager.Renew fail")
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
// audience column and flattening the times to Unix seconds.
func toPersonalAccessToken(t dao.PersonalAccessToken) (types.PersonalAccessToken, error) {
	var audience []string
	if err := json.Unmarshal([]byte(t.Audience), &audience); err != nil {
		return types.PersonalAccessToken{}, err
	}

	token := types.PersonalAccessToken{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		TokenMask:   t.TokenMask,
		RealmName:   t.RealmName,
		Audience:    audience,
		Scope:       t.Scope,
		ExpiresAt:   t.ExpiresAt.Unix(),
		Revoked:     t.Revoked,
		CreatedAt:   t.CreatedAt.Unix(),
		UpdatedAt:   t.UpdatedAt.Unix(),
	}
	// A NULL revoked_at must stay 0, not the Unix epoch of the zero time.
	if t.RevokedAt.Valid {
		token.RevokedAt = t.RevokedAt.Time.Unix()
	}
	return token, nil
}
