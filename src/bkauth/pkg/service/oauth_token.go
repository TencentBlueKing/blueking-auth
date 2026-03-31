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

	"github.com/jmoiron/sqlx"

	"bkauth/pkg/database"
	"bkauth/pkg/database/dao"
	"bkauth/pkg/errorx"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

const OAuthTokenSVC = "OAuthTokenSVC"

// OAuthTokenService defines the interface for OAuth token operations.
type OAuthTokenService interface {
	IssueTokensForAuthorizationCode(
		ctx context.Context, realmName, clientID, tenantID, sub, username string,
		audience []string, policy types.TokenIssuancePolicy,
	) (types.TokenPair, error)
	IssueTokensForDeviceCode(
		ctx context.Context, realmName, clientID, tenantID, sub, username string,
		audience []string, policy types.TokenIssuancePolicy,
	) (types.TokenPair, error)
	RefreshAccessToken(
		ctx context.Context, realmName, refreshToken, clientID string,
		policy types.TokenIssuancePolicy,
	) (types.TokenPair, error)
	GetAccessTokenByTokenHash(ctx context.Context, tokenHash string) (types.ResolvedAccessToken, error)
	RevokeToken(ctx context.Context, tokenHash, clientID string) error
	RevokeByGrantID(ctx context.Context, grantID string) error
}

// oauthTokenService implements OAuthTokenService.
//
// LOCK ORDERING INVARIANT: when a single transaction updates rows in both
// oauth_refresh_token and oauth_access_token, it MUST lock refresh_token rows
// before access_token rows. All existing methods (RefreshAccessToken,
// revokeRefreshTokenWithCascadeTx, RevokeByGrantID) follow this order.
// Violating it will cause deadlocks under concurrent load.
type oauthTokenService struct {
	accessTokenManager  dao.OAuthAccessTokenManager
	refreshTokenManager dao.OAuthRefreshTokenManager
}

// NewOAuthTokenService creates a new OAuthTokenService.
func NewOAuthTokenService() OAuthTokenService {
	return &oauthTokenService{
		accessTokenManager:  dao.NewOAuthAccessTokenManager(),
		refreshTokenManager: dao.NewOAuthRefreshTokenManager(),
	}
}

// preparedTokenPair holds pre-generated random material and DAO structs for a
// token pair, ready to be persisted inside a caller-provided transaction.
// Separating preparation (pure CPU, no DB) from persistence (DB writes) lets
// callers control transaction boundaries while keeping lock duration minimal.
type preparedTokenPair struct {
	accessToken  string
	refreshToken string
	expiresIn    int64

	daoAccessToken  dao.OAuthAccessToken
	daoRefreshToken dao.OAuthRefreshToken
}

// prepareTokenPair generates all random material and builds DAO structs for
// a token pair. This is pure computation with no database access, so it can
// safely run outside a transaction to minimize lock hold time.
// rotationCount should be oauth.InitialRotationCount for initial issuance
// and old.RotationCount+1 for rotation.
func (s *oauthTokenService) prepareTokenPair(
	realmName, grantID, clientID, tenantID, sub, username string,
	audience []string, rotationCount int64,
	policy types.TokenIssuancePolicy,
) (preparedTokenPair, error) {
	now := time.Now()

	accessToken, err := oauth.GenerateToken(policy.Prefix)
	if err != nil {
		return preparedTokenPair{}, err
	}

	jti := oauth.GenerateJTI()

	audienceJSON, err := json.Marshal(audience)
	if err != nil {
		return preparedTokenPair{}, err
	}

	refreshToken, err := oauth.GenerateToken(policy.Prefix)
	if err != nil {
		return preparedTokenPair{}, err
	}

	return preparedTokenPair{
		accessToken:  accessToken,
		refreshToken: refreshToken,
		expiresIn:    policy.AccessTokenTTL,
		daoAccessToken: dao.OAuthAccessToken{
			JTI:       jti,
			TokenHash: oauth.HashToken(accessToken),
			TokenMask: oauth.MaskToken(accessToken),
			GrantID:   grantID,
			ClientID:  clientID,
			RealmName: realmName,
			TenantID:  tenantID,
			Sub:       sub,
			Username:  username,
			Audience:  string(audienceJSON),
			ExpiresAt: now.Add(time.Duration(policy.AccessTokenTTL) * time.Second),
			Revoked:   false,
		},
		daoRefreshToken: dao.OAuthRefreshToken{
			TokenHash:     oauth.HashToken(refreshToken),
			TokenMask:     oauth.MaskToken(refreshToken),
			GrantID:       grantID,
			ClientID:      clientID,
			RealmName:     realmName,
			TenantID:      tenantID,
			Sub:           sub,
			Username:      username,
			Audience:      string(audienceJSON),
			ExpiresAt:     now.Add(time.Duration(policy.RefreshTokenTTL) * time.Second),
			Revoked:       false,
			RotationCount: rotationCount,
		},
	}, nil
}

