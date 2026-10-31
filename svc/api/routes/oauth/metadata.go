package oauth

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
)

// PathMetadata, PathAuthorize, and PathToken are the paths the authorization
// server endpoints are served at, relative to the issuer.
const (
	PathMetadata  = "/.well-known/oauth-authorization-server"
	PathAuthorize = "/oauth/authorize"
	PathToken     = "/oauth/token"
)

// Metadata is the RFC 8414 authorization server metadata document.
type Metadata struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	GrantTypesSupported                        []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported"`
	ClientIDMetadataDocumentSupported          bool     `json:"client_id_metadata_document_supported"`
	AuthorizationResponseIssParameterSupported bool     `json:"authorization_response_iss_parameter_supported"`
}

// MetadataHandler serves the authorization server metadata.
type MetadataHandler struct {
	Devices *clidevice.Service
}

// Method returns the HTTP method this route responds to.
func (h *MetadataHandler) Method() string { return http.MethodGet }

// Path returns the RFC 8414 well-known path for an issuer without a path.
func (h *MetadataHandler) Path() string { return PathMetadata }

// Handle writes the metadata document.
func (h *MetadataHandler) Handle(_ context.Context, s *zen.Session) error {
	s.SkipRequestLog()
	issuer := h.Devices.Issuer()
	s.AddHeader("Cache-Control", "public, max-age=3600")
	return s.JSON(http.StatusOK, Metadata{
		Issuer:                                     issuer,
		AuthorizationEndpoint:                      issuer + PathAuthorize,
		TokenEndpoint:                              issuer + PathToken,
		ResponseTypesSupported:                     []string{"code"},
		GrantTypesSupported:                        []string{grantAuthorizationCode, grantRefreshToken},
		CodeChallengeMethodsSupported:              []string{"S256"},
		TokenEndpointAuthMethodsSupported:          []string{"none"},
		ClientIDMetadataDocumentSupported:          true,
		AuthorizationResponseIssParameterSupported: true,
	})
}
