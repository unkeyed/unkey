// Package canonicalpath reduces a request path to the form app frameworks
// route on, so gateway policies and per-path rate limits cannot be bypassed
// with path variants such as "/admin/", "/admin;x", or "/x/../admin".
//
// It lives outside package policies because the policy engine imports the
// ratelimit package, which also needs the canonical path.
package canonicalpath

import (
	"path"
	"strings"
)

// Clean returns the path as app frameworks route it, without a trailing slash.
// Case is preserved. ";" parameters are stripped before ".." is resolved
// because Tomcat resolves "/public/..;/admin" to "/admin".
func Clean(p string) string {
	if strings.HasPrefix(p, "/") && !strings.Contains(p, ";") {
		return path.Clean(p)
	}
	segments := strings.Split(p, "/")
	for i, segment := range segments {
		segments[i], _, _ = strings.Cut(segment, ";")
	}
	return path.Clean("/" + strings.Join(segments, "/"))
}

// Candidates returns the paths a policy path expression is tested against: the
// cleaned decoded path with and without a trailing slash. The raw path is never
// a candidate, so an earlier policy cannot claim a request through a variant
// the app routes elsewhere.
func Candidates(decoded string) []string {
	cleaned := Clean(decoded)
	if cleaned == "/" {
		return []string{cleaned}
	}
	return []string{cleaned, cleaned + "/"}
}

// HasEncodedSeparator reports whether an escaped path contains an encoded
// slash or backslash. Frameworks disagree on whether these split segments, so
// the gateway cannot know which handler the app will pick and rejects them.
func HasEncodedSeparator(escaped string) bool {
	for i := 0; i+2 < len(escaped); i++ {
		if escaped[i] != '%' {
			continue
		}
		switch strings.ToLower(escaped[i+1 : i+3]) {
		case "2f", "5c":
			return true
		}
	}
	return false
}
