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

var _ = Describe("bluekingRealm ResolveGrantableAudiences", func() {
	ctx := context.Background()

	var (
		ctl        *gomock.Controller
		mockClient *mock.MockLookupClient
		realm      *bluekingRealm
		builtFor   string
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

	resolve := func(audiences ...string) []oauth.AudienceResolution {
		resolutions, err := realm.ResolveGrantableAudiences(ctx, "tencent", audiences)
		require.NoError(GinkgoT(), err)
		require.Len(GinkgoT(), resolutions, len(audiences), "one answer per token")
		for i, aud := range audiences {
			require.Equal(GinkgoT(), aud, resolutions[i].Audience, "answers come back in input order")
		}
		return resolutions
	}

	assertResolved := func(resolution oauth.AudienceResolution) oauth.AudienceDisplay {
		assert.Empty(GinkgoT(), resolution.Reason, "unexpected rejection: %s", resolution.Message)
		require.NotNil(GinkgoT(), resolution.Resource, "%q did not resolve", resolution.Audience)
		assert.Equal(GinkgoT(), resolution.Audience, resolution.Resource.Audience)
		return *resolution.Resource
	}

	assertRejected := func(resolution oauth.AudienceResolution, reason string) {
		assert.Equal(GinkgoT(), reason, resolution.Reason, "%q", resolution.Audience)
		assert.Nil(GinkgoT(), resolution.Resource)
		assert.NotEmpty(GinkgoT(), resolution.Message)
	}

	It("should answer nothing for no tokens", func() {
		resolutions, err := realm.ResolveGrantableAudiences(ctx, "", nil)
		require.NoError(GinkgoT(), err)
		assert.Empty(GinkgoT(), resolutions)
	})

	It("should reject malformed tokens without asking upstream", func() {
		resolutions := resolve("nope", "gateway:bk-log", "gateway:*/api:query_log", "mcp:")

		for _, resolution := range resolutions {
			assertRejected(resolution, oauth.AudienceRejectInvalid)
		}
	})

	It("should resolve the type-wide tokens without asking upstream", func() {
		resolutions := resolve("mcp:*", "gateway:*/api:*")

		assert.Equal(GinkgoT(), displayAllMCPServers, assertResolved(resolutions[0]).DisplayName)
		assert.Equal(GinkgoT(), displayAllAPIs, assertResolved(resolutions[1]).DisplayName)
	})

	Describe("MCP servers", func() {
		It("should resolve an open server, private ones included, as a stored grant renders", func() {
			mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"bk-log-prod-query", "bk-log-prod-hidden"}).
				Return(map[string]bkapigateway.MCPServer{
					"bk-log-prod-query": {
						Name: "bk-log-prod-query", Title: "日志查询", IsPublic: true,
						OAuth2PersonalClientEnabled: true,
					},
					"bk-log-prod-hidden": {
						Name: "bk-log-prod-hidden", IsPublic: false,
						OAuth2PersonalClientEnabled: true,
					},
				}, nil)

			resolutions := resolve("mcp:bk-log-prod-query", "mcp:bk-log-prod-hidden")

			assert.Equal(GinkgoT(), "tencent", builtFor)
			open := assertResolved(resolutions[0])
			assert.Equal(GinkgoT(), typeMCP, open.Type)
			assert.Equal(GinkgoT(), levelMCP, open.Level)
			assert.Equal(GinkgoT(), "bk-log-prod-query", open.Name)
			assert.Equal(GinkgoT(), "日志查询", open.DisplayName)
			assert.Equal(GinkgoT(), oauth.Extras{"is_public": true}, open.Extras)

			hidden := assertResolved(resolutions[1])
			assert.Equal(GinkgoT(), oauth.Extras{"is_public": false}, hidden.Extras,
				"private is no bar to granting; the form marks it instead")
		})

		It("should tell a missing server from one closed to personal tokens", func() {
			mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"gone", "closed"}).
				Return(map[string]bkapigateway.MCPServer{
					"closed": {Name: "closed", IsPublic: true, OAuth2PersonalClientEnabled: false},
				}, nil)

			resolutions := resolve("mcp:gone", "mcp:closed")

			assertRejected(resolutions[0], oauth.AudienceRejectNotFound)
			assert.Contains(GinkgoT(), resolutions[0].Message, `"gone"`)
			assertRejected(resolutions[1], oauth.AudienceRejectNotGrantable)
			assert.Contains(GinkgoT(), resolutions[1].Message, `"closed"`)
		})

		It("should mark every server unavailable when the lookup fails, and no other token", func() {
			mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"s1", "s2"}).
				Return(nil, errors.New("bkapigateway: request failed, status=502 code=BAD_GATEWAY"))

			resolutions := resolve("mcp:s1", "mcp:*", "mcp:s2")

			assertRejected(resolutions[0], oauth.AudienceRejectUnavailable)
			assertResolved(resolutions[1])
			assertRejected(resolutions[2], oauth.AudienceRejectUnavailable)
			assert.NotContains(GinkgoT(), resolutions[0].Message, "bkapigateway",
				"the upstream's own text carries request context the client is not shown")
		})
	})

	Describe("gateway-wide tokens", func() {
		It("should resolve a gateway that exists and reject one that does not", func() {
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"bk-log", "gone"}).
				Return(map[string]bkapigateway.Gateway{
					"bk-log": {Name: "bk-log", IsOfficial: true},
				}, nil)

			resolutions := resolve("gateway:bk-log/api:*", "gateway:gone/api:*")

			display := assertResolved(resolutions[0])
			assert.Equal(GinkgoT(), levelGateway, display.Level)
			assert.Equal(GinkgoT(), "bk-log", display.DisplayName)
			assert.Equal(GinkgoT(), oauth.Extras{"is_official": true}, display.Extras)

			assertRejected(resolutions[1], oauth.AudienceRejectNotFound)
			assert.Contains(GinkgoT(), resolutions[1].Message, `"gone"`)
		})

		It("should mark them unavailable when the gateway lookup fails, but not the single APIs", func() {
			// The gateway lookup only decorates a single API with is_official; its
			// existence is settled by the resource lookup, so losing the former
			// costs that API a label and nothing more.
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"bk-log"}).
				Return(nil, errors.New("timeout"))
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "bk-log", []string{"query_log"}).
				Return(map[string]bkapigateway.Resource{
					"query_log": {Name: "query_log", IsPublic: true, OAuth2PersonalClientEnabled: true},
				}, nil)

			resolutions := resolve("gateway:bk-log/api:*", "gateway:bk-log/api:query_log")

			assertRejected(resolutions[0], oauth.AudienceRejectUnavailable)
			display := assertResolved(resolutions[1])
			assert.Equal(GinkgoT(), oauth.Extras{"is_public": true}, display.Extras,
				"is_official is left out rather than guessed")
		})
	})

	Describe("single APIs", func() {
		It("should resolve an open API and reject missing and closed ones", func() {
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"bk-log"}).
				Return(map[string]bkapigateway.Gateway{"bk-log": {Name: "bk-log", IsOfficial: true}}, nil)
			mockClient.EXPECT().
				LookupReleasedResource(gomock.Any(), "bk-log", []string{"query_log", "gone", "closed"}).
				Return(map[string]bkapigateway.Resource{
					"query_log": {
						Name: "query_log", Description: "查询日志", IsPublic: false,
						OAuth2PersonalClientEnabled: true,
					},
					"closed": {Name: "closed", IsPublic: true, OAuth2PersonalClientEnabled: false},
				}, nil)

			resolutions := resolve(
				"gateway:bk-log/api:query_log", "gateway:bk-log/api:gone", "gateway:bk-log/api:closed")

			display := assertResolved(resolutions[0])
			assert.Equal(GinkgoT(), typeAPI, display.Type)
			assert.Equal(GinkgoT(), levelAPI, display.Level)
			assert.Equal(GinkgoT(), "query_log", display.Name)
			assert.Equal(GinkgoT(), "查询日志", display.DisplayName)
			assert.Equal(GinkgoT(), oauth.Extras{"is_official": true, "is_public": false}, display.Extras)

			assertRejected(resolutions[1], oauth.AudienceRejectNotFound)
			assert.Contains(GinkgoT(), resolutions[1].Message, `"gone"`)
			assertRejected(resolutions[2], oauth.AudienceRejectNotGrantable)
			assert.Contains(GinkgoT(), resolutions[2].Message, `"closed"`)
		})

		It("should keep one gateway's failure to that gateway's tokens", func() {
			mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gone", "down", "alive"}).
				Return(map[string]bkapigateway.Gateway{}, nil)
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gone", []string{"a1", "a2"}).
				Return(nil, &bkapigateway.APIError{StatusCode: 404, Code: "NOT_FOUND"})
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "down", []string{"b1"}).
				Return(nil, &bkapigateway.APIError{StatusCode: 500, Code: "INTERNAL"})
			mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "alive", []string{"c1"}).
				Return(map[string]bkapigateway.Resource{
					"c1": {Name: "c1", OAuth2PersonalClientEnabled: true},
				}, nil)

			resolutions := resolve(
				"gateway:gone/api:a1", "gateway:down/api:b1", "gateway:gone/api:a2", "gateway:alive/api:c1")

			// A 404 on the gateway path is an answer -- there is no such gateway
			// -- while a 500 is the absence of one.
			assertRejected(resolutions[0], oauth.AudienceRejectNotFound)
			assert.Contains(GinkgoT(), resolutions[0].Message, `"gone"`)
			assertRejected(resolutions[1], oauth.AudienceRejectUnavailable)
			assertRejected(resolutions[2], oauth.AudienceRejectNotFound)
			assertResolved(resolutions[3])
		})
	})

	It("should keep each token's answer in place across a mixed list", func() {
		mockClient.EXPECT().LookupMCPServer(gomock.Any(), []string{"s1"}).
			Return(map[string]bkapigateway.MCPServer{
				"s1": {Name: "s1", OAuth2PersonalClientEnabled: true},
			}, nil)
		mockClient.EXPECT().LookupGateway(gomock.Any(), []string{"gw"}).
			Return(map[string]bkapigateway.Gateway{"gw": {Name: "gw"}}, nil)
		mockClient.EXPECT().LookupReleasedResource(gomock.Any(), "gw", []string{"a1"}).
			Return(map[string]bkapigateway.Resource{}, nil)

		resolutions := resolve("gateway:gw/api:a1", "bad", "mcp:s1", "gateway:*/api:*", "gateway:gw/api:*")

		assertRejected(resolutions[0], oauth.AudienceRejectNotFound)
		assertRejected(resolutions[1], oauth.AudienceRejectInvalid)
		assertResolved(resolutions[2])
		assertResolved(resolutions[3])
		assertResolved(resolutions[4])
	})

	It("should error when the realm has no lookup client to ask with", func() {
		_, err := (&bluekingRealm{}).ResolveGrantableAudiences(ctx, "", []string{"mcp:s1"})
		assert.Error(GinkgoT(), err)
	})
})
