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
// slash or backslash, including multiply encoded forms such as "%252f".
// Frameworks disagree on whether these split segments, and an app that decodes
// twice turns "%252f" into "/", so the gateway cannot know which handler the
// app will pick and rejects them.
func HasEncodedSeparator(escaped string) bool {
	for i := 0; i < len(escaped); i++ {
		if escaped[i] != '%' {
			continue
		}
		j := i + 1
		for strings.HasPrefix(escaped[j:], "25") {
			j += 2
		}
		if j+2 > len(escaped) {
			continue
		}
		switch strings.ToLower(escaped[j : j+2]) {
		case "2f", "5c":
			return true
		}
	}
	return false
}
