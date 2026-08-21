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

	Describe("ResolveGrantableResource", func() {
		var (
			lookupCtl    *gomock.Controller
			mockLookup   *mock.MockLookupClient
			lookupRealm  *bluekingRealm
			lookupTenant string
		)

		BeforeEach(func() {
			lookupCtl = gomock.NewController(GinkgoT())
			mockLookup = mock.NewMockLookupClient(lookupCtl)
			lookupTenant = ""
			lookupRealm = &bluekingRealm{
				newLookupClient: func(tenantID string) bkapigateway.LookupClient {
					lookupTenant = tenantID
					return mockLookup
				},
			}
		})

		AfterEach(func() { lookupCtl.Finish() })

		refTo := func(resourceType string, names map[string]string) oauth.GrantableResourceRef {
			return oauth.GrantableResourceRef{Type: resourceType, Names: names}
		}

		It("should reject an unknown type before asking upstream", func() {
			_, err := lookupRealm.ResolveGrantableResource(ctx, "", refTo("nope", nil))

			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceType)
		})

		It("should reject a name keyed by the sibling type's level", func() {
			_, err := lookupRealm.ResolveGrantableResource(ctx, "",
				refTo("mcp", map[string]string{"gateway": "bk-log", "api": "query_log"}))
			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceLevel)

			_, err = lookupRealm.ResolveGrantableResource(ctx, "",
				refTo("api", map[string]string{"gateway": "bk-log", "mcp": "bk-log-prod-query"}))
			assert.ErrorIs(GinkgoT(), err, oauth.ErrUnknownGrantableResourceLevel)
		})

		It("should demand both levels of either type", func() {
			// A gateway on its own is the half-filled form, and for the API type it
			// would otherwise read as that gateway's wildcard -- the broadest grant
			// the type has, from a field the user had not finished.
			for _, ref := range []oauth.GrantableResourceRef{
				refTo("mcp", map[string]string{"gateway": "bk-log"}),
				refTo("mcp", map[string]string{"mcp": "bk-log-prod-query"}),
				refTo("api", map[string]string{"gateway": "bk-log"}),
				refTo("api", map[string]string{"api": "query_log"}),
			} {
				_, err := lookupRealm.ResolveGrantableResource(ctx, "", ref)
				assert.ErrorIs(GinkgoT(), err, oauth.ErrIncompleteGrantableResourceRef,
					"type %q with names %v", ref.Type, ref.Names)
			}
		})

		Context("an MCP server", func() {
			It("should resolve a private one, which no catalog page holds", func() {
				mockLookup.EXPECT().LookupMCPServer(gomock.Any(), []string{"bk-log-prod-query"}).Return(
					map[string]bkapigateway.MCPServer{
						"bk-log-prod-query": {
							Name: "bk-log-prod-query", Title: "日志查询", IsPublic: false,
							// The two switches are independent, and this path reads
							// the personal one alone: a server closed to OAuth
							// clients is still grantable to a personal token.
							OAuth2PersonalClientEnabled: true,
							OAuth2PublicClientEnabled:   false,
						},
					}, nil,
				)

				resource, err := lookupRealm.ResolveGrantableResource(ctx, "tencent",
					refTo("mcp", map[string]string{"gateway": "bk-log", "mcp": "bk-log-prod-query"}))
				require.NoError(GinkgoT(), err)

				assert.Equal(GinkgoT(), "tencent", lookupTenant,
					"the tenant scopes the client, so it is fixed once rather than sent per call")

				assert.Equal(GinkgoT(), "bk-log-prod-query", resource.Name)
				assert.Equal(GinkgoT(), "日志查询", resource.DisplayName)
				assert.Equal(GinkgoT(), "mcp:bk-log-prod-query", resource.Audience)
				assert.Equal(GinkgoT(), oauth.Extras{"is_public": false}, resource.Extras,
					"the form marks it private; the grant itself is no different for it")
			})

			It("should emit a token ValidateAudiences accepts", func() {
				mockLookup.EXPECT().LookupMCPServer(gomock.Any(), gomock.Any()).Return(
					map[string]bkapigateway.MCPServer{
						"bk-log-prod-query": {
							Name: "bk-log-prod-query", OAuth2PersonalClientEnabled: true,
						},
					}, nil,
				)

				resource, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("mcp", map[string]string{"gateway": "bk-log", "mcp": "bk-log-prod-query"}))
				require.NoError(GinkgoT(), err)

				assert.NoError(GinkgoT(), New().ValidateAudiences(ctx, []string{resource.Audience}))
			})

			It("should fall back to the name when upstream has no title", func() {
				mockLookup.EXPECT().LookupMCPServer(gomock.Any(), gomock.Any()).Return(
					map[string]bkapigateway.MCPServer{
						"bk-log-prod-query": {
							Name: "bk-log-prod-query", IsPublic: true,
							OAuth2PersonalClientEnabled: true,
						},
					}, nil,
				)

				resource, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("mcp", map[string]string{"gateway": "bk-log", "mcp": "bk-log-prod-query"}))
				require.NoError(GinkgoT(), err)

				assert.Equal(GinkgoT(), "bk-log-prod-query", resource.DisplayName)
			})

			It("should refuse one that is closed to personal tokens, though it exists", func() {
				// The catalog never offers it either: the search endpoints filter
				// on this field upstream. Naming it outright is what skips that
				// filter, so the check lands here.
				mockLookup.EXPECT().LookupMCPServer(gomock.Any(), []string{"bk-log-prod-query"}).Return(
					map[string]bkapigateway.MCPServer{
						"bk-log-prod-query": {
							Name: "bk-log-prod-query", Title: "日志查询", IsPublic: false,
							OAuth2PersonalClientEnabled: false,
						},
					}, nil,
				)

				_, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("mcp", map[string]string{"gateway": "bk-log", "mcp": "bk-log-prod-query"}))

				require.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotGrantable)
				assert.NotErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound,
					"the name is spelled right, so it must not read as one to go and check")
				assert.Contains(GinkgoT(), err.Error(), "bk-log-prod-query")
			})

			It("should report a name upstream does not know as not found", func() {
				mockLookup.EXPECT().LookupMCPServer(gomock.Any(), []string{"typo"}).Return(
					map[string]bkapigateway.MCPServer{}, nil,
				)

				_, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("mcp", map[string]string{"gateway": "bk-log", "mcp": "typo"}))

				require.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
				assert.Contains(GinkgoT(), err.Error(), "typo",
					"the user has to be told which name to go back and check")
			})

			It("should refuse a server that does not belong to the gateway named", func() {
				// The audience carries the server name alone, so a mistyped gateway
				// would otherwise hand back a grant on a stranger's server without
				// a word about it.
				mockLookup.EXPECT().LookupMCPServer(gomock.Any(), []string{"bk-cmdb-prod-query"}).Return(
					map[string]bkapigateway.MCPServer{
						"bk-cmdb-prod-query": {Name: "bk-cmdb-prod-query"},
					}, nil,
				)

				_, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("mcp", map[string]string{"gateway": "bk-log", "mcp": "bk-cmdb-prod-query"}))

				assert.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
			})

			It("should not take the gateway name itself for one of its servers", func() {
				// "<gateway>" is not "<gateway>-<stage>-...", and without the
				// separator the prefix test would accept it.
				mockLookup.EXPECT().LookupMCPServer(gomock.Any(), []string{"bk-log"}).Return(
					map[string]bkapigateway.MCPServer{"bk-log": {Name: "bk-log"}}, nil,
				)

				_, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("mcp", map[string]string{"gateway": "bk-log", "mcp": "bk-log"}))

				assert.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
			})

			It("should propagate an upstream failure rather than call it a miss", func() {
				// Confirming the server exists is the whole call, so an answer
				// nobody got must not read as "no such server".
				mockLookup.EXPECT().LookupMCPServer(gomock.Any(), gomock.Any()).Return(
					nil, errors.New("upstream down"),
				)

				_, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("mcp", map[string]string{"gateway": "bk-log", "mcp": "bk-log-prod-query"}))

				require.Error(GinkgoT(), err)
				assert.NotErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
			})
		})

		Context("a gateway API", func() {
			It("should resolve a private one and name its gateway in the audience", func() {
				mockLookup.EXPECT().LookupReleasedResource(
					gomock.Any(), "bk-log", []string{"query_log"},
				).Return(map[string]bkapigateway.Resource{
					"query_log": {
						Name: "query_log", Description: "查询日志", IsPublic: false,
						OAuth2PersonalClientEnabled: true,
					},
				}, nil)

				resource, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("api", map[string]string{"gateway": "bk-log", "api": "query_log"}))
				require.NoError(GinkgoT(), err)

				assert.Equal(GinkgoT(), "query_log", resource.Name,
					"the API row's own name, as the catalog names it")
				assert.Equal(GinkgoT(), "查询日志", resource.DisplayName)
				assert.Equal(GinkgoT(), "gateway:bk-log/api:query_log", resource.Audience)
				assert.Equal(GinkgoT(), oauth.Extras{"is_public": false}, resource.Extras)
				assert.NoError(GinkgoT(), New().ValidateAudiences(ctx, []string{resource.Audience}))
			})

			It("should report a gateway upstream 404s as not found", func() {
				mockLookup.EXPECT().LookupReleasedResource(gomock.Any(), "nope", gomock.Any()).Return(
					nil, &bkapigateway.APIError{StatusCode: 404, Code: "NOT_FOUND"},
				)

				_, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("api", map[string]string{"gateway": "nope", "api": "query_log"}))

				require.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
				assert.Contains(GinkgoT(), err.Error(), "nope")
			})

			It("should report an API the gateway has not released as not found", func() {
				mockLookup.EXPECT().LookupReleasedResource(gomock.Any(), "bk-log", []string{"typo"}).Return(
					map[string]bkapigateway.Resource{}, nil,
				)

				_, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("api", map[string]string{"gateway": "bk-log", "api": "typo"}))

				assert.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
			})

			It("should refuse one whose gateway is closed to personal tokens", func() {
				mockLookup.EXPECT().LookupReleasedResource(
					gomock.Any(), "bk-log", []string{"query_log"},
				).Return(map[string]bkapigateway.Resource{
					"query_log": {
						Name: "query_log", Description: "查询日志", IsPublic: true,
						OAuth2PersonalClientEnabled: false,
					},
				}, nil)

				_, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("api", map[string]string{"gateway": "bk-log", "api": "query_log"}))

				require.ErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotGrantable)
				assert.NotErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
				assert.Contains(GinkgoT(), err.Error(), "bk-log",
					"which gateway has to be opened is the actionable half")
			})

			It("should propagate a failure that is not a 404", func() {
				mockLookup.EXPECT().LookupReleasedResource(gomock.Any(), gomock.Any(), gomock.Any()).Return(
					nil, &bkapigateway.APIError{StatusCode: 500, Code: "INTERNAL"},
				)

				_, err := lookupRealm.ResolveGrantableResource(ctx, "",
					refTo("api", map[string]string{"gateway": "bk-log", "api": "query_log"}))

				require.Error(GinkgoT(), err)
				assert.NotErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound)
			})
		})

		It("should error rather than resolve when built without a lookup client", func() {
			_, err := (&bluekingRealm{}).ResolveGrantableResource(ctx, "",
				refTo("mcp", map[string]string{"gateway": "bk-log", "mcp": "bk-log-prod-query"}))

			require.Error(GinkgoT(), err)
			assert.NotErrorIs(GinkgoT(), err, oauth.ErrGrantableResourceNotFound,
				"nothing was looked up, so nothing may be reported as absent")
		})
	})
})
