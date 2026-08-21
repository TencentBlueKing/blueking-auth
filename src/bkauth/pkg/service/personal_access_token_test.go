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

import (
	"context"
	"database/sql"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
	"bkauth/pkg/oauth"
	"bkauth/pkg/service/types"
)

var _ = Describe("PersonalAccessTokenService", func() {
	const (
		realm = "blueking"
		sub   = "u-1"
	)
	policy := types.PersonalTokenPolicy{MaxTTL: 100, MaxActivePerUser: 2}

	var (
		ctl     *gomock.Controller
		manager *mock.MockPersonalAccessTokenManager
		svc     personalAccessTokenService
		ctx     context.Context
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		manager = mock.NewMockPersonalAccessTokenManager(ctl)
		svc = personalAccessTokenService{manager: manager}
		ctx = context.Background()
	})

	AfterEach(func() {
		ctl.Finish()
	})

	// Half a MaxTTL into the future: comfortably inside the window from both ends,
	// so a slow test run cannot drift it out.
	validExpiresAt := func() int64 {
		return time.Now().Unix() + policy.MaxTTL/2
	}

	baseCreateInput := func() types.CreatePersonalAccessTokenInput {
		return types.CreatePersonalAccessTokenInput{
			RealmName: realm,
			Sub:       sub,
			Name:      "ci",
			Audience:  []string{"mcp:demo"},
			ExpiresAt: validExpiresAt(),
		}
	}

	Describe("Create", func() {
		It("rejects a missing expires_at", func() {
			in := baseCreateInput()
			in.ExpiresAt = 0
			_, err := svc.Create(ctx, in, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidExpiresAt)
		})

		// A past timestamp is refused rather than clamped: it would mint a token
		// that is already dead, which is a client bug worth surfacing.
		It("rejects an expires_at in the past", func() {
			in := baseCreateInput()
			in.ExpiresAt = time.Now().Unix() - 1
			_, err := svc.Create(ctx, in, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidExpiresAt)
		})

		It("rejects an expires_at beyond MaxTTL", func() {
			in := baseCreateInput()
			in.ExpiresAt = time.Now().Unix() + policy.MaxTTL + 60
			_, err := svc.Create(ctx, in, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidExpiresAt)
		})

		It("rejects when the active quota is reached", func() {
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(policy.MaxActivePerUser, nil)
			_, err := svc.Create(ctx, baseCreateInput(), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenQuotaExceeded)
		})

		It("mints a token, stores the hash and returns the plaintext once", func() {
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(0, nil)
			manager.EXPECT().CountByOwner(gomock.Any(), realm, sub).Return(1, nil)
			in := baseCreateInput()
			var stored dao.PersonalAccessToken
			manager.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(dao.PersonalAccessToken{})).
				DoAndReturn(func(_ context.Context, t dao.PersonalAccessToken) (int64, error) {
					assert.NotEmpty(GinkgoT(), t.TokenHash)
					assert.NotEmpty(GinkgoT(), t.TokenMask)
					assert.Equal(GinkgoT(), sub, t.Sub)
					assert.False(GinkgoT(), t.Revoked)
					// Create does not echo the expiry back, so the persisted row is the
					// only place to check that the client's timestamp was stored
					// verbatim rather than recomputed from the server's clock.
					assert.Equal(GinkgoT(), in.ExpiresAt, t.ExpiresAt.Unix())
					stored = t
					return 42, nil
				})

			created, err := svc.Create(ctx, in, policy)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), int64(42), created.ID)
			assert.True(GinkgoT(), oauth.IsPersonalToken(created.Token))
			assert.Equal(GinkgoT(), 32, len(oauth.HashToken(created.Token)))
			// The plaintext is returned once and never stored: only its hash reaches the row.
			assert.Equal(GinkgoT(), oauth.HashToken(created.Token), stored.TokenHash)
		})
	})

	// Retention keeps the owner's row count under the DAO list cap, which is what
	// lets ListByOwner hand back a whole list instead of a silently truncated
	// window of one.
	Describe("Create retention sweep", func() {
		expectMint := func() {
			manager.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(dao.PersonalAccessToken{})).
				Return(int64(1), nil)
		}

		It("evicts nothing while the owner is below the target", func() {
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(0, nil)
			manager.EXPECT().CountByOwner(gomock.Any(), realm, sub).
				Return(dao.PersonalAccessTokenRetentionTarget-1, nil)
			expectMint()

			_, err := svc.Create(ctx, baseCreateInput(), policy)
			assert.NoError(GinkgoT(), err)
		})

		It("frees exactly one row when the owner sits on the target", func() {
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(0, nil)
			manager.EXPECT().CountByOwner(gomock.Any(), realm, sub).
				Return(dao.PersonalAccessTokenRetentionTarget, nil)
			manager.EXPECT().DeleteOldestInactiveByOwner(gomock.Any(), realm, sub, 1).Return(int64(1), nil)
			expectMint()

			_, err := svc.Create(ctx, baseCreateInput(), policy)
			assert.NoError(GinkgoT(), err)
		})

		It("catches up in one sweep after concurrent creates overshot the target", func() {
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(0, nil)
			manager.EXPECT().CountByOwner(gomock.Any(), realm, sub).
				Return(dao.PersonalAccessTokenRetentionTarget+4, nil)
			manager.EXPECT().DeleteOldestInactiveByOwner(gomock.Any(), realm, sub, 5).Return(int64(5), nil)
			expectMint()

			_, err := svc.Create(ctx, baseCreateInput(), policy)
			assert.NoError(GinkgoT(), err)
		})

		It("still mints the token when the sweep fails", func() {
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(0, nil)
			manager.EXPECT().CountByOwner(gomock.Any(), realm, sub).
				Return(dao.PersonalAccessTokenRetentionTarget, nil)
			manager.EXPECT().DeleteOldestInactiveByOwner(gomock.Any(), realm, sub, 1).
				Return(int64(0), errors.New("db down"))
			expectMint()

			_, err := svc.Create(ctx, baseCreateInput(), policy)
			assert.NoError(GinkgoT(), err)
		})
	})

	Describe("ListByOwner", func() {
		// There is no server-side filter, so the only logic worth pinning is the row
		// mapping: the audience JSON column, and revoked_at, whose NULL must surface
		// as a nil pointer rather than the zero time — a client cannot tell "never
		// revoked" from "revoked at year zero".
		It("maps every row, decoding audience and leaving revoked_at zero when unset", func() {
			revokedAt := time.Now().UTC().Add(-time.Minute)
			rows := []dao.PersonalAccessToken{
				{ID: 2, Audience: `["mcp:demo"]`, ExpiresAt: time.Now().UTC().Add(time.Hour)},
				{
					ID:        1,
					Audience:  `[]`,
					ExpiresAt: time.Now().UTC().Add(time.Hour),
					Revoked:   true,
					RevokedAt: sql.NullTime{Time: revokedAt, Valid: true},
				},
			}
			manager.EXPECT().ListByOwner(gomock.Any(), realm, sub).Return(rows, nil)

			tokens, err := svc.ListByOwner(ctx, realm, sub)
			assert.NoError(GinkgoT(), err)
			assert.Len(GinkgoT(), tokens, 2)

			assert.Equal(GinkgoT(), []string{"mcp:demo"}, tokens[0].Audience)
			assert.False(GinkgoT(), tokens[0].Revoked)
			assert.Zero(GinkgoT(), tokens[0].RevokedAt)

			assert.Equal(GinkgoT(), []string{}, tokens[1].Audience)
			assert.True(GinkgoT(), tokens[1].Revoked)
			assert.Equal(GinkgoT(), revokedAt.Unix(), tokens[1].RevokedAt)
		})

		It("propagates a malformed audience column as an error", func() {
			manager.EXPECT().ListByOwner(gomock.Any(), realm, sub).
				Return([]dao.PersonalAccessToken{{ID: 1, Audience: "not-json"}}, nil)

			_, err := svc.ListByOwner(ctx, realm, sub)
			assert.Error(GinkgoT(), err)
		})
	})

	Describe("Renew", func() {
		It("rejects an out-of-window expires_at before touching the DB", func() {
			_, err := svc.Renew(ctx, realm, 1, sub, time.Now().Unix()+policy.MaxTTL+60, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidExpiresAt)
		})

		// Rejected before the row is read, so unlike on Update there is not even a
		// stored value it could be compared against.
		It("rejects a missing expires_at", func() {
			_, err := svc.Renew(ctx, realm, 1, sub, 0, policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidExpiresAt)
		})

		It("stores the client's timestamp verbatim", func() {
			active := dao.PersonalAccessToken{
				ID: 1, TokenHash: "h", ExpiresAt: time.Now().UTC().Add(time.Hour),
			}
			target := validExpiresAt()
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(active, nil)
			manager.EXPECT().Renew(gomock.Any(), realm, int64(1), sub, gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ int64, _ string, at time.Time) (int64, error) {
					assert.Equal(GinkgoT(), target, at.Unix())
					return 1, nil
				})

			_, err := svc.Renew(ctx, realm, 1, sub, target, policy)
			assert.NoError(GinkgoT(), err)
		})

		It("maps a missing token to not found", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(dao.PersonalAccessToken{}, nil)
			_, err := svc.Renew(ctx, realm, 1, sub, validExpiresAt(), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})

		It("blocks resurrecting an expired token when the quota is full (bypass prevention)", func() {
			expired := dao.PersonalAccessToken{
				ID:        1,
				TokenHash: "h",
				ExpiresAt: time.Now().UTC().Add(-time.Hour),
				Revoked:   false,
			}
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(expired, nil)
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).Return(policy.MaxActivePerUser, nil)

			_, err := svc.Renew(ctx, realm, 1, sub, validExpiresAt(), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenQuotaExceeded)
		})

		It("extends an already-active token without a quota check even when full", func() {
			active := dao.PersonalAccessToken{
				ID:        1,
				TokenHash: "h",
				ExpiresAt: time.Now().UTC().Add(time.Hour),
				Revoked:   false,
			}
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(active, nil)
			manager.EXPECT().Renew(gomock.Any(), realm, int64(1), sub, gomock.Any()).Return(int64(1), nil)

			hash, err := svc.Renew(ctx, realm, 1, sub, validExpiresAt(), policy)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "h", hash)
		})

		// The UPDATE's revoked = 0 guard cannot report this any more, so the
		// rejection has to happen before the DAO is reached at all -- gomock fails
		// the test if Renew is called, which is the assertion.
		It("refuses a revoked token without issuing the UPDATE", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(
				dao.PersonalAccessToken{
					ID:        1,
					TokenHash: "h",
					ExpiresAt: time.Now().UTC().Add(time.Hour),
					Revoked:   true,
				}, nil)

			_, err := svc.Renew(ctx, realm, 1, sub, validExpiresAt(), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})

		// Renewing to the expiry already stored changes no column, and MySQL counts
		// changed rows: zero here is a no-op, not a missing row.
		It("succeeds when the DAO reports zero affected rows", func() {
			active := dao.PersonalAccessToken{
				ID:        1,
				TokenHash: "h",
				ExpiresAt: time.Now().UTC().Add(time.Hour),
			}
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(active, nil)
			manager.EXPECT().Renew(gomock.Any(), realm, int64(1), sub, gomock.Any()).Return(int64(0), nil)

			hash, err := svc.Renew(ctx, realm, 1, sub, validExpiresAt(), policy)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "h", hash)
		})
	})

	Describe("Revoke", func() {
		It("maps an already-revoked token (rows==0) to not found", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(dao.PersonalAccessToken{ID: 1, TokenHash: "h"}, nil)
			manager.EXPECT().Revoke(gomock.Any(), realm, int64(1), sub).Return(int64(0), nil)

			_, err := svc.Revoke(ctx, realm, 1, sub)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})

		It("returns the token hash on success for cache invalidation", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(dao.PersonalAccessToken{ID: 1, TokenHash: "h"}, nil)
			manager.EXPECT().Revoke(gomock.Any(), realm, int64(1), sub).Return(int64(1), nil)

			hash, err := svc.Revoke(ctx, realm, 1, sub)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "h", hash)
		})
	})

	Describe("GetByTokenHash", func() {
		It("returns a zero-value (client_id empty) resolved token on miss", func() {
			manager.EXPECT().GetByTokenHash(gomock.Any(), "h").Return(dao.PersonalAccessToken{}, nil)
			resolved, err := svc.GetByTokenHash(ctx, "h")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "", resolved.ClientID)
		})

		It("reports client_id == personal for a hit", func() {
			manager.EXPECT().GetByTokenHash(gomock.Any(), "h").Return(dao.PersonalAccessToken{
				ID:        1,
				RealmName: realm,
				Sub:       sub,
				Audience:  `["mcp:demo"]`,
				ExpiresAt: time.Now().UTC().Add(time.Hour),
			}, nil)

			resolved, err := svc.GetByTokenHash(ctx, "h")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), oauth.PersonalAppCode, resolved.ClientID)
			assert.Equal(GinkgoT(), []string{"mcp:demo"}, resolved.Audience)
		})
	})

	Describe("Update", func() {
		// PUT replaces the expiry too, so "unchanged" is only meaningful against a
		// stored row: every case needs one to compare the input with.
		storedWith := func(expiresAt time.Time) dao.PersonalAccessToken {
			return dao.PersonalAccessToken{ID: 1, TokenHash: "h", ExpiresAt: expiresAt}
		}
		inputWith := func(expiresAt int64) types.UpdatePersonalAccessTokenInput {
			return types.UpdatePersonalAccessTokenInput{
				Name: "x", Audience: []string{"mcp:demo"}, ExpiresAt: expiresAt,
			}
		}
		expectUpdate := func(expiresAt time.Time, rows int64) {
			manager.EXPECT().UpdateByIDAndSub(
				gomock.Any(), realm, int64(1), sub, "x", "", `["mcp:demo"]`, expiresAt,
			).Return(rows, nil)
		}

		It("maps a missing token to not found", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(dao.PersonalAccessToken{}, nil)

			_, err := svc.Update(ctx, realm, 1, sub, inputWith(validExpiresAt()), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})

		// Revocation is terminal, and the UPDATE's revoked = 0 guard can no longer
		// report that: its row count is zero for an unchanged save too. gomock
		// fails the test if UpdateByIDAndSub is called, which is the assertion.
		It("refuses a revoked token without issuing the UPDATE", func() {
			revoked := storedWith(time.Now().UTC().Add(time.Hour))
			revoked.Revoked = true
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(revoked, nil)

			_, err := svc.Update(ctx, realm, 1, sub, inputWith(revoked.ExpiresAt.Unix()), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenNotFound)
		})

		// The bug this file's zero-row handling was rewritten for: a user who opens
		// the edit form and saves it untouched submits the stored values back,
		// which changes no column and so is counted as zero affected rows.
		It("succeeds when the DAO reports zero affected rows", func() {
			active := storedWith(time.Now().UTC().Add(time.Hour))
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(active, nil)
			expectUpdate(active.ExpiresAt, 0)

			hash, err := svc.Update(ctx, realm, 1, sub, inputWith(active.ExpiresAt.Unix()), policy)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "h", hash)
		})

		It("returns the token hash on success for cache invalidation", func() {
			active := storedWith(time.Now().UTC().Add(time.Hour))
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(active, nil)
			expectUpdate(active.ExpiresAt, 1)

			hash, err := svc.Update(ctx, realm, 1, sub, inputWith(active.ExpiresAt.Unix()), policy)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "h", hash)
		})

		// The case the unchanged-value exemption exists for: a client that echoes
		// the object back can still rename an expired token, whose own expiry
		// would never pass the policy window. Matching the stored time.Time by
		// equality also pins that it is written back verbatim rather than rebuilt
		// from the client's whole seconds, which would drop the column's
		// sub-second remainder.
		It("renames an expired token when its own past expiry is echoed back", func() {
			expired := storedWith(time.Now().UTC().Add(-time.Hour))
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(expired, nil)
			expectUpdate(expired.ExpiresAt, 1)

			_, err := svc.Update(ctx, realm, 1, sub, inputWith(expired.ExpiresAt.Unix()), policy)
			assert.NoError(GinkgoT(), err)
		})

		It("passes a changed expires_at through to the DAO verbatim", func() {
			active := storedWith(time.Now().UTC().Add(time.Hour))
			in := inputWith(validExpiresAt())

			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).Return(active, nil)
			manager.EXPECT().UpdateByIDAndSub(
				gomock.Any(), realm, int64(1), sub, "x", "", `["mcp:demo"]`,
				gomock.AssignableToTypeOf(time.Time{}),
			).DoAndReturn(func(
				_ context.Context, _ string, _ int64, _, _, _, _ string, at time.Time,
			) (int64, error) {
				assert.Equal(GinkgoT(), in.ExpiresAt, at.Unix())
				return 1, nil
			})

			_, err := svc.Update(ctx, realm, 1, sub, in, policy)
			assert.NoError(GinkgoT(), err)
		})

		It("rejects an out-of-window expires_at", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(storedWith(time.Now().UTC().Add(time.Hour)), nil)

			_, err := svc.Update(ctx, realm, 1, sub, inputWith(time.Now().Unix()+policy.MaxTTL+60), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidExpiresAt)
		})

		// The exemption is for the stored value alone, not for 0: an omitted field
		// is a client that failed to send the whole object, not one asking for the
		// expiry to stay put.
		It("rejects a missing expires_at rather than treating it as unchanged", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(storedWith(time.Now().UTC().Add(time.Hour)), nil)

			_, err := svc.Update(ctx, realm, 1, sub, inputWith(0), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenInvalidExpiresAt)
		})

		// Without this, PUT would be the quota bypass that Renew's guard exists to
		// close: expire N tokens, then push them back into the future one by one.
		It("blocks reviving an expired token through the expiry field when the quota is full", func() {
			manager.EXPECT().GetByIDAndSub(gomock.Any(), realm, int64(1), sub).
				Return(storedWith(time.Now().UTC().Add(-time.Hour)), nil)
			manager.EXPECT().CountActiveByOwner(gomock.Any(), realm, sub).
				Return(policy.MaxActivePerUser, nil)

			_, err := svc.Update(ctx, realm, 1, sub, inputWith(validExpiresAt()), policy)
			assert.ErrorIs(GinkgoT(), err, ErrPersonalTokenQuotaExceeded)
		})
	})
})
