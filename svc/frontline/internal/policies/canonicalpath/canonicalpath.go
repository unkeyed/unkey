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

// EncodedSeparators reports whether an escaped path contains an encoded slash
// or backslash, and whether any of them is encoded more than once, such as
// "%252f". Frameworks disagree on whether "%2f" splits a segment, and an app
// that decodes twice turns "%252f" into "/".
func EncodedSeparators(escaped string) (found, multiple bool) {
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
			found = true
			multiple = multiple || j > i+1
		}
	}
	return found, multiple
}

// HasDotSegment reports whether a decoded path has a "." or ".." segment,
// including one hidden behind ";" parameters such as "..;". ServeMux redirects
// plain literal dot segments, so any that reach the policy engine arrived
// encoded or hidden, and frameworks disagree on whether to resolve them.
func HasDotSegment(decoded string) bool {
	for segment := range strings.SplitSeq(decoded, "/") {
		segment, _, _ = strings.Cut(segment, ";")
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}
