package oauth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
)

const registrationBodyBytesMax = 16 * 1024

// RegistrationResponse is an RFC 7591 section 3.2.1 client information
// response. It lists only the redirect URIs the server kept.
type RegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// RegisterHandler is the dynamic client registration endpoint.
type RegisterHandler struct {
	Devices *clidevice.Service
}

// Method returns the HTTP method this route responds to.
func (h *RegisterHandler) Method() string { return http.MethodPost }

// Path returns the URL path pattern this route matches.
func (h *RegisterHandler) Path() string { return PathRegister }

// Handle registers a public client and returns its signed client id.
func (h *RegisterHandler) Handle(ctx context.Context, s *zen.Session) error {
	s.SkipRequestLog()
	s.AddHeader("Cache-Control", "no-store")

	var registration clidevice.ClientRegistration
	if err := json.NewDecoder(io.LimitReader(s.Request().Body, registrationBodyBytesMax)).Decode(&registration); err != nil {
		return writeError(s, &clidevice.OAuthError{
			Code:        clidevice.OAuthInvalidClientMetadata,
			Description: "The request body must be a JSON client metadata object of at most 16 KiB.",
		})
	}

	client, err := h.Devices.Register(ctx, clidevice.RegisterRequest{
		Registration: registration,
		RemoteIP:     s.Location(),
	})
	if err != nil {
		return writeError(s, err)
	}
	return s.JSON(http.StatusCreated, RegistrationResponse{
		ClientID:                client.ID,
		ClientName:              client.Name,
		RedirectURIs:            client.RedirectURIs,
		GrantTypes:              []string{grantAuthorizationCode, grantRefreshToken},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	})
}
