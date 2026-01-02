package zen

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Bearer extracts and validates a Bearer token from the Authorization header.
// It returns the token string if present and properly formatted.
//
// If the header is missing, malformed, or contains an empty token,
// an appropriate error is returned with the BAD_REQUEST tag.
//
// Example:
//
//	token, err := zen.Bearer(sess)
//	if err != nil {
//	    return err
//	}
//	// Validate the token
func Bearer(s *Session) (string, error) {
	if s == nil {
		return "", fault.New("nil session", fault.Code(codes.Auth.Authentication.Missing.URN()),
			fault.Internal("session is nil"), fault.Public("Invalid session."))
	}

	if s.r == nil {
		return "", fault.New("nil request", fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("session request is nil"), fault.Public("Invalid request."))
	}

	header := s.r.Header.Get("Authorization")
	if header == "" {
		return "", fault.New("empty authorization header", fault.Code(codes.Auth.Authentication.Missing.URN()))
	}

	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "Bearer ") {
		return "", fault.New("invalid format", fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("missing bearer prefix"), fault.Public("Your authorization header is missing the 'Bearer ' prefix."))
	}

	bearer := strings.TrimPrefix(header, "Bearer ")
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return "", fault.New("invalid token", fault.Code(codes.Auth.Authentication.Malformed.URN()))
	}

	return bearer, nil
}
