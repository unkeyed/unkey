package ratelimit

import (
	"net/http"
	"strings"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
)

// extractIdentifier derives the rate limit bucket key from the request.
// A RateLimit policy carries either a single identifier source or a compound
// list; both reduce to a list of parts here. Each part resolves to a string,
// unresolvable parts (missing header, no principal, bad principal path)
// become the shared "unknown" bucket value, and the parts join into one key.
// Returns an empty string only when nothing resolves at all -- no sources
// configured or every part unresolvable -- which the executor treats as a
// configuration error.
func extractIdentifier(
	sess *zen.Session,
	req *http.Request,
	cfg *frontlinev1.RateLimit,
	principal *principal.Principal,
) string {
	sources := cfg.GetIdentifiers()
	if len(sources) == 0 {
		// The single-source key is the raw value, unescaped, so policies
		// written before compound identifiers existed keep their exact
		// bucket keys.
		return extractIdentifierPart(sess, req, cfg.GetIdentifier(), principal)
	}
	if len(sources) == 1 {
		return extractIdentifierPart(sess, req, sources[0], principal)
	}

	var key strings.Builder
	// One upfront grow covers typical composed keys (subjects, paths,
	// header values) so the builder allocates once.
	key.Grow(96)
	resolved := false
	for i, source := range sources {
		if i > 0 {
			key.WriteByte(':')
		}
		part := extractIdentifierPart(sess, req, source, principal)
		if part == "" {
			// Missing dimensions share one bucket instead of failing the
			// request: an anonymous caller under [subject, path] is still
			// limited per path, in the "unknown" subject bucket.
			part = "unknown"
		} else {
			resolved = true
		}
		writeEscapedIdentifierPart(&key, part)
	}
	if !resolved {
		return ""
	}
	return key.String()
}

// writeEscapedIdentifierPart writes part into key, escaping the join
// separator so the composed key is unambiguous. Parts contain
// user-controlled bytes (header values, principal fields, paths), so without
// escaping the tuple ("a:b") would collide with ("a","b"). One pass; the
// common part contains no separator bytes and copies with a single write.
func writeEscapedIdentifierPart(key *strings.Builder, part string) {
	start := 0
	for i := 0; i < len(part); i++ {
		if c := part[i]; c == '\\' || c == ':' {
			key.WriteString(part[start:i])
			key.WriteByte('\\')
			key.WriteByte(c)
			start = i + 1
		}
	}
	key.WriteString(part[start:])
}

// extractIdentifierPart resolves one identifier source to its string value.
// Returns an empty string if the source cannot be resolved (nil config,
// missing header, no principal, etc).
//
// Supported identifier sources:
//   - RemoteIpKey:              client IP via sess.Location()
//   - HeaderKey:                value of the named request header
//   - PathKey:                  request URL path
//   - AuthenticatedSubjectKey:  subject from a principal set by a prior auth policy
//   - PrincipalFieldKey:        dotted-path field from a principal set by a prior auth policy
func extractIdentifierPart(
	sess *zen.Session,
	req *http.Request,
	identifier *frontlinev1.RateLimitIdentifier,
	principal *principal.Principal,
) string {
	if identifier == nil {
		return ""
	}

	switch src := identifier.GetSource().(type) {
	case *frontlinev1.RateLimitIdentifier_RemoteIp:
		return sess.Location()
	case *frontlinev1.RateLimitIdentifier_Header:
		return req.Header.Get(src.Header.GetName())
	case *frontlinev1.RateLimitIdentifier_Path:
		return req.URL.Path
	case *frontlinev1.RateLimitIdentifier_AuthenticatedSubject:
		if principal == nil {
			return ""
		}
		return principal.Subject
	case *frontlinev1.RateLimitIdentifier_PrincipalField:
		if principal == nil {
			return ""
		}
		return principal.ResolveField(src.PrincipalField.GetPath())
	default:
		return ""
	}
}
