package zen

import (
	"crypto/subtle"
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

// StaticAuth extracts a Bearer token from the Authorization header and validates it against an expected value.
// It uses constant-time comparison to prevent timing attacks.
//
// If the header is missing, malformed, or the token doesn't match the expected value,
// an appropriate error is returned.
//
// Example:
//
//	err := zen.StaticAuth(sess, expectedToken)
//	if err != nil {
//	    return err
//	}
//	// Token is valid, proceed
func StaticAuth(s *Session, expectedToken string) error {
	token, err := Bearer(s)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
		return fault.New("invalid token",
			fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
			fault.Internal("token does not match expected value"),
			fault.Public("The provided token is invalid."))
	}

	return nil
}
