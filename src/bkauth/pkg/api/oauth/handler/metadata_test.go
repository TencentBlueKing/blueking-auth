package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bkauth/pkg/config"
	"bkauth/pkg/oauth"
)

var _ = Describe("renderMetadata", func() {
	var (
		w   *httptest.ResponseRecorder
		c   *gin.Context
		cfg *config.Config
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		cfg = &config.Config{
			BKAuthURL: "https://bkauth.example.com",
		}
	})

	parseBody := func() AuthorizationServerMetadata {
		var m AuthorizationServerMetadata
		err := json.Unmarshal(w.Body.Bytes(), &m)
		Expect(err).NotTo(HaveOccurred())
		return m
	}

	It("should return correct metadata fields", func() {
		cfg.OAuth.DCREnabled = false
		renderMetadata(c, cfg, "blueking")

		Expect(w.Code).To(Equal(http.StatusOK))

		m := parseBody()
		base := "https://bkauth.example.com"
		realmName := "blueking"

		Expect(m.Issuer).To(Equal(oauth.IssuerURL(base, realmName)))
		Expect(m.AuthorizationEndpoint).To(Equal(oauth.AuthorizationEndpointURL(base, realmName)))
		Expect(m.TokenEndpoint).To(Equal(oauth.TokenEndpointURL(base, realmName)))
		Expect(m.DeviceAuthorizationEndpoint).To(Equal(oauth.DeviceAuthorizationEndpointURL(base, realmName)))
		Expect(m.IntrospectionEndpoint).To(Equal(oauth.IntrospectionEndpointURL(base, realmName)))
		Expect(m.RevocationEndpoint).To(Equal(oauth.RevocationEndpointURL(base, realmName)))

		Expect(m.ResponseTypesSupported).To(Equal([]string{oauth.ResponseTypeCode}))
		Expect(m.ResponseModesSupported).To(Equal([]string{oauth.ResponseModeQuery}))
		Expect(m.GrantTypesSupported).To(Equal([]string{
			oauth.GrantTypeAuthorizationCode,
			oauth.GrantTypeRefreshToken,
		}))
		Expect(m.CodeChallengeMethodsSupported).To(Equal([]string{oauth.CodeChallengeMethodS256}))
		Expect(m.TokenEndpointAuthMethodsSupported).To(Equal([]string{
			oauth.AuthMethodNone,
			oauth.AuthMethodClientSecretBasic,
			oauth.AuthMethodClientSecretPost,
		}))

		Expect(m.RegistrationEndpoint).To(BeEmpty())
	})

	It("should include registration endpoint when DCR is enabled", func() {
		cfg.OAuth.DCREnabled = true
		renderMetadata(c, cfg, "blueking")

		Expect(w.Code).To(Equal(http.StatusOK))

		m := parseBody()
		Expect(m.RegistrationEndpoint).To(Equal(
			oauth.RegistrationEndpointURL("https://bkauth.example.com", "blueking"),
		))
	})

	It("should build URLs for a different realm", func() {
		renderMetadata(c, cfg, "devops")

		m := parseBody()
		Expect(m.Issuer).To(Equal(oauth.IssuerURL("https://bkauth.example.com", "devops")))
		Expect(m.TokenEndpoint).To(Equal(oauth.TokenEndpointURL("https://bkauth.example.com", "devops")))
	})
})
