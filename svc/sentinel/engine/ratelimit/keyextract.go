package ratelimit

import (
	"net/http"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/zen"
)

// extractIdentifier derives the rate limit bucket key from the request based
// on the configured RateLimitKey source. Returns an empty string if the key
// cannot be resolved (nil config, missing header, no principal, etc).
//
// Supported key sources:
//   - RemoteIpKey:              client IP via sess.Location()
//   - HeaderKey:                value of the named request header
//   - PathKey:                  request URL path
//   - AuthenticatedSubjectKey:  subject from a principal set by a prior auth policy
//   - PrincipalClaimKey:        named claim from a principal set by a prior auth policy
func extractIdentifier(
	sess *zen.Session,
	req *http.Request,
	key *sentinelv1.RateLimitKey,
	principal *sentinelv1.Principal,
) string {
	if key == nil {
		return ""
	}

	switch src := key.GetSource().(type) {
	case *sentinelv1.RateLimitKey_RemoteIp:
		return sess.Location()
	case *sentinelv1.RateLimitKey_Header:
		return req.Header.Get(src.Header.GetName())
	case *sentinelv1.RateLimitKey_Path:
		return req.URL.Path
	case *sentinelv1.RateLimitKey_AuthenticatedSubject:
		if principal == nil {
			return ""
		}
		return principal.GetSubject()
	case *sentinelv1.RateLimitKey_PrincipalClaim:
		if principal == nil {
			return ""
		}
		return principal.GetClaims()[src.PrincipalClaim.GetClaimName()]
	default:
		return ""
	}
}
