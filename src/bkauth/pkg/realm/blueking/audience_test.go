package blueking

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/external/bkapigateway"
	"bkauth/pkg/external/bkapigateway/mock"
	"bkauth/pkg/oauth"
)

var _ = Describe("bluekingRealm audiences", func() {
	ctx := context.Background()

	Describe("ValidateAudiences", func() {
		It("should accept the five gateway-matched formats", func() {
			assert.NoError(GinkgoT(), New().ValidateAudiences(ctx, []string{
				"mcp:log-query",
				"mcp:*",
				"gateway:bk-log/api:query_log",
				"gateway:bk-log/api:*",
				"gateway:*/api:*",
			}))
		})

		It("should reject an empty list", func() {
			assert.Error(GinkgoT(), New().ValidateAudiences(ctx, nil))
		})

		It("should reject a resource URL, which is not a stored token", func() {
			assert.Error(GinkgoT(), New().ValidateAudiences(ctx,
				[]string{"https://bk.example.com/mcp-servers/s1/sse"}))
		})

		It("should reject a gateway token without an api segment", func() {
			assert.Error(GinkgoT(), New().ValidateAudiences(ctx, []string{"gateway:bk-log"}))
		})

		It("should reject a specific api across all gateways", func() {
			assert.Error(GinkgoT(), New().ValidateAudiences(ctx, []string{"gateway:*/api:query_log"}))
		})

		It("should reject empty names", func() {
			assert.Error(GinkgoT(), New().ValidateAudiences(ctx, []string{"mcp:"}))
			assert.Error(GinkgoT(), New().ValidateAudiences(ctx, []string{"gateway:/api:x"}))
			assert.Error(GinkgoT(), New().ValidateAudiences(ctx, []string{"gateway:gw/api:"}))
		})

		It("should reject an unknown prefix", func() {
			assert.Error(GinkgoT(), New().ValidateAudiences(ctx, []string{"service:codecc"}))
		})
	})

	Describe("ResolveAudienceDisplays", func() {
		var (
			ctl        *gomock.Controller
			mockClient *mock.MockLookupClient
			realm      *bluekingRealm
			// builtFor records the tenant the client was built with, which is
			// where the tenant now lands instead of on each call.
			builtFor string
		)

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
			mockClient = mock.NewMockLookupClient(ctl)
			builtFor = ""
			realm = &bluekingRealm{
				newLookupClient: func(tenantID string) bkapigateway.LookupClient {
					builtFor = tenantID
					return mockClient
				},
			}
		})

		AfterEach(func() { ctl.Finish() })

		// resolve reads the renderings back in the order the tokens were given,
		// the way a caller holding a grant list does. The order below is
		// therefore each spec's own input order, not a promise of the realm's,
		// and a token the realm did not render fails the spec here rather than
		// silently shifting the ones after it.
		resolve := func(tenantID string, audiences []string) ([]oauth.AudienceDisplay, error) {
			displays, err := realm.ResolveAudienceDisplays(ctx, tenantID, audiences)
			if err != nil {
				return nil, err
			}

			entries := make([]oauth.AudienceDisplay, 0, len(audiences))
			for _, aud := range audiences {
				display, ok := displays[aud]
				if !ok {
					return nil, fmt.Errorf("no display for %q", aud)
				}
				entries = append(entries, display)
			}
			return entries, nil
		}

		It("should return nothing for no tokens", func() {
			displays, err := realm.ResolveAudienceDisplays(ctx, "", nil)
			require.NoError(GinkgoT(), err)
			assert.Empty(GinkgoT(), displays)
		})

		It("should error on an invalid token", func() {
			_, err := resolve("", []string{"nope"})
			assert.Error(GinkgoT(), err)
		})

		It("should title MCP entries and mark the non-public ones", func() {
			mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"s1", "s2"}).Return(
				map[string]bkapigateway.MCPServer{
					"s1": {Name: "s1", Title: "Server One", IsPublic: true},
					"s2": {Name: "s2", Title: "Server Two", IsPublic: false},
				}, nil,
			)

			entries, err := resolve("tencent", []string{"mcp:s1", "mcp:s2"})
			require.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "tencent", builtFor,
				"the tenant scopes the client, so it is fixed once rather than sent per call")

			require.Len(GinkgoT(), entries, 2)
			assert.Equal(GinkgoT(), typeMCP, entries[0].Type)
			assert.Equal(GinkgoT(), "s1", entries[0].Name,
				"the bare identifier, same as the catalog entry's name")
			assert.Equal(GinkgoT(), "mcp:s1", entries[0].Audience,
				"the token, same as the catalog entry's audience")
			assert.Equal(GinkgoT(), "Server One", entries[0].DisplayName)
			assert.Equal(GinkgoT(), oauth.Extras{"is_public": true}, entries[0].Extras)
			assert.Equal(GinkgoT(), oauth.Extras{"is_public": false}, entries[1].Extras)
		})

		It("should only report levels its own catalog declares", func() {
			mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"s1"}).Return(
				map[string]bkapigateway.MCPServer{}, nil,
			)
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gw"}).Return(
				map[string]bkapigateway.Gateway{}, nil,
			)
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gw", []string{"a1"}).Return(
				map[string]bkapigateway.Resource{}, nil,
			)

			// The invariant a client relies on to label a grant: a level either
			// names one of the type's levels or is empty, standing for the box
			// above the catalog. Nothing else is resolvable.
			declared := map[string]map[string]bool{}
			for _, resourceType := range realm.GrantableResourceTypes() {
				declared[resourceType.Name] = map[string]bool{}
				for _, level := range resourceType.Levels {
					declared[resourceType.Name][level.Name] = true
				}
			}

			entries, err := resolve("", []string{
				"mcp:*", "mcp:s1", "gateway:*/api:*", "gateway:gw/api:*", "gateway:gw/api:a1",
			})
			require.NoError(GinkgoT(), err)

			require.Len(GinkgoT(), entries, 5)
			for _, entry := range entries {
				if entry.Level == oauth.AudienceLevelNone {
					continue
				}
				assert.True(GinkgoT(), declared[entry.Type][entry.Level],
					"level %q is not declared by type %q", entry.Level, entry.Type)
			}
		})

		It("should leave is_public out when the server was not returned", func() {
			mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"gone"}).Return(
				map[string]bkapigateway.MCPServer{}, nil,
			)

			entries, err := resolve("", []string{"mcp:gone"})
			require.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "gone", entries[0].DisplayName)
			assert.Nil(GinkgoT(), entries[0].Extras)
		})

		It("should fall back to names when the lookup fails", func() {
			mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"s1"}).Return(
				nil, errors.New("network error"),
			)

			entries, err := resolve("", []string{"mcp:s1"})
			require.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "s1", entries[0].DisplayName)
			assert.Nil(GinkgoT(), entries[0].Extras)
		})

		It("should render the MCP wildcard without a lookup", func() {
			entries, err := resolve("", []string{"mcp:*"})
			require.NoError(GinkgoT(), err)

			require.Len(GinkgoT(), entries, 1)
			assert.Equal(GinkgoT(), oauth.AudienceLevelNone, entries[0].Level,
				"picked above the catalog, so at no level of it")
			assert.Equal(GinkgoT(), wildcard, entries[0].Name)
			assert.Equal(GinkgoT(), "mcp:*", entries[0].Audience)
			assert.Equal(GinkgoT(), displayAllMCPServers, entries[0].DisplayName)
		})

		It("should describe gateway APIs and flag official gateways", func() {
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"bk-log"}).Return(
				map[string]bkapigateway.Gateway{"bk-log": {Name: "bk-log", IsOfficial: true}}, nil,
			)
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "bk-log", []string{"query_log"}).Return(
				map[string]bkapigateway.Resource{"query_log": {Name: "query_log", Description: "查询日志"}}, nil,
			)

			entries, err := resolve("", []string{"gateway:bk-log/api:query_log"})
			require.NoError(GinkgoT(), err)

			require.Len(GinkgoT(), entries, 1)
			assert.Equal(GinkgoT(), typeAPI, entries[0].Type)
			assert.Equal(GinkgoT(), levelAPI, entries[0].Level)
			assert.Equal(GinkgoT(), "query_log", entries[0].Name,
				"the API row it was picked from, named by the API rather than the gateway above it")
			assert.Equal(GinkgoT(), "gateway:bk-log/api:query_log", entries[0].Audience)
			assert.Equal(GinkgoT(), "查询日志", entries[0].DisplayName)
			assert.Equal(GinkgoT(), oauth.Extras{"is_official": true}, entries[0].Extras)
		})

		It("should repeat a gateway's is_official on every entry naming it", func() {
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gw"}).Return(
				map[string]bkapigateway.Gateway{"gw": {Name: "gw", IsOfficial: true}}, nil,
			)
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gw", []string{"a1", "a2"}).Return(
				map[string]bkapigateway.Resource{}, nil,
			)

			entries, err := resolve("", []string{"gateway:gw/api:a1", "gateway:gw/api:a2"})
			require.NoError(GinkgoT(), err)

			// An item-level entry has no gateway row of its own to hang the fact
			// on, so it rides along on each entry rather than being dropped.
			require.Len(GinkgoT(), entries, 2)
			assert.Equal(GinkgoT(), oauth.Extras{"is_official": true}, entries[0].Extras)
			assert.Equal(GinkgoT(), oauth.Extras{"is_official": true}, entries[1].Extras)
		})

		It("should not let a gateway wildcard hide the specific APIs beside it", func() {
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gw"}).Return(
				map[string]bkapigateway.Gateway{}, nil,
			)
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gw", []string{"a1"}).Return(
				map[string]bkapigateway.Resource{}, nil,
			)

			entries, err := resolve("", []string{"gateway:gw/api:*", "gateway:gw/api:a1"})
			require.NoError(GinkgoT(), err)

			require.Len(GinkgoT(), entries, 2)
			assert.Equal(GinkgoT(), "gateway:gw/api:*", entries[0].Audience)
			assert.Equal(GinkgoT(), levelGateway, entries[0].Level)
			assert.Equal(GinkgoT(), "gw", entries[0].DisplayName,
				"the gateway row said back, since that is the row this was picked from")
			assert.Equal(GinkgoT(), "gateway:gw/api:a1", entries[1].Audience)
			assert.Equal(GinkgoT(), levelAPI, entries[1].Level)
			assert.Equal(GinkgoT(), "a1", entries[1].DisplayName)
		})

		It("should tell the all-gateways grant apart from one gateway's all-APIs grant", func() {
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gw"}).Return(
				map[string]bkapigateway.Gateway{}, nil,
			)

			// The pair a bare "所有 API" on both would have merged: one gateway
			// against every gateway there will ever be. Naming the group-level
			// grant after its gateway separates them without a qualifier.
			entries, err := resolve("", []string{"gateway:*/api:*", "gateway:gw/api:*"})
			require.NoError(GinkgoT(), err)

			require.Len(GinkgoT(), entries, 2)
			assert.Equal(GinkgoT(), oauth.AudienceLevelNone, entries[0].Level)
			assert.Equal(GinkgoT(), wildcard, entries[0].Name)
			assert.Equal(GinkgoT(), displayAllAPIs, entries[0].DisplayName)
			assert.Equal(GinkgoT(), levelGateway, entries[1].Level)
			assert.Equal(GinkgoT(), "gw", entries[1].Name)
			assert.Equal(GinkgoT(), "gw", entries[1].DisplayName)
		})

		It("should render the global gateway wildcard without any lookup", func() {
			entries, err := resolve("", []string{"gateway:*/api:*"})
			require.NoError(GinkgoT(), err)

			require.Len(GinkgoT(), entries, 1)
			assert.Equal(GinkgoT(), "gateway:*/api:*", entries[0].Audience)
			assert.Equal(GinkgoT(), oauth.AudienceLevelNone, entries[0].Level)
			assert.Equal(GinkgoT(), displayAllAPIs, entries[0].DisplayName)
		})

		It("should keep one gateway's failure from costing the others their descriptions", func() {
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gone", "alive"}).Return(
				map[string]bkapigateway.Gateway{"alive": {Name: "alive"}}, nil,
			)
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gone", []string{"a1"}).Return(
				nil, &bkapigateway.APIError{StatusCode: 404, Code: "NOT_FOUND"},
			)
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "alive", []string{"a2"}).Return(
				map[string]bkapigateway.Resource{"a2": {Name: "a2", Description: "still here"}}, nil,
			)

			entries, err := resolve("", []string{"gateway:gone/api:a1", "gateway:alive/api:a2"})
			require.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), "a1", entries[0].DisplayName)
			assert.Equal(GinkgoT(), "still here", entries[1].DisplayName)
		})

		It("should key every kind of token by the token itself", func() {
			mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"s1"}).Return(
				map[string]bkapigateway.MCPServer{}, nil,
			)
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gw2", "gw1"}).Return(
				map[string]bkapigateway.Gateway{}, nil,
			)
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gw2", []string{"a1"}).Return(nil, nil)
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gw1", []string{"a2"}).Return(nil, nil)

			displays, err := realm.ResolveAudienceDisplays(ctx, "",
				[]string{"gateway:gw2/api:a1", "mcp:s1", "gateway:gw1/api:a2"})
			require.NoError(GinkgoT(), err)

			// The key is what lets a caller put an MCP grant back between two
			// gateway grants, which is where the user left it. Mixing the kinds
			// changes nothing about the keying.
			require.Len(GinkgoT(), displays, 3)
			for aud, display := range displays {
				assert.Equal(GinkgoT(), aud, display.Audience,
					"a rendering is keyed by the token it carries")
			}
			assert.Equal(GinkgoT(), typeMCP, displays["mcp:s1"].Type)
			assert.Equal(GinkgoT(), typeAPI, displays["gateway:gw1/api:a2"].Type)
		})

		It("should render a repeated token once and ask upstream once", func() {
			mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"s1"}).Return(
				map[string]bkapigateway.MCPServer{"s1": {Name: "s1", Title: "One", IsPublic: true}}, nil,
			)

			// A rendering depends on nothing but the token, so a token named
			// twice -- by two grant lists on a page, say -- collapses to the one
			// key its callers both read.
			displays, err := realm.ResolveAudienceDisplays(ctx, "", []string{"mcp:s1", "mcp:s1"})
			require.NoError(GinkgoT(), err)
			require.Len(GinkgoT(), displays, 1)
			assert.Equal(GinkgoT(), "One", displays["mcp:s1"].DisplayName)
		})

		// The tokens of a whole page arrive in one flat list, so what these specs
		// pin down is that a name repeated across it costs one upstream call.
		Context("across the tokens of a page", func() {
			It("should query the union of MCP names once", func() {
				mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"s1", "s2"}).Return(
					map[string]bkapigateway.MCPServer{
						"s1": {Name: "s1", Title: "One"},
						"s2": {Name: "s2", Title: "Two"},
					}, nil,
				)

				displays, err := realm.ResolveAudienceDisplays(ctx, "",
					[]string{"mcp:s1", "mcp:s1", "mcp:s2", "mcp:s2"})
				require.NoError(GinkgoT(), err)

				require.Len(GinkgoT(), displays, 2)
				assert.Equal(GinkgoT(), "One", displays["mcp:s1"].DisplayName)
				assert.Equal(GinkgoT(), "Two", displays["mcp:s2"].DisplayName)
			})

			It("should query each gateway once for the union of its api names", func() {
				mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gw1", "gw2"}).Return(
					map[string]bkapigateway.Gateway{"gw1": {Name: "gw1", IsOfficial: true}}, nil,
				)
				mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gw1", []string{"a1", "a2"}).Return(
					map[string]bkapigateway.Resource{
						"a1": {Name: "a1", Description: "first"},
						"a2": {Name: "a2", Description: "second"},
					}, nil,
				)
				mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gw2", []string{"b1"}).Return(
					map[string]bkapigateway.Resource{"b1": {Name: "b1", Description: "other"}}, nil,
				)

				displays, err := realm.ResolveAudienceDisplays(ctx, "", []string{
					"gateway:gw1/api:a1", "gateway:gw2/api:b1",
					"gateway:gw1/api:a2", "gateway:gw1/api:a1",
				})
				require.NoError(GinkgoT(), err)

				require.Len(GinkgoT(), displays, 3)
				assert.Equal(GinkgoT(), "first", displays["gateway:gw1/api:a1"].DisplayName)
				assert.Equal(GinkgoT(), "second", displays["gateway:gw1/api:a2"].DisplayName)
				assert.Equal(GinkgoT(), "other", displays["gateway:gw2/api:b1"].DisplayName)
				assert.Equal(GinkgoT(), oauth.Extras{"is_official": true},
					displays["gateway:gw1/api:a1"].Extras)
			})

			It("should not ask upstream about wildcards", func() {
				displays, err := realm.ResolveAudienceDisplays(ctx, "",
					[]string{"mcp:*", "gateway:*/api:*"})
				require.NoError(GinkgoT(), err)
				assert.Len(GinkgoT(), displays, 2)
			})

			It("should skip the resource lookup for a gateway that only has a wildcard", func() {
				mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gw1"}).Return(
					map[string]bkapigateway.Gateway{}, nil,
				)

				displays, err := realm.ResolveAudienceDisplays(ctx, "", []string{"gateway:gw1/api:*"})
				require.NoError(GinkgoT(), err)

				require.Len(GinkgoT(), displays, 1)
				assert.Equal(GinkgoT(), levelGateway, displays["gateway:gw1/api:*"].Level)
				assert.Equal(GinkgoT(), "gw1", displays["gateway:gw1/api:*"].DisplayName)
			})

			It("should reject the whole page when one token is malformed", func() {
				// Stored tokens are validated on write, so one the realm cannot
				// parse means the two disagree; rendering the rest as if the page
				// were sound would bury that.
				_, err := realm.ResolveAudienceDisplays(ctx, "",
					[]string{"mcp:s1", "gateway:broken"})
				assert.Error(GinkgoT(), err)
			})
		})
	})
})
