package impls

import (
	"errors"
	"time"

	"github.com/agiledragon/gomonkey"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/cache/redis"
	"bkauth/pkg/service"
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
		var patches *gomonkey.Patches
		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})
		AfterEach(func() {
			ctl.Finish()
			patches.Reset()
		})
		It("AppCache Get ok", func() {
			mockService := mock.NewMockAppService(ctl)
			mockApp := types.App{Code: "test"}
			mockService.EXPECT().Get("test").Return(mockApp, nil).AnyTimes()

			patches = gomonkey.ApplyFunc(service.NewAppService,
				func() service.AppService {
					return mockService
				})

			app, err := GetApp("test")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), app, mockApp)
		})
		It("AppCache Get fail", func() {
			mockService := mock.NewMockAppService(ctl)
			mockService.EXPECT().Get("test").Return(types.App{}, errors.New("error")).AnyTimes()

			patches = gomonkey.ApplyFunc(service.NewAppService,
				func() service.AppService {
					return mockService
				})

			app, err := GetApp("test")
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), app, types.App{})
		})
	})
})
