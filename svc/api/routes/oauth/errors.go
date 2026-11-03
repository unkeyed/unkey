package oauth

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
)

// ErrorResponse is an RFC 6749 section 5.2 error body.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// writeError answers with err when it is a [clidevice.OAuthError] and returns
// any other error for the API's error middleware.
func writeError(s *zen.Session, err error) error {
	oauthErr, ok := errors.AsType[*clidevice.OAuthError](err)
	if !ok {
		return err
	}
	return s.JSON(http.StatusBadRequest, ErrorResponse{
		Error:            string(oauthErr.Code),
		ErrorDescription: oauthErr.Description,
	})
}

// rejectRepeated enforces that a parameter appears at most once (RFC 6749
// section 3.1). resource may repeat (RFC 8707); the flow decides how many it
// accepts.
func rejectRepeated(params url.Values) error {
	for name, values := range params {
		if name != "resource" && len(values) > 1 {
			return &clidevice.OAuthError{
				Code:        clidevice.OAuthInvalidRequest,
				Description: name + " was sent more than once.",
			}
		}
	}
	return nil
}
