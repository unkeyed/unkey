package grant

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

const (
	challengeLength      = 43
	verifierLengthMin    = 43
	verifierLengthMax    = 128
	challengeMethodS256  = "S256"
	challengeDigestBytes = sha256.Size
)

// Challenge is an S256 PKCE code challenge: the unpadded base64url SHA-256 of
// the client's verifier. "plain" and absent challenges are unrepresentable, so
// an OAuth request without PKCE cannot exist.
type Challenge struct {
	value string
}

// ParseChallenge accepts only method S256 and a base64url SHA-256 digest.
func ParseChallenge(method string, value string) (Challenge, error) {
	if method != challengeMethodS256 {
		return Challenge{}, invalidChallenge("code_challenge_method must be S256.")
	}
	if len(value) != challengeLength {
		return Challenge{}, invalidChallenge("code_challenge must be a base64url SHA-256 digest.")
	}
	digest, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(digest) != challengeDigestBytes {
		return Challenge{}, invalidChallenge("code_challenge must be a base64url SHA-256 digest.")
	}
	return Challenge{value: value}, nil
}

// Verify reports whether verifier is 43 to 128 unreserved characters (RFC 7636
// section 4.1) whose S256 digest is the challenge.
func (c Challenge) Verify(verifier string) bool {
	if len(verifier) < verifierLengthMin || len(verifier) > verifierLengthMax || !unreserved(verifier) {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	encoded := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(encoded), []byte(c.value)) == 1
}

func unreserved(value string) bool {
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-', r == '.', r == '_', r == '~':
		default:
			return false
		}
	}
	return true
}

func invalidChallenge(message string) error {
	return fault.New(message,
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public(message),
	)
}
