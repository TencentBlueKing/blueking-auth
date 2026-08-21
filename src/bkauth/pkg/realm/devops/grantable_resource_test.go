package devops_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bkauth/pkg/oauth"
	"bkauth/pkg/realm/devops"
)

var _ = Describe("devopsRealm grantable resources", func() {
	var r oauth.Realm
	BeforeEach(func() {
		r = devops.New()
	})
	ctx := context.Background()

	Describe("GrantableResourceTypes", func() {
		It("should offer one type with no select-all box", func() {
			resourceTypes := r.GrantableResourceTypes()

			require.Len(GinkgoT(), resourceTypes, 1)
			assert.Equal(GinkgoT(), "service", resourceTypes[0].Name)
			assert.Equal(GinkgoT(), "蓝盾", resourceTypes[0].DisplayName)
			assert.Empty(GinkgoT(), resourceTypes[0].Audience,
				"the grammar has no all-services token for a select-all box to contribute")
		})

		It("should describe a one-level catalog labelled by what the level holds", func() {
			levels := r.GrantableResourceTypes()[0].Levels

			require.Len(GinkgoT(), levels, 1, "one level means a flat list, no expander")
			assert.Equal(GinkgoT(), "service", levels[0].Name)
			assert.Equal(GinkgoT(), "服务", levels[0].DisplayName,
				"the level holds services; 「蓝盾」 labels the type, not this")
		})
	})

	Describe("ListGrantableResource", func() {
		It("should return the compiled-in services", func() {
			page, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "service"})
			require.NoError(GinkgoT(), err)

			require.NotEmpty(GinkgoT(), page.Results)
			assert.Equal(GinkgoT(), len(page.Results), page.Count)
			assert.Equal(GinkgoT(), "codecc", page.Results[0].Name)
			assert.Equal(GinkgoT(), "CodeCC", page.Results[0].DisplayName)
			assert.Equal(GinkgoT(), "service:codecc", page.Results[0].Audience)
			assert.Empty(GinkgoT(), page.Results[0].Items, "the realm has no group level")
		})

		It("should emit tokens ValidateAudiences accepts", func() {
			page, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "service"})
			require.NoError(GinkgoT(), err)

			for _, entry := range page.Results {
				assert.NoError(GinkgoT(), r.ValidateAudiences(ctx, []string{entry.Audience}))
			}
		})

		It("should order entries the same way on every call, since the source is a map", func() {
			first, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "service"})
			require.NoError(GinkgoT(), err)

			for i := 0; i < 20; i++ {
				again, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "service"})
				require.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), first.Results, again.Results)
			}
		})

		It("should name the same services the display path renders", func() {
			page, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "service"})
			require.NoError(GinkgoT(), err)
			require.NotEmpty(GinkgoT(), page.Results)

			display, err := r.ResolveResourceDisplay(ctx, "service:"+page.Results[0].Name)
			require.NoError(GinkgoT(), err)
			groups := display.([]devops.ResourceDisplay)

			require.Len(GinkgoT(), groups, 1)
			require.Len(GinkgoT(), groups[0].Items, 1)
			assert.Equal(GinkgoT(), groups[0].Items[0].DisplayName, page.Results[0].DisplayName)
		})

		It("should reject an unknown type as a sentinel the web layer can map to 400", func() {
			_, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "gateway"})

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
				Type:     "service",
				Keywords: map[string]string{level: "codecc"},
			})
			assert.NoError(GinkgoT(), err)

			_, err = r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{
				Type:     "service",
				Keywords: map[string]string{"gateway": "codecc"},
			})
			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceLevel)
		})
	})

	Describe("ResolveGrantableResource", func() {
		resolve := func(name string) (oauth.GrantableResource, error) {
			return r.ResolveGrantableResource(ctx, "", oauth.GrantableResourceRef{
				Type:  "service",
				Names: map[string]string{"service": name},
			})
		}

		It("should return the same entry the listing offers", func() {
			page, err := r.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "service"})
			require.NoError(GinkgoT(), err)
			require.NotEmpty(GinkgoT(), page.Results)

			resource, err := resolve(page.Results[0].Name)
			require.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), page.Results[0], resource)
		})

		It("should reject an unknown type", func() {
			_, err := r.ResolveGrantableResource(ctx, "", oauth.GrantableResourceRef{
				Type:  "gateway",
				Names: map[string]string{"service": "codecc"},
			})

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceType)
		})

		It("should resolve only the named services, though any would be storable", func() {
			// ValidateAudiences takes service:<anything>, and that stays true: an
			// unnamed service is grantable, it just has no entry to hand back.
			_, err := resolve("unlisted-service")
			assert.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)

			assert.NoError(GinkgoT(), r.ValidateAudiences(ctx, []string{"service:unlisted-service"}))
		})
	})
})
