package zen

import (
	"strings"

	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
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

	header := s.r.Header.Get("Authorization")
	if header == "" {
		return "", fault.New("empty authorization header", fault.WithCode(codes.Auth.Authentication.Missing.URN()))
	}

	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "Bearer ") {
		return "", fault.New("invalid format", fault.WithCode(codes.Auth.Authentication.Malformed.URN()),
			fault.WithDesc("missing bearer prefix", "Your authorization header is missing the 'Bearer ' prefix."))
	}

	bearer := strings.TrimPrefix(header, "Bearer ")
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return "", fault.New("invalid token", fault.WithCode(codes.Auth.Authentication.Malformed.URN()))
	}

	return bearer, nil

}
