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

	Describe("Name and TokenPrefix", func() {
		It("should return correct values", func() {
			assert.Equal(GinkgoT(), "bk-devops", r.Name())
			assert.Equal(GinkgoT(), "bkci_", r.TokenPrefix())
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
})
