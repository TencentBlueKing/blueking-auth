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
	"bkauth/pkg/util"
)

var _ = Describe("AppCache", func() {
	BeforeEach(func() {
		expiration := 5 * time.Minute
		cli := util.NewTestRedisClient()
		mockCache := redis.NewMockCache(cli, "mockCache", expiration)

		AppCache = mockCache
	})

	It("Key", func() {
		key := AppKey{
			AppCode: "test",
		}
		assert.Equal(GinkgoT(), key.Key(), "test")
	})

	Context("GetApp", func() {
		var ctl *gomock.Controller
		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})
		AfterEach(func() {
			ctl.Finish()
		})
		It("AppCache Get ok", func() {
			mockService := mock.NewMockAppService(ctl)
			mockApp := types.App{Code: "test"}
			mockService.EXPECT().Get(gomock.Any(), "test").Return(mockApp, nil).AnyTimes()

			origRetrieve := retrieveApp
			retrieveApp = func(ctx context.Context, key cache.Key) (interface{}, error) {
				k := key.(AppKey)
				return mockService.Get(ctx, k.AppCode)
			}
			defer func() { retrieveApp = origRetrieve }()

			app, err := GetApp(context.Background(), "test")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), app, mockApp)
		})
		It("AppCache Get fail", func() {
			mockService := mock.NewMockAppService(ctl)
			mockService.EXPECT().Get(gomock.Any(), "test").Return(types.App{}, errors.New("error")).AnyTimes()

			origRetrieve := retrieveApp
			retrieveApp = func(ctx context.Context, key cache.Key) (interface{}, error) {
				k := key.(AppKey)
				return mockService.Get(ctx, k.AppCode)
			}
			defer func() { retrieveApp = origRetrieve }()

			app, err := GetApp(context.Background(), "test")
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), app, types.App{})
		})
	})
})