// persistTokenPairTx inserts both tokens within a caller-provided transaction.
// The access token is inserted first to obtain its ID, which is then set on
// the refresh token's AccessTokenID before the refresh token insert.
func (s *oauthTokenService) persistTokenPairTx(
	ctx context.Context, tx *sqlx.Tx, prepared *preparedTokenPair,
) error {
	accessTokenID, err := s.accessTokenManager.CreateWithTx(ctx, tx, prepared.daoAccessToken)
	if err != nil {
		return err
	}

	prepared.daoRefreshToken.AccessTokenID = accessTokenID

	if _, err := s.refreshTokenManager.CreateWithTx(ctx, tx, prepared.daoRefreshToken); err != nil {
		return err
	}
	return nil
}

// IssueTokensForAuthorizationCode issues tokens for a validated and consumed authorization code.
func (s *oauthTokenService) IssueTokensForAuthorizationCode(
	ctx context.Context,
	realmName, clientID, tenantID, sub, username string,
	audience []string, policy types.TokenIssuancePolicy,
) (types.TokenPair, error) {
	grantID := oauth.GenerateGrantID()
	return s.generateTokenPair(ctx, realmName, tenantID, grantID, clientID, sub, username, audience, policy)
}

// IssueTokensForDeviceCode issues tokens after a device code has been approved (RFC 8628).
func (s *oauthTokenService) IssueTokensForDeviceCode(
	ctx context.Context,
	realmName, clientID, tenantID, sub, username string,
	audience []string, policy types.TokenIssuancePolicy,
) (types.TokenPair, error) {
	grantID := oauth.GenerateGrantID()
	return s.generateTokenPair(ctx, realmName, tenantID, grantID, clientID, sub, username, audience, policy)
}

// generateTokenPair generates an access token and refresh token pair atomically.
// It manages its own transaction and always sets rotationCount=0 (initial issuance).
// For callers that need to embed token creation in a larger transaction or carry
// forward a rotation count, use prepareTokenPair + persistTokenPairTx directly.
func (s *oauthTokenService) generateTokenPair(
	ctx context.Context,
	realmName, tenantID, grantID, clientID, sub, username string,
	audience []string, policy types.TokenIssuancePolicy,
) (types.TokenPair, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthTokenSVC, "generateTokenPair")

	prepared, err := s.prepareTokenPair(
		realmName, grantID, clientID, tenantID, sub, username, audience, oauth.InitialRotationCount, policy,
	)
	if err != nil {
		return types.TokenPair{}, errorWrapf(err, "prepareTokenPair fail")
	}

	tx, err := database.GenerateDefaultDBTx(ctx)
	if err != nil {
		return types.TokenPair{}, errorWrapf(err, "database.GenerateDefaultDBTx fail")
	}
	defer database.RollBackWithLog(tx)

	if err := s.persistTokenPairTx(ctx, tx, &prepared); err != nil {
		return types.TokenPair{}, errorWrapf(err, "persistTokenPairTx fail")
	}
	if err := tx.Commit(); err != nil {
		return types.TokenPair{}, errorWrapf(err, "tx.Commit fail")
	}

	return types.TokenPair{
		AccessToken:  prepared.accessToken,
		ExpiresIn:    prepared.expiresIn,
		RefreshToken: prepared.refreshToken,
	}, nil
}

