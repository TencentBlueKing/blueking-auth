package gpu_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/oauth"
	"bkauth/pkg/realm/gpu"
)

var _ = Describe("gpuRealm", func() {
	var r oauth.Realm
	BeforeEach(func() {
		r = gpu.New()
	})
	ctx := context.Background()

	Describe("Name and TokenPrefix", func() {
		It("should return correct values", func() {
			assert.Equal(GinkgoT(), "bk-gpu", r.Name())
			assert.Equal(GinkgoT(), "bkgpu_", r.TokenPrefix())
		})
	})

	Describe("ValidateResource", func() {
		It("should accept resource:all", func() {
			assert.NoError(GinkgoT(), r.ValidateResource(ctx, "resource:all"))
		})

		It("should reject other resources", func() {
			assert.Error(GinkgoT(), r.ValidateResource(ctx, "service:foo"))
		})

		It("should reject empty input", func() {
			assert.Error(GinkgoT(), r.ValidateResource(ctx, ""))
		})
	})

	Describe("ExtractAudiences", func() {
		It("should return resource:all", func() {
			aud, err := r.ExtractAudiences(ctx, "resource:all")
			require.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{"resource:all"}, aud)
		})

		It("should error on invalid resource", func() {
			_, err := r.ExtractAudiences(ctx, "invalid")
			assert.Error(GinkgoT(), err)
		})
	})

	Describe("ResolveResourceDisplay", func() {
		It("should return correct display", func() {
			display, err := r.ResolveResourceDisplay(ctx, "resource:all")
			require.NoError(GinkgoT(), err)
			groups := display.([]gpu.ResourceDisplay)
			require.Len(GinkgoT(), groups, 1)
			assert.Equal(GinkgoT(), "resource", groups[0].Type)
			assert.Equal(GinkgoT(), "IEG GPU管理平台", groups[0].DisplayName)
			require.Len(GinkgoT(), groups[0].Items, 1)
			assert.Equal(GinkgoT(), "all", groups[0].Items[0].Name)
			assert.Equal(GinkgoT(), "所有", groups[0].Items[0].DisplayName)
		})

		It("should error on invalid resource", func() {
			_, err := r.ResolveResourceDisplay(ctx, "bad")
			assert.Error(GinkgoT(), err)
		})
	})
})
