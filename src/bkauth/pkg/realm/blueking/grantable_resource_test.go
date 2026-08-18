package blueking

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/external/bkapigateway"
	"bkauth/pkg/external/bkapigateway/mock"
	"bkauth/pkg/oauth"
)

var _ = Describe("bluekingRealm grantable resources", func() {
	ctx := context.Background()

	var (
		ctl        *gomock.Controller
		mockClient *mock.MockSearchClient
		realm      *bluekingRealm
		// builtFor records the tenant the client was built with, which is where
		// the tenant now lands instead of on the query.
		builtFor string
	)

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())
		mockClient = mock.NewMockSearchClient(ctl)
		builtFor = ""
		realm = &bluekingRealm{
			newSearchClient: func(tenantID string) bkapigateway.SearchClient {
				builtFor = tenantID
				return mockClient
			},
		}
	})

	AfterEach(func() { ctl.Finish() })

	Describe("GrantableResourceTypes", func() {
		It("should offer a select-all token per type", func() {
			resourceTypes := realm.GrantableResourceTypes()
			require.Len(GinkgoT(), resourceTypes, 2)
			assert.Equal(GinkgoT(), "mcp", resourceTypes[0].Name)
			assert.Equal(GinkgoT(), "mcp:*", resourceTypes[0].Audience)
			assert.Equal(GinkgoT(), "api", resourceTypes[1].Name)
			assert.Equal(GinkgoT(), "gateway:*/api:*", resourceTypes[1].Audience,
				"the type is named after what it grants, the audience after its grammar")
		})

		It("should describe both catalogs as gateways over their members", func() {
			resourceTypes := realm.GrantableResourceTypes()

			assert.Equal(GinkgoT(), []oauth.GrantableResourceLevel{
				{Name: "gateway", DisplayName: "网关"},
				{Name: "mcp", DisplayName: "MCP"},
			}, resourceTypes[0].Levels,
				"the gateway level is listed though its rows are not grantable: "+
					"levels describe the tree, selectability is per row")

			assert.Equal(GinkgoT(), []oauth.GrantableResourceLevel{
				{Name: "gateway", DisplayName: "网关"},
				{Name: "api", DisplayName: "API"},
			}, resourceTypes[1].Levels)
		})

		It("should return tokens ValidateAudiences accepts", func() {
			for _, resourceType := range realm.GrantableResourceTypes() {
				assert.NoError(GinkgoT(), New().ValidateAudiences(ctx, []string{resourceType.Audience}))
			}
		})
	})

	Describe("ListGrantableResource", func() {
		It("should reject an unknown type", func() {
			_, err := realm.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "nope"})
			assert.Error(GinkgoT(), err)
		})

		It("should reject a keyword aimed at the sibling type's level", func() {
			// The two types share their outer level and differ in their inner
			// one, which makes the sibling's name the mistake most likely to be
			// sent. No upstream call is expected: the query is rejected before
			// one is made.
			_, err := realm.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{
				Type:     "mcp",
				Keywords: map[string]string{"api": "query"},
			})
			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceLevel)

			_, err = realm.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{
				Type:     "api",
				Keywords: map[string]string{"mcp": "log"},
			})
			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceLevel)
		})

		It("should accept every level it declares, so the two agree", func() {
			// Guards the pair that would otherwise drift: Levels is what the
			// frontend is told to send, and this is what accepts it.
			for _, resourceType := range realm.GrantableResourceTypes() {
				for _, level := range resourceType.Levels {
					mockClient.EXPECT().SearchMCPServer(gomock.Any(), gomock.Any()).
						Return(bkapigateway.MCPServerPage{}, nil).AnyTimes()
					mockClient.EXPECT().SearchResource(gomock.Any(), gomock.Any()).
						Return(bkapigateway.ResourcePage{}, nil).AnyTimes()

					_, err := realm.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{
						Type:     resourceType.Name,
						Keywords: map[string]string{level.Name: "x"},
					})
					assert.NoError(GinkgoT(), err, "type %q rejected its own level %q",
						resourceType.Name, level.Name)
				}
			}
		})

		It("should map MCP search hits onto unselectable gateway rows", func() {
			mockClient.EXPECT().SearchMCPServer(gomock.Any(), bkapigateway.MCPServerQuery{
				OAuthClientType: bkapigateway.OAuthClientTypePersonal,
				GatewayName:     "bk-",
				MCPServerName:   "query",
				Limit:           10,
				Offset:          20,
			}).Return(bkapigateway.MCPServerPage{
				Count: 7,
				Results: []bkapigateway.GatewayMCPServers{{
					Name:       "bk-log",
					IsOfficial: true,
					MCPServers: []bkapigateway.MCPServerItem{
						{Name: "log-query", Title: "日志查询"},
						{Name: "no-title"},
					},
				}},
			}, nil)

			page, err := realm.ListGrantableResource(ctx, "tencent", oauth.GrantableResourceQuery{
				Type:     "mcp",
				Keywords: map[string]string{"gateway": "bk-", "mcp": "query"},
				Limit:    10,
				Offset:   20,
			})
			require.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "tencent", builtFor,
				"the tenant scopes the client, so it is fixed once rather than sent per query")

			assert.Equal(GinkgoT(), 7, page.Count)
			require.Len(GinkgoT(), page.Results, 1)

			gateway := page.Results[0]
			assert.Equal(GinkgoT(), "bk-log", gateway.Name)
			assert.Equal(GinkgoT(), oauth.Extras{"is_official": true}, gateway.Extras)
			assert.Empty(GinkgoT(), gateway.Audience, "no token spans one gateway's MCP servers")

			require.Len(GinkgoT(), gateway.Items, 2)
			assert.Equal(GinkgoT(), "日志查询", gateway.Items[0].DisplayName)
			assert.Equal(GinkgoT(), "mcp:log-query", gateway.Items[0].Audience)
			assert.Equal(GinkgoT(), "no-title", gateway.Items[1].DisplayName)
		})

		It("should make the gateway row itself grantable for the API type", func() {
			mockClient.EXPECT().SearchResource(gomock.Any(), bkapigateway.ResourceQuery{
				OAuthClientType: bkapigateway.OAuthClientTypePersonal,
			}).Return(bkapigateway.ResourcePage{
				Count: 1,
				Results: []bkapigateway.GatewayResources{{
					Name: "bk-log",
					Resources: []bkapigateway.ResourceItem{
						{Name: "query_log", Description: "查询日志"},
						{Name: "no_desc"},
					},
				}},
			}, nil)

			page, err := realm.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "api"})
			require.NoError(GinkgoT(), err)

			gateway := page.Results[0]
			assert.Equal(GinkgoT(), "gateway:bk-log/api:*", gateway.Audience,
				"ticking the gateway grants every API of it, including ones released later")
			assert.Equal(GinkgoT(), oauth.Extras{"is_official": false}, gateway.Extras)

			// One row per grant: the token above appears nowhere below, so a
			// grant read back has exactly one row it could have come from.
			require.Len(GinkgoT(), gateway.Items, 2)
			assert.Equal(GinkgoT(), "查询日志", gateway.Items[0].DisplayName)
			assert.Equal(GinkgoT(), "gateway:bk-log/api:query_log", gateway.Items[0].Audience)
			assert.Equal(GinkgoT(), "no_desc", gateway.Items[1].DisplayName)
		})

		It("should emit tokens ValidateAudiences accepts", func() {
			mockClient.EXPECT().SearchResource(gomock.Any(), gomock.Any()).Return(
				bkapigateway.ResourcePage{
					Count: 1,
					Results: []bkapigateway.GatewayResources{{
						Name:      "bk-log",
						Resources: []bkapigateway.ResourceItem{{Name: "query_log"}},
					}},
				}, nil)

			page, err := realm.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "api"})
			require.NoError(GinkgoT(), err)

			gateway := page.Results[0]
			for _, row := range append([]oauth.GrantableResource{gateway}, gateway.Items...) {
				assert.NoError(GinkgoT(), New().ValidateAudiences(ctx, []string{row.Audience}),
					"every grantable row, the gateway one included")
			}
		})

		It("should propagate an upstream failure", func() {
			mockClient.EXPECT().SearchMCPServer(gomock.Any(), gomock.Any()).Return(
				bkapigateway.MCPServerPage{}, errors.New("upstream down"),
			)

			_, err := realm.ListGrantableResource(ctx, "", oauth.GrantableResourceQuery{Type: "mcp"})
			assert.Error(GinkgoT(), err)
		})
	})
})