// GetAccessTokenByTokenHash retrieves an access token record by its token hash.
// Callers are responsible for hashing the raw token via oauth.HashToken before calling this method,
// so that the raw token never enters the service/cache layer.
// Returns a zero-value ResolvedAccessToken (ClientID == "") when the token does not exist;
// callers should check this to distinguish "not found" from a valid record.
// Revoked/expired checks are NOT performed here — callers decide how to interpret the token state.
func (s *oauthTokenService) GetAccessTokenByTokenHash(
	ctx context.Context, tokenHash string,
) (types.ResolvedAccessToken, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthTokenSVC, "GetAccessTokenByTokenHash")

	daoToken, err := s.accessTokenManager.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return types.ResolvedAccessToken{}, errorWrapf(err, "accessTokenManager.GetByTokenHash fail")
	}

	// not found — return zero-value struct so the result is cacheable as a negative entry
	if daoToken.ID == 0 {
		return types.ResolvedAccessToken{}, nil
	}

	var audience []string
	if err := json.Unmarshal([]byte(daoToken.Audience), &audience); err != nil {
		return types.ResolvedAccessToken{}, errorWrapf(err, "json.Unmarshal audience fail")
	}

	return types.ResolvedAccessToken{
		ClientID:  daoToken.ClientID,
		RealmName: daoToken.RealmName,
		TenantID:  daoToken.TenantID,
		Sub:       daoToken.Sub,
		Username:  daoToken.Username,
		Audience:  audience,

		ExpiresAt: daoToken.ExpiresAt.Unix(),
		Revoked:   daoToken.Revoked,
	}, nil
}

