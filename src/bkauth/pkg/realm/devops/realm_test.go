package devops_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/oauth"
	"bkauth/pkg/realm/devops"
)

var _ = Describe("devopsRealm", func() {
	var r oauth.Realm
	BeforeEach(func() {
		r = devops.New()
	})
	ctx := context.Background()

	Describe("Name", func() {
		It("should return correct value", func() {
			assert.Equal(GinkgoT(), "bk-devops", r.Name())
		})
	})

	Describe("ValidateResource", func() {
		It("should accept valid service resource", func() {
			assert.NoError(GinkgoT(), r.ValidateResource(ctx, "service:codecc"))
		})

		It("should accept multiple services", func() {
			assert.NoError(GinkgoT(), r.ValidateResource(ctx, "service:codecc,service:pipeline"))
		})

		It("should error on invalid prefix", func() {
			assert.Error(GinkgoT(), r.ValidateResource(ctx, "mcp:foo"))
		})

		It("should error on empty name", func() {
			assert.Error(GinkgoT(), r.ValidateResource(ctx, "service:"))
		})

		It("should error on empty input", func() {
			assert.Error(GinkgoT(), r.ValidateResource(ctx, ""))
		})
	})

	Describe("ExtractAudiences", func() {
		It("should dedup audiences", func() {
			aud, err := r.ExtractAudiences(ctx, "service:codecc,service:pipeline,service:codecc")
			require.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{"service:codecc", "service:pipeline"}, aud)
		})
	})

	Describe("ResolveResourceDisplay", func() {
		It("should parse a single service", func() {
			display, err := r.ResolveResourceDisplay(ctx, "service:codecc")
			require.NoError(GinkgoT(), err)
			groups := display.([]devops.ResourceDisplay)
			require.Len(GinkgoT(), groups, 1)
			assert.Equal(GinkgoT(), "service", groups[0].Type)
			assert.Equal(GinkgoT(), "蓝盾", groups[0].DisplayName)
			require.Len(GinkgoT(), groups[0].Items, 1)
			assert.Equal(GinkgoT(), "codecc", groups[0].Items[0].Name)
		})

		It("should parse multiple services", func() {
			display, err := r.ResolveResourceDisplay(ctx, "service:codecc,service:pipeline,service:turbo")
			require.NoError(GinkgoT(), err)
			groups := display.([]devops.ResourceDisplay)
			require.Len(GinkgoT(), groups, 1)
			assert.Len(GinkgoT(), groups[0].Items, 3)
		})

		It("should tolerate spaces and trailing commas", func() {
			display, err := r.ResolveResourceDisplay(ctx, " service:codecc , service:pipeline , ")
			require.NoError(GinkgoT(), err)
			groups := display.([]devops.ResourceDisplay)
			assert.Len(GinkgoT(), groups[0].Items, 2)
		})

		It("should error on empty input", func() {
			_, err := r.ResolveResourceDisplay(ctx, "")
			assert.Error(GinkgoT(), err)
		})
	})

	Describe("ValidateAudiences", func() {
		It("should accept service tokens", func() {
			assert.NoError(GinkgoT(), r.ValidateAudiences(ctx, []string{"service:codecc", "service:pipeline"}))
		})

		It("should reject an empty list", func() {
			assert.Error(GinkgoT(), r.ValidateAudiences(ctx, nil))
		})

		It("should reject a token without the service prefix", func() {
			assert.Error(GinkgoT(), r.ValidateAudiences(ctx, []string{"codecc"}))
		})

		It("should reject an empty service name", func() {
			assert.Error(GinkgoT(), r.ValidateAudiences(ctx, []string{"service:"}))
		})
	})

	Describe("ResolveAudienceDisplays", func() {
		It("should render one entry per token, keyed by that token", func() {
			displays, err := r.ResolveAudienceDisplays(ctx, "tencent",
				[]string{"service:codecc", "service:pipeline"})
			require.NoError(GinkgoT(), err)

			require.Len(GinkgoT(), displays, 2)
			assert.Equal(GinkgoT(), "service:codecc", displays["service:codecc"].Audience)
			assert.Equal(GinkgoT(), "service:pipeline", displays["service:pipeline"].Audience)
		})

		It("should carry the same name, display name and token as the catalog entry", func() {
			displays, err := r.ResolveAudienceDisplays(ctx, "", []string{"service:codecc"})
			require.NoError(GinkgoT(), err)

			entry := displays["service:codecc"]
			assert.Equal(GinkgoT(), "codecc", entry.Name)
			assert.Equal(GinkgoT(), "CodeCC", entry.DisplayName)
			assert.Equal(GinkgoT(), "service:codecc", entry.Audience)
			assert.Equal(GinkgoT(), "service", entry.Level,
				"a one-level catalog has no level above, and no all-services token either")
			assert.Equal(GinkgoT(), "service", entry.Type,
				"the same code GrantableResourceTypes reports, so 蓝盾 is looked up once")
		})

		It("should fall back to the bare name for a service it cannot name", func() {
			displays, err := r.ResolveAudienceDisplays(ctx, "", []string{"service:pipeline"})
			require.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "pipeline", displays["service:pipeline"].DisplayName,
				"an unlisted service is still a valid grant")
		})

		It("should render a repeated token once", func() {
			displays, err := r.ResolveAudienceDisplays(ctx, "",
				[]string{"service:codecc", "service:codecc"})
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
