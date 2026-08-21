package gpu_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/oauth"
	"bkauth/pkg/realm/gpu"
)

var _ = Describe("gpuRealm grantable resources", func() {
	var r oauth.Realm
	BeforeEach(func() {
		r = gpu.New()
	})
	ctx := context.Background()

	Describe("GrantableResourceTypes", func() {
		It("should offer one type with no select-all box", func() {
			resourceTypes := r.GrantableResourceTypes()

			require.Len(GinkgoT(), resourceTypes, 1)
			assert.Equal(GinkgoT(), "resource", resourceTypes[0].Name)
			assert.Equal(GinkgoT(), "IEG GPU 管理平台", resourceTypes[0].DisplayName)
			assert.Empty(GinkgoT(), resourceTypes[0].Audience,
				"the sole entry carries resource:all, so the type must not offer it again")
		})

		It("should describe a one-level catalog labelled by what the level holds", func() {
			levels := r.GrantableResourceTypes()[0].Levels

			require.Len(GinkgoT(), levels, 1, "one level means a flat list, no expander")
			assert.Equal(GinkgoT(), "resource", levels[0].Name)
			assert.Equal(GinkgoT(), "资源", levels[0].DisplayName,
				"the level holds resources; 「IEG GPU 管理平台」 labels the type, not this")
		})
	})

	Describe("ListGrantableResource", func() {
		It("should return the single entry", func() {
			page, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "resource"})
			require.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), 1, page.Count)
			require.Len(GinkgoT(), page.Results, 1)
			assert.Equal(GinkgoT(), "all", page.Results[0].Name)
			assert.Equal(GinkgoT(), "所有", page.Results[0].DisplayName)
			assert.Equal(GinkgoT(), "resource:all", page.Results[0].Audience)
			assert.Empty(GinkgoT(), page.Results[0].Items, "the realm has no group level")
		})

		It("should emit a token ValidateAudiences accepts", func() {
			page, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "resource"})
			require.NoError(GinkgoT(), err)

			audiences := make([]string, 0, len(page.Results))
			for _, entry := range page.Results {
				audiences = append(audiences, entry.Audience)
			}
			assert.NoError(GinkgoT(), r.ValidateAudiences(ctx, audiences))
		})

		It("should name the same entry the display path renders", func() {
			page, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "resource"})
			require.NoError(GinkgoT(), err)

			display, err := r.ResolveResourceDisplay(ctx, "resource:all")
			require.NoError(GinkgoT(), err)
			groups := display.([]gpu.ResourceDisplay)

			require.Len(GinkgoT(), groups, 1)
			require.Len(GinkgoT(), groups[0].Items, 1)
			require.Len(GinkgoT(), page.Results, 1)
			assert.Equal(GinkgoT(), groups[0].Items[0].Name, page.Results[0].Name)
			assert.Equal(GinkgoT(), groups[0].Items[0].DisplayName, page.Results[0].DisplayName)
		})

		It("should reject an unknown type as a sentinel the web layer can map to 400", func() {
			_, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "mcp"})

			assert.True(GinkgoT(), errors.Is(err, oauth.ErrUnknownGrantableResourceType))
		})

		It("should reject an empty type rather than defaulting to its only one", func() {
			_, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{})

			assert.Error(GinkgoT(), err)
		})

		It("should filter on the level it declares and reject any other", func() {
			// The level is read back from the metadata rather than written out,
			// so this guards the pair: what the frontend is told to send, and
			// what accepts it.
			level := r.GrantableResourceTypes()[0].Levels[0].Name

			_, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{
				Type:     "resource",
				Keywords: map[string]string{level: "all"},
			})
			assert.NoError(GinkgoT(), err)

			_, err = r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{
				Type:     "resource",
				Keywords: map[string]string{"service": "all"},
			})
			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceLevel)
		})
	})

	Describe("ResolveGrantableResource", func() {
		It("should return the same entry the listing offers", func() {
			// Nothing is withheld from this catalog, so the two paths must not be
			// able to disagree about the realm's one entry.
			page, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "resource"})
			require.NoError(GinkgoT(), err)

			resource, err := r.ResolveGrantableResource(ctx, "", oauth.GrantableResourceRef{
				Type:  "resource",
				Names: map[string]string{"resource": "all"},
			})
			require.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), page.Results[0], resource)
		})

		It("should reject an unknown type", func() {
			_, err := r.ResolveGrantableResource(ctx, "", oauth.GrantableResourceRef{
				Type:  "mcp",
				Names: map[string]string{"resource": "all"},
			})

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceType)
		})

		It("should report a name the realm does not have as not found", func() {
			_, err := r.ResolveGrantableResource(ctx, "", oauth.GrantableResourceRef{
				Type:  "resource",
				Names: map[string]string{"resource": "some-gpu"},
			})

			assert.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
		})
	})

})
