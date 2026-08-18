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

	Describe("Name", func() {
		It("should return correct value", func() {
			assert.Equal(GinkgoT(), "bk-gpu", r.Name())
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
			assert.Equal(GinkgoT(), "IEG GPU 管理平台", groups[0].DisplayName)
			require.Len(GinkgoT(), groups[0].Items, 1)
			assert.Equal(GinkgoT(), "all", groups[0].Items[0].Name)
			assert.Equal(GinkgoT(), "所有", groups[0].Items[0].DisplayName)
		})

		It("should error on invalid resource", func() {
			_, err := r.ResolveResourceDisplay(ctx, "bad")
			assert.Error(GinkgoT(), err)
		})
	})

	Describe("ValidateAudiences", func() {
		It("should accept exactly the one token", func() {
			assert.NoError(GinkgoT(), r.ValidateAudiences(ctx, []string{"resource:all"}))
		})

		It("should reject an empty list", func() {
			assert.Error(GinkgoT(), r.ValidateAudiences(ctx, nil))
		})

		It("should reject any other token", func() {
			assert.Error(GinkgoT(), r.ValidateAudiences(ctx, []string{"resource:some"}))
		})

		It("should reject the token repeated, which no caller should produce", func() {
			assert.Error(GinkgoT(), r.ValidateAudiences(ctx, []string{"resource:all", "resource:all"}))
		})
	})

	Describe("ResolveAudienceDisplays", func() {
		It("should render one entry carrying the token it renders", func() {
			displays, err := r.ResolveAudienceDisplays(ctx, "tencent", []string{"resource:all"})
			require.NoError(GinkgoT(), err)
			require.Len(GinkgoT(), displays, 1)

			entry := displays["resource:all"]
			assert.Equal(GinkgoT(), "all", entry.Name,
				"the bare identifier, same as the catalog entry's name")
			assert.Equal(GinkgoT(), "resource:all", entry.Audience)
			assert.Equal(GinkgoT(), "所有", entry.DisplayName)
			assert.Equal(GinkgoT(), "resource", entry.Level,
				"resource:all is the sole entry of a one-level catalog, not a select-all above it")
			assert.Equal(GinkgoT(), "resource", entry.Type,
				"a lone 所有 says nothing; the type carries the label above it")
		})

		It("should render the token repeated once, unlike ValidateAudiences", func() {
			// The cardinality rule governs what may be written. Re-imposing it on
			// the render path would turn a grant list stored before it into a page
			// that will not render at all.
			displays, err := r.ResolveAudienceDisplays(ctx, "",
				[]string{"resource:all", "resource:all"})
			require.NoError(GinkgoT(), err)
			assert.Len(GinkgoT(), displays, 1)
		})

		It("should return nothing for no tokens", func() {
			displays, err := r.ResolveAudienceDisplays(ctx, "", nil)
			require.NoError(GinkgoT(), err)
			assert.Empty(GinkgoT(), displays)
		})

		It("should error on an invalid token", func() {
			_, err := r.ResolveAudienceDisplays(ctx, "", []string{"bad"})
			assert.Error(GinkgoT(), err)
		})
	})
})
