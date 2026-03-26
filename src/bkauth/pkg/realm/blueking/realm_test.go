package blueking

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/external/bkapigateway/mock"
	"bkauth/pkg/oauth"
)

var _ = Describe("bluekingRealm", func() {
	var r oauth.Realm
	BeforeEach(func() {
		r = New()
	})
	ctx := context.Background()

	Describe("Name and TokenPrefix", func() {
		It("should return correct values", func() {
			assert.Equal(GinkgoT(), "blueking", r.Name())
			assert.Equal(GinkgoT(), "bk_", r.TokenPrefix())
		})
	})

	Describe("ValidateResource", func() {
		It("should accept valid MCP resource", func() {
			assert.NoError(GinkgoT(), r.ValidateResource(ctx, "mcp:s1"))
		})

		It("should accept valid gateway resource", func() {
			assert.NoError(GinkgoT(), r.ValidateResource(ctx, "gateway:gw:api:get_host"))
		})

		It("should accept URL format", func() {
			assert.NoError(GinkgoT(), r.ValidateResource(ctx, "https://bk.example.com/mcp-servers/s1/sse"))
		})

		It("should accept mixed resources", func() {
			assert.NoError(GinkgoT(), r.ValidateResource(ctx, "mcp:s1,gateway:gw:api:*"))
		})

		It("should error on empty input", func() {
			assert.Error(GinkgoT(), r.ValidateResource(ctx, ""))
		})

		It("should error on unknown prefix", func() {
			assert.Error(GinkgoT(), r.ValidateResource(ctx, "unknown:foo"))
		})

		It("should error on gateway without API segment", func() {
			assert.Error(GinkgoT(), r.ValidateResource(ctx, "gateway:gw"))
		})
	})

	Describe("ExtractAudiences", func() {
		It("should extract MCP audiences", func() {
			aud, err := r.ExtractAudiences(ctx, "mcp:s1,mcp:s2")
			require.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{"mcp:s1", "mcp:s2"}, aud)
		})

		It("should dedup gateway audiences", func() {
			aud, err := r.ExtractAudiences(ctx,
				"gateway:gw:api:a1,gateway:gw:api:a2,gateway:gw:api:*",
			)
			require.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{"gateway:gw"}, aud)
		})

		It("should handle mixed types", func() {
			aud, err := r.ExtractAudiences(ctx,
				"mcp:s1,gateway:gw1:api:*,mcp:s2,gateway:gw2:api:a1",
			)
			require.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{"mcp:s1", "gateway:gw1", "mcp:s2", "gateway:gw2"}, aud)
		})

		It("should handle URL format", func() {
			aud, err := r.ExtractAudiences(ctx,
				"https://bk.example.com/mcp-servers/my-server/sse",
			)
			require.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), []string{"mcp:my-server"}, aud)
		})
	})

	Describe("ResolveResourceDisplay", func() {
		It("should parse a single MCP resource", func() {
			display, err := r.ResolveResourceDisplay(ctx, "mcp:bk-cmdb-mcp-server")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups, 1)
			assert.Equal(GinkgoT(), "mcp", groups[0].Type)
			assert.Equal(GinkgoT(), "网关 MCP Server", groups[0].DisplayName)
			require.Len(GinkgoT(), groups[0].Items, 1)
			assert.Equal(GinkgoT(), "bk-cmdb-mcp-server", groups[0].Items[0].Name)
		})

		It("should parse multiple MCP resources into one group", func() {
			display, err := r.ResolveResourceDisplay(ctx, "mcp:s1,mcp:s2,mcp:s3")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups, 1)
			assert.Len(GinkgoT(), groups[0].Items, 3)
		})

		It("should parse gateway with wildcard API", func() {
			display, err := r.ResolveResourceDisplay(ctx, "gateway:bk-apigateway-a:api:*")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups, 1)
			assert.Equal(GinkgoT(), "gateway", groups[0].Type)
			require.Len(GinkgoT(), groups[0].Items, 1)
			item := groups[0].Items[0]
			assert.Equal(GinkgoT(), "bk-apigateway-a", item.Name)
			require.Len(GinkgoT(), item.Items, 1)
			assert.Equal(GinkgoT(), "*", item.Items[0].Name)
			assert.Equal(GinkgoT(), "所有 API", item.Items[0].DisplayName)
		})

		It("should parse gateway with specific APIs", func() {
			display, err := r.ResolveResourceDisplay(ctx, "gateway:gw1:api:get_host,gateway:gw1:api:get_app")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups, 1)
			require.Len(GinkgoT(), groups[0].Items, 1)
			item := groups[0].Items[0]
			assert.Equal(GinkgoT(), "gw1", item.Name)
			require.Len(GinkgoT(), item.Items, 2)
		})

		It("should let wildcard override specific APIs", func() {
			display, err := r.ResolveResourceDisplay(ctx,
				"gateway:gw1:api:get_host,gateway:gw1:api:*,gateway:gw1:api:get_app",
			)
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			item := groups[0].Items[0]
			require.Len(GinkgoT(), item.Items, 1)
			assert.Equal(GinkgoT(), "*", item.Items[0].Name)
		})

		It("should handle mixed MCP and gateway types", func() {
			display, err := r.ResolveResourceDisplay(ctx,
				"mcp:s1,gateway:gw1:api:*,mcp:s2,gateway:gw2:api:a1,gateway:gw2:api:a2",
			)
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups, 2)
			assert.Equal(GinkgoT(), "mcp", groups[0].Type)
			assert.Len(GinkgoT(), groups[0].Items, 2)
			assert.Equal(GinkgoT(), "gateway", groups[1].Type)
			assert.Len(GinkgoT(), groups[1].Items, 2)
		})

		It("should parse URL format", func() {
			display, err := r.ResolveResourceDisplay(ctx,
				"https://bk.example.com/mcp-servers/bk-cmdb-mcp-server/sse",
			)
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups, 1)
			assert.Equal(GinkgoT(), "bk-cmdb-mcp-server", groups[0].Items[0].Name)
		})

		It("should tolerate spaces and trailing commas", func() {
			display, err := r.ResolveResourceDisplay(ctx, " mcp:s1 , mcp:s2 , ")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			assert.Len(GinkgoT(), groups[0].Items, 2)
		})

		It("should dedup duplicate gateway APIs within same gateway", func() {
			display, err := r.ResolveResourceDisplay(ctx, "gateway:gw:api:a1,gateway:gw:api:a1")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			assert.Len(GinkgoT(), groups[0].Items[0].Items, 1)
		})

		It("should error on empty input", func() {
			_, err := r.ResolveResourceDisplay(ctx, "")
			assert.Error(GinkgoT(), err)
		})

		It("should error on empty name after prefix", func() {
			_, err := r.ResolveResourceDisplay(ctx, "mcp:")
			assert.Error(GinkgoT(), err)
		})
	})

	Describe("ResolveResourceDisplay with MCPServerClient", func() {
		var (
			ctl        *gomock.Controller
			mockClient *mock.MockMCPServerClient
			realm      *bluekingRealm
		)

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
			mockClient = mock.NewMockMCPServerClient(ctl)
			realm = &bluekingRealm{mcpServerClient: mockClient}
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("should use title from MCPServerClient as DisplayName", func() {
			mockClient.EXPECT().BatchQueryTitles(gomock.Any(), []string{"bk-cmdb"}).Return(
				map[string]string{"bk-cmdb": "CMDB MCP Server"}, nil,
			)

			display, err := realm.ResolveResourceDisplay(ctx, "mcp:bk-cmdb")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups, 1)
			assert.Equal(GinkgoT(), "bk-cmdb", groups[0].Items[0].Name)
			assert.Equal(GinkgoT(), "CMDB MCP Server", groups[0].Items[0].DisplayName)
		})

		It("should batch query multiple MCP names", func() {
			mockClient.EXPECT().BatchQueryTitles(gomock.Any(), []string{"s1", "s2"}).Return(
				map[string]string{"s1": "Server One", "s2": "Server Two"}, nil,
			)

			display, err := realm.ResolveResourceDisplay(ctx, "mcp:s1,mcp:s2")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups[0].Items, 2)
			assert.Equal(GinkgoT(), "Server One", groups[0].Items[0].DisplayName)
			assert.Equal(GinkgoT(), "Server Two", groups[0].Items[1].DisplayName)
		})

		It("should fall back to name when title not found for a specific MCP", func() {
			mockClient.EXPECT().BatchQueryTitles(gomock.Any(), []string{"s1", "s2"}).Return(
				map[string]string{"s1": "Server One"}, nil,
			)

			display, err := realm.ResolveResourceDisplay(ctx, "mcp:s1,mcp:s2")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			assert.Equal(GinkgoT(), "Server One", groups[0].Items[0].DisplayName)
			assert.Equal(GinkgoT(), "s2", groups[0].Items[1].DisplayName)
		})

		It("should fall back to name when BatchQueryTitles returns error", func() {
			mockClient.EXPECT().BatchQueryTitles(gomock.Any(), []string{"s1"}).Return(
				nil, errors.New("network error"),
			)

			display, err := realm.ResolveResourceDisplay(ctx, "mcp:s1")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			assert.Equal(GinkgoT(), "s1", groups[0].Items[0].DisplayName)
		})

		It("should dedup MCP names before querying", func() {
			mockClient.EXPECT().BatchQueryTitles(gomock.Any(), []string{"s1"}).Return(
				map[string]string{"s1": "Server One"}, nil,
			)

			display, err := realm.ResolveResourceDisplay(ctx, "mcp:s1,mcp:s1")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups[0].Items, 1)
			assert.Equal(GinkgoT(), "Server One", groups[0].Items[0].DisplayName)
		})

		It("should not call BatchQueryTitles when only gateway resources present", func() {
			display, err := realm.ResolveResourceDisplay(ctx, "gateway:gw:api:*")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups, 1)
			assert.Equal(GinkgoT(), "gateway", groups[0].Type)
		})

		It("should handle mixed MCP and gateway with title resolution", func() {
			mockClient.EXPECT().BatchQueryTitles(gomock.Any(), []string{"s1"}).Return(
				map[string]string{"s1": "Server One"}, nil,
			)

			display, err := realm.ResolveResourceDisplay(ctx, "mcp:s1,gateway:gw:api:get_host")
			require.NoError(GinkgoT(), err)
			groups := display.([]ResourceGroup)
			require.Len(GinkgoT(), groups, 2)
			assert.Equal(GinkgoT(), "Server One", groups[0].Items[0].DisplayName)
			assert.Equal(GinkgoT(), "gw", groups[1].Items[0].Name)
		})
	})
})