// RefreshAccessToken rotates a refresh token: validates the presented token,
// revokes it together with its associated access token, and issues a fresh pair.
//
// # Token revocation policy (RFC 6819 / draft-ietf-oauth-security-topics)
//
// Per the OAuth 2.0 Security BCP, refresh token rotation MUST immediately
// invalidate the old refresh token upon use to prevent replay attacks.
// Invalidating the old access token is RECOMMENDED (the BCP says SHOULD for
// refresh-token revocation cascading to access tokens; some implementations
// allow the short-lived access token to expire naturally).
//
// We choose the stricter policy — both old tokens are revoked atomically in
// the same transaction that issues the new pair. This ensures:
//   - The old refresh token cannot be replayed after rotation.
//   - The old access token cannot be used after the holder has received a
//     replacement, closing the window for a stolen access token.
//   - The revoke-and-issue is all-or-nothing: if issuing the new pair fails,
//     the old tokens remain valid and the client can retry.
//
// # Why validation lives in the service layer
//
// The checks (existence, revocation, expiry, client ownership) are domain
// invariants of the token rotation operation, not HTTP-layer concerns. Keeping
// them here ensures that every caller — current and future (handlers, gRPC,
// cron jobs, tests) — gets the full security guarantee without duplicating
// validation logic. Moving them to the handler would make the service method
// "unsafe to call" and create an implicit, fragile contract that each caller
// must independently uphold.
//
// # Concurrency strategy — two-phase CAS
//
// Phase 1 (outside tx): read the refresh token and validate immutable
// attributes (existence, client ownership, expiry). These fields never change
// after creation, so the snapshot is reliable regardless of concurrent access.
// A token that is already revoked at this stage indicates a replay of a
// previously consumed token; the entire grant family is revoked defensively.
//
// Phase 2 (inside tx): atomically claim the token via
//
//	UPDATE ... SET revoked=1 WHERE id=? AND revoked=0
//
// If RowsAffected==0, a concurrent request consumed it first. The same tx
// then revokes the old access token and inserts the new pair, so the
// operation is all-or-nothing.
//
// Compared to SELECT ... FOR UPDATE:
//   - Equivalent safety: the CAS UPDATE serializes concurrent consumers.
//   - Lower lock duration: row lock held only for the UPDATE+INSERT span,
//     not for the validation phase.
//   - Cheaper rejection: invalid requests (not-found, expired, client mismatch)
//     are rejected before opening a transaction.
//   - Finer replay signal: "already revoked on read" (likely replay) vs
//     "CAS lost" (concurrent race) can be distinguished, enabling targeted
//     grant-family revocation only for the former.
func (s *oauthTokenService) RefreshAccessToken(
	ctx context.Context,
	realmName, refreshToken, clientID string, policy types.TokenIssuancePolicy,
) (types.TokenPair, error) {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthTokenSVC, "RefreshAccessToken")

	// ---- Phase 1: read + immutable-attribute validation (no tx, no lock) ----

	tokenHash := oauth.HashToken(refreshToken)
	daoRefreshToken, err := s.refreshTokenManager.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return types.TokenPair{}, errorWrapf(err, "refreshTokenManager.GetByTokenHash fail")
	}

	if daoRefreshToken.ID == 0 {
		return types.TokenPair{}, oauth.ErrInvalidRefreshToken
	}

	// Realm and clientID are immutable attributes bound at token issuance.
	// Check them before any stateful validation (revoked / expired / rotation)
	// so that a request with wrong credentials cannot trigger side effects
	// such as grant-family revocation in the replay-detection path.
	if daoRefreshToken.RealmName != realmName {
		return types.TokenPair{}, oauth.ErrRealmMismatch
	}

	if daoRefreshToken.ClientID != clientID {
		return types.TokenPair{}, oauth.ErrClientMismatch
	}

	if daoRefreshToken.Revoked {
		// The token was already consumed before this request arrived.
		// This could be either (a) a legitimate client racing (e.g. timeout
		// retry, multiple tabs) or (b) an attacker replaying a stolen token.
		//
		// We use the time elapsed since revocation to distinguish: reuse
		// within the grace period is treated as a benign race; reuse after
		// the grace period is treated as a replay attack and triggers
		// family-wide revocation. See oauth.ReplayDetectionGracePeriod for
		// the full rationale and trade-off analysis.
		if time.Since(daoRefreshToken.UpdatedAt) > oauth.ReplayDetectionGracePeriod {
			_ = s.RevokeByGrantID(ctx, daoRefreshToken.GrantID)
		}
		return types.TokenPair{}, oauth.ErrRefreshTokenRevoked
	}

	// Expiry is checked after revocation intentionally: a token that is both
	// revoked and expired should still enter the replay-detection path above,
	// because presenting a long-dead token is a stronger signal of token
	// theft than natural expiration.
	if time.Now().After(daoRefreshToken.ExpiresAt) {
		return types.TokenPair{}, oauth.ErrRefreshTokenExpired
	}

	// NOTE: MaxRefreshTokenRotations is currently a constant, so the "> 0"
	// guard is always true. Keep it here because we plan to make this value
	// configurable per-environment; once it becomes a runtime config, the
	// zero-value will mean "unlimited rotations" and this guard will matter.
	if oauth.MaxRefreshTokenRotations > 0 && daoRefreshToken.RotationCount >= oauth.MaxRefreshTokenRotations {
		// Proactively revoke the entire grant family so that the current
		// (still technically valid) refresh token and its associated access
		// token cannot be used again. Without this, the token pair would
		// remain valid until natural expiry, leaving an open window despite
		// the rotation limit being reached.
		_ = s.RevokeByGrantID(ctx, daoRefreshToken.GrantID)
		return types.TokenPair{}, oauth.ErrRotationLimitExceeded
	}

	var audience []string
	if err := json.Unmarshal([]byte(daoRefreshToken.Audience), &audience); err != nil {
		return types.TokenPair{}, errorWrapf(err, "json.Unmarshal audience fail")
	}

	// Pre-generate all random material outside the transaction to minimize
	// the time the transaction holds locks.
	prepared, err := s.prepareTokenPair(
		realmName, daoRefreshToken.GrantID, clientID, daoRefreshToken.TenantID,
		daoRefreshToken.Sub, daoRefreshToken.Username,
		audience, daoRefreshToken.RotationCount+1,
		policy,
	)
	if err != nil {
		return types.TokenPair{}, errorWrapf(err, "prepareTokenPair fail")
	}

	// ---- Phase 2: single tx { CAS revoke old + issue new } ----

	tx, err := database.GenerateDefaultDBTx(ctx)
	if err != nil {
		return types.TokenPair{}, errorWrapf(err, "database.GenerateDefaultDBTx fail")
	}
	defer database.RollBackWithLog(tx)

	// Optimistic lock via CAS: UPDATE … SET revoked=1 WHERE id=? AND revoked=0.
	// No explicit row lock (SELECT … FOR UPDATE) is taken beforehand; instead we
	// optimistically assume the token is still unclaimed and let the database's
	// own row-level lock during UPDATE arbitrate concurrent consumers.
	// RowsAffected==1 means we won; ==0 means another request claimed it first.
	rows, err := s.refreshTokenManager.RevokeIfNotRevokedWithTx(ctx, tx, daoRefreshToken.ID)
	if err != nil {
		return types.TokenPair{}, errorWrapf(err, "refreshTokenManager.RevokeIfNotRevokedWithTx fail")
	}
	if rows == 0 {
		return types.TokenPair{}, oauth.ErrRefreshTokenRevoked
	}

	if _, err := s.accessTokenManager.RevokeWithTx(ctx, tx, daoRefreshToken.AccessTokenID); err != nil {
		return types.TokenPair{}, errorWrapf(err, "accessTokenManager.RevokeWithTx fail")
	}

	if err := s.persistTokenPairTx(ctx, tx, &prepared); err != nil {
		return types.TokenPair{}, errorWrapf(err, "persistTokenPairTx fail")
	}

	if err := tx.Commit(); err != nil {
		return types.TokenPair{}, errorWrapf(err, "tx.Commit fail")
	}

	return types.TokenPair{
		AccessToken:  prepared.accessToken,
		ExpiresIn:    prepared.expiresIn,
		RefreshToken: prepared.refreshToken,
	}, nil
}

