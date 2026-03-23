package impls

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/cache"
	"bkauth/pkg/cache/redis"
	"bkauth/pkg/service/mock"
	"bkauth/pkg/service/types"
)

var _ = Describe("AccessTokenCache", func() {
	BeforeEach(func() {
		cli := newTestRedisClient()
		AccessTokenCache = redis.NewMockCache(cli, "mockAccessToken", 5*time.Minute)
	})

	It("Key", func() {
		key := AccessTokenHashKey{TokenHash: "abc123"}
		assert.Equal(GinkgoT(), "abc123", key.Key())
	})

	Context("GetAccessTokenByTokenHash", func() {
		var ctl *gomock.Controller
		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})
		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			mockSvc := mock.NewMockOAuthTokenService(ctl)
			mockSvc.EXPECT().
				GetAccessTokenByTokenHash(gomock.Any(), "hash1").
				Return(types.ResolvedAccessToken{
					ClientID:  "client-1",
					RealmName: "blueking",
					Sub:       "user-1",
					Username:  "admin",
					Audience:  []string{"aud-1"},
					ExpiresAt: 1700000000,
				}, nil).
				AnyTimes()

			origRetrieve := retrieveAccessTokenByTokenHash
			retrieveAccessTokenByTokenHash = func(ctx context.Context, key cache.Key) (interface{}, error) {
				k := key.(AccessTokenHashKey)
				return mockSvc.GetAccessTokenByTokenHash(ctx, k.TokenHash)
			}
			defer func() { retrieveAccessTokenByTokenHash = origRetrieve }()

			token, err := GetAccessTokenByTokenHash(context.Background(), "hash1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "client-1", token.ClientID)
			assert.Equal(GinkgoT(), "blueking", token.RealmName)
			assert.Equal(GinkgoT(), "user-1", token.Sub)
			assert.Equal(GinkgoT(), "admin", token.Username)
			assert.Equal(GinkgoT(), []string{"aud-1"}, token.Audience)
			assert.Equal(GinkgoT(), int64(1700000000), token.ExpiresAt)
		})

		It("error", func() {
			mockSvc := mock.NewMockOAuthTokenService(ctl)
			mockSvc.EXPECT().
				GetAccessTokenByTokenHash(gomock.Any(), "hash1").
				Return(types.ResolvedAccessToken{}, errors.New("db error")).
				AnyTimes()

			origRetrieve := retrieveAccessTokenByTokenHash
			retrieveAccessTokenByTokenHash = func(ctx context.Context, key cache.Key) (interface{}, error) {
				k := key.(AccessTokenHashKey)
				return mockSvc.GetAccessTokenByTokenHash(ctx, k.TokenHash)
			}
			defer func() { retrieveAccessTokenByTokenHash = origRetrieve }()

			_, err := GetAccessTokenByTokenHash(context.Background(), "hash1")
			assert.Error(GinkgoT(), err)
		})

		It("not found returns zero value", func() {
			mockSvc := mock.NewMockOAuthTokenService(ctl)
			mockSvc.EXPECT().
				GetAccessTokenByTokenHash(gomock.Any(), "hash-missing").
				Return(types.ResolvedAccessToken{}, nil).
				AnyTimes()

			origRetrieve := retrieveAccessTokenByTokenHash
			retrieveAccessTokenByTokenHash = func(ctx context.Context, key cache.Key) (interface{}, error) {
				k := key.(AccessTokenHashKey)
				return mockSvc.GetAccessTokenByTokenHash(ctx, k.TokenHash)
			}
			defer func() { retrieveAccessTokenByTokenHash = origRetrieve }()

			token, err := GetAccessTokenByTokenHash(context.Background(), "hash-missing")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "", token.ClientID)
			assert.Equal(GinkgoT(), int64(0), token.ExpiresAt)
		})
	})

	It("DeleteAccessTokenCache", func() {
		err := DeleteAccessTokenCache(context.Background(), "hash1")
		assert.NoError(GinkgoT(), err)
	})
})
