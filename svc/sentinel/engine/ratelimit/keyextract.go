package ratelimit

import (
	"net/http"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/engine/principal"
)

// extractIdentifier derives the rate limit bucket key from the request based
// on the configured RateLimitIdentifier source. Returns an empty string if the
// identifier cannot be resolved (nil config, missing header, no principal, etc).
//
// Supported identifier sources:
//   - RemoteIpKey:              client IP via sess.Location()
//   - HeaderKey:                value of the named request header
//   - PathKey:                  request URL path
//   - AuthenticatedSubjectKey:  subject from a principal set by a prior auth policy
//   - PrincipalFieldKey:        dotted-path field from a principal set by a prior auth policy
func extractIdentifier(
	sess *zen.Session,
	req *http.Request,
	identifier *sentinelv1.RateLimitIdentifier,
	principal *principal.Principal,
) string {
	if identifier == nil {
		return ""
	}

	switch src := identifier.GetSource().(type) {
	case *sentinelv1.RateLimitIdentifier_RemoteIp:
		return sess.Location()
	case *sentinelv1.RateLimitIdentifier_Header:
		return req.Header.Get(src.Header.GetName())
	case *sentinelv1.RateLimitIdentifier_Path:
		return req.URL.Path
	case *sentinelv1.RateLimitIdentifier_AuthenticatedSubject:
		if principal == nil {
			return ""
		}
		return principal.Subject
	case *sentinelv1.RateLimitIdentifier_PrincipalField:
		if principal == nil {
			return ""
		}
		return principal.ResolveField(src.PrincipalField.GetPath())
	default:
		return ""
	}
}