// revokeRefreshTokenWithCascadeTx revokes a refresh token and its associated
// access token within a caller-provided transaction.
func (s *oauthTokenService) revokeRefreshTokenWithCascadeTx(
	ctx context.Context, tx *sqlx.Tx,
	refreshTokenID, accessTokenID int64,
) error {
	if _, err := s.refreshTokenManager.RevokeWithTx(ctx, tx, refreshTokenID); err != nil {
		return err
	}
	if _, err := s.accessTokenManager.RevokeWithTx(ctx, tx, accessTokenID); err != nil {
		return err
	}
	return nil
}

// revokeRefreshTokenWithCascade atomically revokes a refresh token and its
// associated access token. It manages its own transaction; used by RevokeToken.
func (s *oauthTokenService) revokeRefreshTokenWithCascade(
	ctx context.Context, refreshTokenID, accessTokenID int64,
) error {
	tx, err := database.GenerateDefaultDBTx(ctx)
	if err != nil {
		return err
	}
	defer database.RollBackWithLog(tx)

	if err := s.revokeRefreshTokenWithCascadeTx(ctx, tx, refreshTokenID, accessTokenID); err != nil {
		return err
	}
	return tx.Commit()
}

// RevokeToken revokes an access token or refresh token (RFC 7009).
// Lookup order: access_token table first, then refresh_token table.
// Revoking a refresh_token cascades to its associated access_token (RFC 7009 SHOULD);
// revoking an access_token does NOT cascade to its refresh_token (RFC 7009 MAY — we opt out).
//
// Per RFC 7009 Section 2.1, the method always returns nil (success) for non-infrastructure errors
// — including token-not-found, client mismatch, and already-revoked cases —
// to prevent callers from probing token existence.
func (s *oauthTokenService) RevokeToken(ctx context.Context, tokenHash, clientID string) error {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthTokenSVC, "RevokeToken")

	accessToken, err := s.accessTokenManager.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return errorWrapf(err, "accessTokenManager.GetByTokenHash fail")
	}
	if accessToken.ID != 0 {
		if clientID != "" && accessToken.ClientID != clientID {
			return nil
		}
		if accessToken.Revoked {
			return nil
		}
		if _, err := s.accessTokenManager.Revoke(ctx, accessToken.ID); err != nil {
			return errorWrapf(err, "accessTokenManager.Revoke fail")
		}
		return nil
	}

	refreshToken, err := s.refreshTokenManager.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return errorWrapf(err, "refreshTokenManager.GetByTokenHash fail")
	}
	if refreshToken.ID != 0 {
		if clientID != "" && refreshToken.ClientID != clientID {
			return nil
		}
		if refreshToken.Revoked {
			return nil
		}

		if err := s.revokeRefreshTokenWithCascade(
			ctx,
			refreshToken.ID,
			refreshToken.AccessTokenID,
		); err != nil {
			return errorWrapf(err, "revokeRefreshTokenWithCascade fail")
		}
		return nil
	}

	return nil
}

// RevokeByGrantID revokes all tokens in a token family (same grant).
//
// Lock ordering: refresh_token table first, then access_token table.
// This matches the order used by RefreshAccessToken and
// revokeRefreshTokenWithCascadeTx to prevent deadlocks when concurrent
// requests operate on tokens within the same grant family.
func (s *oauthTokenService) RevokeByGrantID(ctx context.Context, grantID string) error {
	errorWrapf := errorx.NewLayerFunctionErrorWrapf(OAuthTokenSVC, "RevokeByGrantID")

	tx, err := database.GenerateDefaultDBTx(ctx)
	if err != nil {
		return errorWrapf(err, "database.GenerateDefaultDBTx fail")
	}
	defer database.RollBackWithLog(tx)

	if _, err := s.refreshTokenManager.RevokeByGrantIDWithTx(ctx, tx, grantID); err != nil {
		return errorWrapf(err, "refreshTokenManager.RevokeByGrantIDWithTx fail")
	}
	if _, err := s.accessTokenManager.RevokeByGrantIDWithTx(ctx, tx, grantID); err != nil {
		return errorWrapf(err, "accessTokenManager.RevokeByGrantIDWithTx fail")
	}
	if err := tx.Commit(); err != nil {
		return errorWrapf(err, "tx.Commit fail")
	}

	return nil
}
